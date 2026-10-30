package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/senthilnasa/mailvex/core/crypto"
	"github.com/senthilnasa/mailvex/core/domain"
	"github.com/senthilnasa/mailvex/core/repository"
)

func TestPersistentMemoryRepositoryRestart(t *testing.T) {
	tempDir := t.TempDir()
	ctx := context.Background()

	// 1. Initial repository instance
	repo1, err := repository.NewPersistentMemoryRepository(tempDir)
	if err != nil {
		t.Fatalf("failed to create repo1: %v", err)
	}

	orgID := "org_test"
	if err := repo1.CreateOrganization(ctx, &domain.Organization{
		ID:        orgID,
		Name:      "Test Org",
		Slug:      "test-org",
		Status:    "active",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("failed to create org: %v", err)
	}

	tenantID := "tenant_test"
	if err := repo1.CreateTenant(ctx, &domain.Tenant{
		ID:             tenantID,
		OrganizationID: orgID,
		Name:           "Test Tenant",
		Slug:           "test-tenant",
		Environment:    "live",
		Status:         "active",
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}); err != nil {
		t.Fatalf("failed to create tenant: %v", err)
	}

	domainID := "dom_test_123"
	if err := repo1.CreateDomain(ctx, &domain.Domain{
		ID:        domainID,
		TenantID:  tenantID,
		Name:      "crm.krea.edu.in",
		Type:      "public",
		Status:    "verified",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("failed to create domain: %v", err)
	}

	if err := repo1.SaveDKIMKey(ctx, &domain.DKIMKey{
		ID:                  "key_dkim_123",
		DomainID:            domainID,
		TenantID:            tenantID,
		Selector:            "mvx",
		Algorithm:           "rsa2048",
		EncryptedPrivateKey: "test-priv-key",
		PublicKey:           "test-pub-key",
		Status:              "active",
		CreatedAt:           time.Now().UTC(),
	}); err != nil {
		t.Fatalf("failed to save dkim key: %v", err)
	}

	apiKeyID := "key_api_123"
	token := "mvx_live_zrUDzzGQoy3wh4xJpMoOHsMD1SwAPsft"
	keyHash := crypto.HashAPIKey(token)
	if err := repo1.CreateAPIKey(ctx, &domain.APIKey{
		ID:          apiKeyID,
		TenantID:    tenantID,
		Name:        "Test Live Key",
		Prefix:      token[:12],
		KeyHash:     keyHash,
		LastFour:    token[len(token)-4:],
		Scopes:      []string{"send.email", "messages.read"},
		Environment: "live",
		RateLimit:   100,
		CreatedAt:   time.Now().UTC(),
	}); err != nil {
		t.Fatalf("failed to create api key: %v", err)
	}

	// Flush and close repo1
	if err := repo1.Close(); err != nil {
		t.Fatalf("failed to close repo1: %v", err)
	}

	// 2. Reopen repository from same directory (simulating container restart)
	repo2, err := repository.NewPersistentMemoryRepository(tempDir)
	if err != nil {
		t.Fatalf("failed to create repo2 from persisted directory: %v", err)
	}
	defer repo2.Close()

	// Verify org restored
	org, err := repo2.GetOrganization(ctx, orgID)
	if err != nil || org.Name != "Test Org" {
		t.Fatalf("expected org 'Test Org', got: %v (err: %v)", org, err)
	}

	// Verify tenant restored
	tenant, err := repo2.GetTenant(ctx, tenantID)
	if err != nil || tenant.Name != "Test Tenant" {
		t.Fatalf("expected tenant 'Test Tenant', got: %v (err: %v)", tenant, err)
	}

	// Verify domain restored
	dom, err := repo2.GetDomain(ctx, tenantID, domainID)
	if err != nil || dom.Name != "crm.krea.edu.in" || dom.Status != "verified" {
		t.Fatalf("expected verified domain 'crm.krea.edu.in', got: %v (err: %v)", dom, err)
	}

	// Verify DKIM key restored
	dkimKey, err := repo2.GetActiveDKIMKey(ctx, domainID)
	if err != nil || dkimKey.Selector != "mvx" || dkimKey.PublicKey != "test-pub-key" {
		t.Fatalf("expected active dkim key, got: %v (err: %v)", dkimKey, err)
	}

	// Verify API key restored and hash lookup works
	apiKey, err := repo2.GetAPIKeyByHash(ctx, keyHash)
	if err != nil || apiKey.ID != apiKeyID {
		t.Fatalf("expected api key lookup to succeed for restored hash, got: %v (err: %v)", apiKey, err)
	}
}
