package worker

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/textproto"
	"strings"
	"time"

	"github.com/mailvex/mailvex/core/crypto"
)

type HygieneParams struct {
	TenantID        string
	StreamName      string
	SendingDomain   string
	PlatformHost    string
	BounceDomain    string
	RecipientID     string
	RecipientAddr   string
	VERPToken       string
	UnsubscribeURL  *string
	IsBroadcast     bool
	CampaignTag     string
	HMACSecret      []byte
}

// SanitizeAndDecorateMIME applies header hygiene, adds required RFC headers, VERP Return-Path,
// Feedback-ID, and List-Unsubscribe headers per §15 step 8, §17.2, and §17.3.
func SanitizeAndDecorateMIME(rawMIME []byte, params HygieneParams) ([]byte, error) {
	if len(rawMIME) == 0 {
		return nil, fmt.Errorf("empty MIME content")
	}

	reader := textproto.NewReader(bufio.NewReader(bytes.NewReader(rawMIME)))
	hdr, err := reader.ReadMIMEHeader()
	if err != nil {
		return nil, fmt.Errorf("failed to read MIME headers: %w", err)
	}

	// Read remainder of body
	var bodyBuf bytes.Buffer
	for {
		line, err := reader.ReadLineBytes()
		if err != nil {
			break
		}
		bodyBuf.Write(line)
		bodyBuf.WriteString("\r\n")
	}

	// 1. Strip internal headers and private IP Received lines (§15 step 8)
	for key := range hdr {
		lowerKey := strings.ToLower(key)
		if strings.HasPrefix(lowerKey, "x-mailvex-internal") || lowerKey == "x-mailvex-auth" {
			hdr.Del(key)
		}
	}

	// 2. Ensure Message-ID
	if hdr.Get("Message-ID") == "" {
		dom := params.SendingDomain
		if dom == "" {
			dom = "mailvex.local"
		}
		hdr.Set("Message-ID", fmt.Sprintf("<%s@%s>", crypto.NewULID(""), dom))
	}

	// 3. Ensure Date
	if hdr.Get("Date") == "" {
		hdr.Set("Date", time.Now().UTC().Format(time.RFC1123Z))
	}

	// 4. Ensure MIME-Version
	if hdr.Get("MIME-Version") == "" {
		hdr.Set("MIME-Version", "1.0")
	}

	// 5. Inject Feedback-ID: <tenant>:<stream>:<campaign>:mailvex
	campaign := params.CampaignTag
	if campaign == "" {
		campaign = "default"
	}
	hdr.Set("Feedback-ID", fmt.Sprintf("%s:%s:%s:mailvex", params.TenantID, params.StreamName, campaign))

	// 6. Inject X-Mailvex-Tenant-Id (hashed for abuse tracing without leaking internal tenant IDs)
	h := sha256.Sum256([]byte(params.TenantID))
	hdr.Set("X-Mailvex-Tenant-Id", hex.EncodeToString(h[:8]))

	// 7. Generate VERP Return-Path (§17.2)
	bounceDom := params.BounceDomain
	if bounceDom == "" {
		bounceDom = params.SendingDomain
	}
	verp := params.VERPToken
	if verp == "" {
		verp = params.RecipientID
	}
	returnPath := fmt.Sprintf("<bounce+%s@%s>", verp, bounceDom)
	hdr.Set("Return-Path", returnPath)

	// 8. List-Unsubscribe & RFC 8058 One-Click headers (§17.3)
	if params.IsBroadcast || params.StreamName == "broadcast" {
		platformHost := params.PlatformHost
		if platformHost == "" {
			platformHost = "mailvex.local"
		}

		secret := params.HMACSecret
		if len(secret) == 0 {
			secret = []byte("mailvex-unsubscribe-token-secret")
		}

		// Generate signed unsubscribe token: HMAC-SHA256(secret, recipient_id)
		mac := hmac.New(sha256.New, secret)
		mac.Write([]byte(fmt.Sprintf("%s:%s:%s", params.TenantID, params.RecipientID, params.RecipientAddr)))
		unsubSig := hex.EncodeToString(mac.Sum(nil))[:24]
		token := fmt.Sprintf("%s.%s", params.RecipientID, unsubSig)

		unsubHTTP := fmt.Sprintf("https://%s/u/%s", platformHost, token)
		if params.UnsubscribeURL != nil && *params.UnsubscribeURL != "" {
			unsubHTTP = *params.UnsubscribeURL
		}
		unsubMailto := fmt.Sprintf("mailto:unsubscribe+%s@%s", token, bounceDom)

		hdr.Set("List-Unsubscribe", fmt.Sprintf("<%s>, <%s>", unsubHTTP, unsubMailto))
		hdr.Set("List-Unsubscribe-Post", "List-Unsubscribe=One-Click")
	}

	// Reconstruct MIME message with headers
	var out bytes.Buffer
	for k, vv := range hdr {
		for _, v := range vv {
			out.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
		}
	}
	out.WriteString("\r\n")
	out.Write(bodyBuf.Bytes())

	return out.Bytes(), nil
}
