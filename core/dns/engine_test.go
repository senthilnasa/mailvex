package dns

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mailvex/mailvex/core/domain"
)

func TestDNSEngineAndVerification(t *testing.T) {
	ctx := context.Background()
	stub := NewStubResolver()
	engine := NewEngine(stub)

	dom := &domain.Domain{
		ID:             "dom_123",
		TenantID:       "tenant_123",
		Name:           "example.com",
		Type:           "public",
		Status:         "pending",
		OwnershipToken: "token_abc_xyz",
	}

	dkim := &domain.DKIMKey{
		ID:        "dkim_123",
		DomainID:  "dom_123",
		Selector:  "mvx",
		PublicKey: "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQE...",
		Status:    "pending",
	}

	platformHost := "mail.platform.test"

	// 1. Initially without DNS records, verification must fail
	results, allPassed := engine.VerifyDomain(ctx, dom, dkim, platformHost)
	if allPassed {
		t.Fatalf("expected verification to fail on empty DNS")
	}

	// 2. Populate stub resolver with correct records
	stub.TXTRecords["_mailvex-verify.example.com"] = []string{
		fmt.Sprintf("mailvex-verify=%s", dom.OwnershipToken),
	}
	stub.TXTRecords["mvx._domainkey.example.com"] = []string{
		fmt.Sprintf("v=DKIM1; k=rsa; p=%s", dkim.PublicKey),
	}
	stub.TXTRecords["example.com"] = []string{
		fmt.Sprintf("v=spf1 include:%s -all", platformHost),
	}
	stub.TXTRecords["_dmarc.example.com"] = []string{
		fmt.Sprintf("v=DMARC1; p=none; rua=mailto:dmarc-reports@%s", platformHost),
	}

	// 3. Re-run verification: all checks should pass
	results, allPassed = engine.VerifyDomain(ctx, dom, dkim, platformHost)
	if !allPassed {
		t.Fatalf("expected all checks to pass, got results: %+v", results)
	}

	// 4. Test Degraded and Suspended status progression (§10.1)
	dom.Status = "verified"
	newStatus, changed := CheckDegradedStatus(dom, false, 72)
	if !changed || newStatus != "degraded" {
		t.Fatalf("expected transition to degraded on DNS failure, got status %s changed=%v", newStatus, changed)
	}

	// Within grace period (e.g. 10 hours later), should remain degraded
	tenHoursAgo := time.Now().UTC().Add(-10 * time.Hour)
	dom.DegradedAt = &tenHoursAgo
	newStatus, _ = CheckDegradedStatus(dom, false, 72)
	if newStatus != "degraded" {
		t.Fatalf("expected to remain degraded within 72h grace period, got %s", newStatus)
	}

	// Beyond grace period (e.g. 75 hours later), should transition to suspended
	pastGrace := time.Now().UTC().Add(-75 * time.Hour)
	dom.DegradedAt = &pastGrace
	newStatus, changed = CheckDegradedStatus(dom, false, 72)
	if !changed || newStatus != "suspended" {
		t.Fatalf("expected suspended after grace period, got status %s changed=%v", newStatus, changed)
	}

	// When DNS recovers, should return to verified
	dom.Status = "degraded"
	newStatus, changed = CheckDegradedStatus(dom, true, 72)
	if !changed || newStatus != "verified" {
		t.Fatalf("expected recovery to verified, got status %s changed=%v", newStatus, changed)
	}
}
