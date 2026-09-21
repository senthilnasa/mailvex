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

func TestLocalKEKProvider(t *testing.T) {
	ctx := context.Background()
	kek := make([]byte, 32)
	io.ReadFull(rand.Reader, kek)

	provider, err := NewLocalKEKProvider("local-v1", kek)
	if err != nil {
		t.Fatalf("unexpected error creating LocalKEKProvider: %v", err)
	}

	if provider.KeyID() != "local-v1" {
		t.Errorf("expected KeyID local-v1, got %s", provider.KeyID())
	}

	dek := make([]byte, 32)
	io.ReadFull(rand.Reader, dek)

	wrapped, err := provider.WrapDEK(ctx, dek)
	if err != nil {
		t.Fatalf("WrapDEK failed: %v", err)
	}

	unwrapped, err := provider.UnwrapDEK(ctx, wrapped)
	if err != nil {
		t.Fatalf("UnwrapDEK failed: %v", err)
	}

	if string(unwrapped) != string(dek) {
		t.Errorf("unwrapped DEK mismatch: got %x, want %x", unwrapped, dek)
	}

	// Corrupted ciphertext
	_, err = provider.UnwrapDEK(ctx, "invalid-base64!")
	if err == nil {
		t.Error("expected error on invalid base64 unwrap")
	}
}

func TestVaultTransitKEKProvider(t *testing.T) {
	ctx := context.Background()

	// Mock Vault Transit Server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token := r.Header.Get("X-Vault-Token"); token != "s.mocktoken123" {
			http.Error(w, `{"errors":["permission denied"]}`, http.StatusForbidden)
			return
		}

		if strings.Contains(r.URL.Path, "/v1/transit/encrypt/") {
			var body struct {
				Plaintext string `json:"plaintext"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			// Simulate Vault transit ciphertext format
			resp := map[string]interface{}{
				"data": map[string]interface{}{
					"ciphertext": "vault:v1:" + body.Plaintext,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		if strings.Contains(r.URL.Path, "/v1/transit/decrypt/") {
			var body struct {
				Ciphertext string `json:"ciphertext"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			// Strip prefix
			pt := strings.TrimPrefix(body.Ciphertext, "vault:v1:")
			resp := map[string]interface{}{
				"data": map[string]interface{}{
					"plaintext": pt,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	vaultProvider, err := NewVaultTransitKEKProvider(VaultTransitConfig{
		Address:   server.URL,
		Token:     "s.mocktoken123",
		KeyName:   "test-kek",
		Namespace: "prod",
	}, server.Client())
	if err != nil {
		t.Fatalf("failed to create VaultTransitKEKProvider: %v", err)
	}

	dek := make([]byte, 32)
	io.ReadFull(rand.Reader, dek)

	wrapped, err := vaultProvider.WrapDEK(ctx, dek)
	if err != nil {
		t.Fatalf("Vault WrapDEK failed: %v", err)
	}

	if !strings.HasPrefix(wrapped, "vault:v1:") {
		t.Errorf("expected vault:v1: prefix in ciphertext, got: %s", wrapped)
	}

	unwrapped, err := vaultProvider.UnwrapDEK(ctx, wrapped)
	if err != nil {
		t.Fatalf("Vault UnwrapDEK failed: %v", err)
	}

	if string(unwrapped) != string(dek) {
		t.Errorf("unwrapped DEK mismatch: got %x, want %x", unwrapped, dek)
	}
}

func TestRotateDEK(t *testing.T) {
	ctx := context.Background()

	kek1 := make([]byte, 32)
	kek2 := make([]byte, 32)
	io.ReadFull(rand.Reader, kek1)
	io.ReadFull(rand.Reader, kek2)

	p1, _ := NewLocalKEKProvider("kek-v1", kek1)
	p2, _ := NewLocalKEKProvider("kek-v2", kek2)

	dek := make([]byte, 32)
	io.ReadFull(rand.Reader, dek)

	// Encrypt sample customer secret with the DEK
	secretPlaintext := []byte("private-dkim-rsa-key-material")
	encSecret, err := EncryptSecret(dek, secretPlaintext)
	if err != nil {
		t.Fatalf("EncryptSecret failed: %v", err)
	}

	// Wrap DEK with p1
	wrapped1, err := p1.WrapDEK(ctx, dek)
	if err != nil {
		t.Fatalf("WrapDEK with p1 failed: %v", err)
	}

	// Rotate DEK to p2
	wrapped2, err := RotateDEK(ctx, p1, p2, wrapped1)
	if err != nil {
		t.Fatalf("RotateDEK failed: %v", err)
	}

	// Unwrap with p2
	recoveredDEK, err := p2.UnwrapDEK(ctx, wrapped2)
	if err != nil {
		t.Fatalf("Unwrap with p2 failed: %v", err)
	}

	// Verify the recovered DEK can still decrypt the secret
	decryptedSecret, err := DecryptSecret(recoveredDEK, encSecret)
	if err != nil {
		t.Fatalf("DecryptSecret with rotated DEK failed: %v", err)
	}

	if string(decryptedSecret) != string(secretPlaintext) {
		t.Errorf("decrypted secret mismatch: got %s, want %s", string(decryptedSecret), string(secretPlaintext))
	}
}
