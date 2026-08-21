// Package testutil содержит переиспользуемые тестовые помощники для unit-тестов
// service-слоя. Импортируется только из _test.go файлов.
package testutil

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// FakeTx реализует pgx.Tx через embedding nil-интерфейса: делегировать некуда,
// поэтому переопределены только Commit/Rollback. Любой другой метод (Query, Exec,
// Conn...), если вдруг вызван, паникует на nil-указателе.
//
// Commit/Rollback используют общий флаг Closed, воспроизводя реальную семантику
// pgx: pgx.BeginFunc всегда вызывает Rollback через defer, даже после успешного
// Commit — второй вызов обязан быть no-op'ом (pgx.ErrTxClosed), а не повторным
// действием.
type FakeTx struct {
	pgx.Tx
	Closed     bool
	Committed  bool
	RolledBack bool
}

func (f *FakeTx) Commit(ctx context.Context) error {
	if f.Closed {
		return pgx.ErrTxClosed
	}
	f.Closed = true
	f.Committed = true
	return nil
}

func (f *FakeTx) Rollback(ctx context.Context) error {
	if f.Closed {
		return pgx.ErrTxClosed
	}
	f.Closed = true
	f.RolledBack = true
	return nil
}

// MockBeginner реализует db.Beginner (единственный метод — Begin) для тестов,
// открывающих pgx.BeginFunc без реальной БД.
type MockBeginner struct {
	Tx     *FakeTx
	Err    error
	Called bool
}

func (m *MockBeginner) Begin(ctx context.Context) (pgx.Tx, error) {
	m.Called = true
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Tx, nil
}
