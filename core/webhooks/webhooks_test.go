package webhooks_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mailvex/mailvex/core/crypto"
	"github.com/mailvex/mailvex/core/domain"
	"github.com/mailvex/mailvex/core/repository"
	"github.com/mailvex/mailvex/core/webhooks"
)

func TestWebhookSignatureGenerationAndVerification(t *testing.T) {
	secret := webhooks.GenerateSecret()
	timestamp := time.Now().UTC().Unix()
	payload := []byte(`{"id":"evt_123","event":"message.delivered","data":{"message_id":"msg_456"}}`)

	header := webhooks.FormatSignatureHeader(timestamp, secret, nil, nil, payload)

	// Valid verification
	valid, err := webhooks.VerifySignature(payload, header, []string{secret}, 300*time.Second)
	if err != nil || !valid {
		t.Fatalf("expected valid signature, got valid=%v, err=%v", valid, err)
	}

	// Invalid payload tampering
	tamperedPayload := []byte(`{"id":"evt_123","event":"message.delivered","data":{"message_id":"msg_999"}}`)
	valid, err = webhooks.VerifySignature(tamperedPayload, header, []string{secret}, 300*time.Second)
	if valid || err == nil {
		t.Fatalf("expected signature mismatch on tampered payload, got valid=%v", valid)
	}

	// Wrong secret
	wrongSecret := webhooks.GenerateSecret()
	valid, err = webhooks.VerifySignature(payload, header, []string{wrongSecret}, 300*time.Second)
	if valid || err == nil {
		t.Fatalf("expected signature mismatch with wrong secret, got valid=%v", valid)
	}
}

func TestWebhookSecretRotation24HourOverlap(t *testing.T) {
	oldSecret := webhooks.GenerateSecret()
	newSecret := webhooks.GenerateSecret()
	timestamp := time.Now().UTC().Unix()
	payload := []byte(`{"id":"evt_rotate","event":"message.bounced"}`)

	// Active 24-hour overlap window
	overlapExpiresAt := time.Now().UTC().Add(24 * time.Hour)
	header := webhooks.FormatSignatureHeader(timestamp, newSecret, &oldSecret, &overlapExpiresAt, payload)

	// Receiver using new secret verifies successfully
	validNew, err := webhooks.VerifySignature(payload, header, []string{newSecret}, 300*time.Second)
	if err != nil || !validNew {
		t.Fatalf("expected receiver with new secret to verify, got valid=%v, err=%v", validNew, err)
	}

	// Receiver still using old secret also verifies successfully! (Zero downtime)
	validOld, err := webhooks.VerifySignature(payload, header, []string{oldSecret}, 300*time.Second)
	if err != nil || !validOld {
		t.Fatalf("expected receiver with old secret during overlap to verify, got valid=%v, err=%v", validOld, err)
	}

	// When overlap expires, old secret should not be signed
	expiredTime := time.Now().UTC().Add(-1 * time.Hour)
	expiredHeader := webhooks.FormatSignatureHeader(timestamp, newSecret, &oldSecret, &expiredTime, payload)
	validOldAfterExpiry, _ := webhooks.VerifySignature(payload, expiredHeader, []string{oldSecret}, 300*time.Second)
	if validOldAfterExpiry {
		t.Fatalf("expected old secret signature to be omitted after expiration")
	}
}

func TestWebhookReplayAttackToleranceRejection(t *testing.T) {
	secret := webhooks.GenerateSecret()
	payload := []byte(`{"event":"message.sent"}`)

	// Expired timestamp (10 minutes ago, tolerance is 5 minutes)
	expiredTimestamp := time.Now().UTC().Add(-10 * time.Minute).Unix()
	header := webhooks.FormatSignatureHeader(expiredTimestamp, secret, nil, nil, payload)

	valid, err := webhooks.VerifySignature(payload, header, []string{secret}, 300*time.Second)
	if valid || err != webhooks.ErrReplayAttack {
		t.Fatalf("expected ErrReplayAttack for old timestamp, got valid=%v, err=%v", valid, err)
	}
}

