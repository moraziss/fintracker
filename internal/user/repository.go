package user

import "context"

type Repository interface {
	Create(ctx context.Context, email, passwordHash string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
}
