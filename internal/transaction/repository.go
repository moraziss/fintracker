package transaction

import (
	"context"

	"github.com/moraziss/fintracker/internal/db"
)

type Repository interface {
	Create(ctx context.Context, t *Transaction) (*Transaction, error)
	List(ctx context.Context, userID int64, f ListFilter) ([]*Transaction, error)
	SoftDelete(ctx context.Context, id, userID int64) error
	WithTx(q db.Querier) Repository
	GetForUpdate(ctx context.Context, id, userID int64) (*Transaction, error)
	Update(ctx context.Context, t *Transaction) (*Transaction, error)
}
