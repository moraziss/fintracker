package analytics

import (
	"context"
	"fmt"
	"strings"
)

func (r *PostgresRepository) Summary(ctx context.Context, userID int64, f Filter) (*Summary, error) {
	var sb strings.Builder
	sb.WriteString(`
		SELECT
			COALESCE(SUM(t.amount) FILTER (WHERE t.type = 'income'), 0),
			COALESCE(SUM(t.amount) FILTER (WHERE t.type = 'expense'), 0)
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
	if f.From != nil {
		add("t.occurred_at >=", *f.From)
	}
	if f.To != nil {
		add("t.occurred_at <=", *f.To)
	}

	var s Summary
	if err := r.q.QueryRow(ctx, sb.String(), args...).Scan(&s.Income, &s.Expense); err != nil {
		return nil, err
	}
	s.Net = s.Income.Sub(s.Expense)
	return &s, nil
}
