package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
)

type DKIMKeyPair struct {
	PrivateKeyPEM string
	PublicKeyB64  string // base64 encoded DER public key for p= tag
}

// GenerateDKIMKey creates an RSA 2048-bit key pair formatted for DKIM
func GenerateDKIMKey(bits int) (*DKIMKeyPair, error) {
	if bits < 2048 {
		return nil, errors.New("DKIM RSA key length under 2048 bits refused")
	}

	privKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	privASN1, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}

	privBlock := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privASN1,
	}
	privPEM := string(pem.EncodeToMemory(privBlock))

	pubASN1, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public key: %w", err)
	}

	pubB64 := base64.StdEncoding.EncodeToString(pubASN1)

	return &DKIMKeyPair{
		PrivateKeyPEM: privPEM,
		PublicKeyB64:  pubB64,
	}, nil
}
