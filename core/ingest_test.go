package core_test

import (
	"context"
	"testing"
	"time"

	"github.com/senthilnasa/mailvex/core"
	"github.com/senthilnasa/mailvex/core/domain"
	"github.com/senthilnasa/mailvex/core/repository"
	"github.com/senthilnasa/mailvex/core/storage"
)

func TestIngestEngineRetainsCcAndBccRecipients(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemoryRepository()
	store := storage.NewMemoryStorage()
	engine := core.NewIngestionEngine(repo, store)

	// Setup organization, tenant, and verified domain
	orgID := "org_test_cc_bcc"
	_ = repo.CreateOrganization(ctx, &domain.Organization{
		ID:        orgID,
		Name:      "Test Org",
		Slug:      "test-org",
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	})

	tenantID := "tenant_test_cc_bcc"
	_ = repo.CreateTenant(ctx, &domain.Tenant{
		ID:             tenantID,
		OrganizationID: orgID,
		Name:           "Test Tenant",
		Slug:           "test-tenant",
		Environment:    "live",
		Status:         "active",
		CreatedAt:      time.Now().UTC(),
	})

	domID := "dom_test_cc_bcc"
	if err := repo.CreateDomain(ctx, &domain.Domain{
		ID:        domID,
		TenantID:  tenantID,
		Name:      "example.org",
		Status:    "verified",
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("failed to create domain: %v", err)
	}

	req := &core.IngestRequest{
		TenantID:    tenantID,
		Source:      "api",
		FromAddress: "mailer@example.org",
		To:          []string{"alice@example.com"},
		Cc:          []string{"bob@example.com", "carol@example.com"},
		Bcc:         []string{"secret@example.com"},
		Subject:     "Quarterly Report",
		BodyText:    "Please find attached the report.",
	}

	res, err := engine.Ingest(ctx, req)
	if err != nil {
		t.Fatalf("unexpected ingest error: %v", err)
	}

	rcpts, err := repo.GetMessageRecipients(ctx, tenantID, res.MessageID)
	if err != nil {
		t.Fatalf("failed to get message recipients: %v", err)
	}

	if len(rcpts) != 4 {
		t.Fatalf("expected 4 recipients (1 To + 2 Cc + 1 Bcc), got %d", len(rcpts))
	}

	toCount := 0
	ccCount := 0
	bccCount := 0
	for _, r := range rcpts {
		switch r.Type {
		case "to":
			toCount++
			if r.Address != "alice@example.com" {
				t.Errorf("unexpected To address: %s", r.Address)
			}
		case "cc":
			ccCount++
		case "bcc":
			bccCount++
			if r.Address != "secret@example.com" {
				t.Errorf("unexpected Bcc address: %s", r.Address)
			}
		default:
			t.Errorf("unknown recipient type: %s", r.Type)
		}
	}

	if toCount != 1 || ccCount != 2 || bccCount != 1 {
		t.Errorf("expected 1 To, 2 Cc, 1 Bcc; got %d To, %d Cc, %d Bcc", toCount, ccCount, bccCount)
	}
}

func TestParseCleanAddress(t *testing.T) {
	tests := []struct {
		input        string
		expectedAddr string
		expectedDom  string
		expectErr    bool
	}{
		{"senthil prabhu k <erpadmin@krea.edu.in>", "erpadmin@krea.edu.in", "krea.edu.in", false},
		{"<erpadmin@krea.edu.in>", "erpadmin@krea.edu.in", "krea.edu.in", false},
		{"erpadmin@krea.edu.in", "erpadmin@krea.edu.in", "krea.edu.in", false},
		{"No Reply <noreply@crm.krea.edu.in>", "noreply@crm.krea.edu.in", "crm.krea.edu.in", false},
		{"invalid-address", "", "", true},
		{"", "", "", true},
	}

	for _, tc := range tests {
		addr, dom, err := core.ParseCleanAddress(tc.input)
		if tc.expectErr {
			if err == nil {
				t.Errorf("expected error for %q, got nil", tc.input)
			}
		} else {
			if err != nil {
				t.Errorf("unexpected error for %q: %v", tc.input, err)
			}
			if addr != tc.expectedAddr {
				t.Errorf("for %q, expected addr %q, got %q", tc.input, tc.expectedAddr, addr)
			}
			if dom != tc.expectedDom {
				t.Errorf("for %q, expected dom %q, got %q", tc.input, tc.expectedDom, dom)
			}
		}
	}
}
