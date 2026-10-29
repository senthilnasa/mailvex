package retention

import (
	"context"
	"errors"

	"github.com/senthilnasa/mailvex/core/domain"
	"github.com/senthilnasa/mailvex/core/repository"
)

type RetentionService struct {
	repo repository.Repository
}

func NewRetentionService(repo repository.Repository) *RetentionService {
	return &RetentionService{repo: repo}
}

func (s *RetentionService) GetPolicy(ctx context.Context, tenantID string) (*domain.RetentionPolicy, error) {
	if tenantID == "" {
		return nil, errors.New("tenant_id is required")
	}
	return s.repo.GetRetentionPolicy(ctx, tenantID)
}

func (s *RetentionService) UpdatePolicy(ctx context.Context, policy *domain.RetentionPolicy) error {
	if policy == nil || policy.TenantID == "" {
		return errors.New("tenant_id is required")
	}
	if policy.MessageRetentionDays < 1 || policy.EventsRetentionDays < 1 || policy.AttemptsRetentionDays < 1 || policy.RawMIMERetentionDays < 1 {
		return errors.New("retention days must be at least 1")
	}
	return s.repo.UpdateRetentionPolicy(ctx, policy)
}

func (s *RetentionService) ExecutePurge(ctx context.Context, tenantID string) (*domain.RetentionPurgeSummary, error) {
	if tenantID == "" {
		return nil, errors.New("tenant_id is required")
	}
	return s.repo.ExecuteRetentionPurge(ctx, tenantID)
}
