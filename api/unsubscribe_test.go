package api

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/senthilnasa/mailvex/core/audit"
	"github.com/senthilnasa/mailvex/core/crypto"
	"github.com/senthilnasa/mailvex/core/domain"
	"github.com/senthilnasa/mailvex/core/repository"
)

func TestUnsubscribeEndpoints(t *testing.T) {
	repo := repository.NewMemoryRepository()
	auditSvc := audit.NewService(repo)
	kek := bytes.Repeat([]byte{0x04}, 32)
	enc, _ := crypto.NewEnvelopeEncryption(kek)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	srv := NewServer(repo, auditSvc, enc, logger)
	ctx := context.Background()

	// Seed tenant & message
	tenant := &domain.Tenant{ID: "t_unsub", OrganizationID: "org_1", Name: "Unsub Tenant", Slug: "unsub-tenant", Environment: "live"}
	_ = repo.CreateTenant(ctx, tenant)

	stream := &domain.MessageStream{ID: "str_broadcast", TenantID: "t_unsub", Name: "broadcast", Type: "broadcast"}
	_ = repo.CreateStream(ctx, stream)

	msg := &domain.Message{
		ID:              "msg_unsub_1",
		TenantID:        "t_unsub",
		StreamID:        "str_broadcast",
		DomainID:        "dom_1",
		FromAddress:     "newsletter@example.com",
		Subject:         "Weekly Newsletter",
		MessageIDHeader: "<msg_unsub_1@example.com>",
		Status:          "delivered",
		Environment:     "live",
		CreatedAt:       time.Now().UTC(),
	}

	rcpt := &domain.MessageRecipient{
		ID:            "rcpt_unsub_1",
		MessageID:     "msg_unsub_1",
		TenantID:      "t_unsub",
		Address:       "subscriber@example.com",
		AddressDomain: "example.com",
		Status:        "delivered",
		CreatedAt:     time.Now().UTC(),
	}

	_ = repo.SaveMessage(ctx, msg, []*domain.MessageRecipient{rcpt}, nil, nil)

	// Create valid HMAC token
	secret := []byte("mailvex-unsubscribe-token-secret")
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(fmt.Sprintf("%s:%s:%s", "t_unsub", "rcpt_unsub_1", "subscriber@example.com")))
	validSig := hex.EncodeToString(mac.Sum(nil))[:24]
	validToken := fmt.Sprintf("rcpt_unsub_1.%s", validSig)

	// 1. Test GET /u/{token} (Hosted confirmation page)
	req := httptest.NewRequest(http.MethodGet, "/u/"+validToken, nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on GET /u/{token}, got: %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Confirm Unsubscribe") || !strings.Contains(body, "subscriber@example.com") {
		t.Fatalf("expected hosted unsubscribe HTML page, got: %s", body)
	}

	// 2. Test POST /u/{token} with RFC 8058 One-Click
	postReq := httptest.NewRequest(http.MethodPost, "/u/"+validToken, strings.NewReader("List-Unsubscribe=One-Click"))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(postW, postReq)

	if postW.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on POST /u/{token}, got: %d, body: %s", postW.Code, postW.Body.String())
	}

	// Verify suppression is created in repository
	isSupp, reason, err := repo.CheckSuppression(ctx, "t_unsub", "str_broadcast", "subscriber@example.com")
	if err != nil || !isSupp {
		t.Fatalf("expected address to be suppressed after one-click unsubscribe, isSupp=%v, err=%v", isSupp, err)
	}
	if reason != "unsubscribe" && !strings.Contains(reason, "Unsubscribe") {
		t.Fatalf("expected unsubscribe reason, got: %s", reason)
	}

	// 3. Test tampered token returns 403 Forbidden
	tamperedReq := httptest.NewRequest(http.MethodPost, "/u/rcpt_unsub_1.invalidSignature123456", strings.NewReader("List-Unsubscribe=One-Click"))
	tamperedW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(tamperedW, tamperedReq)

	if tamperedW.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden on tampered token, got: %d", tamperedW.Code)
	}
}
