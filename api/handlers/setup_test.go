package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/senthilnasa/mailvex/api/handlers"
	"github.com/senthilnasa/mailvex/core/audit"
	"github.com/senthilnasa/mailvex/core/domain"
	"github.com/senthilnasa/mailvex/core/repository"
)

func TestSetupExecuteHandler_IdempotenceAndSelfHealing(t *testing.T) {
	repo := repository.NewMemoryRepository()
	auditSvc := audit.NewService(repo)
	// Pass nil encryption to verify automatic fallback
	api := handlers.NewAPI(repo, auditSvc, nil)

	payload := map[string]string{
		"admin_email":    "admin@mailvex.local",
		"admin_password": "supersecretpassword123!",
		"org_name":       "Acme Corp",
		"tenant_name":    "Acme Prod",
		"profile":        "homelab",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/v1/setup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	api.SetupExecuteHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["organization_id"] == nil || resp["admin_user_id"] == nil || resp["tenant_id"] == nil {
		t.Fatalf("expected org, user, and tenant IDs in response, got %v", resp)
	}

	// Verify that repeating setup returns 409 Conflict (setup_already_completed)
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("POST", "/v1/setup", bytes.NewReader(body))
	api.SetupExecuteHandler(w2, req2)

	if w2.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict on repeated setup, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestSetupExecuteHandler_PreExistingResourcesRecovery(t *testing.T) {
	repo := repository.NewMemoryRepository()
	auditSvc := audit.NewService(repo)
	api := handlers.NewAPI(repo, auditSvc, nil)

	// Pre-create org and user simulating an interrupted first setup run
	_ = repo.CreateOrganization(t.Context(), &domain.Organization{
		ID:     "org_existing",
		Name:   "Acme Corp",
		Slug:   "acme-corp",
		Status: "active",
	})
	_ = repo.CreateUser(t.Context(), &domain.User{
		ID:             "user_existing",
		OrganizationID: "org_existing",
		Email:          "admin@mailvex.local",
		PasswordHash:   "hash",
		Status:         "active",
	})

	payload := map[string]string{
		"admin_email":    "admin@mailvex.local",
		"admin_password": "supersecretpassword123!",
		"org_name":       "Acme Corp",
		"tenant_name":    "Acme Prod",
		"profile":        "homelab",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/v1/setup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	api.SetupExecuteHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201 Created even with pre-existing resources, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["organization_id"] != "org_existing" || resp["admin_user_id"] != "user_existing" {
		t.Fatalf("expected existing org and user to be reused, got %v", resp)
	}
}

func TestResetAdminHandler(t *testing.T) {
	repo := repository.NewMemoryRepository()
	auditSvc := audit.NewService(repo)
	api := handlers.NewAPI(repo, auditSvc, nil)

	// Call ResetAdminHandler to seed platform admin
	payload := map[string]string{
		"email":    "ADMIN@EXAMPLE.COM",
		"password": "NewSecretPassword123!",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/v1/auth/reset-admin", bytes.NewReader(body))
	w := httptest.NewRecorder()
	api.ResetAdminHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	// Verify login works case-insensitively with the new password
	loginPayload := map[string]string{
		"email":    "admin@example.com",
		"password": "NewSecretPassword123!",
	}
	loginBody, _ := json.Marshal(loginPayload)
	loginReq := httptest.NewRequest("POST", "/v1/auth/login", bytes.NewReader(loginBody))
	loginW := httptest.NewRecorder()
	api.LoginHandler(loginW, loginReq)

	if loginW.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on login, got %d: %s", loginW.Code, loginW.Body.String())
	}
}

