package auth

import "errors"

var ErrInvalidCredentials = errors.New("auth: invalid email or password")
var ErrTokenNotFound = errors.New("auth: refresh token not found")
