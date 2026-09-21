package privacy_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mailvex/mailvex/core/domain"
	"github.com/mailvex/mailvex/core/privacy"
	"github.com/mailvex/mailvex/core/repository"
)

func TestPrivacyExportAndErasure(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := privacy.NewPrivacyService(repo)
	ctx := context.Background()

	tenantID := "tenant_privacy_test"
	email := "user.gdpr@example.com"

	// Setup seed data
	msgID := "msg_priv_1"
	bodyRef := "blob://body_priv_1"
	rawMIME := "blob://mime_priv_1"
	msg := &domain.Message{
		ID:         msgID,
		TenantID:   tenantID,
		StreamID:   "stream_tx",
		BodyRef:    &bodyRef,
		RawMIMERef: &rawMIME,
		CreatedAt:  time.Now().UTC(),
	}

	rcptID := "rcpt_priv_1"
	rcpt := &domain.MessageRecipient{
		ID:            rcptID,
		MessageID:     msgID,
		TenantID:      tenantID,
		Address:       email,
		AddressDomain: "example.com",
		Status:        "delivered",
		CreatedAt:     time.Now().UTC(),
	}

	evt := &domain.MessageEvent{
		ID:          "evt_priv_1",
		TenantID:    tenantID,
		MessageID:   msgID,
		RecipientID: &rcptID,
		Type:        "message.delivered",
		OccurredAt:  time.Now().UTC(),
		CreatedAt:   time.Now().UTC(),
	}

	err := repo.SaveMessage(ctx, msg, []*domain.MessageRecipient{rcpt}, []*domain.MessageEvent{evt}, nil)
	if err != nil {
		t.Fatalf("failed to save message: %v", err)
	}

	err = repo.AddSuppression(ctx, &domain.Suppression{
		ID:            "sup_priv_1",
		TenantID:      tenantID,
		Address:       email,
		AddressDomain: "example.com",
		Type:          "manual",
		CreatedAt:     time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("failed to add suppression: %v", err)
	}

	// 1. Test Export
	exp, err := svc.RequestExport(ctx, tenantID, email)
	if err != nil {
		t.Fatalf("failed to request export: %v", err)
	}
	if exp.EmailAddress != email {
		t.Errorf("expected email %s, got %s", email, exp.EmailAddress)
	}
	if len(exp.Messages) != 1 {
		t.Errorf("expected 1 message recipient, got %d", len(exp.Messages))
	}
	if len(exp.Events) != 1 {
		t.Errorf("expected 1 event, got %d", len(exp.Events))
	}
	if len(exp.Suppressions) != 1 {
		t.Errorf("expected 1 suppression, got %d", len(exp.Suppressions))
	}

	// 2. Test Erasure
	req, err := svc.RequestErasure(ctx, tenantID, email)
	if err != nil {
		t.Fatalf("failed to request erasure: %v", err)
	}
	if req.Status != "completed" {
		t.Errorf("expected status completed, got %s", req.Status)
	}
	if req.ResultSummary == nil || !strings.Contains(*req.ResultSummary, "pseudonymized") {
		t.Errorf("expected result summary containing pseudonymized, got %v", req.ResultSummary)
	}

	// Verify recipient address in message is scrubbed
	storedMsg, recs, _, err := repo.GetMessage(ctx, tenantID, msgID)
	if err != nil {
		t.Fatalf("failed to get message: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("expected 1 recipient, got %d", len(recs))
	}
	if strings.Contains(recs[0].Address, email) {
		t.Fatalf("recipient address was not scrubbed: %s", recs[0].Address)
	}
	if storedMsg.BodyRef != nil || storedMsg.RawMIMERef != nil {
		t.Fatalf("message bodies and raw MIME were not cleared upon erasure")
	}

	// Verify suppression address is hashed
	sups, err := repo.ListSuppressions(ctx, tenantID, 10)
	if err != nil {
		t.Fatalf("failed to list suppressions: %v", err)
	}
	if len(sups) != 1 {
		t.Fatalf("expected 1 suppression, got %d", len(sups))
	}
	if strings.Contains(sups[0].Address, email) {
		t.Fatalf("cleartext email still present in suppressions: %s", sups[0].Address)
	}
	if !strings.HasPrefix(sups[0].Address, "erased-hash-") {
		t.Fatalf("expected erased-hash prefix, got %s", sups[0].Address)
	}

	// Check privacy requests list
	reqs, err := svc.ListRequests(ctx, tenantID)
	if err != nil {
		t.Fatalf("failed to list requests: %v", err)
	}
	if len(reqs) < 2 { // export + erasure
		t.Errorf("expected at least 2 privacy requests, got %d", len(reqs))
	}
}
