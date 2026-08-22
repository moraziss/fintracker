package auth

import (
	"context"
	"time"

	"github.com/moraziss/fintracker/internal/db"
)

type RefreshTokenRepository interface {
	Create(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) (*RefreshToken, error)
	GetByHash(ctx context.Context, tokenHash string) (*RefreshToken, error)
	Revoke(ctx context.Context, id int64) error
	WithTx(q db.Querier) RefreshTokenRepository
}
