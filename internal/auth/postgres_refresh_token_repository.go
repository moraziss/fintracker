package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/morazss/fintracker/internal/db"
)

type PostgresRefreshTokenRepository struct {
	q db.Querier
}

func NewPostgresRefreshTokenRepository(q db.Querier) *PostgresRefreshTokenRepository {
	return &PostgresRefreshTokenRepository{q: q}
}

// WithTx — тот же конструктор, другой Querier. Ни Create, ни GetByHash, ни Revoke
// не меняются ни строкой — они уже написаны против интерфейса, не против *pgxpool.Pool.
func (r *PostgresRefreshTokenRepository) WithTx(q db.Querier) RefreshTokenRepository {
	return &PostgresRefreshTokenRepository{q: q}
}

func (r *PostgresRefreshTokenRepository) Create(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) (*RefreshToken, error) {
	const query = `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, token_hash, expires_at, created_at, revoked_at
	`
	var t RefreshToken
	err := r.q.QueryRow(ctx, query, userID, tokenHash, expiresAt).Scan(
		&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.CreatedAt, &t.RevokedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *PostgresRefreshTokenRepository) GetByHash(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	const query = `
		SELECT id, user_id, token_hash, expires_at, created_at, revoked_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`
	var t RefreshToken
	err := r.q.QueryRow(ctx, query, tokenHash).Scan(
		&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.CreatedAt, &t.RevokedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTokenNotFound
		}
		return nil, err
	}
	return &t, nil
}

func (r *PostgresRefreshTokenRepository) Revoke(ctx context.Context, id int64) error {
	const query = `UPDATE refresh_tokens SET revoked_at = now() WHERE id = $1`
	_, err := r.q.Exec(ctx, query, id)
	return err
}
