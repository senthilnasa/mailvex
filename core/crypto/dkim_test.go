package crypto

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/emersion/go-msgauth/dkim"
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
	if !strings.HasPrefix(signedStr, "DKIM-Signature:") {
		t.Fatalf("expected DKIM-Signature header, got: %s", signedStr[:100])
	}
	if !strings.Contains(signedStr, "d=example.com") || !strings.Contains(signedStr, "s=default") {
		t.Fatalf("expected domain and selector in signature, got: %s", signedStr[:200])
	}
	if !strings.Contains(signedStr, "bh=") || !strings.Contains(signedStr, "b=") {
		t.Fatalf("expected bh= and b= tags in signature, got: %s", signedStr[:150])
	}
}

func TestSignAndVerifyDKIM(t *testing.T) {
	keyPair, err := GenerateDKIMKey(2048)
	if err != nil {
		t.Fatalf("failed to generate DKIM key: %v", err)
	}

	rawEmail := []byte("From: erpadmin@crm.krea.edu.in\r\nTo: senthilprabu005@gmail.com\r\nSubject: DKIM Test\r\nDate: Sat, 31 Oct 2026 02:46:21 +0000\r\nMessage-ID: <01m67xkpgkmqjdnmgm35nkya2z@crm.krea.edu.in>\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n<p>Testing DKIM signature verification.</p>\r\n")

	signed, err := SignDKIM(rawEmail, "crm.krea.edu.in", "mvx", keyPair.PrivateKeyPEM)
	if err != nil {
		t.Fatalf("failed to sign email: %v", err)
	}

	// Verify using go-msgauth
	verifications, err := dkim.VerifyWithOptions(bytes.NewReader(signed), &dkim.VerifyOptions{
		LookupTXT: func(domain string) ([]string, error) {
			if domain == "mvx._domainkey.crm.krea.edu.in" {
				return []string{"v=DKIM1; k=rsa; p=" + keyPair.PublicKeyB64}, nil
			}
			return nil, fmt.Errorf("domain not found")
		},
	})
	if err != nil {
		t.Fatalf("verification error: %v", err)
	}
	if len(verifications) == 0 {
		t.Fatalf("no verifications returned")
	}
	for _, v := range verifications {
		if v.Err != nil {
			t.Fatalf("verification failed: %v", v.Err)
		}
	}
}

