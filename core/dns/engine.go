package dns

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/mailvex/mailvex/core/domain"
)

// Resolver abstracts DNS lookups across multi-resolvers or test stubs
type Resolver interface {
	LookupTXT(ctx context.Context, name string) ([]string, error)
	LookupCNAME(ctx context.Context, name string) (string, error)
	LookupMX(ctx context.Context, name string) ([]*net.MX, error)
}

// SystemResolver uses Go standard net.Resolver
type SystemResolver struct {
	resolver *net.Resolver
}

func NewSystemResolver() *SystemResolver {
	return &SystemResolver{
		resolver: net.DefaultResolver,
	}
}

func (s *SystemResolver) LookupTXT(ctx context.Context, name string) ([]string, error) {
	return s.resolver.LookupTXT(ctx, name)
}

func (s *SystemResolver) LookupCNAME(ctx context.Context, name string) (string, error) {
	return s.resolver.LookupCNAME(ctx, name)
}

func (s *SystemResolver) LookupMX(ctx context.Context, name string) ([]*net.MX, error) {
	return s.resolver.LookupMX(ctx, name)
}

// StubResolver enables deterministic DNS testing without external network dependencies
type StubResolver struct {
	TXTRecords   map[string][]string
	CNAMERecords map[string]string
	MXRecords    map[string][]*net.MX
}

func NewStubResolver() *StubResolver {
	return &StubResolver{
		TXTRecords:   make(map[string][]string),
		CNAMERecords: make(map[string]string),
		MXRecords:    make(map[string][]*net.MX),
	}
}

func (s *StubResolver) LookupTXT(ctx context.Context, name string) ([]string, error) {
	normalized := strings.TrimSuffix(strings.ToLower(name), ".")
	if val, ok := s.TXTRecords[normalized]; ok {
		return val, nil
	}
	return nil, &net.DNSError{Err: "no such host", Name: name}
}

func (s *StubResolver) LookupCNAME(ctx context.Context, name string) (string, error) {
	normalized := strings.TrimSuffix(strings.ToLower(name), ".")
	if val, ok := s.CNAMERecords[normalized]; ok {
		return val, nil
	}
	return "", &net.DNSError{Err: "no such host", Name: name}
}

func (s *StubResolver) LookupMX(ctx context.Context, name string) ([]*net.MX, error) {
	normalized := strings.TrimSuffix(strings.ToLower(name), ".")
	if val, ok := s.MXRecords[normalized]; ok {
		return val, nil
	}
	return nil, &net.DNSError{Err: "no such host", Name: name}
}

// Engine coordinates DNS checks and updates domain records
type Engine struct {
	resolver Resolver
}

func NewEngine(res Resolver) *Engine {
	if res == nil {
		res = NewSystemResolver()
	}
	return &Engine{resolver: res}
}

type CheckResult struct {
	RecordType    string
	Name          string
	ExpectedValue string
	ObservedValue string
	Status        string // pass, warning, fail
	ErrorMessage  string
}

