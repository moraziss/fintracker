package user_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/moraziss/fintracker/internal/user"
)

type mockRepository struct {
	CreateFunc     func(ctx context.Context, email, passwordHash string) (*user.User, error)
	GetByEmailFunc func(ctx context.Context, email string) (*user.User, error)
}

func (m *mockRepository) Create(ctx context.Context, email, passwordHash string) (*user.User, error) {
	return m.CreateFunc(ctx, email, passwordHash)
}

func (m *mockRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	return m.GetByEmailFunc(ctx, email)
}

func TestService_Register(t *testing.T) {
	tests := []struct {
		name             string
		email            string
		password         string
		createErr        error // что должен вернуть Create (nil = успех)
		createShouldCall bool  // ожидаем ли вообще вызов Create
		wantErr          error // что должен вернуть Register
	}{
		{
			name:             "success",
			email:            "test@example.com",
			password:         "correct horse battery staple",
			createShouldCall: true,
		},
		{
			name:             "password too long",
			email:            "test@example.com",
			password:         strings.Repeat("a", 73), // 73 байта > лимита в 72
			createShouldCall: false,
			wantErr:          user.ErrPasswordTooLong,
		},
		{
			name:             "email already taken",
			email:            "taken@example.com",
			password:         "somepassword",
			createErr:        user.ErrEmailTaken,
			createShouldCall: true,
			wantErr:          user.ErrEmailTaken,
		},
		{
			name:             "repository error",
			email:            "test@example.com",
			password:         "somepassword",
			createErr:        errors.New("db: connection refused"),
			createShouldCall: true,
			wantErr:          errors.New("db: connection refused"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			repo := &mockRepository{
				CreateFunc: func(ctx context.Context, email, passwordHash string) (*user.User, error) {
					called = true
					require.NoError(t, bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(tt.password)))
					if tt.createErr != nil {
						return nil, tt.createErr
					}
					return &user.User{Email: email}, nil
				},
			}

			svc := user.NewService(repo)
			got, err := svc.Register(context.Background(), tt.email, tt.password)

			require.Equal(t, tt.createShouldCall, called, "Create вызван не так, как ожидалось")

			if tt.wantErr != nil {
				require.EqualError(t, err, tt.wantErr.Error())
				require.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			require.Equal(t, tt.email, got.Email)
		})
	}
}
