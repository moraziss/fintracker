package account

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/moraziss/fintracker/internal/db"
)

type PostgresRepository struct {
	q db.Querier
}

func NewPostgresRepository(q db.Querier) *PostgresRepository {
	return &PostgresRepository{q: q}
}

func (r *PostgresRepository) WithTx(q db.Querier) Repository {
	return &PostgresRepository{q: q}
}

func (r *PostgresRepository) ApplyBalanceDelta(ctx context.Context, accountID, userID int64, delta decimal.Decimal) error {
	const query = `
		UPDATE accounts
		SET balance = balance + $1
		WHERE id = $2 AND user_id = $3
	`
	tag, err := r.q.Exec(ctx, query, delta, accountID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
