package transaction

import (
	"encoding/base64"
	"encoding/json"
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

type Cursor struct {
	OccurredAt time.Time `json:"o"`
	ID         int64     `json:"i"`
}

func EncodeCursor(c Cursor) string {
	b, _ := json.Marshal(c)
	return base64.URLEncoding.EncodeToString(b)
}

func DecodeCursor(s string) (*Cursor, error) {
	b, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	var c Cursor
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

type ListFilter struct {
	AccountID  *int64
	CategoryID *int64
	Type       *Type
	From       *time.Time
	To         *time.Time
	Cursor     *Cursor
	Limit      int
}

type UpdateRequest struct {
	CategoryID  *int64          `json:"category_id"`
	Type        string          `json:"type" validate:"required,oneof=income expense"`
	Amount      decimal.Decimal `json:"amount"`
	Description *string         `json:"description"`
	OccurredAt  *time.Time      `json:"occurred_at"`
}
