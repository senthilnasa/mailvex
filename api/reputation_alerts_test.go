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

func setupTestServerForReputationAlerts(t *testing.T) (*Server, repository.Repository, string, string) {
	repo := repository.NewMemoryRepository()
	auditSvc := audit.NewService(repo)
	kek := bytes.Repeat([]byte{0x06}, 32)
	enc, _ := crypto.NewEnvelopeEncryption(kek)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	srv := NewServer(repo, auditSvc, enc, logger)

	ctx := context.Background()
	orgID := crypto.NewULID(crypto.PrefixOrg)
	_ = repo.CreateOrganization(ctx, &domain.Organization{
		ID:        orgID,
		Name:      "Phase 11 Org",
		Slug:      "phase11-org",
		CreatedAt: time.Now().UTC(),
	})

	tenantID := crypto.NewULID(crypto.PrefixTenant)
	userID := crypto.NewULID(crypto.PrefixUser)

	_ = repo.CreateTenant(ctx, &domain.Tenant{
		ID:             tenantID,
		OrganizationID: orgID,
		Name:           "Phase 11 Tenant",
		Slug:           "phase11-test",
		Environment:    "live",
		Status:         "active",
		CreatedAt:      time.Now().UTC(),
	})
	_ = repo.CreateUser(ctx, &domain.User{
		ID:             userID,
		OrganizationID: orgID,
		Email:          "admin@phase11.test",
		CreatedAt:      time.Now().UTC(),
	})
	_ = repo.AddTenantMembership(ctx, &domain.TenantMembership{
		ID:        crypto.NewULID(crypto.PrefixMembership),
		TenantID:  tenantID,
		UserID:    userID,
		RoleID:    "role_tenant_owner",
		CreatedAt: time.Now().UTC(),
	})

	return srv, repo, tenantID, userID
}

func TestReputationAndDNSBLEndpoints(t *testing.T) {
	srv, repo, tenantID, userID := setupTestServerForReputationAlerts(t)

	// 1. GET /v1/reputation/overview
	req, _ := http.NewRequest("GET", "/v1/reputation/overview", nil)
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-Tenant-ID", tenantID)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}
	var overview domain.ReputationOverview
	if err := json.Unmarshal(rec.Body.Bytes(), &overview); err != nil {
		t.Fatalf("failed to decode reputation overview: %v", err)
	}
	if overview.OverallStatus == "" {
		t.Errorf("expected overall_status in overview")
	}

	// 2. GET /v1/dnsbl/providers
	req, _ = http.NewRequest("GET", "/v1/dnsbl/providers", nil)
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-Tenant-ID", tenantID)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}
	var provResp map[string][]domain.DNSBLProvider
	_ = json.Unmarshal(rec.Body.Bytes(), &provResp)
	if len(provResp["data"]) < 4 {
		t.Errorf("expected seeded providers, got %d", len(provResp["data"]))
	}

	// 3. POST /v1/dnsbl/check
	checkPayload := map[string]string{
		"ip_address_id": "ip_1",
		"ip":            "192.0.2.1",
	}
	checkBody, _ := json.Marshal(checkPayload)
	req, _ = http.NewRequest("POST", "/v1/dnsbl/check", bytes.NewReader(checkBody))
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from dnsbl check, got %d: %s", rec.Code, rec.Body.String())
	}

	// 4. GET /v1/dnsbl/results
	req, _ = http.NewRequest("GET", "/v1/dnsbl/results?ip_address_id=ip_1", nil)
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-Tenant-ID", tenantID)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from dnsbl results, got %d", rec.Code)
	}
	var resResp map[string][]domain.DNSBLResult
	_ = json.Unmarshal(rec.Body.Bytes(), &resResp)
	if len(resResp["data"]) == 0 {
		t.Errorf("expected check results returned")
	}
	_ = repo
}

