package tracking_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mailvex/mailvex/core/repository"
	"github.com/mailvex/mailvex/core/tracking"
)

func TestTrackingTokens(t *testing.T) {
	repo := repository.NewMemoryRepository()
	secret := []byte("test-tracking-secret-key-32-chars!")
	engine := tracking.NewTrackingEngine(secret, repo)

	tenantID := "tenant_track_1"
	messageID := "msg_track_1"
	recipientID := "rcpt_track_1"
	linkID := "link_track_1"

	// 1. Open Token
	openToken := engine.SignOpenToken(tenantID, messageID, recipientID)
	vTenant, vMsg, vRcpt, err := engine.VerifyOpenToken(openToken)
	if err != nil {
		t.Fatalf("failed to verify open token: %v", err)
	}
	if vTenant != tenantID || vMsg != messageID || vRcpt != recipientID {
		t.Errorf("token payload mismatch: %s, %s, %s", vTenant, vMsg, vRcpt)
	}

	// Tampered open token rejection
	tampered := openToken[:len(openToken)-4] + "AAAA"
	_, _, _, err = engine.VerifyOpenToken(tampered)
	if err == nil {
		t.Fatalf("expected error for tampered token, got nil")
	}

	// 2. Click Token
	clickToken := engine.SignClickToken(tenantID, messageID, recipientID, linkID)
	cTenant, cMsg, cRcpt, cLink, err := engine.VerifyClickToken(clickToken)
	if err != nil {
		t.Fatalf("failed to verify click token: %v", err)
	}
	if cTenant != tenantID || cMsg != messageID || cRcpt != recipientID || cLink != linkID {
		t.Errorf("click token mismatch: %s, %s, %s, %s", cTenant, cMsg, cRcpt, cLink)
	}

	// Tampered click token rejection
	tamperedClick := clickToken[:len(clickToken)-4] + "ZZZZ"
	_, _, _, _, err = engine.VerifyClickToken(tamperedClick)
	if err == nil {
		t.Fatalf("expected error for tampered click token, got nil")
	}
}

func TestTransparentGIF(t *testing.T) {
	if len(tracking.TransparentGIF) != 43 {
		t.Errorf("expected 43 bytes GIF, got %d", len(tracking.TransparentGIF))
	}
	if string(tracking.TransparentGIF[:6]) != "GIF89a" {
		t.Errorf("expected GIF89a header, got %s", string(tracking.TransparentGIF[:6]))
	}
}

func TestBotClassification(t *testing.T) {
	now := time.Now().UTC()
	deliveredAt := now.Add(-5 * time.Minute)

	// Google Image Proxy
	isBot, reason := tracking.ClassifyHit("GoogleImageProxy", "66.249.90.1", deliveredAt, now)
	if !isBot || reason != "google_image_proxy" {
		t.Errorf("expected GoogleImageProxy bot classification, got %v (%s)", isBot, reason)
	}

	// Microsoft SafeLinks
	isBot, reason = tracking.ClassifyHit("Mozilla/5.0 (Windows NT 10.0; Win64; x64) BingPreview/1.0b", "40.76.0.1", deliveredAt, now)
	if !isBot || reason != "microsoft_safelinks" {
		t.Errorf("expected Microsoft SafeLinks classification, got %v (%s)", isBot, reason)
	}

	// Fast prefetch hit (< 2s after delivery)
	isBot, reason = tracking.ClassifyHit("Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X)", "1.2.3.4", now.Add(-500*time.Millisecond), now)
	if !isBot || reason != "prefetch_instant_hit" {
		t.Errorf("expected fast prefetch instant hit classification, got %v (%s)", isBot, reason)
	}

	// Normal Human Hit (5 min after delivery, standard desktop UA)
	isBot, reason = tracking.ClassifyHit("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36", "98.12.34.56", deliveredAt, now)
	if isBot {
		t.Errorf("expected human hit not to be classified as bot, got reason: %s", reason)
	}
}

func TestIPTruncation(t *testing.T) {
	ipv4 := tracking.TruncateIP("192.168.1.42")
	if ipv4 != "192.168.1.0" {
		t.Errorf("expected 192.168.1.0, got %s", ipv4)
	}

	ipv6 := tracking.TruncateIP("2001:0db8:85a3:0000:0000:8a2e:0370:7334")
	if !strings.HasPrefix(ipv6, "2001:db8:85a3::") && !strings.HasPrefix(ipv6, "2001:0db8:85a3::") {
		t.Errorf("expected /48 masked IPv6, got %s", ipv6)
	}
}

func TestRewriteHTML(t *testing.T) {
	repo := repository.NewMemoryRepository()
	secret := []byte("test-tracking-secret-key-32-chars!")
	engine := tracking.NewTrackingEngine(secret, repo)
	ctx := context.Background()

	rawHTML := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
  <h1>Welcome</h1>
  <p><a href="https://example.com/pricing">Check Pricing</a></p>
  <p><a href="mailto:support@example.com">Contact Us</a></p>
  <p><a href="https://example.com/unsub" data-mvx-notrack="true">Opt-out</a></p>
</body>
</html>`

	rewritten, err := engine.RewriteHTML(ctx, rawHTML, "tenant_1", "msg_1", "rcpt_1", "https://track.mailvex.com", true, true)
	if err != nil {
		t.Fatalf("RewriteHTML failed: %v", err)
	}

	// 1. Check open pixel was injected
	if !strings.Contains(rewritten, `https://track.mailvex.com/o/`) {
		t.Errorf("tracking pixel was not injected")
	}

	// 2. Check trackable link was rewritten
	if !strings.Contains(rewritten, `https://track.mailvex.com/c/`) {
		t.Errorf("pricing link was not rewritten to trackable URL")
	}

	// 3. Check mailto was NOT rewritten
	if !strings.Contains(rewritten, `href="mailto:support@example.com"`) {
		t.Errorf("mailto link was incorrectly rewritten")
	}

	// 4. Check data-mvx-notrack was NOT rewritten
	if !strings.Contains(rewritten, `href="https://example.com/unsub" data-mvx-notrack="true"`) {
		t.Errorf("notrack link was incorrectly rewritten")
	}
}
