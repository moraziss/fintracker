package analytics

import (
	"context"

	"github.com/morazss/fintracker/internal/db"
)

type Repository interface {
	Summary(ctx context.Context, userID int64, f Filter) (*Summary, error)
}

type PostgresRepository struct {
	q db.Querier
}

func NewPostgresRepository(q db.Querier) *PostgresRepository {
	return &PostgresRepository{q: q}
}