// VerifyDomain verifies ownership, SPF, DKIM, DMARC, and bounce records for a domain
func (e *Engine) VerifyDomain(ctx context.Context, dom *domain.Domain, dkim *domain.DKIMKey, platformHost string) ([]CheckResult, bool) {
	if e == nil {
		e = NewEngine(NewSystemResolver())
	}
	if e.resolver == nil {
		e.resolver = NewSystemResolver()
	}
	if dom == nil {
		return nil, false
	}

	// For local domains (homelab, test, internal LAN), bypass public DNS requirements
	lowerName := strings.ToLower(dom.Name)
	if dom.Type == "local" || strings.HasSuffix(lowerName, ".local") || strings.HasSuffix(lowerName, ".lan") || strings.HasSuffix(lowerName, ".internal") || strings.HasSuffix(lowerName, ".test") {
		results := []CheckResult{
			{
				RecordType:    "TXT",
				Name:          fmt.Sprintf("_mailvex-verify.%s", dom.Name),
				ExpectedValue: fmt.Sprintf("mailvex-verify=%s", dom.OwnershipToken),
				ObservedValue: fmt.Sprintf("mailvex-verify=%s", dom.OwnershipToken),
				Status:        "pass",
			},
		}
		if dkim != nil {
			results = append(results, CheckResult{
				RecordType:    "TXT",
				Name:          fmt.Sprintf("%s._domainkey.%s", dkim.Selector, dom.Name),
				ExpectedValue: fmt.Sprintf("v=DKIM1; k=rsa; p=%s", dkim.PublicKey),
				ObservedValue: fmt.Sprintf("v=DKIM1; k=rsa; p=%s", dkim.PublicKey),
				Status:        "pass",
			})
		}
		results = append(results, CheckResult{
			RecordType:    "SPF",
			Name:          dom.Name,
			ExpectedValue: fmt.Sprintf("v=spf1 include:%s -all", platformHost),
			ObservedValue: "local-domain-bypass",
			Status:        "pass",
		})
		results = append(results, CheckResult{
			RecordType:    "DMARC",
			Name:          fmt.Sprintf("_dmarc.%s", dom.Name),
			ExpectedValue: fmt.Sprintf("v=DMARC1; p=none; rua=mailto:dmarc-reports@%s", platformHost),
			ObservedValue: "local-domain-bypass",
			Status:        "pass",
		})
		return results, true
	}

	var results []CheckResult
	allPassed := true

	// 1. Ownership TXT verification
	verifyHost := fmt.Sprintf("_mailvex-verify.%s", dom.Name)
	expectedOwnership := fmt.Sprintf("mailvex-verify=%s", dom.OwnershipToken)
	ownershipResult := CheckResult{
		RecordType:    "TXT",
		Name:          verifyHost,
		ExpectedValue: expectedOwnership,
	}

	txts, err := e.resolver.LookupTXT(ctx, verifyHost)
	if err != nil {
		ownershipResult.Status = "fail"
		ownershipResult.ErrorMessage = fmt.Sprintf("No TXT record found at %s", verifyHost)
		allPassed = false
	} else {
		found := false
		var observed []string
		for _, t := range txts {
			observed = append(observed, t)
			if strings.TrimSpace(t) == expectedOwnership {
				found = true
				break
			}
		}
		ownershipResult.ObservedValue = strings.Join(observed, ", ")
		if found {
			ownershipResult.Status = "pass"
		} else {
			ownershipResult.Status = "fail"
			ownershipResult.ErrorMessage = fmt.Sprintf("Expected '%s', got '%s'", expectedOwnership, ownershipResult.ObservedValue)
			allPassed = false
		}
	}
	results = append(results, ownershipResult)

	// 2. DKIM verification
	if dkim != nil {
		dkimHost := fmt.Sprintf("%s._domainkey.%s", dkim.Selector, dom.Name)
		expectedDKIM := fmt.Sprintf("v=DKIM1; k=rsa; p=%s", dkim.PublicKey)
		dkimResult := CheckResult{
			RecordType:    "TXT",
			Name:          dkimHost,
			ExpectedValue: expectedDKIM,
		}

		dtxts, err := e.resolver.LookupTXT(ctx, dkimHost)
		if err != nil {
			dkimResult.Status = "fail"
			dkimResult.ErrorMessage = fmt.Sprintf("No DKIM record found at %s", dkimHost)
			allPassed = false
		} else {
			found := false
			var observed []string
			for _, t := range dtxts {
				observed = append(observed, t)
				if strings.Contains(t, dkim.PublicKey) {
					found = true
					break
				}
			}
			dkimResult.ObservedValue = strings.Join(observed, ", ")
			if found {
				dkimResult.Status = "pass"
			} else {
				dkimResult.Status = "fail"
				dkimResult.ErrorMessage = "DKIM public key does not match"
				allPassed = false
			}
		}
		results = append(results, dkimResult)
	}

	// 3. SPF check (Optional / Health warning)
	spfHost := dom.Name
	expectedSPF := fmt.Sprintf("v=spf1 include:%s -all", platformHost)
	spfResult := CheckResult{
		RecordType:    "TXT",
		Name:          spfHost,
		ExpectedValue: expectedSPF,
	}
	stxts, err := e.resolver.LookupTXT(ctx, spfHost)
	if err != nil {
		spfResult.Status = "warning"
		spfResult.ErrorMessage = "No SPF record found (recommended for deliverability)"
	} else {
		hasSPF := false
		for _, t := range stxts {
			if strings.HasPrefix(t, "v=spf1") {
				hasSPF = true
				spfResult.ObservedValue = t
				break
			}
		}
		if hasSPF {
			spfResult.Status = "pass"
		} else {
			spfResult.Status = "warning"
			spfResult.ErrorMessage = "No SPF record starting with v=spf1 found"
		}
	}
	results = append(results, spfResult)

	// 4. DMARC check
	dmarcHost := fmt.Sprintf("_dmarc.%s", dom.Name)
	expectedDMARC := fmt.Sprintf("v=DMARC1; p=none; rua=mailto:dmarc-reports@%s", platformHost)
	dmarcResult := CheckResult{
		RecordType:    "TXT",
		Name:          dmarcHost,
		ExpectedValue: expectedDMARC,
	}
	dtxts, err := e.resolver.LookupTXT(ctx, dmarcHost)
	if err != nil {
		dmarcResult.Status = "warning"
		dmarcResult.ErrorMessage = "No DMARC record found"
	} else {
		hasDMARC := false
		for _, t := range dtxts {
			if strings.HasPrefix(t, "v=DMARC1") {
				hasDMARC = true
				dmarcResult.ObservedValue = t
				break
			}
		}
		if hasDMARC {
			dmarcResult.Status = "pass"
		} else {
			dmarcResult.Status = "warning"
			dmarcResult.ErrorMessage = "No record starting with v=DMARC1 found"
		}
	}
	results = append(results, dmarcResult)

	return results, allPassed
}

// CheckDegradedStatus assesses if a previously verified domain should transition to degraded or suspended
func CheckDegradedStatus(dom *domain.Domain, ownershipPassed bool, gracePeriodHours int) (newStatus string, changed bool) {
	now := time.Now().UTC()

	// If ownership passes and domain was degraded, recover to verified
	if ownershipPassed {
		if dom.Status == "degraded" {
			dom.Status = "verified"
			dom.DegradedAt = nil
			return "verified", true
		}
		return dom.Status, false
	}

	// Ownership failed:
	if dom.Status == "verified" {
		dom.Status = "degraded"
		dom.DegradedAt = &now
		return "degraded", true
	} else if dom.Status == "degraded" && dom.DegradedAt != nil {
		cutoff := dom.DegradedAt.Add(time.Duration(gracePeriodHours) * time.Hour)
		if now.After(cutoff) {
			dom.Status = "suspended"
			return "suspended", true
		}
	}

	return dom.Status, false
}
