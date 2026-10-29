package smtp

import (
	"context"
	"strings"
	"testing"

	gosmtp "github.com/emersion/go-smtp"
	"github.com/senthilnasa/mailvex/core"
	"github.com/senthilnasa/mailvex/core/crypto"
	"github.com/senthilnasa/mailvex/core/domain"
	"github.com/senthilnasa/mailvex/core/repository"
	"github.com/senthilnasa/mailvex/core/storage"
)

func setupTestSMTPEnvironment(t *testing.T) (repository.Repository, *Backend) {
	repo := repository.NewMemoryRepository()
	store := storage.NewMemoryStorage()
	ingest := core.NewIngestionEngine(repo, store)
	backend := NewBackend(repo, ingest)

	ctx := context.Background()

	// 1. Create Organization & Tenant
	orgID := crypto.NewULID(crypto.PrefixOrg)
	_ = repo.CreateOrganization(ctx, &domain.Organization{
		ID: orgID, Name: "Acme Corp", Slug: "acme", Status: "active",
	})
	tenantID := crypto.NewULID(crypto.PrefixTenant)
	_ = repo.CreateTenant(ctx, &domain.Tenant{
		ID: tenantID, OrganizationID: orgID, Name: "Acme Main", Slug: "acme-main", Environment: "development", Status: "active",
	})

	// 2. Add verified sending domain
	_ = repo.CreateDomain(ctx, &domain.Domain{
		ID: crypto.NewULID(crypto.PrefixDomain), TenantID: tenantID, Name: "acme.test", Type: "public", Status: "verified", OwnershipToken: "tok123",
	})

	// 3. Add default message stream
	_ = repo.CreateStream(ctx, &domain.MessageStream{
		ID: crypto.NewULID(crypto.PrefixStream), TenantID: tenantID, Name: "transactional", Type: "transactional", IsDefault: true,
	})

	// 4. Create SMTP Credential
	pwdHash, _ := crypto.HashPassword("SecretPassword123")
	_ = repo.CreateSMTPCredential(ctx, &domain.SMTPCredential{
		ID:           "smtp_user1",
		TenantID:     tenantID,
		Username:     "smtp_user1",
		PasswordHash: pwdHash,
		RateLimit:    100,
	})

	// 5. Add Trusted LAN network and registered device
	_ = repo.AddTrustedNetwork(ctx, &domain.TrustedNetwork{
		ID: "tn_1", TenantID: tenantID, CIDR: "192.168.1.0/24", Description: "Home Lab LAN",
	})
	_ = repo.CreateDevice(ctx, &domain.Device{
		ID: "dev_cam1", TenantID: tenantID, Name: "Driveway Camera", Type: "camera", ExpectedIP: "192.168.1.50", AllowedFrom: []string{"camera@acme.test"},
	})

	return repo, backend
}

func TestSMTPAuthenticatedSubmission(t *testing.T) {
	repo, backend := setupTestSMTPEnvironment(t)
	ctx := context.Background()

	// Create session from standard external IP
	session := &Session{
		backend:  backend,
		clientIP: "203.0.113.10",
	}

	// 1. Authenticate via SASL PLAIN
	saslServer, err := session.Auth("PLAIN")
	if err != nil {
		t.Fatalf("failed to get sasl server: %v", err)
	}

	// SASL PLAIN format: \x00username\x00password
	authPayload := []byte("\x00smtp_user1\x00SecretPassword123")
	_, done, err := saslServer.Next(authPayload)
	if err != nil || !done || !session.authenticated {
		t.Fatalf("authentication failed: err=%v, done=%v, authenticated=%v", err, done, session.authenticated)
	}

	// 2. Issue MAIL FROM
	if err := session.Mail("sender@acme.test", nil); err != nil {
		t.Fatalf("MAIL FROM failed: %v", err)
	}

	// 3. Issue RCPT TO
	if err := session.Rcpt("recipient@example.com", nil); err != nil {
		t.Fatalf("RCPT TO failed: %v", err)
	}

	// 4. Transmit DATA
	rawEmail := strings.NewReader("Subject: Production Invoice\r\nTo: recipient@example.com\r\n\r\nThank you for your order.")
	if err := session.Data(rawEmail); err != nil {
		t.Fatalf("DATA submission failed: %v", err)
	}

	// 5. Verify ingestion into repository and development sink
	tenants, _ := repo.ListTenantsByOrg(ctx, "org_")
	var tenantID string
	for _, tr := range tenants {
		tenantID = tr.ID
		break
	}

	devMsgs, err := repo.ListDevelopmentMessages(ctx, session.tenantID)
	if err != nil || len(devMsgs) != 1 {
		t.Fatalf("expected 1 message in development sink, got %d err=%v", len(devMsgs), err)
	}

	// Verify Outbox item created
	pendingOutbox, _ := repo.ListPendingOutbox(ctx, 10)
	if len(pendingOutbox) != 1 {
		t.Fatalf("expected 1 outbox record created for delivery hint, got %d", len(pendingOutbox))
	}
	_ = tenantID
}

