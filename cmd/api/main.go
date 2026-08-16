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

	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/register", userHandler.Register)
	mux.HandleFunc("POST /auth/login", authHandler.Login)
	mux.HandleFunc("POST /auth/refresh", authHandler.Refresh)

	log.Println("listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("failed to start HTTP server: %v", err)
	}

}
