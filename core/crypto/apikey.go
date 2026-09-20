package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
)

const (
	APIKeyPrefixLive = "mvx_live_"
	APIKeyPrefixTest = "mvx_test_"
	base62Chars      = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

// GenerateAPIKey generates a secure 32-character base62 random token with the proper prefix
func GenerateAPIKey(env string) (rawKey, keyHash, prefix, lastFour string, err error) {
	if env == "test" {
		prefix = APIKeyPrefixTest
	} else {
		prefix = APIKeyPrefixLive
	}

	randomPart, err := randomBase62(32)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to generate random key: %w", err)
	}

	rawKey = prefix + randomPart
	keyHash = HashAPIKey(rawKey)
	lastFour = rawKey[len(rawKey)-4:]

	return rawKey, keyHash, prefix, lastFour, nil
}

// HashAPIKey returns the SHA-256 hex string of the API key for constant-time DB lookups
func HashAPIKey(rawKey string) string {
	sum := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(sum[:])
}

// ValidateAPIKeyFormat checks if key begins with mvx_live_ or mvx_test_
func ValidateAPIKeyFormat(key string) bool {
	return strings.HasPrefix(key, APIKeyPrefixLive) || strings.HasPrefix(key, APIKeyPrefixTest)
}

func randomBase62(n int) (string, error) {
	bytes := make([]byte, n)
	maxVal := big.NewInt(int64(len(base62Chars)))
	for i := 0; i < n; i++ {
		num, err := rand.Int(rand.Reader, maxVal)
		if err != nil {
			return "", err
		}
		bytes[i] = base62Chars[num.Int64()]
	}
	return string(bytes), nil
}
