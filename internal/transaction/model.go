package transaction

import (
	"time"

	"github.com/shopspring/decimal"
)

type Type string

const (
	TypeIncome  Type = "income"
	TypeExpense Type = "expense"
)

type Transaction struct {
	ID          int64           `json:"id"`
	AccountID   int64           `json:"account_id"`
	CategoryID  *int64          `json:"category_id"`
	Type        Type            `json:"type"`
	Amount      decimal.Decimal `json:"amount"`
	Description *string         `json:"description"`
	OccurredAt  time.Time       `json:"occurred_at"`
	CreatedAt   time.Time       `json:"created_at"`
	DeletedAt   *time.Time      `json:"deleted_at,omitempty"`
}
