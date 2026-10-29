package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/senthilnasa/mailvex/core/audit"
	"github.com/senthilnasa/mailvex/core/crypto"
	"github.com/senthilnasa/mailvex/core/logger"
	"github.com/senthilnasa/mailvex/core/repository"
)

func setupTestServer(t *testing.T) (*Server, repository.Repository) {
	repo := repository.NewMemoryRepository()
	auditService := audit.NewService(repo)
	kek := []byte("01234567890123456789012345678901") // 32 bytes
	enc, err := crypto.NewEnvelopeEncryption(kek)
	if err != nil {
		t.Fatalf("failed to create envelope encryption: %v", err)
	}
	appLogger := logger.New("error", true)
	server := NewServer(repo, auditService, enc, appLogger)
	return server, repo
}

func TestHealthAndReadiness(t *testing.T) {
	server, _ := setupTestServer(t)

	// Liveness
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on /health, got %d", w.Code)
	}

	// Readiness
	req = httptest.NewRequest("GET", "/ready", nil)
	w = httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on /ready, got %d", w.Code)
	}
}

func TestSetupWizardAndAuthFlow(t *testing.T) {
	server, repo := setupTestServer(t)

	// Step 1: Check setup status initially (should be false)
	req := httptest.NewRequest("GET", "/v1/setup", nil)
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on /v1/setup, got %d", w.Code)
	}
	var statusResp map[string]any
	_ = json.NewDecoder(w.Body).Decode(&statusResp)
	if statusResp["setup_completed"] != false {
		t.Fatalf("expected setup_completed = false initially")
	}

	// Step 2: Execute setup wizard
	setupPayload := map[string]string{
		"profile":        "standard",
		"admin_email":    "admin@example.com",
		"admin_password": "SuperSecretPassword123!",
		"org_name":       "Primary Org",
		"tenant_name":    "Production Tenant",
	}
	body, _ := json.Marshal(setupPayload)
	req = httptest.NewRequest("POST", "/v1/setup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 on /v1/setup execution, got %d body=%s", w.Code, w.Body.String())
	}

	var setupRes map[string]any
	_ = json.NewDecoder(w.Body).Decode(&setupRes)
	adminUserID := setupRes["admin_user_id"].(string)
	tenantID := setupRes["tenant_id"].(string)

	// Step 3: Attempting setup again should return 409 Conflict
	req = httptest.NewRequest("POST", "/v1/setup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 conflict when repeating setup, got %d", w.Code)
	}

	// Step 4: Login with admin credentials
	loginPayload := map[string]string{
		"email":    "admin@example.com",
		"password": "SuperSecretPassword123!",
	}
	loginBody, _ := json.Marshal(loginPayload)
	req = httptest.NewRequest("POST", "/v1/auth/login", bytes.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on login, got %d body=%s", w.Code, w.Body.String())
	}

	// Step 5: Test /v1/auth/me
	req = httptest.NewRequest("GET", "/v1/auth/me", nil)
	req.Header.Set("X-User-ID", adminUserID)
	req.Header.Set("X-Tenant-ID", tenantID)
	w = httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on /v1/auth/me, got %d", w.Code)
	}

	// Step 6: Create another tenant
	newTenantPayload := map[string]string{
		"name": "Staging Tenant",
		"slug": "staging",
	}
	tenantBody, _ := json.Marshal(newTenantPayload)
	req = httptest.NewRequest("POST", "/v1/tenants", bytes.NewReader(tenantBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", adminUserID)
	req.Header.Set("X-Tenant-ID", tenantID)
	w = httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 on /v1/tenants, got %d body=%s", w.Code, w.Body.String())
	}

	// Step 7: Check audit logs recorded
	req = httptest.NewRequest("GET", "/v1/audit-logs", nil)
	req.Header.Set("X-User-ID", adminUserID)
	w = httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on /v1/audit-logs, got %d", w.Code)
	}
	var auditResp map[string]any
	_ = json.NewDecoder(w.Body).Decode(&auditResp)
	logs := auditResp["data"].([]any)
	if len(logs) == 0 {
		t.Fatalf("expected audit logs to be recorded")
	}

	// Verify tenant DEK was created and encrypted with envelope encryption
	activeKey, err := repo.GetActiveTenantKey(req.Context(), tenantID)
	if err != nil || activeKey.EncryptedDEK == "" {
		t.Fatalf("expected tenant key to be created with encrypted DEK: %v", err)
	}
}
