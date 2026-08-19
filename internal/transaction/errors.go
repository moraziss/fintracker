package transaction

import "errors"

var ErrInvalidAmount = errors.New("amount must be positive")
var ErrNotFound = errors.New("transaction not found")
