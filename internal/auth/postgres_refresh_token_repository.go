package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRefreshTokenRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRefreshTokenRepository(pool *pgxpool.Pool) *PostgresRefreshTokenRepository {
	return &PostgresRefreshTokenRepository{pool: pool}
}

func (r *PostgresRefreshTokenRepository) Create(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) (*RefreshToken, error) {
	const query = `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, token_hash, expires_at, created_at, revoked_at
	`
	var t RefreshToken
	err := r.pool.QueryRow(ctx, query, userID, tokenHash, expiresAt).Scan(
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
	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(
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
	_, err := r.pool.Exec(ctx, query, id)
	return err
}
