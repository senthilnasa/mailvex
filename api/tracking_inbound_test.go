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

	"github.com/mailvex/mailvex/core/audit"
	"github.com/mailvex/mailvex/core/crypto"
	"github.com/mailvex/mailvex/core/domain"
	"github.com/mailvex/mailvex/core/repository"
	"github.com/mailvex/mailvex/core/tracking"
)

func setupTrackingInboundTestEnv(t *testing.T) (*Server, repository.Repository, *tracking.TrackingEngine, string, string) {
	repo := repository.NewMemoryRepository()
	auditSvc := audit.NewService(repo)
	kek := bytes.Repeat([]byte{0x06}, 32)
	enc, _ := crypto.NewEnvelopeEncryption(kek)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	server := NewServer(repo, auditSvc, enc, logger)

	trackSecret := []byte("mailvex-default-tracking-hmac-secret-key-32")
	engine := tracking.NewTrackingEngine(trackSecret, repo)

	ctx := context.Background()

	orgID := crypto.NewULID(crypto.PrefixOrg)
	_ = repo.CreateOrganization(ctx, &domain.Organization{
		ID:        orgID,
		Name:      "Track Org",
		Slug:      "track-org",
		CreatedAt: time.Now().UTC(),
	})

	tenantID := crypto.NewULID(crypto.PrefixTenant)
	_ = repo.CreateTenant(ctx, &domain.Tenant{
		ID:             tenantID,
		OrganizationID: orgID,
		Name:           "Track Tenant",
		Slug:           "track-tenant",
		Status:         "active",
		CreatedAt:      time.Now().UTC(),
	})

	userID := crypto.NewULID(crypto.PrefixUser)
	_ = repo.CreateUser(ctx, &domain.User{
		ID:             userID,
		OrganizationID: orgID,
		Email:          "dev@track.local",
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

	return server, repo, engine, tenantID, userID
}

func TestOpenPixelEndpoint(t *testing.T) {
	server, repo, engine, tenantID, _ := setupTrackingInboundTestEnv(t)
	ctx := context.Background()

	msgID := "msg_open_test"
	rcptID := "rcpt_open_test"

	// Seed recipient with past delivery time
	past := time.Now().UTC().Add(-5 * time.Minute)
	_ = repo.UpdateRecipientDelivery(ctx, rcptID, "delivered", 1, nil, nil, &past, nil, nil, nil)

	token := engine.SignOpenToken(tenantID, msgID, rcptID)

	// Hit GET /o/{token}
	req := httptest.NewRequest(http.MethodGet, "/o/"+token, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0 Safari/537.36")
	req.RemoteAddr = "198.51.100.23:54321"

	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "image/gif" {
		t.Errorf("expected Content-Type image/gif, got %s", w.Header().Get("Content-Type"))
	}
	if !bytes.Equal(w.Body.Bytes(), tracking.TransparentGIF) {
		t.Errorf("expected body to match TransparentGIF")
	}

	// Verify tracking event recorded
	events, err := repo.ListTrackingEvents(ctx, tenantID, msgID)
	if err != nil {
		t.Fatalf("failed to list tracking events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 tracking event, got %d", len(events))
	}
	if events[0].Type != "open" {
		t.Errorf("expected event type open, got %s", events[0].Type)
	}
	if events[0].IP != "198.51.100.0" {
		t.Errorf("expected truncated IP 198.51.100.0, got %s", events[0].IP)
	}
}

func TestClickRedirectEndpoint(t *testing.T) {
	server, repo, engine, tenantID, _ := setupTrackingInboundTestEnv(t)
	ctx := context.Background()

	msgID := "msg_click_test"
	rcptID := "rcpt_click_test"
	linkID := "link_click_test"
	targetURL := "https://myproduct.example.com/checkout?ref=mailvex"

	// Seed tracking link
	_ = repo.SaveTrackingLink(ctx, &domain.TrackingLink{
		ID:        linkID,
		TenantID:  tenantID,
		MessageID: msgID,
		LinkID:    linkID,
		TargetURL: targetURL,
		CreatedAt: time.Now().UTC(),
	})

	token := engine.SignClickToken(tenantID, msgID, rcptID, linkID)

	// Hit GET /c/{token}
	req := httptest.NewRequest(http.MethodGet, "/c/"+token, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)")
	req.RemoteAddr = "203.0.113.88:12345"

	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected HTTP 302 Found, got %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != targetURL {
		t.Errorf("expected redirect to %s, got %s", targetURL, loc)
	}

	// Tampered token test -> must return 404 (no open redirect)
	tamperedReq := httptest.NewRequest(http.MethodGet, "/c/tampered-bad-token", nil)
	wBad := httptest.NewRecorder()
	server.Handler().ServeHTTP(wBad, tamperedReq)

	if wBad.Code != http.StatusNotFound {
		t.Errorf("expected 404 for tampered token, got %d", wBad.Code)
	}
}

func TestInboundRoutesHTTPEndpoints(t *testing.T) {
	server, _, _, tenantID, userID := setupTrackingInboundTestEnv(t)

	// 1. Create Inbound Route (POST /v1/inbound/routes)
	targetWebhook := "https://webhook.site/inbound"
	createBody, _ := json.Marshal(map[string]any{
		"priority":      5,
		"match_type":    "wildcard",
		"pattern":       "tickets-*@domain.com",
		"action":        "webhook",
		"action_target": targetWebhook,
		"is_active":     true,
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/inbound/routes", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("X-User-ID", userID)

	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create route failed with status %d: %s", w.Code, w.Body.String())
	}

	var createdRoute domain.InboundRoute
	_ = json.NewDecoder(w.Body).Decode(&createdRoute)
	if createdRoute.Pattern != "tickets-*@domain.com" {
		t.Errorf("unexpected pattern: %s", createdRoute.Pattern)
	}

	// 2. Test Inbound Route Match (POST /v1/inbound/routes/test)
	testBody, _ := json.Marshal(map[string]string{
		"address": "tickets-999@domain.com",
	})
	testReq := httptest.NewRequest(http.MethodPost, "/v1/inbound/routes/test", bytes.NewReader(testBody))
	testReq.Header.Set("Content-Type", "application/json")
	testReq.Header.Set("X-Tenant-ID", tenantID)
	testReq.Header.Set("X-User-ID", userID)

	wTest := httptest.NewRecorder()
	server.Handler().ServeHTTP(wTest, testReq)

	if wTest.Code != http.StatusOK {
		t.Fatalf("test match failed with status %d: %s", wTest.Code, wTest.Body.String())
	}

	var matchResp struct {
		Matched bool                 `json:"matched"`
		Route   *domain.InboundRoute `json:"route"`
	}
	_ = json.NewDecoder(wTest.Body).Decode(&matchResp)
	if !matchResp.Matched || matchResp.Route == nil || matchResp.Route.ID != createdRoute.ID {
		t.Errorf("expected route match, got %+v", matchResp)
	}

	// 3. List Inbound Routes (GET /v1/inbound/routes)
	listReq := httptest.NewRequest(http.MethodGet, "/v1/inbound/routes", nil)
	listReq.Header.Set("X-Tenant-ID", tenantID)
	listReq.Header.Set("X-User-ID", userID)

	wList := httptest.NewRecorder()
	server.Handler().ServeHTTP(wList, listReq)

	if wList.Code != http.StatusOK {
		t.Fatalf("list routes failed with status %d: %s", wList.Code, wList.Body.String())
	}

	var listResp struct {
		Data []*domain.InboundRoute `json:"data"`
	}
	_ = json.NewDecoder(wList.Body).Decode(&listResp)
	if len(listResp.Data) != 1 {
		t.Errorf("expected 1 route, got %d", len(listResp.Data))
	}

	// 4. Delete Inbound Route (DELETE /v1/inbound/routes/{id})
	delReq := httptest.NewRequest(http.MethodDelete, "/v1/inbound/routes/"+createdRoute.ID, nil)
	delReq.Header.Set("X-Tenant-ID", tenantID)
	delReq.Header.Set("X-User-ID", userID)

	wDel := httptest.NewRecorder()
	server.Handler().ServeHTTP(wDel, delReq)

	if wDel.Code != http.StatusNoContent {
		t.Errorf("expected 204 No Content for delete, got %d", wDel.Code)
	}
}
