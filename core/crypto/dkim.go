package crypto

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
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

	// Split message into headers and body
	parts := bytes.SplitN(rawMIME, []byte("\r\n\r\n"), 2)
	var headerBytes, bodyBytes []byte
	if len(parts) == 2 {
		headerBytes = parts[0]
		bodyBytes = parts[1]
	} else {
		// Fallback for \n\n
		parts = bytes.SplitN(rawMIME, []byte("\n\n"), 2)
		if len(parts) == 2 {
			headerBytes = parts[0]
			bodyBytes = parts[1]
		} else {
			headerBytes = rawMIME
			bodyBytes = []byte{}
		}
	}

	// 1. Canonicalize body (relaxed)
	canonicalBody := canonicalizeBodyRelaxed(bodyBytes)
	bodyHash := sha256.Sum256(canonicalBody)
	bhB64 := base64.StdEncoding.EncodeToString(bodyHash[:])

	// Headers to sign
	headersToSign := []string{"from", "to", "subject", "date", "message-id"}
	var signedHeadersList []string

	// Parse headers into key -> value
	rawHeaders := parseHeaders(headerBytes)
	var canonicalHeaders bytes.Buffer
	for _, hName := range headersToSign {
		for _, h := range rawHeaders {
			if strings.EqualFold(h.key, hName) {
				signedHeadersList = append(signedHeadersList, hName)
				canonicalHeaders.WriteString(canonicalizeHeaderRelaxed(h.key, h.val))
				canonicalHeaders.WriteString("\r\n")
				break
			}
		}
	}

	hTag := strings.Join(signedHeadersList, ":")
	if hTag == "" {
		hTag = "from:to:subject"
	}

	// 2. Prepare DKIM-Signature header without b=
	dkimHeaderPrefix := fmt.Sprintf("v=1; a=rsa-sha256; c=relaxed/relaxed; d=%s; s=%s; bh=%s; h=%s; b=",
		domain, selector, bhB64, hTag)

	dkimHeaderCanonical := canonicalizeHeaderRelaxed("dkim-signature", dkimHeaderPrefix)
	canonicalHeaders.WriteString(dkimHeaderCanonical)

	// 3. Compute RSA-SHA256 signature
	headerHash := sha256.Sum256(canonicalHeaders.Bytes())
	sig, err := rsa.SignPKCS1v15(rand.Reader, rsaPriv, crypto.SHA256, headerHash[:])
	if err != nil {
		return nil, fmt.Errorf("failed to sign header hash: %w", err)
	}
	sigB64 := base64.StdEncoding.EncodeToString(sig)

	finalDKIMHeader := fmt.Sprintf("DKIM-Signature: %s%s\r\n", dkimHeaderPrefix, sigB64)

	// Prepend DKIM-Signature to raw MIME
	var result bytes.Buffer
	result.WriteString(finalDKIMHeader)
	result.Write(rawMIME)

	return result.Bytes(), nil
}

type headerKV struct {
	key string
	val string
}

func parseHeaders(headerBytes []byte) []headerKV {
	lines := strings.Split(string(headerBytes), "\n")
	var headers []headerKV
	var currentKey, currentVal string

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if len(line) == 0 {
			continue
		}
		if line[0] == ' ' || line[0] == '\t' {
			currentVal += " " + strings.TrimSpace(line)
		} else {
			if currentKey != "" {
				headers = append(headers, headerKV{key: currentKey, val: currentVal})
			}
			colonIdx := strings.Index(line, ":")
			if colonIdx > 0 {
				currentKey = strings.TrimSpace(line[:colonIdx])
				currentVal = strings.TrimSpace(line[colonIdx+1:])
			}
		}
	}
	if currentKey != "" {
		headers = append(headers, headerKV{key: currentKey, val: currentVal})
	}
	return headers
}

func canonicalizeHeaderRelaxed(k, v string) string {
	k = strings.ToLower(strings.TrimSpace(k))
	// Compress multiple spaces
	v = strings.Join(strings.Fields(v), " ")
	return k + ":" + v
}

func canonicalizeBodyRelaxed(body []byte) []byte {
	if len(body) == 0 {
		return []byte{}
	}
	lines := bytes.Split(body, []byte("\n"))
	var processed [][]byte

	for _, line := range lines {
		line = bytes.TrimRight(line, "\r")
		// Collapse whitespace
		fields := bytes.Fields(line)
		collapsed := bytes.Join(fields, []byte(" "))
		processed = append(processed, collapsed)
	}

	// Remove trailing empty lines
	for len(processed) > 0 && len(processed[len(processed)-1]) == 0 {
		processed = processed[:len(processed)-1]
	}

	if len(processed) == 0 {
		return []byte{}
	}

	var result bytes.Buffer
	for _, l := range processed {
		result.Write(l)
		result.WriteString("\r\n")
	}
	return result.Bytes()
}

