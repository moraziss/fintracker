package auth

import (
	"context"
	"errors"
	"time"

	"github.com/morazss/fintracker/internal/user"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	users         user.Repository
	tokens        *TokenIssuer
	refreshTokens RefreshTokenRepository
}

func NewService(users user.Repository, tokens *TokenIssuer, refreshTokens RefreshTokenRepository) *Service {
	return &Service{users: users, tokens: tokens, refreshTokens: refreshTokens}
}

func (s *Service) Login(ctx context.Context, email, password string) (*TokenPair, error) {
	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return s.issueTokenPair(ctx, u.ID)
}

func (s *Service) Refresh(ctx context.Context, rawRefreshToken string) (*TokenPair, error) {
	hash := hashRefreshToken(rawRefreshToken)

	stored, err := s.refreshTokens.GetByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, ErrTokenNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if stored.RevokedAt != nil || time.Now().After(stored.ExpiresAt) {
		return nil, ErrInvalidCredentials
	}

	if err := s.refreshTokens.Revoke(ctx, stored.ID); err != nil {
		return nil, err
	}

	return s.issueTokenPair(ctx, stored.UserID)
}

func (s *Service) issueTokenPair(ctx context.Context, userID int64) (*TokenPair, error) {
	accessToken, err := s.tokens.GenerateAccessToken(userID)
	if err != nil {
		return nil, err
	}

	rawRefresh, err := generateRefreshToken()
	if err != nil {
		return nil, err
	}

	_, err = s.refreshTokens.Create(ctx, userID, hashRefreshToken(rawRefresh), time.Now().Add(RefreshTokenTTL))
	if err != nil {
		return nil, err
	}

	return &TokenPair{AccessToken: accessToken, RefreshToken: rawRefresh}, nil
}
