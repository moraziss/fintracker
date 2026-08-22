package transaction_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/moraziss/fintracker/internal/account"
	"github.com/moraziss/fintracker/internal/db"
	"github.com/moraziss/fintracker/internal/testutil"
	"github.com/moraziss/fintracker/internal/transaction"
)

type mockAccountRepository struct {
	ApplyBalanceDeltaFunc func(ctx context.Context, accountID, userID int64, delta decimal.Decimal) error

	applyCalled  bool
	appliedDelta decimal.Decimal
}

func (m *mockAccountRepository) ApplyBalanceDelta(ctx context.Context, accountID, userID int64, delta decimal.Decimal) error {
	m.applyCalled = true
	m.appliedDelta = delta
	return m.ApplyBalanceDeltaFunc(ctx, accountID, userID, delta)
}

func (m *mockAccountRepository) WithTx(q db.Querier) account.Repository {
	return m
}

type mockTransactionRepository struct {
	CreateFunc       func(ctx context.Context, t *transaction.Transaction) (*transaction.Transaction, error)
	ListFunc         func(ctx context.Context, userID int64, f transaction.ListFilter) ([]*transaction.Transaction, error)
	SoftDeleteFunc   func(ctx context.Context, id, userID int64) error
	GetForUpdateFunc func(ctx context.Context, id, userID int64) (*transaction.Transaction, error)
	UpdateFunc       func(ctx context.Context, t *transaction.Transaction) (*transaction.Transaction, error)

	createCalled bool
	updateCalled bool

	receivedListFilter transaction.ListFilter
	softDeleteCalled   bool
}

func (m *mockTransactionRepository) Create(ctx context.Context, t *transaction.Transaction) (*transaction.Transaction, error) {
	m.createCalled = true
	return m.CreateFunc(ctx, t)
}
func (m *mockTransactionRepository) List(ctx context.Context, userID int64, f transaction.ListFilter) ([]*transaction.Transaction, error) {
	m.receivedListFilter = f
	return m.ListFunc(ctx, userID, f)
}
func (m *mockTransactionRepository) SoftDelete(ctx context.Context, id, userID int64) error {
	m.softDeleteCalled = true
	return m.SoftDeleteFunc(ctx, id, userID)
}
func (m *mockTransactionRepository) WithTx(q db.Querier) transaction.Repository {
	return m
}
func (m *mockTransactionRepository) GetForUpdate(ctx context.Context, id, userID int64) (*transaction.Transaction, error) {
	return m.GetForUpdateFunc(ctx, id, userID)
}
func (m *mockTransactionRepository) Update(ctx context.Context, t *transaction.Transaction) (*transaction.Transaction, error) {
	m.updateCalled = true
	return m.UpdateFunc(ctx, t)
}

