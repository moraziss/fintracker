package main

import (
	"context"
	"log"
	"net/http"
	"os"

	pgxdecimal "github.com/jackc/pgx-shopspring-decimal"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/morazss/fintracker/internal/account"
	"github.com/morazss/fintracker/internal/auth"
	"github.com/morazss/fintracker/internal/transaction"
	"github.com/morazss/fintracker/internal/user"
)

func main() {
	ctx := context.Background()

	poolConfig, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("invalid DATABASE_URL: %v", err)
	}
	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		pgxdecimal.Register(conn.TypeMap())
		return nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		log.Fatalf("unable to create connection pool: %v", err)
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

	publicMux := http.NewServeMux()
	publicMux.HandleFunc("POST /auth/register", userHandler.Register)
	publicMux.HandleFunc("POST /auth/login", authHandler.Login)
	publicMux.HandleFunc("POST /auth/refresh", authHandler.Refresh)

	protectedMux := http.NewServeMux()
	protectedMux.HandleFunc("POST /transactions", transactionHandler.Create)
	protectedMux.HandleFunc("GET /transactions", transactionHandler.List)
	protectedMux.HandleFunc("DELETE /transactions/{id}", transactionHandler.Delete)

	rootMux := http.NewServeMux()
	rootMux.Handle("/auth/", publicMux)
	rootMux.Handle("/", auth.RequireAuth(tokenIssuer)(protectedMux))

	log.Println("listening on :8080")
	if err := http.ListenAndServe(":8080", rootMux); err != nil {
		log.Fatal(err)
	}
}
