package transaction

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/moraziss/fintracker/internal/account"
	"github.com/moraziss/fintracker/internal/db"
)

const (
	defaultListLimit = 20
	maxListLimit     = 100
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

func (s *Service) List(ctx context.Context, userID int64, f ListFilter) ([]*Transaction, *Cursor, error) {
	if f.Limit <= 0 || f.Limit > maxListLimit {
		f.Limit = defaultListLimit
	}

	// запрашиваем на одну строку больше, чтобы понять, есть ли следующая
	// страница, без отдельного COUNT-запроса
	fetch := f
	fetch.Limit = f.Limit + 1

	items, err := s.transactions.List(ctx, userID, fetch)
	if err != nil {
		return nil, nil, err
	}

	var next *Cursor
	if len(items) > f.Limit {
		last := items[f.Limit-1]
		next = &Cursor{OccurredAt: last.OccurredAt, ID: last.ID}
		items = items[:f.Limit]
	}

	return items, next, nil
}

func (s *Service) Delete(ctx context.Context, id, userID int64) error {
	return s.transactions.SoftDelete(ctx, id, userID)
}

func (s *Service) Update(ctx context.Context, id, userID int64, req UpdateRequest) (*Transaction, error) {
	if req.Amount.Sign() <= 0 {
		return nil, ErrInvalidAmount
	}

	newDelta := req.Amount
	if req.Type == string(TypeExpense) {
		newDelta = newDelta.Neg()
	}

	occurredAt := time.Now()
	if req.OccurredAt != nil {
		occurredAt = *req.OccurredAt
	}

	var updated *Transaction
	err := pgx.BeginFunc(ctx, s.beginner, func(tx pgx.Tx) error {
		transactionsTx := s.transactions.WithTx(tx)

		existing, err := transactionsTx.GetForUpdate(ctx, id, userID)
		if err != nil {
			return err
		}

		oldDelta := existing.Amount
		if existing.Type == TypeExpense {
			oldDelta = oldDelta.Neg()
		}

		net := newDelta.Sub(oldDelta)
		if !net.IsZero() {
			accountsTx := s.accounts.WithTx(tx)
			if err := accountsTx.ApplyBalanceDelta(ctx, existing.AccountID, userID, net); err != nil {
				return err
			}
		}

		existing.CategoryID = req.CategoryID
		existing.Type = Type(req.Type)
		existing.Amount = req.Amount
		existing.Description = req.Description
		existing.OccurredAt = occurredAt

		updated, err = transactionsTx.Update(ctx, existing)
		return err
	})
	if err != nil {
		return nil, err
	}

	return updated, nil
}
