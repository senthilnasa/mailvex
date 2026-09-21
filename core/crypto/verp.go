package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidVERPFormat = errors.New("invalid VERP address format")
	ErrInvalidVERPToken  = errors.New("invalid or tampered VERP token")
)

var defaultVERPSecret = []byte("mailvex-default-verp-secret-key-32b")

// GenerateVERPToken generates a signed VERP token embedding recipientID with HMAC-SHA256
func GenerateVERPToken(recipientID string, secret []byte) string {
	if len(secret) == 0 {
		secret = defaultVERPSecret
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(recipientID))
	sig := hex.EncodeToString(mac.Sum(nil))[:16]
	return fmt.Sprintf("%s.%s", recipientID, sig)
}

// VerifyAndDecodeVERPToken parses and validates a signed VERP token, returning recipientID
func VerifyAndDecodeVERPToken(token string, secret []byte) (string, error) {
	if len(secret) == 0 {
		secret = defaultVERPSecret
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		// Fallback: if token has no dot, check if it's a bare recipient ID (e.g. ULID)
		if len(parts) == 1 && len(token) >= 10 {
			return token, nil
		}
		return "", ErrInvalidVERPToken
	}

	recipientID := parts[0]
	sig := parts[1]

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(recipientID))
	expectedSig := hex.EncodeToString(mac.Sum(nil))[:16]

	if !hmac.Equal([]byte(sig), []byte(expectedSig)) {
		return "", ErrInvalidVERPToken
	}

	return recipientID, nil
}

// FormatVERPAddress formats a full VERP email address: bounce+<token>@<bounceDomain>
func FormatVERPAddress(recipientID, bounceDomain string, secret []byte) string {
	token := GenerateVERPToken(recipientID, secret)
	return fmt.Sprintf("bounce+%s@%s", token, bounceDomain)
}

// ParseVERPAddress extracts and verifies the recipientID from a VERP bounce email address
// e.g. "bounce+rcpt123.abcdef0123456789@bounce.mailvex.com" or "<bounce+...>"
func ParseVERPAddress(address string, secret []byte) (string, error) {
	// Strip enclosing angle brackets and whitespace
	cleaned := strings.TrimSpace(address)
	cleaned = strings.TrimPrefix(cleaned, "<")
	cleaned = strings.TrimSuffix(cleaned, ">")

	atIdx := strings.LastIndex(cleaned, "@")
	if atIdx == -1 {
		return "", ErrInvalidVERPFormat
	}
	localPart := cleaned[:atIdx]

	plusIdx := strings.Index(localPart, "+")
	if plusIdx == -1 {
		return "", ErrInvalidVERPFormat
	}

	prefix := localPart[:plusIdx]
	if !strings.EqualFold(prefix, "bounce") {
		return "", ErrInvalidVERPFormat
	}

	token := localPart[plusIdx+1:]
	if token == "" {
		return "", ErrInvalidVERPFormat
	}

	return VerifyAndDecodeVERPToken(token, secret)
}
