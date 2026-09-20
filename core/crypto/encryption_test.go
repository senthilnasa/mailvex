package crypto

import (
	"bytes"
	"testing"
)

func TestEnvelopeEncryption(t *testing.T) {
	masterKEK := []byte("12345678901234567890123456789012") // 32 bytes

	enc, err := NewEnvelopeEncryption(masterKEK)
	if err != nil {
		t.Fatalf("failed to create envelope encryption: %v", err)
	}

	// Test 1: Generate DEK
	dek, err := enc.GenerateDEK()
	if err != nil {
		t.Fatalf("failed to generate DEK: %v", err)
	}
	if len(dek) != 32 {
		t.Fatalf("expected 32-byte DEK, got %d", len(dek))
	}

	// Test 2: Wrap DEK
	wrappedDEK, err := enc.EncryptDEK(dek)
	if err != nil {
		t.Fatalf("failed to wrap DEK: %v", err)
	}

	// Test 3: Unwrap DEK
	unwrappedDEK, err := enc.DecryptDEK(wrappedDEK)
	if err != nil {
		t.Fatalf("failed to unwrap DEK: %v", err)
	}
	if !bytes.Equal(dek, unwrappedDEK) {
		t.Fatalf("unwrapped DEK does not match original DEK")
	}

	// Test 4: Secret encryption with DEK
	secretPayload := []byte("-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEA0...")
	encSecret, err := EncryptSecret(unwrappedDEK, secretPayload)
	if err != nil {
		t.Fatalf("failed to encrypt secret: %v", err)
	}

	decSecret, err := DecryptSecret(unwrappedDEK, encSecret)
	if err != nil {
		t.Fatalf("failed to decrypt secret: %v", err)
	}
	if !bytes.Equal(secretPayload, decSecret) {
		t.Fatalf("decrypted secret does not match original")
	}

	// Test 5: Tampered ciphertext must fail
	tampered := wrappedDEK[:len(wrappedDEK)-4] + "AAAA"
	if _, err := enc.DecryptDEK(tampered); err == nil {
		t.Fatalf("expected decryption error for tampered ciphertext")
	}
}
