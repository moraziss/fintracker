package analytics

import (
	"context"

	"github.com/morazss/fintracker/internal/db"
)

type Repository interface {
	Summary(ctx context.Context, userID int64, f Filter) (*Summary, error)
	ByCategory(ctx context.Context, userID int64, f Filter) ([]*CategoryTotal, error)
	Trend(ctx context.Context, userID int64, f Filter) ([]*DailyTrend, error)
}

type PostgresRepository struct {
	q db.Querier
}

func NewPostgresRepository(q db.Querier) *PostgresRepository {
	return &PostgresRepository{q: q}
}
