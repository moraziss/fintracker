package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/morazss/fintracker/internal/auth"

	"github.com/morazss/fintracker/internal/user"
)

func main() {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("unable to create connection pool: %v", err)
	}
	defer pool.Close()

	tokenIssuer := auth.NewTokenIssuer([]byte(os.Getenv("JWT_SECRET")))

	userRepo := user.NewPostgresRepository(pool)
	userService := user.NewService(userRepo)
	userHandler := user.NewHandler(userService)

	refreshTokenRepo := auth.NewPostgresRefreshTokenRepository(pool)

	authService := auth.NewService(userRepo, tokenIssuer, refreshTokenRepo)
	authHandler := auth.NewHandler(authService)

	publicMux := http.NewServeMux()
	publicMux.HandleFunc("POST /auth/register", userHandler.Register)
	publicMux.HandleFunc("POST /auth/login", authHandler.Login)
	publicMux.HandleFunc("POST /auth/refresh", authHandler.Refresh)

	protectedMux := http.NewServeMux()
	// с Недели 3 сюда просто добавляются новые маршруты — уже под защитой

	rootMux := http.NewServeMux()
	rootMux.Handle("/auth/", publicMux)
	rootMux.Handle("/", auth.RequireAuth(tokenIssuer)(protectedMux))

	log.Println("listening on :8080")
	if err := http.ListenAndServe(":8080", rootMux); err != nil {
		log.Fatal(err)
	}
}
