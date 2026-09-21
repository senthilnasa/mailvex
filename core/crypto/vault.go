package crypto

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// KEKProvider represents a pluggable Key Encryption Key provider for envelope encryption (§19.2)
type KEKProvider interface {
	WrapDEK(ctx context.Context, dek []byte) (string, error)
	UnwrapDEK(ctx context.Context, wrappedB64 string) ([]byte, error)
	KeyID() string
}

// LocalKEKProvider implements KEKProvider using a local 32-byte AES-256 key
type LocalKEKProvider struct {
	keyID string
	kek   []byte
}

// NewLocalKEKProvider creates a local AES-256-GCM KEK provider
func NewLocalKEKProvider(keyID string, kek []byte) (*LocalKEKProvider, error) {
	if len(kek) != 32 {
		return nil, errors.New("master KEK must be exactly 32 bytes (AES-256)")
	}
	if keyID == "" {
		keyID = "local-kek-v1"
	}
	return &LocalKEKProvider{keyID: keyID, kek: kek}, nil
}

func (p *LocalKEKProvider) KeyID() string {
	return p.keyID
}

func (p *LocalKEKProvider) WrapDEK(ctx context.Context, dek []byte) (string, error) {
	block, err := aes.NewCipher(p.kek)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, dek, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (p *LocalKEKProvider) UnwrapDEK(ctx context.Context, wrappedB64 string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(wrappedB64)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(p.kek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("malformed encrypted DEK: ciphertext too short")
	}
	nonce, actualCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	dek, err := gcm.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt DEK: %w", err)
	}
	return dek, nil
}

// VaultTransitConfig contains settings for connecting to HashiCorp Vault Transit Engine
type VaultTransitConfig struct {
	Address   string        // e.g. "https://vault.internal:8200"
	Token     string        // Vault authentication token
	Namespace string        // Optional Vault enterprise namespace
	KeyName   string        // Transit key name, e.g. "mailvex-kek"
	Timeout   time.Duration // HTTP request timeout
}

// VaultTransitKEKProvider implements KEKProvider via HashiCorp Vault Transit Secret Engine
type VaultTransitKEKProvider struct {
	cfg    VaultTransitConfig
	client *http.Client
}

// NewVaultTransitKEKProvider creates a HashiCorp Vault Transit KEK provider
func NewVaultTransitKEKProvider(cfg VaultTransitConfig, customClient *http.Client) (*VaultTransitKEKProvider, error) {
	if cfg.Address == "" {
		return nil, errors.New("vault address is required")
	}
	if cfg.KeyName == "" {
		cfg.KeyName = "mailvex-master-kek"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}
	client := customClient
	if client == nil {
		client = &http.Client{Timeout: cfg.Timeout}
	}
	return &VaultTransitKEKProvider{cfg: cfg, client: client}, nil
}

func (v *VaultTransitKEKProvider) KeyID() string {
	return fmt.Sprintf("vault:%s", v.cfg.KeyName)
}

func (v *VaultTransitKEKProvider) WrapDEK(ctx context.Context, dek []byte) (string, error) {
	plaintextB64 := base64.StdEncoding.EncodeToString(dek)
	reqBody, err := json.Marshal(map[string]interface{}{
		"plaintext": plaintextB64,
	})
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/v1/transit/encrypt/%s", strings.TrimRight(v.cfg.Address, "/"), v.cfg.KeyName)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if v.cfg.Token != "" {
		req.Header.Set("X-Vault-Token", v.cfg.Token)
	}
	if v.cfg.Namespace != "" {
		req.Header.Set("X-Vault-Namespace", v.cfg.Namespace)
	}

	resp, err := v.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("vault transit encrypt request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("vault transit encrypt returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var vaultResp struct {
		Data struct {
			Ciphertext string `json:"ciphertext"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&vaultResp); err != nil {
		return "", fmt.Errorf("failed to decode vault encrypt response: %w", err)
	}
	if vaultResp.Data.Ciphertext == "" {
		return "", errors.New("vault returned empty ciphertext")
	}

	return vaultResp.Data.Ciphertext, nil
}

func (v *VaultTransitKEKProvider) UnwrapDEK(ctx context.Context, wrapped string) ([]byte, error) {
	reqBody, err := json.Marshal(map[string]interface{}{
		"ciphertext": wrapped,
	})
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/v1/transit/decrypt/%s", strings.TrimRight(v.cfg.Address, "/"), v.cfg.KeyName)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if v.cfg.Token != "" {
		req.Header.Set("X-Vault-Token", v.cfg.Token)
	}
	if v.cfg.Namespace != "" {
		req.Header.Set("X-Vault-Namespace", v.cfg.Namespace)
	}

	resp, err := v.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vault transit decrypt request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("vault transit decrypt returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var vaultResp struct {
		Data struct {
			Plaintext string `json:"plaintext"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&vaultResp); err != nil {
		return nil, fmt.Errorf("failed to decode vault decrypt response: %w", err)
	}
	if vaultResp.Data.Plaintext == "" {
		return nil, errors.New("vault returned empty plaintext")
	}

	rawDEK, err := base64.StdEncoding.DecodeString(vaultResp.Data.Plaintext)
	if err != nil {
		return nil, fmt.Errorf("failed to base64 decode vault plaintext: %w", err)
	}
	return rawDEK, nil
}

// KeyRotationResult summarizes a DEK re-wrapping execution
type KeyRotationResult struct {
	TotalRotated int           `json:"total_rotated"`
	Errors       []string      `json:"errors,omitempty"`
	Duration     time.Duration `json:"duration"`
	OldKeyID     string        `json:"old_key_id"`
	NewKeyID     string        `json:"new_key_id"`
}

// RotateDEK re-encrypts a tenant DEK from an old KEK provider to a new KEK provider
func RotateDEK(ctx context.Context, oldProvider KEKProvider, newProvider KEKProvider, oldEncryptedDEK string) (string, error) {
	rawDEK, err := oldProvider.UnwrapDEK(ctx, oldEncryptedDEK)
	if err != nil {
		return "", fmt.Errorf("failed to unwrap DEK with old KEK (%s): %w", oldProvider.KeyID(), err)
	}

	if len(rawDEK) != 32 {
		return "", fmt.Errorf("unwrapped DEK has invalid length %d (expected 32)", len(rawDEK))
	}

	newEncryptedDEK, err := newProvider.WrapDEK(ctx, rawDEK)
	if err != nil {
		return "", fmt.Errorf("failed to wrap DEK with new KEK (%s): %w", newProvider.KeyID(), err)
	}

	// Verify immediately by unwrapping with new provider
	verifiedDEK, err := newProvider.UnwrapDEK(ctx, newEncryptedDEK)
	if err != nil {
		return "", fmt.Errorf("verification of newly wrapped DEK failed: %w", err)
	}
	if !bytes.Equal(rawDEK, verifiedDEK) {
		return "", errors.New("verification mismatch: re-wrapped DEK does not match original DEK")
	}

	return newEncryptedDEK, nil
}
