package health

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/mailvex/mailvex/core/domain"
	"github.com/mailvex/mailvex/core/repository"
)

type mockResolver struct {
	ptrs map[string][]string
	ips  map[string][]net.IP
	txts map[string][]string
}

func (m *mockResolver) LookupAddr(ctx context.Context, addr string) ([]string, error) {
	if ptrs, ok := m.ptrs[addr]; ok {
		return ptrs, nil
	}
	return nil, fmt.Errorf("no ptr for %s", addr)
}

func (m *mockResolver) LookupIP(ctx context.Context, network, host string) ([]net.IP, error) {
	if ips, ok := m.ips[host]; ok {
		return ips, nil
	}
	return nil, fmt.Errorf("no ip for %s", host)
}

func (m *mockResolver) LookupTXT(ctx context.Context, name string) ([]string, error) {
	if txts, ok := m.txts[name]; ok {
		return txts, nil
	}
	return nil, fmt.Errorf("no txt for %s", name)
}

func TestFCrDNSVerification(t *testing.T) {
	mock := &mockResolver{
		ptrs: map[string][]string{
			"198.51.100.25": {"mail.example.com."},
			"198.51.100.26": {"unconfirmed.example.com."},
		},
		ips: map[string][]net.IP{
			"mail.example.com":        {net.ParseIP("198.51.100.25")},
			"unconfirmed.example.com": {net.ParseIP("203.0.113.99")}, // Mismatch!
		},
	}

	evaluator := NewFCrDNSEvaluator(mock)
	ctx := context.Background()

	// 1. Confirmed FCrDNS
	res, err := evaluator.Verify(ctx, "198.51.100.25")
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !res.IsConfirmed || res.ConfirmedDomain != "mail.example.com" {
		t.Errorf("Expected confirmed FCrDNS, got: %+v", res)
	}

	// 2. Mismatch (Unconfirmed)
	resUnconf, err := evaluator.Verify(ctx, "198.51.100.26")
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if resUnconf.IsConfirmed {
		t.Errorf("Expected unconfirmed FCrDNS due to IP mismatch, got: %+v", resUnconf)
	}

	// 3. Missing PTR
	resMissing, err := evaluator.Verify(ctx, "198.51.100.99")
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if resMissing.IsConfirmed {
		t.Errorf("Expected unconfirmed for missing PTR, got: %+v", resMissing)
	}
}

func TestComplianceEvaluation(t *testing.T) {
	repo := repository.NewMemoryRepository()
	ctx := context.Background()

	// Seed org, tenant, and domain in repo as verified
	_ = repo.CreateOrganization(ctx, &domain.Organization{ID: "org_default", Name: "Test Org", Slug: "test-org", Status: "active"})
	_ = repo.CreateTenant(ctx, &domain.Tenant{ID: "tenant_default", OrganizationID: "org_default", Name: "Default Tenant", Slug: "default-tenant", Environment: "live", Status: "active"})
	_ = repo.CreateDomain(ctx, &domain.Domain{
		ID:       "dom_acme",
		TenantID: "tenant_default",
		Name:     "acme.com",
		Status:   "verified",
	})

	mock := &mockResolver{
		ptrs: map[string][]string{
			"198.51.100.25": {"mail.acme.com."},
		},
		ips: map[string][]net.IP{
			"mail.acme.com": {net.ParseIP("198.51.100.25")},
		},
		txts: map[string][]string{
			"acme.com":        {"v=spf1 include:_spf.mailvex.com ~all"},
			"_dmarc.acme.com": {"v=DMARC1; p=reject; rua=mailto:dmarc@acme.com; pct=100"},
		},
	}

	evaluator := NewComplianceEvaluator(repo, mock)

	check, err := evaluator.EvaluateDomain(ctx, "tenant_default", "acme.com", "198.51.100.25")
	if err != nil {
		t.Fatalf("EvaluateDomain failed: %v", err)
	}

	// Should have perfect score (20+20+20+15+15+10 = 100)
	if check.OverallScore != 100 {
		t.Errorf("Expected 100 overall score, got %d. Check: %+v", check.OverallScore, check)
	}
	if check.SPFStatus != "pass" || check.DKIMStatus != "pass" || check.DMARCStatus != "pass" || check.FCrDNSStatus != "pass" {
		t.Errorf("Expected all pass statuses, got: SPF=%s, DKIM=%s, DMARC=%s, FCrDNS=%s",
			check.SPFStatus, check.DKIMStatus, check.DMARCStatus, check.FCrDNSStatus)
	}

	// Verify saved to repo
	saved, err := repo.GetLatestComplianceCheck(ctx, "tenant_default", "acme.com")
	if err != nil || saved == nil {
		t.Fatalf("Saved compliance check not found in repo: %v", err)
	}
	if saved.OverallScore != 100 {
		t.Errorf("Expected saved score 100, got %d", saved.OverallScore)
	}
}

func TestTLSCheckResultStrings(t *testing.T) {
	checker := NewTLSChecker()
	if checker.dialTimeout != 5*time.Second {
		t.Errorf("Expected 5s dial timeout, got %v", checker.dialTimeout)
	}
}