func TestService_Create(t *testing.T) {
	amount100 := decimal.NewFromInt(100)
	amount50 := decimal.NewFromInt(50)
	applyErrSentinel := errors.New("account: apply failed")
	createErrSentinel := errors.New("db: insert failed")

	tests := []struct {
		name             string
		req              transaction.CreateRequest
		applyErr         error
		createErr        error
		wantErr          error
		wantBeginCalled  bool
		wantApplyCalled  bool
		wantApplyDelta   decimal.Decimal
		wantCreateCalled bool
		wantCommit       bool
		wantRollback     bool
	}{
		{
			name:             "success income",
			req:              transaction.CreateRequest{AccountID: 1, Type: string(transaction.TypeIncome), Amount: amount100},
			wantBeginCalled:  true,
			wantApplyCalled:  true,
			wantApplyDelta:   amount100,
			wantCreateCalled: true,
			wantCommit:       true,
		},
		{
			name:             "success expense — delta инвертируется",
			req:              transaction.CreateRequest{AccountID: 1, Type: string(transaction.TypeExpense), Amount: amount50},
			wantBeginCalled:  true,
			wantApplyCalled:  true,
			wantApplyDelta:   amount50.Neg(),
			wantCreateCalled: true,
			wantCommit:       true,
		},
		{
			name:    "zero amount",
			req:     transaction.CreateRequest{AccountID: 1, Type: string(transaction.TypeIncome), Amount: decimal.Zero},
			wantErr: transaction.ErrInvalidAmount,
			// wantBeginCalled и всё остальное — false по умолчанию: до pgx.BeginFunc дело не доходит
		},
		{
			name:    "negative amount",
			req:     transaction.CreateRequest{AccountID: 1, Type: string(transaction.TypeIncome), Amount: decimal.NewFromInt(-10)},
			wantErr: transaction.ErrInvalidAmount,
		},
		{
			name:             "balance apply fails — Create не вызывается",
			req:              transaction.CreateRequest{AccountID: 1, Type: string(transaction.TypeIncome), Amount: amount100},
			applyErr:         applyErrSentinel,
			wantErr:          applyErrSentinel,
			wantBeginCalled:  true,
			wantApplyCalled:  true,
			wantApplyDelta:   amount100,
			wantCreateCalled: false,
			wantRollback:     true,
		},
		{
			name:             "repository create fails",
			req:              transaction.CreateRequest{AccountID: 1, Type: string(transaction.TypeIncome), Amount: amount100},
			createErr:        createErrSentinel,
			wantErr:          createErrSentinel,
			wantBeginCalled:  true,
			wantApplyCalled:  true,
			wantApplyDelta:   amount100,
			wantCreateCalled: true,
			wantRollback:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accounts := &mockAccountRepository{
				ApplyBalanceDeltaFunc: func(ctx context.Context, accountID, userID int64, delta decimal.Decimal) error {
					return tt.applyErr
				},
			}
			transactions := &mockTransactionRepository{
				CreateFunc: func(ctx context.Context, tr *transaction.Transaction) (*transaction.Transaction, error) {
					if tt.createErr != nil {
						return nil, tt.createErr
					}
					return &transaction.Transaction{ID: 1, AccountID: tr.AccountID, Amount: tr.Amount, Type: tr.Type}, nil
				},
			}
			tx := &testutil.FakeTx{}
			beginner := &testutil.MockBeginner{Tx: tx}

			svc := transaction.NewService(transactions, accounts, beginner)
			got, err := svc.Create(context.Background(), 7, tt.req)

			require.Equal(t, tt.wantBeginCalled, beginner.Called, "Begin вызван не так, как ожидалось")
			require.Equal(t, tt.wantApplyCalled, accounts.applyCalled, "ApplyBalanceDelta вызван не так, как ожидалось")
			require.Equal(t, tt.wantCreateCalled, transactions.createCalled, "Create вызван не так, как ожидалось")
			require.Equal(t, tt.wantCommit, tx.Committed, "Commit вызван не так, как ожидалось")
			require.Equal(t, tt.wantRollback, tx.RolledBack, "Rollback вызван не так, как ожидалось")

			if tt.wantApplyCalled {
				require.True(t, tt.wantApplyDelta.Equal(accounts.appliedDelta),
					"delta: want %s, got %s", tt.wantApplyDelta, accounts.appliedDelta)
			}

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
		})
	}
}

