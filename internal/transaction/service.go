package transaction

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/morazss/fintracker/internal/account"
	"github.com/morazss/fintracker/internal/db"
)

type Service struct {
	transactions Repository
	accounts     account.Repository
	beginner     db.Beginner
}

func NewService(transactions Repository, accounts account.Repository, beginner db.Beginner) *Service {
	return &Service{transactions: transactions, accounts: accounts, beginner: beginner}
}

func (s *Service) Create(ctx context.Context, userID int64, req CreateRequest) (*Transaction, error) {
	if req.Amount.Sign() <= 0 {
		return nil, ErrInvalidAmount
	}

	delta := req.Amount
	if req.Type == string(TypeExpense) {
		delta = delta.Neg()
	}

	occurredAt := time.Now()
	if req.OccurredAt != nil {
		occurredAt = *req.OccurredAt
	}

	t := &Transaction{
		AccountID:   req.AccountID,
		CategoryID:  req.CategoryID,
		Type:        Type(req.Type),
		Amount:      req.Amount,
		Description: req.Description,
		OccurredAt:  occurredAt,
	}

	var created *Transaction
	err := pgx.BeginFunc(ctx, s.beginner, func(tx pgx.Tx) error {
		accountsTx := s.accounts.WithTx(tx)
		if err := accountsTx.ApplyBalanceDelta(ctx, req.AccountID, userID, delta); err != nil {
			return err
		}

		transactionsTx := s.transactions.WithTx(tx)
		var err error
		created, err = transactionsTx.Create(ctx, t)
		return err
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}
