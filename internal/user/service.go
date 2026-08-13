package user

import (
	"context"

	"golang.org/x/crypto/bcrypt"
)

type Service interface {
	Register(ctx context.Context, email, password string) (*User, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Register(ctx context.Context, email, password string) (*User, error) {
	if len(password) > 72 {
		return nil, ErrPasswordTooLong
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return s.repo.Create(ctx, email, string(hash))
}
