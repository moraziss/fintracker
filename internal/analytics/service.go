package analytics

import "context"

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Summary(ctx context.Context, userID int64, f Filter) (*Summary, error) {
	return s.repo.Summary(ctx, userID, f)
}
