package transaction_test

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/morazss/fintracker/internal/account"
	"github.com/morazss/fintracker/internal/db"
	"github.com/morazss/fintracker/internal/testutil"
	"github.com/morazss/fintracker/internal/transaction"
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
}

func (m *mockTransactionRepository) Create(ctx context.Context, t *transaction.Transaction) (*transaction.Transaction, error) {
	m.createCalled = true
	return m.CreateFunc(ctx, t)
}
func (m *mockTransactionRepository) List(ctx context.Context, userID int64, f transaction.ListFilter) ([]*transaction.Transaction, error) {
	return m.ListFunc(ctx, userID, f)
}
func (m *mockTransactionRepository) SoftDelete(ctx context.Context, id, userID int64) error {
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
