package crypto

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/emersion/go-msgauth/dkim"
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

// SignDKIM signs a MIME message using RFC 6376 with RSA-SHA256 and relaxed/relaxed canonicalization
func SignDKIM(rawMIME []byte, domain, selector, privateKeyPEM string) ([]byte, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the private key")
	}

	privKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		// Fallback to PKCS1
		var err2 error
		privKey, err2 = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("failed to parse private key: %w (pkcs1: %v)", err, err2)
		}
	}

	rsaPriv, ok := privKey.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("not an RSA private key")
	}

	options := &dkim.SignOptions{
		Domain:                 domain,
		Selector:               selector,
		Signer:                 rsaPriv,
		Hash:                   crypto.SHA256,
		HeaderCanonicalization: dkim.CanonicalizationRelaxed,
		BodyCanonicalization:   dkim.CanonicalizationRelaxed,
		HeaderKeys:             []string{"From", "To", "Subject", "Date", "Message-ID"},
	}

	var signed bytes.Buffer
	if err := dkim.Sign(&signed, bytes.NewReader(rawMIME), options); err != nil {
		return nil, fmt.Errorf("failed to sign DKIM: %w", err)
	}

	return signed.Bytes(), nil
}

