package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/moraziss/fintracker/internal/db"
	"github.com/moraziss/fintracker/internal/user"
)

type Service struct {
	users         user.Repository
	tokens        *TokenIssuer
	refreshTokens RefreshTokenRepository
	beginner      db.Beginner
}

func NewService(users user.Repository, tokens *TokenIssuer, refreshTokens RefreshTokenRepository, beginner db.Beginner) *Service {
	return &Service{users: users, tokens: tokens, refreshTokens: refreshTokens, beginner: beginner}
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

	return s.issueTokenPair(ctx, s.refreshTokens, u.ID)
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

	// Revoke старого + Create нового — одна DB-транзакция: либо оба эффекта
	// применяются, либо ни один. pgx.BeginFunc сам коммитит при nil и
	// откатывает при любой ошибке из fn.
	var pair *TokenPair
	err = pgx.BeginFunc(ctx, s.beginner, func(tx pgx.Tx) error {
		txRepo := s.refreshTokens.WithTx(tx)

		if err := txRepo.Revoke(ctx, stored.ID); err != nil {
			return err
		}

		newPair, err := s.issueTokenPair(ctx, txRepo, stored.UserID)
		if err != nil {
			return err
		}
		pair = newPair
		return nil
	})
	if err != nil {
		return nil, err
	}

	return pair, nil
}

// issueTokenPair теперь принимает репозиторий параметром, а не берёт s.refreshTokens
// напрямую: Login вызывает его вне транзакции (там нечего откатывать — только один
// Create), Refresh — с tx-scoped репозиторием, чтобы Create попал в ту же транзакцию,
// что и Revoke чуть выше.
func (s *Service) issueTokenPair(ctx context.Context, refreshTokens RefreshTokenRepository, userID int64) (*TokenPair, error) {
	accessToken, err := s.tokens.GenerateAccessToken(userID)
	if err != nil {
		return nil, err
	}

	rawRefresh, err := generateRefreshToken()
	if err != nil {
		return nil, err
	}

	_, err = refreshTokens.Create(ctx, userID, hashRefreshToken(rawRefresh), time.Now().Add(RefreshTokenTTL))
	if err != nil {
		return nil, err
	}

	return &TokenPair{AccessToken: accessToken, RefreshToken: rawRefresh}, nil
}
