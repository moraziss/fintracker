package transaction

import (
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/shopspring/decimal"
)

var validate = validator.New()

type CreateRequest struct {
	AccountID   int64           `json:"account_id" validate:"required"`
	CategoryID  *int64          `json:"category_id"`
	Type        string          `json:"type" validate:"required,oneof=income expense"`
	Amount      decimal.Decimal `json:"amount"`
	Description *string         `json:"description"`
	OccurredAt  *time.Time      `json:"occurred_at"`
}
