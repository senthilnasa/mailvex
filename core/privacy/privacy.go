package privacy

import (
	"context"
	"errors"
	"net/mail"
	"strings"

	"github.com/mailvex/mailvex/core/domain"
	"github.com/mailvex/mailvex/core/repository"
)

type PrivacyService struct {
	repo repository.Repository
}

func NewPrivacyService(repo repository.Repository) *PrivacyService {
	return &PrivacyService{repo: repo}
}

// RequestErasure implements GDPR Article 17 / DPDP right-to-be-forgotten
func (s *PrivacyService) RequestErasure(ctx context.Context, tenantID, email string) (*domain.PrivacyRequest, error) {
	normEmail := strings.TrimSpace(strings.ToLower(email))
	if _, err := mail.ParseAddress(normEmail); err != nil {
		return nil, errors.New("invalid email address")
	}

	return s.repo.ExecutePrivacyErasure(ctx, tenantID, normEmail)
}

// RequestExport implements GDPR Article 15 right to data portability / access
func (s *PrivacyService) RequestExport(ctx context.Context, tenantID, email string) (*domain.PrivacyExportReport, error) {
	normEmail := strings.TrimSpace(strings.ToLower(email))
	if _, err := mail.ParseAddress(normEmail); err != nil {
		return nil, errors.New("invalid email address")
	}

	return s.repo.ExecutePrivacyExport(ctx, tenantID, normEmail)
}

func (s *PrivacyService) ListRequests(ctx context.Context, tenantID string) ([]*domain.PrivacyRequest, error) {
	return s.repo.ListPrivacyRequests(ctx, tenantID)
}

func (s *PrivacyService) GetRequest(ctx context.Context, tenantID, id string) (*domain.PrivacyRequest, error) {
	return s.repo.GetPrivacyRequest(ctx, tenantID, id)
}