func TestWarmupEndpoints(t *testing.T) {
	srv, _, tenantID, userID := setupTestServerForReputationAlerts(t)

	// 1. GET /v1/warmup/plans
	req, _ := http.NewRequest("GET", "/v1/warmup/plans", nil)
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-Tenant-ID", tenantID)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	// 2. POST /v1/warmup/ips (enroll IP)
	enrollPayload := map[string]string{
		"ip_address_id": "ip_warm_1",
		"ip":            "198.51.100.25",
		"plan_id":       "wplan_default",
	}
	body, _ := json.Marshal(enrollPayload)
	req, _ = http.NewRequest("POST", "/v1/warmup/ips", bytes.NewReader(body))
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created from enroll warmup, got %d: %s", rec.Code, rec.Body.String())
	}
	var state domain.IPWarmupState
	_ = json.Unmarshal(rec.Body.Bytes(), &state)
	if state.CurrentDay != 1 || state.Status != "warming" {
		t.Errorf("unexpected warmup state: %+v", state)
	}

	// 3. POST /v1/warmup/ips/{id}/hold
	holdBody, _ := json.Marshal(map[string]string{"reason": "Reputation dip"})
	req, _ = http.NewRequest("POST", "/v1/warmup/ips/ip_warm_1/hold", bytes.NewReader(holdBody))
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from hold, got %d", rec.Code)
	}

	// 4. POST /v1/warmup/ips/{id}/resume
	req, _ = http.NewRequest("POST", "/v1/warmup/ips/ip_warm_1/resume", nil)
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-Tenant-ID", tenantID)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from resume, got %d", rec.Code)
	}

	// 5. POST /v1/warmup/ips/{id}/advance
	advBody, _ := json.Marshal(map[string]bool{"force": true})
	req, _ = http.NewRequest("POST", "/v1/warmup/ips/ip_warm_1/advance", bytes.NewReader(advBody))
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from advance, got %d", rec.Code)
	}
	var stateAdv domain.IPWarmupState
	_ = json.Unmarshal(rec.Body.Bytes(), &stateAdv)
	if stateAdv.CurrentDay != 2 {
		t.Errorf("expected day 2, got %d", stateAdv.CurrentDay)
	}
}

func TestAlertsAndNotificationEndpoints(t *testing.T) {
	srv, _, tenantID, userID := setupTestServerForReputationAlerts(t)

	// 1. POST /v1/alerts/channels
	chanPayload := map[string]any{
		"name": "Ops Webhook",
		"type": "generic_webhook",
		"config": `{"url":"https://example.com/alerts","secret":"shh"}`,
	}
	chBody, _ := json.Marshal(chanPayload)
	req, _ := http.NewRequest("POST", "/v1/alerts/channels", bytes.NewReader(chBody))
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created from create channel, got %d: %s", rec.Code, rec.Body.String())
	}
	var ch domain.NotificationChannel
	_ = json.Unmarshal(rec.Body.Bytes(), &ch)

	// 2. POST /v1/alerts/rules
	rulePayload := map[string]any{
		"name":                "High Bounce Rate",
		"metric_type":         "bounce_rate",
		"comparison_operator": ">",
		"threshold":           0.05,
		"severity":            "critical",
		"cooldown_minutes":    30,
		"channels":            []string{ch.ID},
	}
	ruleBody, _ := json.Marshal(rulePayload)
	req, _ = http.NewRequest("POST", "/v1/alerts/rules", bytes.NewReader(ruleBody))
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created from create rule, got %d: %s", rec.Code, rec.Body.String())
	}

	// 3. POST /v1/alerts/evaluate
	req, _ = http.NewRequest("POST", "/v1/alerts/evaluate", nil)
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-Tenant-ID", tenantID)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from evaluate alerts, got %d", rec.Code)
	}

	// 4. GET /v1/alerts/notifications
	req, _ = http.NewRequest("GET", "/v1/alerts/notifications", nil)
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-Tenant-ID", tenantID)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from list notifications, got %d", rec.Code)
	}
}
