package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const AccessTokenTTL = 15 * time.Minute

type Claims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

type TokenIssuer struct {
	secret []byte
}

func NewTokenIssuer(secret []byte) *TokenIssuer {
	return &TokenIssuer{secret: secret}
}

func (i *TokenIssuer) GenerateAccessToken(userID int64) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(AccessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(i.secret)
}
