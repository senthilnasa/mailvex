package mailvex

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestGoSDKSendAndBatch(t *testing.T) {
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-api-key" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/v1/send" && r.Method == http.MethodPost {
			var body SendEmailRequest
			_ = json.NewDecoder(r.Body).Decode(&body)
			resp := SendEmailResponse{
				MessageID: "msg_sdk_123",
				Status:    "queued",
				Accepted:  body.To,
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.URL.Path == "/v1/send/batch" && r.Method == http.MethodPost {
			var body SendBatchRequest
			_ = json.NewDecoder(r.Body).Decode(&body)
			resp := SendBatchResponse{
				BatchID:    "batch_sdk_456",
				Total:      len(body.Recipients),
				MessageIDs: []string{"msg_b1", "msg_b2"},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL), WithHTTPClient(server.Client()))

	// Test Send
	sendResp, err := client.Send(ctx, &SendEmailRequest{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "SDK Test",
		HTML:    "<p>Hello</p>",
	})
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if sendResp.MessageID != "msg_sdk_123" {
		t.Errorf("expected msg_sdk_123, got %s", sendResp.MessageID)
	}

	// Test Send with response returning {"id": "..."}
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg_id_only","status":"queued"}`))
	}))
	defer server2.Close()
	client2 := NewClient("test-key", WithBaseURL(server2.URL), WithHTTPClient(server2.Client()))
	sendResp2, err := client2.Send(ctx, &SendEmailRequest{
		From:    "sender@example.com",
		To:      []string{"rcpt@example.com"},
		Subject: "ID Only Test",
	})
	if err != nil {
		t.Fatalf("Send ID only failed: %v", err)
	}
	if sendResp2.MessageID != "msg_id_only" {
		t.Errorf("expected msg_id_only fallback, got %s", sendResp2.MessageID)
	}

	// Test SendBatch
	batchResp, err := client.SendBatch(ctx, &SendBatchRequest{
		From:    "sender@example.com",
		Subject: "Batch Test",
		Recipients: []BatchRecipient{
			{To: "u1@example.com"},
			{To: "u2@example.com"},
		},
	})
	if err != nil {
		t.Fatalf("SendBatch failed: %v", err)
	}
	if batchResp.BatchID != "batch_sdk_456" {
		t.Errorf("expected batch_sdk_456, got %s", batchResp.BatchID)
	}
}

func TestGoSDKRetries(t *testing.T) {
	ctx := context.Background()
	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		att := atomic.AddInt32(&attempts, 1)
		if att == 1 {
			http.Error(w, `{"error":"temporary server error"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "msg_recovered"})
	}))
	defer server.Close()

	client := NewClient("key", WithBaseURL(server.URL), WithHTTPClient(server.Client()), WithMaxRetries(2))

	msg, err := client.GetMessage(ctx, "msg_recovered")
	if err != nil {
		t.Fatalf("expected retry to succeed, got: %v", err)
	}
	if msg.ID != "msg_recovered" {
		t.Errorf("expected msg_recovered, got %s", msg.ID)
	}
	if atomic.LoadInt32(&attempts) != 2 {
		t.Errorf("expected 2 attempts, got %d", atomic.LoadInt32(&attempts))
	}
}

func TestVerifyWebhookSignature(t *testing.T) {
	payload := []byte(`{"event":"delivered","message_id":"msg_100"}`)
	secret := "whsec_supersecretkey123"
	timestamp := fmt.Sprintf("%d", time.Now().Unix())

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp + "." + string(payload)))
	validSig := hex.EncodeToString(mac.Sum(nil))

	header := fmt.Sprintf("t=%s,v1=%s", timestamp, validSig)

	// Valid check
	if !VerifyWebhookSignature(payload, header, secret) {
		t.Error("expected valid signature check to pass")
	}

	// Tampered payload
	tamperedPayload := []byte(`{"event":"bounced","message_id":"msg_100"}`)
	if VerifyWebhookSignature(tamperedPayload, header, secret) {
		t.Error("expected tampered payload check to fail")
	}

	// Wrong secret
	if VerifyWebhookSignature(payload, header, "wrong-secret") {
		t.Error("expected wrong secret check to fail")
	}
}
