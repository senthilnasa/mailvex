package dns

import (
	"context"
	"strings"
	"testing"

	"github.com/senthilnasa/mailvex/core/domain"
)

func TestSPFHostFallsBackToPlatformHost(t *testing.T) {
	t.Setenv("MVX_SPF_HOST", "")
	if got := SPFHost("mail.platform.test"); got != "mail.platform.test" {
		t.Fatalf("expected platform host fallback, got %q", got)
	}

	t.Setenv("MVX_SPF_HOST", " _spf.mail.platform.test. ")
	if got := SPFHost("mail.platform.test"); got != "_spf.mail.platform.test" {
		t.Fatalf("expected MVX_SPF_HOST to win, got %q", got)
	}
	if got := ExpectedSPF(SPFHost("mail.platform.test")); got != "v=spf1 include:_spf.mail.platform.test -all" {
		t.Fatalf("unexpected SPF record %q", got)
	}
}

func TestSPFIncludes(t *testing.T) {
	cases := []struct {
		record string
		want   bool
	}{
		{"v=spf1 include:_spf.example.com -all", true},
		{"v=spf1 +include:_SPF.Example.com. ~all", true},
		{"v=spf1 include:_spf.google.com include:_spf.example.com ~all", true},
		{"v=spf1 include:example.com -all", false},
		{"v=spf1 ip4:203.0.113.5 -all", false},
	}
	for _, c := range cases {
		if got := spfIncludes(c.record, "_spf.example.com"); got != c.want {
			t.Errorf("spfIncludes(%q) = %v, want %v", c.record, got, c.want)
		}
	}
}

func TestVerifyDomainWarnsWhenSPFMissesInclude(t *testing.T) {
	t.Setenv("MVX_SPF_HOST", "_spf.mail.platform.test")
	stub := NewStubResolver()
	engine := NewEngine(stub)
	dom := &domain.Domain{ID: "dom_1", Name: "example.com", Type: "public", OwnershipToken: "tok"}

	spfCheck := func() CheckResult {
		results, _ := engine.VerifyDomain(context.Background(), dom, nil, "mail.platform.test")
		for _, r := range results {
			if strings.HasPrefix(r.ExpectedValue, "v=spf1") {
				return r
			}
		}
		t.Fatal("no SPF check in results")
		return CheckResult{}
	}

	// Pointing at the platform host (a CNAME with no SPF) is what causes PERMERROR.
	stub.TXTRecords["example.com"] = []string{"v=spf1 include:mail.platform.test -all"}
	if r := spfCheck(); r.Status != "warning" || r.ExpectedValue != "v=spf1 include:_spf.mail.platform.test -all" {
		t.Fatalf("expected warning against _spf host, got %+v", r)
	}

	stub.TXTRecords["example.com"] = []string{"v=spf1 include:_spf.mail.platform.test -all"}
	if r := spfCheck(); r.Status != "pass" {
		t.Fatalf("expected pass, got %+v", r)
	}
}
