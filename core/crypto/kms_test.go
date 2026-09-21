package crypto

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAWSKMSKEKProvider(t *testing.T) {
	ctx := context.Background()

	// Mock AWS KMS Server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := r.Header.Get("X-Amz-Target")
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")

		if target == "TrentService.Encrypt" {
			var body struct {
				KeyId     string `json:"KeyId"`
				Plaintext string `json:"Plaintext"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// Simulate KMS ciphertext blob (prepend kms:v1:)
			resp := map[string]interface{}{
				"CiphertextBlob": "kms:v1:" + body.Plaintext,
				"KeyId":          body.KeyId,
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if target == "TrentService.Decrypt" {
			var body struct {
				KeyId          string `json:"KeyId"`
				CiphertextBlob string `json:"CiphertextBlob"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			pt := strings.TrimPrefix(body.CiphertextBlob, "kms:v1:")
			resp := map[string]interface{}{
				"Plaintext": pt,
				"KeyId":     body.KeyId,
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		http.Error(w, "unknown action", http.StatusBadRequest)
	}))
	defer server.Close()

	kmsProvider, err := NewAWSKMSKEKProvider(AWSKMSConfig{
		KeyID:    "arn:aws:kms:us-east-1:123456789012:key/test-mailvex-key",
		Region:   "us-east-1",
		Endpoint: server.URL,
	}, server.Client())
	if err != nil {
		t.Fatalf("failed to create AWSKMSKEKProvider: %v", err)
	}

	if !strings.HasPrefix(kmsProvider.KeyID(), "awskms:") {
		t.Errorf("expected awskms: prefix in KeyID, got %s", kmsProvider.KeyID())
	}

	dek := make([]byte, 32)
	io.ReadFull(rand.Reader, dek)

	wrapped, err := kmsProvider.WrapDEK(ctx, dek)
	if err != nil {
		t.Fatalf("WrapDEK with KMS failed: %v", err)
	}

	if !strings.HasPrefix(wrapped, "kms:v1:") {
		t.Errorf("expected kms:v1: prefix in wrapped DEK, got %s", wrapped)
	}

	unwrapped, err := kmsProvider.UnwrapDEK(ctx, wrapped)
	if err != nil {
		t.Fatalf("UnwrapDEK with KMS failed: %v", err)
	}

	if string(unwrapped) != string(dek) {
		t.Errorf("unwrapped DEK mismatch: got %x, want %x", unwrapped, dek)
	}

	// Test rotation from LocalKEK to AWS KMS
	kekLocal := make([]byte, 32)
	io.ReadFull(rand.Reader, kekLocal)
	localProv, _ := NewLocalKEKProvider("local-kek", kekLocal)

	wrappedLocal, _ := localProv.WrapDEK(ctx, dek)
	rotatedToKMS, err := RotateDEK(ctx, localProv, kmsProvider, wrappedLocal)
	if err != nil {
		t.Fatalf("RotateDEK to KMS failed: %v", err)
	}

	recoveredDEK, err := kmsProvider.UnwrapDEK(ctx, rotatedToKMS)
	if err != nil {
		t.Fatalf("UnwrapDEK after rotation failed: %v", err)
	}
	if string(recoveredDEK) != string(dek) {
		t.Errorf("rotated DEK mismatch: got %x, want %x", recoveredDEK, dek)
	}
}
