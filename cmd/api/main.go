package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	pgxdecimal "github.com/jackc/pgx-shopspring-decimal"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/morazss/fintracker/internal/analytics"

	"github.com/morazss/fintracker/internal/account"
	"github.com/morazss/fintracker/internal/auth"
	"github.com/morazss/fintracker/internal/transaction"
	"github.com/morazss/fintracker/internal/user"
)

func main() {
	if err := run(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	poolConfig, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		return fmt.Errorf("invalid DATABASE_URL: %w", err)
	}
	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		pgxdecimal.Register(conn.TypeMap())
		return nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return fmt.Errorf("unable to create connection pool: %w", err)
	}
	defer pool.Close()

	tokenIssuer := auth.NewTokenIssuer([]byte(os.Getenv("JWT_SECRET")))

	userRepo := user.NewPostgresRepository(pool)
	userService := user.NewService(userRepo)
	userHandler := user.NewHandler(userService)

	refreshTokenRepo := auth.NewPostgresRefreshTokenRepository(pool)
	authService := auth.NewService(userRepo, tokenIssuer, refreshTokenRepo, pool)
	authHandler := auth.NewHandler(authService)

	accountRepo := account.NewPostgresRepository(pool)
	transactionRepo := transaction.NewPostgresRepository(pool)
	transactionService := transaction.NewService(transactionRepo, accountRepo, pool)
	transactionHandler := transaction.NewHandler(transactionService)

	analyticsRepo := analytics.NewPostgresRepository(pool)
	analyticsService := analytics.NewService(analyticsRepo)
	analyticsHandler := analytics.NewHandler(analyticsService)

	publicMux := http.NewServeMux()
	publicMux.HandleFunc("POST /auth/register", userHandler.Register)
	publicMux.HandleFunc("POST /auth/login", authHandler.Login)
	publicMux.HandleFunc("POST /auth/refresh", authHandler.Refresh)

	protectedMux := http.NewServeMux()
	protectedMux.HandleFunc("POST /transactions", transactionHandler.Create)
	protectedMux.HandleFunc("GET /transactions", transactionHandler.List)
	protectedMux.HandleFunc("DELETE /transactions/{id}", transactionHandler.Delete)
	protectedMux.HandleFunc("PUT /transactions/{id}", transactionHandler.Update)
	protectedMux.HandleFunc("GET /analytics/summary", analyticsHandler.Summary)
	protectedMux.HandleFunc("GET /analytics/by-category", analyticsHandler.ByCategory)
	protectedMux.HandleFunc("GET /analytics/trend", analyticsHandler.Trend)

	rootMux := http.NewServeMux()
	rootMux.Handle("/auth/", publicMux)
	rootMux.Handle("/", auth.RequireAuth(tokenIssuer)(protectedMux))

	srv := &http.Server{
		Addr:    ":8080",
		Handler: rootMux,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Println("listening on :8080")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	notifyCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	case <-notifyCtx.Done():
		log.Println("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	log.Println("server stopped gracefully")
	return nil
}
