package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/senthilnasa/mailvex/core/audit"
	"github.com/senthilnasa/mailvex/core/crypto"
	"github.com/senthilnasa/mailvex/core/domain"
	"github.com/senthilnasa/mailvex/core/repository"
)

func setupWebhookTestEnvironment(t *testing.T) (*Server, repository.Repository, string, string) {
	repo := repository.NewMemoryRepository()
	auditSvc := audit.NewService(repo)
	kek := bytes.Repeat([]byte{0x04}, 32)
	enc, _ := crypto.NewEnvelopeEncryption(kek)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	server := NewServer(repo, auditSvc, enc, logger)

	ctx := context.Background()

	orgID := crypto.NewULID(crypto.PrefixOrg)
	_ = repo.CreateOrganization(ctx, &domain.Organization{
		ID:        orgID,
		Name:      "Webhook Org",
		Slug:      "webhook-org",
		CreatedAt: time.Now().UTC(),
	})

	tenantID := crypto.NewULID(crypto.PrefixTenant)
	_ = repo.CreateTenant(ctx, &domain.Tenant{
		ID:             tenantID,
		OrganizationID: orgID,
		Name:           "Webhook Tenant",
		Slug:           "webhook-tenant",
		Status:         "active",
		CreatedAt:      time.Now().UTC(),
	})

	userID := crypto.NewULID(crypto.PrefixUser)
	_ = repo.CreateUser(ctx, &domain.User{
		ID:             userID,
		OrganizationID: orgID,
		Email:          "dev@webhook.local",
		Status:         "active",
		CreatedAt:      time.Now().UTC(),
	})

	_ = repo.AddTenantMembership(ctx, &domain.TenantMembership{
		ID:        crypto.NewULID(crypto.PrefixMembership),
		TenantID:  tenantID,
		UserID:    userID,
		RoleID:    "role_tenant_owner",
		CreatedAt: time.Now().UTC(),
	})

	return server, repo, tenantID, userID
}

func TestWebhookHTTPEndpoints(t *testing.T) {
	server, _, tenantID, userID := setupWebhookTestEnvironment(t)

	// Setup mock webhook receiver server
	mockReceiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer mockReceiver.Close()

	// 1. Create Webhook Endpoint (POST /v1/webhooks)
	createBody, _ := json.Marshal(map[string]any{
		"url":         mockReceiver.URL,
		"description": "Production webhook receiver",
		"events":      []string{"message.delivered", "message.bounced"},
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("X-User-ID", userID)

	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	var createResp struct {
		Data domain.WebhookEndpoint `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &createResp)
	endpointID := createResp.Data.ID
	if endpointID == "" || createResp.Data.Secret == "" {
		t.Fatalf("expected valid endpoint ID and secret, got %+v", createResp.Data)
	}

	// 2. List Webhooks (GET /v1/webhooks)
	req = httptest.NewRequest(http.MethodGet, "/v1/webhooks", nil)
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("X-User-ID", userID)
	w = httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}
	var listResp struct {
		Data []domain.WebhookEndpoint `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &listResp)
	if len(listResp.Data) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(listResp.Data))
	}

	// 3. Get Webhook (GET /v1/webhooks/{id})
	req = httptest.NewRequest(http.MethodGet, "/v1/webhooks/"+endpointID, nil)
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("X-User-ID", userID)
	w = httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	// 4. Update Webhook (PATCH /v1/webhooks/{id})
	updatedDesc := "Updated production receiver"
	updateBody, _ := json.Marshal(map[string]any{
		"description": updatedDesc,
		"events":      []string{"*"},
	})
	req = httptest.NewRequest(http.MethodPatch, "/v1/webhooks/"+endpointID, bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("X-User-ID", userID)
	w = httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on update, got %d: %s", w.Code, w.Body.String())
	}

	// 5. Rotate Secret (POST /v1/webhooks/{id}/rotate-secret)
	req = httptest.NewRequest(http.MethodPost, "/v1/webhooks/"+endpointID+"/rotate-secret", nil)
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("X-User-ID", userID)
	w = httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on rotate-secret, got %d: %s", w.Code, w.Body.String())
	}
	var rotateResp struct {
		Data             domain.WebhookEndpoint `json:"data"`
		OverlapExpiresAt string                 `json:"overlap_expires_at"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &rotateResp)
	if rotateResp.Data.PreviousSecret == nil || *rotateResp.Data.PreviousSecret != createResp.Data.Secret {
		t.Fatalf("expected previous_secret to match original secret, got %v", rotateResp.Data.PreviousSecret)
	}

	// 6. Test Ping (POST /v1/webhooks/{id}/test)
	req = httptest.NewRequest(http.MethodPost, "/v1/webhooks/"+endpointID+"/test", nil)
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("X-User-ID", userID)
	w = httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on test ping, got %d: %s", w.Code, w.Body.String())
	}

	// 7. List Deliveries (GET /v1/webhooks/{id}/deliveries)
	req = httptest.NewRequest(http.MethodGet, "/v1/webhooks/"+endpointID+"/deliveries", nil)
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("X-User-ID", userID)
	w = httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on list deliveries, got %d: %s", w.Code, w.Body.String())
	}
	var deliveriesResp struct {
		Data []domain.WebhookDelivery `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &deliveriesResp)
	if len(deliveriesResp.Data) != 1 {
		t.Fatalf("expected 1 delivery record, got %d", len(deliveriesResp.Data))
	}
	if deliveriesResp.Data[0].EventType != "webhook.test" {
		t.Fatalf("expected webhook.test event type, got %s", deliveriesResp.Data[0].EventType)
	}

	// 8. Delete Webhook (DELETE /v1/webhooks/{id})
	req = httptest.NewRequest(http.MethodDelete, "/v1/webhooks/"+endpointID, nil)
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("X-User-ID", userID)
	w = httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on delete, got %d: %s", w.Code, w.Body.String())
	}

	// Verify deleted
	req = httptest.NewRequest(http.MethodGet, "/v1/webhooks/"+endpointID, nil)
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("X-User-ID", userID)
	w = httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found after delete, got %d", w.Code)
	}
}
