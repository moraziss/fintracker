package auth

import (
	"net/http"
	"strings"
)

func RequireAuth(tokens *TokenIssuer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			const prefix = "Bearer "

			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, prefix) {
				http.Error(w, "missing or malformed Authorization header", http.StatusUnauthorized)
				return
			}

			claims, err := tokens.ParseAccessToken(strings.TrimPrefix(authHeader, prefix))
			if err != nil {
				http.Error(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}

			ctx := contextWithUserID(r.Context(), claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
