package analytics_test

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/morazss/fintracker/internal/analytics"
)

type mockRepository struct {
	SummaryFunc    func(ctx context.Context, userID int64, f analytics.Filter) (*analytics.Summary, error)
	ByCategoryFunc func(ctx context.Context, userID int64, f analytics.Filter) ([]*analytics.CategoryTotal, error)
	TrendFunc      func(ctx context.Context, userID int64, f analytics.Filter) ([]*analytics.DailyTrend, error)
}

func (m *mockRepository) Summary(ctx context.Context, userID int64, f analytics.Filter) (*analytics.Summary, error) {
	return m.SummaryFunc(ctx, userID, f)
}

func (m *mockRepository) ByCategory(ctx context.Context, userID int64, f analytics.Filter) ([]*analytics.CategoryTotal, error) {
	return m.ByCategoryFunc(ctx, userID, f)
}

func (m *mockRepository) Trend(ctx context.Context, userID int64, f analytics.Filter) ([]*analytics.DailyTrend, error) {
	return m.TrendFunc(ctx, userID, f)
}

func accountID(id int64) *int64 { return &id }

func TestService_Summary(t *testing.T) {
	wantResult := &analytics.Summary{
		Income:  decimal.NewFromInt(500),
		Expense: decimal.NewFromInt(200),
		Net:     decimal.NewFromInt(300),
	}
	repoErrSentinel := errors.New("db: query failed")
	filter := analytics.Filter{AccountID: accountID(1)}

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
			var receivedFilter analytics.Filter
			repo := &mockRepository{
				SummaryFunc: func(ctx context.Context, userID int64, f analytics.Filter) (*analytics.Summary, error) {
					receivedFilter = f
					if tt.repoErr != nil {
						return nil, tt.repoErr
					}
					return wantResult, nil
				},
			}
			svc := analytics.NewService(repo)

			got, err := svc.Summary(context.Background(), 7, filter)

			require.Equal(t, filter, receivedFilter, "фильтр должен дойти до репозитория без изменений")

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.Same(t, wantResult, got, "Service.Summary обязан вернуть тот же объект, а не копию")
		})
	}
}

func TestService_ByCategory(t *testing.T) {
	wantResult := []*analytics.CategoryTotal{
		{CategoryID: accountID(1), Income: decimal.NewFromInt(300), Expense: decimal.Zero},
	}
	repoErrSentinel := errors.New("db: query failed")
	filter := analytics.Filter{AccountID: accountID(1)}

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
			var receivedFilter analytics.Filter
			repo := &mockRepository{
				ByCategoryFunc: func(ctx context.Context, userID int64, f analytics.Filter) ([]*analytics.CategoryTotal, error) {
					receivedFilter = f
					if tt.repoErr != nil {
						return nil, tt.repoErr
					}
					return wantResult, nil
				},
			}
			svc := analytics.NewService(repo)

			got, err := svc.ByCategory(context.Background(), 7, filter)

			require.Equal(t, filter, receivedFilter, "фильтр должен дойти до репозитория без изменений")

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, wantResult, got)
		})
	}
}

func TestService_Trend(t *testing.T) {
	wantResult := []*analytics.DailyTrend{
		{Income: decimal.NewFromInt(100), Expense: decimal.Zero, Net: decimal.NewFromInt(100), Cumulative: decimal.NewFromInt(100)},
	}
	repoErrSentinel := errors.New("db: query failed")
	filter := analytics.Filter{AccountID: accountID(1)}

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
			var receivedFilter analytics.Filter
			repo := &mockRepository{
				TrendFunc: func(ctx context.Context, userID int64, f analytics.Filter) ([]*analytics.DailyTrend, error) {
					receivedFilter = f
					if tt.repoErr != nil {
						return nil, tt.repoErr
					}
					return wantResult, nil
				},
			}
			svc := analytics.NewService(repo)

			got, err := svc.Trend(context.Background(), 7, filter)

			require.Equal(t, filter, receivedFilter, "фильтр должен дойти до репозитория без изменений")

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, wantResult, got)
		})
	}
}