func TestService_Update(t *testing.T) {
	amount100 := decimal.NewFromInt(100)
	amount150 := decimal.NewFromInt(150)
	applyErrSentinel := errors.New("account: apply failed")
	updateErrSentinel := errors.New("db: update failed")
	getErrSentinel := transaction.ErrNotFound

	tests := []struct {
		name             string
		req              transaction.UpdateRequest
		existing         *transaction.Transaction
		getErr           error
		applyErr         error
		updateErr        error
		wantErr          error
		wantBeginCalled  bool
		wantApplyCalled  bool
		wantApplyDelta   decimal.Decimal
		wantUpdateCalled bool
		wantCommit       bool
		wantRollback     bool
	}{
		{
			name:             "amount increased, тип не менялся",
			req:              transaction.UpdateRequest{Type: string(transaction.TypeIncome), Amount: amount150},
			existing:         &transaction.Transaction{ID: 1, AccountID: 5, Type: transaction.TypeIncome, Amount: amount100},
			wantBeginCalled:  true,
			wantApplyCalled:  true,
			wantApplyDelta:   decimal.NewFromInt(50), // 150 - 100
			wantUpdateCalled: true,
			wantCommit:       true,
		},
		{
			name:             "тип сменился income -> expense, сумма та же — net удваивается",
			req:              transaction.UpdateRequest{Type: string(transaction.TypeExpense), Amount: amount100},
			existing:         &transaction.Transaction{ID: 2, AccountID: 5, Type: transaction.TypeIncome, Amount: amount100},
			wantBeginCalled:  true,
			wantApplyCalled:  true,
			wantApplyDelta:   decimal.NewFromInt(-200), // newDelta(-100) - oldDelta(+100)
			wantUpdateCalled: true,
			wantCommit:       true,
		},
		{
			name:             "сумма и тип не менялись — баланс не трогаем",
			req:              transaction.UpdateRequest{Type: string(transaction.TypeIncome), Amount: amount100},
			existing:         &transaction.Transaction{ID: 3, AccountID: 5, Type: transaction.TypeIncome, Amount: amount100},
			wantBeginCalled:  true,
			wantApplyCalled:  false, // net.IsZero() — ветка ApplyBalanceDelta пропускается
			wantUpdateCalled: true,
			wantCommit:       true,
		},
		{
			name:    "invalid new amount",
			req:     transaction.UpdateRequest{Type: string(transaction.TypeIncome), Amount: decimal.Zero},
			wantErr: transaction.ErrInvalidAmount,
		},
		{
			name:            "транзакция не найдена / чужая",
			req:             transaction.UpdateRequest{Type: string(transaction.TypeIncome), Amount: amount100},
			getErr:          getErrSentinel,
			wantErr:         getErrSentinel,
			wantBeginCalled: true,
			wantRollback:    true,
		},
		{
			name:             "balance apply fails — Update не вызывается",
			req:              transaction.UpdateRequest{Type: string(transaction.TypeIncome), Amount: amount150},
			existing:         &transaction.Transaction{ID: 4, AccountID: 5, Type: transaction.TypeIncome, Amount: amount100},
			applyErr:         applyErrSentinel,
			wantErr:          applyErrSentinel,
			wantBeginCalled:  true,
			wantApplyCalled:  true,
			wantApplyDelta:   decimal.NewFromInt(50),
			wantUpdateCalled: false,
			wantRollback:     true,
		},
		{
			name:             "repository update fails",
			req:              transaction.UpdateRequest{Type: string(transaction.TypeIncome), Amount: amount150},
			existing:         &transaction.Transaction{ID: 5, AccountID: 5, Type: transaction.TypeIncome, Amount: amount100},
			updateErr:        updateErrSentinel,
			wantErr:          updateErrSentinel,
			wantBeginCalled:  true,
			wantApplyCalled:  true,
			wantApplyDelta:   decimal.NewFromInt(50),
			wantUpdateCalled: true,
			wantRollback:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accounts := &mockAccountRepository{
				ApplyBalanceDeltaFunc: func(ctx context.Context, accountID, userID int64, delta decimal.Decimal) error {
					return tt.applyErr
				},
			}
			transactions := &mockTransactionRepository{
				GetForUpdateFunc: func(ctx context.Context, id, userID int64) (*transaction.Transaction, error) {
					if tt.getErr != nil {
						return nil, tt.getErr
					}
					return tt.existing, nil
				},
				UpdateFunc: func(ctx context.Context, tr *transaction.Transaction) (*transaction.Transaction, error) {
					if tt.updateErr != nil {
						return nil, tt.updateErr
					}
					return tr, nil
				},
			}
			tx := &testutil.FakeTx{}
			beginner := &testutil.MockBeginner{Tx: tx}

			svc := transaction.NewService(transactions, accounts, beginner)
			got, err := svc.Update(context.Background(), 1, 7, tt.req)

			require.Equal(t, tt.wantBeginCalled, beginner.Called, "Begin вызван не так, как ожидалось")
			require.Equal(t, tt.wantApplyCalled, accounts.applyCalled, "ApplyBalanceDelta вызван не так, как ожидалось")
			require.Equal(t, tt.wantUpdateCalled, transactions.updateCalled, "Update вызван не так, как ожидалось")
			require.Equal(t, tt.wantCommit, tx.Committed, "Commit вызван не так, как ожидалось")
			require.Equal(t, tt.wantRollback, tx.RolledBack, "Rollback вызван не так, как ожидалось")

			if tt.wantApplyCalled {
				require.True(t, tt.wantApplyDelta.Equal(accounts.appliedDelta),
					"delta: want %s, got %s", tt.wantApplyDelta, accounts.appliedDelta)
			}

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
		})
	}
}

