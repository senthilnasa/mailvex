package crypto

import (
	"testing"
)

func TestVERPTokenGenerationAndVerification(t *testing.T) {
	secret := []byte("super-secret-test-key-12345")
	rcptID := "rcpt_01J7ABCDEF123456789"

	token := GenerateVERPToken(rcptID, secret)
	if token == "" {
		t.Fatalf("expected non-empty token")
	}

	decoded, err := VerifyAndDecodeVERPToken(token, secret)
	if err != nil {
		t.Fatalf("failed to decode valid token: %v", err)
	}
	if decoded != rcptID {
		t.Fatalf("expected %s, got %s", rcptID, decoded)
	}

	// Tampered recipient ID
	tamperedToken := "rcpt_TAMPERED." + token[len(rcptID)+1:]
	_, err = VerifyAndDecodeVERPToken(tamperedToken, secret)
	if err == nil {
		t.Fatalf("expected error for tampered recipient ID")
	}

	// Tampered signature
	tamperedSig := token[:len(token)-2] + "00"
	_, err = VerifyAndDecodeVERPToken(tamperedSig, secret)
	if err == nil {
		t.Fatalf("expected error for tampered signature")
	}

	// Different secret
	otherSecret := []byte("another-secret-key-67890")
	_, err = VerifyAndDecodeVERPToken(token, otherSecret)
	if err == nil {
		t.Fatalf("expected error for different secret")
	}
}

func TestVERPAddressFormattingAndParsing(t *testing.T) {
	secret := []byte("test-key")
	rcptID := "rcpt_0190abcdef"
	bounceDomain := "bounce.mailvex.com"

	addr := FormatVERPAddress(rcptID, bounceDomain, secret)
	expectedPrefix := "bounce+" + rcptID + "."
	if len(addr) < len(expectedPrefix) || addr[:len(expectedPrefix)] != expectedPrefix {
		t.Fatalf("unexpected addr: %s", addr)
	}

	// Parse valid formatted address
	parsed, err := ParseVERPAddress(addr, secret)
	if err != nil {
		t.Fatalf("failed to parse valid address: %v", err)
	}
	if parsed != rcptID {
		t.Fatalf("expected %s, got %s", rcptID, parsed)
	}

	// Parse with angle brackets
	angleAddr := "<" + addr + ">"
	parsedAngle, err := ParseVERPAddress(angleAddr, secret)
	if err != nil {
		t.Fatalf("failed to parse angle bracket address: %v", err)
	}
	if parsedAngle != rcptID {
		t.Fatalf("expected %s, got %s", rcptID, parsedAngle)
	}

	// Invalid addresses
	badAddresses := []string{
		"not-an-email",
		"user@example.com",
		"bounce@example.com",
		"bounce+@example.com",
		"other+token@example.com",
	}
	for _, bad := range badAddresses {
		_, err := ParseVERPAddress(bad, secret)
		if err == nil {
			t.Errorf("expected error for invalid address: %s", bad)
		}
	}
}
