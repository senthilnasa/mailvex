package tracking

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/mailvex/mailvex/core/crypto"
	"github.com/mailvex/mailvex/core/domain"
	"github.com/mailvex/mailvex/core/repository"
)

// Transparent 1x1 GIF tracking pixel (RFC 2046 compliant)
var TransparentGIF = []byte{
	0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00,
	0x80, 0x00, 0x00, 0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x21,
	0xf9, 0x04, 0x01, 0x00, 0x00, 0x00, 0x00, 0x2c, 0x00, 0x00,
	0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x44,
	0x01, 0x00, 0x3b,
}

var (
	ErrInvalidToken = errors.New("invalid or tampered tracking token")
	ErrTokenExpired = errors.New("tracking token has expired")
)

type TrackingEngine struct {
	secretKey []byte
	repo      repository.Repository
}

func NewTrackingEngine(secretKey []byte, repo repository.Repository) *TrackingEngine {
	if len(secretKey) == 0 {
		secretKey = []byte("mailvex-default-tracking-hmac-secret-key-32")
	}
	return &TrackingEngine{
		secretKey: secretKey,
		repo:      repo,
	}
}

// SignOpenToken creates an HMAC-signed token for open pixel tracking
// Format: base64url("open:" + tenantID + ":" + messageID + ":" + recipientID + ":" + hmac)
func (e *TrackingEngine) SignOpenToken(tenantID, messageID, recipientID string) string {
	payload := fmt.Sprintf("open:%s:%s:%s", tenantID, messageID, recipientID)
	mac := hmac.New(sha256.New, e.secretKey)
	mac.Write([]byte(payload))
	sig := mac.Sum(nil)
	tokenStr := fmt.Sprintf("%s:%x", payload, sig)
	return base64.RawURLEncoding.EncodeToString([]byte(tokenStr))
}

// VerifyOpenToken validates an open pixel token and extracts payload
func (e *TrackingEngine) VerifyOpenToken(token string) (tenantID, messageID, recipientID string, err error) {
	data, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", "", "", ErrInvalidToken
	}
	parts := strings.Split(string(data), ":")
	if len(parts) != 5 || parts[0] != "open" {
		return "", "", "", ErrInvalidToken
	}

	payload := fmt.Sprintf("open:%s:%s:%s", parts[1], parts[2], parts[3])
	mac := hmac.New(sha256.New, e.secretKey)
	mac.Write([]byte(payload))
	expectedSig := fmt.Sprintf("%x", mac.Sum(nil))

	if !hmac.Equal([]byte(parts[4]), []byte(expectedSig)) {
		return "", "", "", ErrInvalidToken
	}

	return parts[1], parts[2], parts[3], nil
}

// SignClickToken creates an HMAC-signed token for click tracking
// Format: base64url("click:" + tenantID + ":" + messageID + ":" + recipientID + ":" + linkID + ":" + hmac)
func (e *TrackingEngine) SignClickToken(tenantID, messageID, recipientID, linkID string) string {
	payload := fmt.Sprintf("click:%s:%s:%s:%s", tenantID, messageID, recipientID, linkID)
	mac := hmac.New(sha256.New, e.secretKey)
	mac.Write([]byte(payload))
	sig := mac.Sum(nil)
	tokenStr := fmt.Sprintf("%s:%x", payload, sig)
	return base64.RawURLEncoding.EncodeToString([]byte(tokenStr))
}

// VerifyClickToken validates a click token and extracts linkID
func (e *TrackingEngine) VerifyClickToken(token string) (tenantID, messageID, recipientID, linkID string, err error) {
	data, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", "", "", "", ErrInvalidToken
	}
	parts := strings.Split(string(data), ":")
	if len(parts) != 6 || parts[0] != "click" {
		return "", "", "", "", ErrInvalidToken
	}

	payload := fmt.Sprintf("click:%s:%s:%s:%s", parts[1], parts[2], parts[3], parts[4])
	mac := hmac.New(sha256.New, e.secretKey)
	mac.Write([]byte(payload))
	expectedSig := fmt.Sprintf("%x", mac.Sum(nil))

	if !hmac.Equal([]byte(parts[5]), []byte(expectedSig)) {
		return "", "", "", "", ErrInvalidToken
	}

	return parts[1], parts[2], parts[3], parts[4], nil
}