func TestService_List(t *testing.T) {
	t1 := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 8, 1, 10, 0, 1, 0, time.UTC)
	t3 := time.Date(2026, 8, 1, 10, 0, 2, 0, time.UTC)
	t4 := time.Date(2026, 8, 1, 10, 0, 3, 0, time.UTC)

	fourItems := []*transaction.Transaction{
		{ID: 101, OccurredAt: t1},
		{ID: 102, OccurredAt: t2},
		{ID: 103, OccurredAt: t3},
		{ID: 104, OccurredAt: t4},
	}
	repoErrSentinel := errors.New("db: query failed")

	tests := []struct {
		name           string
		filter         transaction.ListFilter
		repoItems      []*transaction.Transaction
		repoErr        error
		wantFetchLimit int
		wantItemsLen   int
		wantCursor     *transaction.Cursor
		wantErr        error
	}{
		{
			name:           "limit не задан — дефолт 20, запрашиваем 21",
			filter:         transaction.ListFilter{Limit: 0},
			repoItems:      fourItems, // 4 < 21 — следующей страницы нет
			wantFetchLimit: 21,
			wantItemsLen:   4,
			wantCursor:     nil,
		},
		{
			name:           "limit превышает максимум — сброс к дефолту, НЕ clamp до 100",
			filter:         transaction.ListFilter{Limit: 500},
			repoItems:      fourItems,
			wantFetchLimit: 21, // не 101
			wantItemsLen:   4,
			wantCursor:     nil,
		},
		{
			name:           "отрицательный limit — тоже сброс к дефолту",
			filter:         transaction.ListFilter{Limit: -5},
			repoItems:      fourItems,
			wantFetchLimit: 21,
			wantItemsLen:   4,
			wantCursor:     nil,
		},
		{
			name:           "ровно limit элементов — следующей страницы нет",
			filter:         transaction.ListFilter{Limit: 4},
			repoItems:      fourItems, // len == f.Limit, не больше
			wantFetchLimit: 5,
			wantItemsLen:   4,
			wantCursor:     nil,
		},
		{
			name:           "limit+1 элементов — есть следующая страница",
			filter:         transaction.ListFilter{Limit: 3},
			repoItems:      fourItems, // len(4) > f.Limit(3)
			wantFetchLimit: 4,
			wantItemsLen:   3,
			wantCursor:     &transaction.Cursor{OccurredAt: t3, ID: 103}, // items[f.Limit-1] = items[2]
		},
		{
			name:    "repository error",
			filter:  transaction.ListFilter{Limit: 3},
			repoErr: repoErrSentinel,
			wantErr: repoErrSentinel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transactions := &mockTransactionRepository{
				ListFunc: func(ctx context.Context, userID int64, f transaction.ListFilter) ([]*transaction.Transaction, error) {
					if tt.repoErr != nil {
						return nil, tt.repoErr
					}
					return tt.repoItems, nil
				},
			}
			svc := transaction.NewService(transactions, nil, nil) // accounts/beginner не нужны — List их не трогает

			items, cursor, err := svc.List(context.Background(), 7, tt.filter)

			require.Equal(t, tt.wantFetchLimit, transactions.receivedListFilter.Limit,
				"в репозиторий должен был уйти limit+1")

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, items)
				require.Nil(t, cursor)
				return
			}

			require.NoError(t, err)
			require.Len(t, items, tt.wantItemsLen)
			require.Equal(t, tt.wantCursor, cursor)
		})
	}
}

func TestService_Delete(t *testing.T) {
	repoErrSentinel := errors.New("db: not found or not owned")

	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "success"},
		{name: "repository error", repoErr: repoErrSentinel, wantErr: repoErrSentinel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transactions := &mockTransactionRepository{
				SoftDeleteFunc: func(ctx context.Context, id, userID int64) error {
					return tt.repoErr
				},
			}
			svc := transaction.NewService(transactions, nil, nil)

			err := svc.Delete(context.Background(), 1, 7)

			require.True(t, transactions.softDeleteCalled)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
