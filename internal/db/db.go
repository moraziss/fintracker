package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Querier — минимальный набор методов, которым пользуются наши репозитории.
// *pgxpool.Pool и pgx.Tx оба реализуют его "бесплатно" за счёт структурной
// типизации Go (совпадения сигнатур достаточно, explicit implements не нужен) —
// поэтому один и тот же репозиторий одинаково работает и вне транзакции
// (получив пул), и внутри неё (получив tx), без дублирования SQL под каждый случай.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Beginner — ровно то, что требует pgx.BeginFunc (interface{ Begin(ctx) (Tx, error) }).
// Называем и держим отдельным маленьким интерфейсом, а не *pgxpool.Pool напрямую в
// полях Service, — тот же принцип DI через интерфейсы, что уже применён для
// Repository/TokenIssuer: тесты сервисного слоя смогут подставить фейковый Beginner,
// не поднимая реальный Postgres. pgx.BeginFunc эту сигнатуру НЕ требует по имени —
// достаточно структурного совпадения; тип нужен только нам, ради читаемости DI.
type Beginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}