// ClassifyHit analyzes User-Agent and timing to filter bot prefetching (§19)
func ClassifyHit(userAgent, ip string, deliveryTime, hitTime time.Time) (bool, string) {
	uaLower := strings.ToLower(userAgent)

	// 1. Google Image Proxy
	if strings.Contains(uaLower, "googleimageproxy") {
		return true, "google_image_proxy"
	}

	// 2. Apple Mail Privacy Protection (MPP) proxy
	if strings.Contains(uaLower, "applemail") || strings.Contains(uaLower, "mozilla/5.0") && strings.Contains(uaLower, "applewebkit") && strings.Contains(uaLower, "proxy") {
		return true, "apple_mpp"
	}

	// 3. Microsoft SafeLinks Scanner
	if strings.Contains(uaLower, "bingpreview") || strings.Contains(uaLower, "safelinks") || strings.Contains(uaLower, "msft") {
		return true, "microsoft_safelinks"
	}

	// 4. Security scanners, link preview bots, crawlers
	knownBots := []string{
		"bot", "spider", "crawler", "scanner", "curl", "wget", "python", "go-http-client",
		"facebookexternalhit", "slackbot", "twitterbot", "whatsapp", "telegram",
	}
	for _, b := range knownBots {
		if strings.Contains(uaLower, b) {
			return true, "scanner_or_crawler"
		}
	}

	// 5. Fast automated hits (< 2s after delivery timestamp) (§19)
	if !deliveryTime.IsZero() && !hitTime.IsZero() {
		if hitTime.Sub(deliveryTime) < 2*time.Second && hitTime.Sub(deliveryTime) > -2*time.Second {
			return true, "prefetch_instant_hit"
		}
	}

	return false, ""
}

// TruncateIP applies IP privacy pseudonymization (IPv4 /24, IPv6 /48) (§19, §23.4)
func TruncateIP(ipStr string) string {
	parsed := net.ParseIP(strings.TrimSpace(ipStr))
	if parsed == nil {
		return ipStr
	}
	if v4 := parsed.To4(); v4 != nil {
		return fmt.Sprintf("%d.%d.%d.0", v4[0], v4[1], v4[2])
	}
	// IPv6 /48
	mask := net.CIDRMask(48, 128)
	masked := parsed.Mask(mask)
	return masked.String()
}

var linkRegex = regexp.MustCompile(`(?i)<a\s+([^>]*?)href=["']([^"']+)["']([^>]*)>`)

// RewriteHTML processes HTML message content, rewrites trackable links to signed /c/ tokens,
// and injects the 1x1 tracking pixel /o/ token before </body> (§19)
func (e *TrackingEngine) RewriteHTML(ctx context.Context, htmlBody, tenantID, messageID, recipientID, baseURL string, openTrack, clickTrack bool) (string, error) {
	res := htmlBody

	if clickTrack {
		res = linkRegex.ReplaceAllStringFunc(res, func(tag string) string {
			// Skip mailto:, tel:, internal anchor #, or elements with data-mvx-notrack
			if strings.Contains(tag, "data-mvx-notrack") {
				return tag
			}

			submatches := linkRegex.FindStringSubmatch(tag)
			if len(submatches) < 4 {
				return tag
			}
			beforeHref := submatches[1]
			targetURL := submatches[2]
			afterHref := submatches[3]

			targetLower := strings.ToLower(targetURL)
			if strings.HasPrefix(targetLower, "mailto:") ||
				strings.HasPrefix(targetLower, "tel:") ||
				strings.HasPrefix(targetLower, "#") ||
				strings.HasPrefix(targetLower, "javascript:") {
				return tag
			}

			linkID := crypto.NewULID(crypto.PrefixTrackingLink)
			linkObj := &domain.TrackingLink{
				ID:        linkID,
				TenantID:  tenantID,
				MessageID: messageID,
				LinkID:    linkID,
				TargetURL: targetURL,
				CreatedAt: time.Now().UTC(),
			}
			if err := e.repo.SaveTrackingLink(ctx, linkObj); err != nil {
				return tag // fallback to original
			}

			token := e.SignClickToken(tenantID, messageID, recipientID, linkID)
			cleanBase := strings.TrimRight(baseURL, "/")
			trackingURL := fmt.Sprintf("%s/c/%s", cleanBase, token)

			return fmt.Sprintf(`<a %shref="%s"%s>`, beforeHref, trackingURL, afterHref)
		})
	}

	if openTrack {
		openToken := e.SignOpenToken(tenantID, messageID, recipientID)
		cleanBase := strings.TrimRight(baseURL, "/")
		pixelTag := fmt.Sprintf(`<img src="%s/o/%s" width="1" height="1" alt="" style="display:none !important;" />`, cleanBase, openToken)

		if idx := strings.LastIndex(strings.ToLower(res), "</body>"); idx != -1 {
			res = res[:idx] + pixelTag + "\n" + res[idx:]
		} else {
			res = res + "\n" + pixelTag
		}
	}

	return res, nil
}
