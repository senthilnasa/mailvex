package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/senthilnasa/mailvex/core/crypto"
	"github.com/senthilnasa/mailvex/core/domain"
	"github.com/senthilnasa/mailvex/core/repository"
)

var (
	ErrImpersonationReasonRequired = errors.New("impersonation requires an explicit, audited reason")
	ErrImpersonationUnauthorized   = errors.New("only platform admins and owners can impersonate tenants")
	ErrTenantNotFound              = errors.New("target tenant not found")
)

const (
	MaskedSupportNotice = "[MASKED - SUPPORT ROLE ACCESS]"
)

// ImpersonationManager coordinates platform admin impersonation sessions and audit logs
type ImpersonationManager struct {
	repo repository.Repository
}

func NewImpersonationManager(repo repository.Repository) *ImpersonationManager {
	return &ImpersonationManager{repo: repo}
}

// StartImpersonation begins an impersonation session with an explicit audit trail
func (m *ImpersonationManager) StartImpersonation(
	ctx context.Context,
	actorUserID, targetTenantID, reason, accessType, ipAddress, userAgent string,
	isSupportMasking bool,
) (*domain.ImpersonationSession, error) {
	reasonClean := strings.TrimSpace(reason)
	if reasonClean == "" {
		return nil, ErrImpersonationReasonRequired
	}

	// Verify target tenant exists
	tenant, err := m.repo.GetTenant(ctx, targetTenantID)
	if err != nil || tenant == nil {
		return nil, ErrTenantNotFound
	}

	if accessType != "read_write" {
		accessType = "read_only"
	}

	session := &domain.ImpersonationSession{
		ID:               crypto.NewULID(crypto.PrefixImpersonationSession),
		ActorUserID:      actorUserID,
		TargetTenantID:   targetTenantID,
		Reason:           reasonClean,
		AccessType:       accessType,
		IsSupportMasking: isSupportMasking,
		IPAddress:        ipAddress,
		UserAgent:        userAgent,
		StartedAt:        time.Now().UTC(),
	}

	if err := m.repo.CreateImpersonationSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to save impersonation session: %w", err)
	}

	// Record audit log
	auditAction := "admin.impersonation.start"
	_ = m.repo.RecordAuditLog(ctx, &domain.AuditLog{
		ID:          crypto.NewULID(crypto.PrefixAudit),
		TenantID:    &targetTenantID,
		ActorUserID: &actorUserID,
		Action:      auditAction,
		ResourceType: "tenant",
		ResourceID:  targetTenantID,
		Reason:      &reasonClean,
		IPAddress:   &ipAddress,
		UserAgent:   &userAgent,
		OccurredAt:  time.Now().UTC(),
	})

	return session, nil
}

// ExitImpersonation terminates an active impersonation session
func (m *ImpersonationManager) ExitImpersonation(ctx context.Context, sessionID, actorUserID string) error {
	if err := m.repo.EndImpersonationSession(ctx, sessionID); err != nil {
		return err
	}

	exitAction := "admin.impersonation.exit"
	_ = m.repo.RecordAuditLog(ctx, &domain.AuditLog{
		ID:          crypto.NewULID(crypto.PrefixAudit),
		ActorUserID: &actorUserID,
		Action:      exitAction,
		ResourceType: "impersonation_session",
		ResourceID:  sessionID,
		OccurredAt:  time.Now().UTC(),
	})

	return nil
}

// MaskMessageBody applies Support-role body masking if required (§7.4)
func MaskMessageBody(body string, isSupport bool) string {
	if !isSupport {
		return body
	}
	return MaskedSupportNotice
}
