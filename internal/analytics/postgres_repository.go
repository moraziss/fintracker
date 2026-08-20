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

func (r *PostgresRepository) ByCategory(ctx context.Context, userID int64, f Filter) ([]*CategoryTotal, error) {
	var sb strings.Builder
	sb.WriteString(`
		SELECT
			c.id,
			c.name,
			COALESCE(SUM(t.amount) FILTER (WHERE t.type = 'income'), 0),
			COALESCE(SUM(t.amount) FILTER (WHERE t.type = 'expense'), 0)
		FROM transactions t
		JOIN accounts a ON a.id = t.account_id
		LEFT JOIN categories c ON c.id = t.category_id
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

	sb.WriteString(" GROUP BY c.id ORDER BY c.name")

	rows, err := r.q.Query(ctx, sb.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*CategoryTotal
	for rows.Next() {
		var c CategoryTotal
		if err := rows.Scan(&c.CategoryID, &c.CategoryName, &c.Income, &c.Expense); err != nil {
			return nil, err
		}
		items = append(items, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *PostgresRepository) Trend(ctx context.Context, userID int64, f Filter) ([]*DailyTrend, error) {
	var sb strings.Builder
	sb.WriteString(`
		WITH daily AS (
			SELECT
				t.occurred_at AS day,
				COALESCE(SUM(t.amount) FILTER (WHERE t.type = 'income'), 0) AS income,
				COALESCE(SUM(t.amount) FILTER (WHERE t.type = 'expense'), 0) AS expense
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

	sb.WriteString(`
			GROUP BY t.occurred_at
		)
		SELECT
			day,
			income,
			expense,
			income - expense,
			SUM(income - expense) OVER (ORDER BY day)
		FROM daily
		ORDER BY day`)

	rows, err := r.q.Query(ctx, sb.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*DailyTrend
	for rows.Next() {
		var d DailyTrend
		if err := rows.Scan(&d.Date, &d.Income, &d.Expense, &d.Net, &d.Cumulative); err != nil {
			return nil, err
		}
		items = append(items, &d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
