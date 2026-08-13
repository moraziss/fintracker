package user

import "errors"

var (
	ErrEmailTaken      = errors.New("user: email already registered")
	ErrNotFound        = errors.New("user: not found")
	ErrPasswordTooLong = errors.New("user: password exceeds 72 bytes")
)
