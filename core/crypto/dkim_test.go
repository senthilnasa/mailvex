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

func TestSignDKIM(t *testing.T) {
	keyPair, err := GenerateDKIMKey(2048)
	if err != nil {
		t.Fatalf("failed to generate DKIM key: %v", err)
	}

	rawEmail := []byte("From: sender@example.com\r\nTo: recipient@example.com\r\nSubject: Hello World\r\nDate: Mon, 01 Jan 2026 00:00:00 +0000\r\nMessage-ID: <msg123@example.com>\r\n\r\nThis is the email body.\r\n")

	signed, err := SignDKIM(rawEmail, "example.com", "default", keyPair.PrivateKeyPEM)
	if err != nil {
		t.Fatalf("failed to sign email with DKIM: %v", err)
	}

	signedStr := string(signed)
	if !strings.HasPrefix(signedStr, "DKIM-Signature: v=1; a=rsa-sha256; c=relaxed/relaxed; d=example.com; s=default;") {
		t.Fatalf("expected DKIM-Signature header, got: %s", signedStr[:100])
	}
	if !strings.Contains(signedStr, "bh=") || !strings.Contains(signedStr, "b=") {
		t.Fatalf("expected bh= and b= tags in signature, got: %s", signedStr[:150])
	}
}

