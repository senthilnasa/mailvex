package dnsbl

import (
	"context"
	"net"
	"testing"

	"github.com/mailvex/mailvex/core/repository"
)

type mockResolver struct {
	responses map[string][]string
}

func (m *mockResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	if addrs, found := m.responses[host]; found {
		return addrs, nil
	}
	return nil, &net.DNSError{IsNotFound: true, Err: "no such host", Name: host}
}

func TestReverseIP(t *testing.T) {
	ipv4 := "192.0.2.1"
	rev4, err := ReverseIP(ipv4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rev4 != "1.2.0.192" {
		t.Errorf("expected 1.2.0.192, got %s", rev4)
	}

	ipv6 := "2001:db8::1"
	rev6, err := ReverseIP(ipv6)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedPrefix := "1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0"
	if len(rev6) < 32 || rev6[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("expected ipv6 reversed starting with %s, got %s", expectedPrefix, rev6)
	}

	_, err = ReverseIP("not-an-ip")
	if err == nil {
		t.Errorf("expected error for invalid IP")
	}
}

func TestCheckIPListedAndClean(t *testing.T) {
	repo := repository.NewMemoryRepository()
	resolver := &mockResolver{
		responses: map[string][]string{
			"1.2.0.192.zen.spamhaus.org": {"127.0.0.2"},
		},
	}
	checker := NewChecker(repo, resolver)

	ctx := context.Background()
	p, err := repo.GetDNSBLProvider(ctx, "dnsbl_zen")
	if err != nil {
		t.Fatalf("failed to get zen provider: %v", err)
	}

	// 1. Listed IP
	resListed, err := checker.CheckIP(ctx, "ip_1", "192.0.2.1", p)
	if err != nil {
		t.Fatalf("CheckIP failed: %v", err)
	}
	if !resListed.IsListed {
		t.Errorf("expected IP to be listed")
	}
	if resListed.ResponseCode == nil || *resListed.ResponseCode != "127.0.0.2" {
		t.Errorf("expected response code 127.0.0.2")
	}
	if resListed.ReturnMeaning == nil || *resListed.ReturnMeaning != "SBL - Direct UBE sources" {
		t.Errorf("expected return meaning from json map, got %v", resListed.ReturnMeaning)
	}

	// 2. Clean IP
	resClean, err := checker.CheckIP(ctx, "ip_2", "192.0.2.2", p)
	if err != nil {
		t.Fatalf("CheckIP failed: %v", err)
	}
	if resClean.IsListed {
		t.Errorf("expected clean IP not to be listed")
	}

	// 3. Check All Providers
	results, err := checker.CheckIPAllProviders(ctx, "ip_1", "192.0.2.1")
	if err != nil {
		t.Fatalf("CheckIPAllProviders failed: %v", err)
	}
	if len(results) < 4 {
		t.Errorf("expected at least 4 provider results, got %d", len(results))
	}
}
