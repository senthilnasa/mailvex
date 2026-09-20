package crypto

import (
	"strings"
	"testing"
)

func TestDKIMKeyGeneration(t *testing.T) {
	// Test 1: Sub-2048 keys refused (§10.5)
	if _, err := GenerateDKIMKey(1024); err == nil {
		t.Fatalf("expected error generating 1024-bit key, got nil")
	}

	// Test 2: Standard 2048-bit key generation
	keyPair, err := GenerateDKIMKey(2048)
	if err != nil {
		t.Fatalf("failed to generate 2048-bit DKIM key: %v", err)
	}

	if !strings.Contains(keyPair.PrivateKeyPEM, "BEGIN PRIVATE KEY") {
		t.Fatalf("expected valid PKCS#8 private key PEM, got: %s", keyPair.PrivateKeyPEM)
	}

	if len(keyPair.PublicKeyB64) < 100 {
		t.Fatalf("expected substantial base64 public key, got length: %d", len(keyPair.PublicKeyB64))
	}
}
