package analytics

import (
	"time"

	"github.com/shopspring/decimal"
)

type Summary struct {
	Income  decimal.Decimal `json:"income"`
	Expense decimal.Decimal `json:"expense"`
	Net     decimal.Decimal `json:"net"`
}

type CategoryTotal struct {
	CategoryID   *int64          `json:"category_id"`
	CategoryName *string         `json:"category_name"`
	Income       decimal.Decimal `json:"income"`
	Expense      decimal.Decimal `json:"expense"`
}

type DailyTrend struct {
	Date       time.Time       `json:"date"`
	Income     decimal.Decimal `json:"income"`
	Expense    decimal.Decimal `json:"expense"`
	Net        decimal.Decimal `json:"net"`
	Cumulative decimal.Decimal `json:"cumulative"`
}
