package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// EnvelopeEncryption manages KEK wrapping of DEKs and secret encryption
type EnvelopeEncryption struct {
	kek []byte
}

func NewEnvelopeEncryption(kek []byte) (*EnvelopeEncryption, error) {
	if len(kek) != 32 {
		return nil, fmt.Errorf("master KEK must be exactly 32 bytes (AES-256), got %d bytes", len(kek))
	}
	return &EnvelopeEncryption{kek: kek}, nil
}

// GenerateDEK creates a cryptographically secure 256-bit data encryption key
func (e *EnvelopeEncryption) GenerateDEK() ([]byte, error) {
	dek := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, dek); err != nil {
		return nil, fmt.Errorf("failed to generate random DEK: %w", err)
	}
	return dek, nil
}

// EncryptDEK wraps a tenant DEK with the master KEK using AES-256-GCM
func (e *EnvelopeEncryption) EncryptDEK(dek []byte) (string, error) {
	block, err := aes.NewCipher(e.kek)
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

// DecryptDEK unwraps an encrypted tenant DEK using the master KEK
func (e *EnvelopeEncryption) DecryptDEK(encryptedDEKB64 string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedDEKB64)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(e.kek)
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

// EncryptSecret encrypts an arbitrary secret (DKIM key, webhook secret, body) using a tenant DEK
func EncryptSecret(dek []byte, plaintext []byte) (string, error) {
	if len(dek) != 32 {
		return "", errors.New("tenant DEK must be exactly 32 bytes")
	}

	block, err := aes.NewCipher(dek)
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

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptSecret decrypts an encrypted secret using a tenant DEK
func DecryptSecret(dek []byte, encryptedSecretB64 string) ([]byte, error) {
	if len(dek) != 32 {
		return nil, errors.New("tenant DEK must be exactly 32 bytes")
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encryptedSecretB64)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("malformed encrypted secret: ciphertext too short")
	}

	nonce, actualCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt secret: %w", err)
	}

	return plaintext, nil
}