func TestWebhookCircuitBreakerAutoDisable(t *testing.T) {
	repo := repository.NewMemoryRepository()
	ctx := context.Background()
	tenantID := crypto.NewULID(crypto.PrefixTenant)

	// Failing HTTP server (returns 500)
	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal crash"}`))
	}))
	defer failServer.Close()

	endpointID := crypto.NewULID("whe_")
	endpoint := &domain.WebhookEndpoint{
		ID:                  endpointID,
		TenantID:            tenantID,
		URL:                 failServer.URL,
		Events:              []string{"*"},
		Secret:              webhooks.GenerateSecret(),
		Status:              "active",
		ConsecutiveFailures: 0,
		CreatedAt:           time.Now().UTC(),
		UpdatedAt:           time.Now().UTC(),
	}
	if err := repo.CreateWebhookEndpoint(ctx, endpoint); err != nil {
		t.Fatalf("failed to create endpoint: %v", err)
	}

	dispatcher := webhooks.NewDispatcher(repo, failServer.Client())

	// Dispatch 9 failures -> should still be active with 9 failures
	for i := 0; i < 9; i++ {
		_ = dispatcher.DispatchEvent(ctx, tenantID, fmt.Sprintf("evt_%d", i), "message.delivered", map[string]string{"foo": "bar"})
	}

	ep, err := repo.GetWebhookEndpoint(ctx, tenantID, endpointID)
	if err != nil {
		t.Fatalf("failed to get endpoint: %v", err)
	}
	if ep.Status != "active" || ep.ConsecutiveFailures != 9 {
		t.Fatalf("expected active status with 9 failures, got %s / %d", ep.Status, ep.ConsecutiveFailures)
	}

	// 10th failure -> circuit breaker triggers and sets status to disabled_failing
	_ = dispatcher.DispatchEvent(ctx, tenantID, "evt_10", "message.delivered", map[string]string{"foo": "bar"})

	ep, err = repo.GetWebhookEndpoint(ctx, tenantID, endpointID)
	if err != nil {
		t.Fatalf("failed to get endpoint: %v", err)
	}
	if ep.Status != "disabled_failing" {
		t.Fatalf("expected disabled_failing status, got %s", ep.Status)
	}
	if ep.ConsecutiveFailures != 10 {
		t.Fatalf("expected 10 consecutive failures, got %d", ep.ConsecutiveFailures)
	}

	// Once disabled_failing, it should NOT receive further dispatches
	activeEndpoints, _ := repo.GetActiveWebhookEndpointsForEvent(ctx, tenantID, "message.delivered")
	if len(activeEndpoints) != 0 {
		t.Fatalf("expected 0 active endpoints, got %d", len(activeEndpoints))
	}
}

func TestWebhookTestPing(t *testing.T) {
	repo := repository.NewMemoryRepository()
	ctx := context.Background()
	tenantID := crypto.NewULID(crypto.PrefixTenant)

	var receivedHeaders http.Header
	var receivedBody []byte

	// Successful HTTP server
	successServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		receivedBody = make([]byte, r.ContentLength)
		_, _ = r.Body.Read(receivedBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"received":true}`))
	}))
	defer successServer.Close()

	endpointID := crypto.NewULID("whe_")
	endpoint := &domain.WebhookEndpoint{
		ID:        endpointID,
		TenantID:  tenantID,
		URL:       successServer.URL,
		Events:    []string{"message.delivered"},
		Secret:    webhooks.GenerateSecret(),
		Status:    "active",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	_ = repo.CreateWebhookEndpoint(ctx, endpoint)

	dispatcher := webhooks.NewDispatcher(repo, successServer.Client())

	del, err := dispatcher.DispatchPing(ctx, tenantID, endpointID)
	if err != nil {
		t.Fatalf("ping failed: %v", err)
	}

	if del.StatusCode == nil || *del.StatusCode != 200 {
		t.Fatalf("expected status 200, got %v", del.StatusCode)
	}
	if del.EventType != "webhook.test" {
		t.Fatalf("expected webhook.test event type, got %s", del.EventType)
	}
	if receivedHeaders.Get("X-Mailvex-Signature") == "" {
		t.Fatalf("expected signature header to be present")
	}

	// Verify delivery log exists
	logs, err := repo.ListWebhookDeliveries(ctx, tenantID, endpointID, 10, 0)
	if err != nil || len(logs) != 1 {
		t.Fatalf("expected 1 delivery record, got %d, err=%v", len(logs), err)
	}
}
