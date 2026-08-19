package transaction

import (
	"context"
	"fmt"
	"strings"

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

func (r *PostgresRepository) List(ctx context.Context, userID int64, f ListFilter) ([]*Transaction, error) {
	var sb strings.Builder
	sb.WriteString(`
		SELECT t.id, t.account_id, t.category_id, t.type, t.amount, t.description, t.occurred_at, t.created_at, t.deleted_at
		FROM transactions t
		JOIN accounts a ON a.id = t.account_id
		WHERE a.user_id = $1 AND t.deleted_at IS NULL`)
	args := []any{userID}

	add := func(clause string, value any) {
		args = append(args, value)
		fmt.Fprintf(&sb, " AND %s $%d", clause, len(args))
	}

	if f.AccountID != nil {
		add("t.account_id =", *f.AccountID)
	}
	if f.CategoryID != nil {
		add("t.category_id =", *f.CategoryID)
	}
	if f.Type != nil {
		add("t.type =", *f.Type)
	}
	if f.From != nil {
		add("t.occurred_at >=", *f.From)
	}
	if f.To != nil {
		add("t.occurred_at <=", *f.To)
	}
	if f.Cursor != nil {
		args = append(args, f.Cursor.OccurredAt, f.Cursor.ID)
		fmt.Fprintf(&sb, " AND (t.occurred_at, t.id) < ($%d, $%d)", len(args)-1, len(args))
	}

	sb.WriteString(" ORDER BY t.occurred_at DESC, t.id DESC")
	args = append(args, f.Limit)
	fmt.Fprintf(&sb, " LIMIT $%d", len(args))

	rows, err := r.q.Query(ctx, sb.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*Transaction
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(
			&t.ID, &t.AccountID, &t.CategoryID, &t.Type,
			&t.Amount, &t.Description, &t.OccurredAt,
			&t.CreatedAt, &t.DeletedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, &t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *PostgresRepository) SoftDelete(ctx context.Context, id, userID int64) error {
	const query = `
		UPDATE transactions t
		SET deleted_at = now()
		FROM accounts a
		WHERE t.id = $1 AND t.account_id = a.id AND a.user_id = $2 AND t.deleted_at IS NULL
	`
	tag, err := r.q.Exec(ctx, query, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
