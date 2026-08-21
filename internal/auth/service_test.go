package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/morazss/fintracker/internal/testutil"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/morazss/fintracker/internal/auth"
	"github.com/morazss/fintracker/internal/db"
	"github.com/morazss/fintracker/internal/user"
)

var testTokenIssuer = auth.NewTokenIssuer([]byte("test-secret"))

// --- mocks ---

type mockUserRepository struct {
	GetByEmailFunc func(ctx context.Context, email string) (*user.User, error)
	CreateFunc     func(ctx context.Context, email, passwordHash string) (*user.User, error)
}

func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	return m.GetByEmailFunc(ctx, email)
}

func (m *mockUserRepository) Create(ctx context.Context, email, passwordHash string) (*user.User, error) {
	return m.CreateFunc(ctx, email, passwordHash)
}

type mockRefreshTokenRepository struct {
	CreateFunc    func(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) (*auth.RefreshToken, error)
	GetByHashFunc func(ctx context.Context, tokenHash string) (*auth.RefreshToken, error)
	RevokeFunc    func(ctx context.Context, id int64) error

	revokeCalled bool
}

func (m *mockRefreshTokenRepository) Create(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) (*auth.RefreshToken, error) {
	return m.CreateFunc(ctx, userID, tokenHash, expiresAt)
}

func (m *mockRefreshTokenRepository) GetByHash(ctx context.Context, tokenHash string) (*auth.RefreshToken, error) {
	return m.GetByHashFunc(ctx, tokenHash)
}

func (m *mockRefreshTokenRepository) Revoke(ctx context.Context, id int64) error {
	m.revokeCalled = true
	return m.RevokeFunc(ctx, id)
}

func (m *mockRefreshTokenRepository) WithTx(q db.Querier) auth.RefreshTokenRepository {
	return m // моку неважно, в транзакции он или нет — саму транзакционность Postgres здесь не проверяем
}

// --- Login ---

func TestService_Login(t *testing.T) {
	const correctPassword = "correct-password"
	hash, err := bcrypt.GenerateFromPassword([]byte(correctPassword), bcrypt.DefaultCost)
	require.NoError(t, err)

	validUser := &user.User{ID: 42, Email: "test@example.com", PasswordHash: string(hash)}
	sentinelDBErr := errors.New("db: connection refused")

	tests := []struct {
		name          string
		password      string
		getByEmailErr error
		wantErr       error
	}{
		{name: "success", password: correctPassword},
		{name: "user not found", password: correctPassword, getByEmailErr: user.ErrNotFound, wantErr: auth.ErrInvalidCredentials},
		{name: "wrong password", password: "wrong-password", wantErr: auth.ErrInvalidCredentials},
		{name: "repository error", password: correctPassword, getByEmailErr: sentinelDBErr, wantErr: sentinelDBErr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users := &mockUserRepository{
				GetByEmailFunc: func(ctx context.Context, email string) (*user.User, error) {
					if tt.getByEmailErr != nil {
						return nil, tt.getByEmailErr
					}
					return validUser, nil
				},
			}
			refreshTokens := &mockRefreshTokenRepository{
				CreateFunc: func(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) (*auth.RefreshToken, error) {
					return &auth.RefreshToken{ID: 1, UserID: userID}, nil
				},
			}

			svc := auth.NewService(users, testTokenIssuer, refreshTokens, nil) // beginner не нужен — Login транзакций не открывает

			got, err := svc.Login(context.Background(), validUser.Email, tt.password)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)

			claims, err := testTokenIssuer.ParseAccessToken(got.AccessToken)
			require.NoError(t, err)
			require.Equal(t, validUser.ID, claims.UserID)
		})
	}
}

// --- Refresh ---

func TestService_Refresh(t *testing.T) {
	future := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Hour)
	revokedAt := time.Now().Add(-time.Minute)
	beginErrSentinel := errors.New("db: connection refused")
	revokeErrSentinel := errors.New("db: revoke failed")

	tests := []struct {
		name             string
		getByHashErr     error
		storedToken      *auth.RefreshToken
		beginErr         error
		revokeErr        error
		wantErr          error
		wantBeginCalled  bool
		wantRevokeCalled bool
		wantCommit       bool
		wantRollback     bool
	}{
		{
			name:             "success",
			storedToken:      &auth.RefreshToken{ID: 1, UserID: 42, ExpiresAt: future},
			wantBeginCalled:  true,
			wantRevokeCalled: true,
			wantCommit:       true,
		},
		{
			name:         "token not found",
			getByHashErr: auth.ErrTokenNotFound,
			wantErr:      auth.ErrInvalidCredentials,
		},
		{
			name:        "token expired",
			storedToken: &auth.RefreshToken{ID: 2, UserID: 42, ExpiresAt: past},
			wantErr:     auth.ErrInvalidCredentials,
		},
		{
			name:        "token revoked",
			storedToken: &auth.RefreshToken{ID: 3, UserID: 42, ExpiresAt: future, RevokedAt: &revokedAt},
			wantErr:     auth.ErrInvalidCredentials,
		},
		{
			name:            "begin fails",
			storedToken:     &auth.RefreshToken{ID: 4, UserID: 42, ExpiresAt: future},
			beginErr:        beginErrSentinel,
			wantErr:         beginErrSentinel,
			wantBeginCalled: true,
		},
		{
			name:             "revoke fails inside tx",
			storedToken:      &auth.RefreshToken{ID: 5, UserID: 42, ExpiresAt: future},
			revokeErr:        revokeErrSentinel,
			wantErr:          revokeErrSentinel,
			wantBeginCalled:  true,
			wantRevokeCalled: true,
			wantRollback:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockRefreshTokenRepository{
				GetByHashFunc: func(ctx context.Context, tokenHash string) (*auth.RefreshToken, error) {
					if tt.getByHashErr != nil {
						return nil, tt.getByHashErr
					}
					return tt.storedToken, nil
				},
				RevokeFunc: func(ctx context.Context, id int64) error {
					return tt.revokeErr
				},
				CreateFunc: func(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) (*auth.RefreshToken, error) {
					return &auth.RefreshToken{ID: 99, UserID: userID}, nil
				},
			}

			tx := &testutil.FakeTx{}
			beginner := &testutil.MockBeginner{Tx: tx, Err: tt.beginErr}

			svc := auth.NewService(nil, testTokenIssuer, repo, beginner) // users не нужен — Refresh его не трогает

			got, err := svc.Refresh(context.Background(), "some-raw-refresh-token")

			require.Equal(t, tt.wantBeginCalled, beginner.Called, "Begin вызван не так, как ожидалось")
			require.Equal(t, tt.wantRevokeCalled, repo.revokeCalled, "Revoke вызван не так, как ожидалось")
			require.Equal(t, tt.wantCommit, tx.Committed, "Commit вызван не так, как ожидалось")
			require.Equal(t, tt.wantRollback, tx.RolledBack, "Rollback вызван не так, как ожидалось")

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			require.NotEmpty(t, got.AccessToken)
			require.NotEmpty(t, got.RefreshToken)

			claims, err := testTokenIssuer.ParseAccessToken(got.AccessToken)
			require.NoError(t, err)
			require.Equal(t, tt.storedToken.UserID, claims.UserID)
		})
	}
}
