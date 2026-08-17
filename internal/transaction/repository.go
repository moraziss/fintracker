package transaction

import (
	"context"

	"github.com/morazss/fintracker/internal/db"
)

type Repository interface {
	Create(ctx context.Context, t *Transaction) (*Transaction, error)
	WithTx(q db.Querier) Repository
}
