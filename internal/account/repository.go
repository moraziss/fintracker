package account

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/morazss/fintracker/internal/db"
)

// Repository — сознательно минимальный: единственный метод, который реально
// нужен на этом шаге. Методы добавляются по мере доказанной необходимости.
type Repository interface {
	// ApplyBalanceDelta атомарно прибавляет delta к балансу (delta отрицательна
	// для расходов). Проверка владельца — тем же запросом: если счёт не найден
	// ИЛИ принадлежит другому пользователю — ErrNotFound в обоих случаях.
	ApplyBalanceDelta(ctx context.Context, accountID, userID int64, delta decimal.Decimal) error
	WithTx(q db.Querier) Repository
}
