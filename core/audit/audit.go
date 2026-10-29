package audit

import (
	"context"
	"time"

	"github.com/senthilnasa/mailvex/core/crypto"
	"github.com/senthilnasa/mailvex/core/domain"
	"github.com/senthilnasa/mailvex/core/repository"
)

type Service struct {
	repo repository.Repository
}

func NewService(repo repository.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Log(ctx context.Context, log *domain.AuditLog) error {
	if log.ID == "" {
		log.ID = crypto.NewULID(crypto.PrefixAudit)
	}
	if log.OccurredAt.IsZero() {
		log.OccurredAt = time.Now().UTC()
	}
	return s.repo.RecordAuditLog(ctx, log)
}
