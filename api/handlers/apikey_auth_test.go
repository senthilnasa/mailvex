package handlers_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/senthilnasa/mailvex/api/handlers"
	"github.com/senthilnasa/mailvex/api/middleware"
	"github.com/senthilnasa/mailvex/core/audit"
	"github.com/senthilnasa/mailvex/core/domain"
	"github.com/senthilnasa/mailvex/core/repository"
)

func TestHandlersWithAPIKeyContext_NoUserPanic(t *testing.T) {
	repo := repository.NewMemoryRepository()
	auditSvc := audit.NewService(repo)
	api := handlers.NewAPI(repo, auditSvc, nil)

	ctx := context.Background()
	orgID := "org_api_key_test"
	_ = repo.CreateOrganization(ctx, &domain.Organization{
		ID:        orgID,
		Name:      "Test Org",
		Slug:      "test-org",
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	})

	tenantID := "tenant_api_key_test"
	_ = repo.CreateTenant(ctx, &domain.Tenant{
		ID:             tenantID,
		OrganizationID: orgID,
		Name:           "Test Tenant",
		Slug:           "test-tenant",
		Environment:    "live",
		Status:         "active",
		CreatedAt:      time.Now().UTC(),
	})

	apiKey := &domain.APIKey{
		ID:          "key_test_123",
		TenantID:    tenantID,
		Name:        "Test Key",
		Environment: "live",
		Scopes:      []string{"*"},
	}

	// 1. Test ListTenantsHandler with API Key
	req1 := httptest.NewRequest("GET", "/v1/tenants", nil)
	ctx1 := context.WithValue(req1.Context(), middleware.APIKeyKey, apiKey)
	ctx1 = context.WithValue(ctx1, middleware.TenantIDKey, tenantID)
	req1 = req1.WithContext(ctx1)
	w1 := httptest.NewRecorder()

	api.ListTenantsHandler(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for ListTenants with API Key, got %d: %s", w1.Code, w1.Body.String())
	}

	// 2. Test CreateSMTPCredentialHandler with API Key (should not panic on &user.ID)
	req2 := httptest.NewRequest("POST", "/v1/smtp-credentials", bytes.NewReader([]byte(`{"username":"device_camera"}`)))
	ctx2 := context.WithValue(req2.Context(), middleware.APIKeyKey, apiKey)
	ctx2 = context.WithValue(ctx2, middleware.TenantIDKey, tenantID)
	req2 = req2.WithContext(ctx2)
	w2 := httptest.NewRecorder()

	api.CreateSMTPCredentialHandler(w2, req2)
	if w2.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for CreateSMTPCredential with API Key, got %d: %s", w2.Code, w2.Body.String())
	}

	// 3. Test CreateDomainHandler with API Key (should not panic on &user.ID)
	req3 := httptest.NewRequest("POST", "/v1/domains", bytes.NewReader([]byte(`{"name":"test-api-domain.com","type":"public"}`)))
	ctx3 := context.WithValue(req3.Context(), middleware.APIKeyKey, apiKey)
	ctx3 = context.WithValue(ctx3, middleware.TenantIDKey, tenantID)
	req3 = req3.WithContext(ctx3)
	w3 := httptest.NewRecorder()

	api.CreateDomainHandler(w3, req3)
	if w3.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for CreateDomain with API Key, got %d: %s", w3.Code, w3.Body.String())
	}
}