func TestSMTPDeviceIPAuthenticatedSubmission(t *testing.T) {
	repo, backend := setupTestSMTPEnvironment(t)

	// Simulated session originating from registered IoT device IP: 192.168.1.50
	session := &Session{
		backend:  backend,
		clientIP: "192.168.1.50",
	}

	// Issue MAIL FROM without explicit AUTH:
	// Backend detects trusted network and registered device -> authenticates session
	if err := session.Mail("camera@acme.test", nil); err != nil {
		t.Fatalf("IoT device submission failed on MAIL FROM: %v", err)
	}

	if !session.authenticated || session.device == nil {
		t.Fatalf("expected session to be authenticated via registered device IP")
	}

	// Issue RCPT TO
	if err := session.Rcpt("alerts@acme.test", nil); err != nil {
		t.Fatalf("IoT RCPT TO failed: %v", err)
	}

	// Transmit IoT alert
	rawAlert := strings.NewReader("Subject: Motion Detected\r\n\r\nDriveway motion alert at front gate.")
	if err := session.Data(rawAlert); err != nil {
		t.Fatalf("IoT DATA submission failed: %v", err)
	}

	// Verify message ingested
	devMsgs, err := repo.ListDevelopmentMessages(context.Background(), session.tenantID)
	if err != nil || len(devMsgs) != 1 {
		t.Fatalf("expected 1 message in sink for IoT alert, got %d", len(devMsgs))
	}
}

func TestOpenRelayPrevention(t *testing.T) {
	_, backend := setupTestSMTPEnvironment(t)

	// Unauthenticated session from untrusted IP
	session := &Session{
		backend:       backend,
		clientIP:      "198.51.100.25",
		authenticated: false,
	}

	if err := session.Mail("spammer@evil.test", nil); err != nil {
		t.Fatalf("MAIL FROM should be accepted pending RCPT check: %v", err)
	}

	// Attempting to deliver to external domain (not hosted on Mailvex)
	err := session.Rcpt("victim@target.test", nil)
	if err == nil {
		t.Fatalf("OPEN RELAY VULNERABILITY: expected relaying denied for unauthenticated external delivery")
	}

	smtpErr, ok := err.(*gosmtp.SMTPError)
	if !ok || smtpErr.Code != 550 {
		t.Fatalf("expected SMTP 550 Relaying denied, got %v", err)
	}
}

func TestBruteForceProtection(t *testing.T) {
	_, backend := setupTestSMTPEnvironment(t)
	attackerIP := "203.0.113.99"

	session := &Session{
		backend:  backend,
		clientIP: attackerIP,
	}

	// Submit 5 bad passwords
	for i := 0; i < 5; i++ {
		saslServer, _ := session.Auth("PLAIN")
		_, _, _ = saslServer.Next([]byte("\x00smtp_user1\x00WrongPassword"))
	}

	// Check directly against backend blocked check:
	backend.mu.Lock()
	_, blocked := backend.blockedIPs[attackerIP]
	backend.mu.Unlock()

	if !blocked {
		t.Fatalf("expected IP %s to be blocked after 5 failed authentication attempts", attackerIP)
	}
}
