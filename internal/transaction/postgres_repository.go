package transaction

import (
	"context"

	"github.com/morazss/fintracker/internal/db"
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

func (r *PostgresRepository) Create(ctx context.Context, t *Transaction) (*Transaction, error) {
	const query = `
		INSERT INTO transactions (account_id, category_id, type, amount, description, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, account_id, category_id, type, amount, description, occurred_at, created_at, deleted_at
	`
	var created Transaction
	err := r.q.QueryRow(ctx, query,
		t.AccountID, t.CategoryID, t.Type, t.Amount, t.Description, t.OccurredAt,
	).Scan(
		&created.ID, &created.AccountID, &created.CategoryID, &created.Type,
		&created.Amount, &created.Description, &created.OccurredAt,
		&created.CreatedAt, &created.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return &created, nil
}
