package analytics

import "time"

type Filter struct {
	AccountID *int64
	From      *time.Time
	To        *time.Time
}
