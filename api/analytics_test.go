package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mailvex/mailvex/core/audit"
	"github.com/mailvex/mailvex/core/crypto"
	"github.com/mailvex/mailvex/core/domain"
	"github.com/mailvex/mailvex/core/repository"
)

func setupAnalyticsTestEnvironment(t *testing.T) (*Server, repository.Repository, string, string) {
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
		Name:      "Analytics Org",
		Slug:      "analytics-org",
		CreatedAt: time.Now().UTC(),
	})

	tenantID := crypto.NewULID(crypto.PrefixTenant)
	_ = repo.CreateTenant(ctx, &domain.Tenant{
		ID:             tenantID,
		OrganizationID: orgID,
		Name:           "Analytics Tenant",
		Slug:           "analytics-tenant",
		Status:         "active",
		CreatedAt:      time.Now().UTC(),
	})

	userID := crypto.NewULID(crypto.PrefixUser)
	_ = repo.CreateUser(ctx, &domain.User{
		ID:             userID,
		OrganizationID: orgID,
		Email:          "admin@analytics.local",
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

func TestAnalyticsHTTPEndpoints(t *testing.T) {
	server, repo, tenantID, userID := setupAnalyticsTestEnvironment(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// Seed rollups: 100 sent, 90 delivered, 10 bounced (8 hard, 2 soft), 1 complaint
	for i := 0; i < 100; i++ {
		prov := "google"
		if i >= 50 {
			prov = "microsoft"
		}
		_ = repo.RecordAnalyticsEvent(ctx, tenantID, "message.sent", nil, nil, prov, "welcome", 0, now)
	}

	for i := 0; i < 90; i++ {
		prov := "google"
		latency := 3.2
		if i >= 50 {
			prov = "microsoft"
			latency = 25.0
		}
		_ = repo.RecordAnalyticsEvent(ctx, tenantID, "message.delivered", nil, nil, prov, "welcome", latency, now)
	}

	for i := 0; i < 8; i++ {
		_ = repo.RecordAnalyticsEvent(ctx, tenantID, "hard_bounce", nil, nil, "google", "welcome", 0, now)
	}
	for i := 0; i < 2; i++ {
		_ = repo.RecordAnalyticsEvent(ctx, tenantID, "soft_bounce", nil, nil, "microsoft", "welcome", 0, now)
	}
	_ = repo.RecordAnalyticsEvent(ctx, tenantID, "message.complained", nil, nil, "google", "welcome", 0, now)

	// 1. Test GET /v1/analytics/overview
	reqOverview := httptest.NewRequest(http.MethodGet, "/v1/analytics/overview", nil)
	reqOverview.Header.Set("X-User-ID", userID)
	reqOverview.Header.Set("X-Tenant-ID", tenantID)
	wOverview := httptest.NewRecorder()
	server.Handler().ServeHTTP(wOverview, reqOverview)

	if wOverview.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on GET /v1/analytics/overview, got %d: %s", wOverview.Code, wOverview.Body.String())
	}

	var summary domain.AnalyticsSummary
	_ = json.Unmarshal(wOverview.Body.Bytes(), &summary)
	if summary.Sent != 100 || summary.Delivered != 90 {
		t.Fatalf("expected 100 sent and 90 delivered, got sent=%d, delivered=%d", summary.Sent, summary.Delivered)
	}
	if summary.DeliveryRate != 0.90 {
		t.Fatalf("expected 0.90 delivery rate, got %f", summary.DeliveryRate)
	}
	if summary.BounceRate != 0.10 {
		t.Fatalf("expected 0.10 bounce rate, got %f", summary.BounceRate)
	}
	if !summary.ComplaintWarning {
		t.Fatalf("expected ComplaintWarning to be true (complaint rate 1/90 > 0.10%%)")
	}

	// 2. Test GET /v1/analytics/time-series
	reqTS := httptest.NewRequest(http.MethodGet, "/v1/analytics/time-series?interval=1h", nil)
	reqTS.Header.Set("X-User-ID", userID)
	reqTS.Header.Set("X-Tenant-ID", tenantID)
	wTS := httptest.NewRecorder()
	server.Handler().ServeHTTP(wTS, reqTS)

	if wTS.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on GET /v1/analytics/time-series, got %d", wTS.Code)
	}

	var tsResp map[string][]domain.TimeSeriesPoint
	_ = json.Unmarshal(wTS.Body.Bytes(), &tsResp)
	if len(tsResp["data"]) == 0 {
		t.Fatalf("expected at least 1 time series point")
	}

	// 3. Test GET /v1/analytics/providers
	reqProv := httptest.NewRequest(http.MethodGet, "/v1/analytics/providers", nil)
	reqProv.Header.Set("X-User-ID", userID)
	reqProv.Header.Set("X-Tenant-ID", tenantID)
	wProv := httptest.NewRecorder()
	server.Handler().ServeHTTP(wProv, reqProv)

	if wProv.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on GET /v1/analytics/providers, got %d", wProv.Code)
	}

	var provResp map[string][]domain.ProviderBreakdown
	_ = json.Unmarshal(wProv.Body.Bytes(), &provResp)
	if len(provResp["data"]) < 2 {
		t.Fatalf("expected breakdown for google and microsoft, got %d", len(provResp["data"]))
	}

	// 4. Test GET /v1/analytics/latency
	reqLat := httptest.NewRequest(http.MethodGet, "/v1/analytics/latency", nil)
	reqLat.Header.Set("X-User-ID", userID)
	reqLat.Header.Set("X-Tenant-ID", tenantID)
	wLat := httptest.NewRecorder()
	server.Handler().ServeHTTP(wLat, reqLat)

	if wLat.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on GET /v1/analytics/latency, got %d", wLat.Code)
	}

	var hist domain.LatencyHistogram
	_ = json.Unmarshal(wLat.Body.Bytes(), &hist)
	if hist.Under5s != 50 || hist.Under30s != 40 {
		t.Fatalf("expected 50 under 5s and 40 under 30s, got under5s=%d, under30s=%d", hist.Under5s, hist.Under30s)
	}

	// 5. Test POST /v1/analytics/export (CSV export)
	reqExp := httptest.NewRequest(http.MethodPost, "/v1/analytics/export", strings.NewReader(`{"format":"csv"}`))
	reqExp.Header.Set("X-User-ID", userID)
	reqExp.Header.Set("X-Tenant-ID", tenantID)
	reqExp.Header.Set("Content-Type", "application/json")
	wExp := httptest.NewRecorder()
	server.Handler().ServeHTTP(wExp, reqExp)

	if wExp.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created on export, got %d: %s", wExp.Code, wExp.Body.String())
	}

	var exp domain.AnalyticsExport
	_ = json.Unmarshal(wExp.Body.Bytes(), &exp)
	if exp.Status != "completed" || exp.FileURL == nil {
		t.Fatalf("expected completed export with file URL")
	}

	// 6. Test GET /v1/analytics/exports/{id}/download
	downloadPath := fmt.Sprintf("/v1/analytics/exports/%s/download", exp.ID)
	reqDown := httptest.NewRequest(http.MethodGet, downloadPath, nil)
	reqDown.Header.Set("X-User-ID", userID)
	reqDown.Header.Set("X-Tenant-ID", tenantID)
	wDown := httptest.NewRecorder()
	server.Handler().ServeHTTP(wDown, reqDown)

	if wDown.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on CSV download, got %d: %s", wDown.Code, wDown.Body.String())
	}
	if wDown.Header().Get("Content-Type") != "text/csv; charset=utf-8" {
		t.Fatalf("expected text/csv Content-Type, got: %s", wDown.Header().Get("Content-Type"))
	}
	csvContent := wDown.Body.String()
	if !strings.Contains(csvContent, "delivery_rate_pct") {
		t.Fatalf("expected CSV content, got: %s", csvContent)
	}
}
