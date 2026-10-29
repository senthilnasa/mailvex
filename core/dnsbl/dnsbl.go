package dnsbl

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/senthilnasa/mailvex/core/crypto"
	"github.com/senthilnasa/mailvex/core/domain"
	"github.com/senthilnasa/mailvex/core/repository"
)

// DNSResolver abstracts host lookups for DNSBL checks
type DNSResolver interface {
	LookupHost(ctx context.Context, host string) ([]string, error)
}

type DefaultDNSResolver struct {
	resolver *net.Resolver
}

func NewDefaultDNSResolver() *DefaultDNSResolver {
	return &DefaultDNSResolver{
		resolver: net.DefaultResolver,
	}
}

func (r *DefaultDNSResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	return r.resolver.LookupHost(ctx, host)
}

// Checker performs DNSBL queries across configured providers
type Checker struct {
	repo     repository.Repository
	resolver DNSResolver
}

func NewChecker(repo repository.Repository, resolver DNSResolver) *Checker {
	if resolver == nil {
		resolver = NewDefaultDNSResolver()
	}
	return &Checker{
		repo:     repo,
		resolver: resolver,
	}
}

// ReverseIP converts an IPv4 or IPv6 address into its reverse DNSBL query representation
func ReverseIP(ipStr string) (string, error) {
	ip := net.ParseIP(strings.TrimSpace(ipStr))
	if ip == nil {
		return "", fmt.Errorf("invalid IP address: %s", ipStr)
	}

	// IPv4
	if ipv4 := ip.To4(); ipv4 != nil {
		return fmt.Sprintf("%d.%d.%d.%d", ipv4[3], ipv4[2], ipv4[1], ipv4[0]), nil
	}

	// IPv6: reverse 32 nibbles
	var nibbles []string
	for i := len(ip) - 1; i >= 0; i-- {
		b := ip[i]
		nibbles = append(nibbles, fmt.Sprintf("%x", b&0x0F))
		nibbles = append(nibbles, fmt.Sprintf("%x", (b>>4)&0x0F))
	}
	return strings.Join(nibbles, "."), nil
}

// CheckIP checks a single IP against a specific DNSBL provider
func (c *Checker) CheckIP(ctx context.Context, ipAddressID, ipStr string, provider *domain.DNSBLProvider) (*domain.DNSBLResult, error) {
	revIP, err := ReverseIP(ipStr)
	if err != nil {
		return nil, err
	}

	queryHost := fmt.Sprintf("%s.%s", revIP, provider.QueryPattern)

	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	addrs, err := c.resolver.LookupHost(ctxTimeout, queryHost)
	result := &domain.DNSBLResult{
		ID:           crypto.NewULID(crypto.PrefixDNSBLResult),
		IPAddressID:  ipAddressID,
		IPAddress:    ipStr,
		ProviderID:   provider.ID,
		ProviderName: provider.Name,
		DelistURL:    provider.DelistURL,
		CheckedAt:    time.Now().UTC(),
	}

	if err != nil {
		// Not found or lookup error
		result.IsListed = false
		if err := c.repo.RecordDNSBLResult(ctx, result); err != nil {
			return nil, err
		}
		return result, nil
	}

	if len(addrs) > 0 {
		result.IsListed = true
		respCode := addrs[0]
		result.ResponseCode = &respCode

		// Parse return code meaning from JSON map if available
		if provider.ReturnCodeMeanings != "" {
			var meanings map[string]string
			if err := json.Unmarshal([]byte(provider.ReturnCodeMeanings), &meanings); err == nil {
				if meaning, found := meanings[respCode]; found {
					result.ReturnMeaning = &meaning
				}
			}
		}

		if result.ReturnMeaning == nil {
			m := fmt.Sprintf("Listed (response: %s)", respCode)
			result.ReturnMeaning = &m
		}
	}

	if err := c.repo.RecordDNSBLResult(ctx, result); err != nil {
		return nil, err
	}

	return result, nil
}

// CheckIPAllProviders runs checks across all enabled DNSBL providers
func (c *Checker) CheckIPAllProviders(ctx context.Context, ipAddressID, ipStr string) ([]*domain.DNSBLResult, error) {
	providers, err := c.repo.ListDNSBLProviders(ctx)
	if err != nil {
		return nil, err
	}

	var results []*domain.DNSBLResult
	for _, p := range providers {
		if !p.IsEnabled {
			continue
		}
		res, err := c.CheckIP(ctx, ipAddressID, ipStr, p)
		if err != nil {
			continue
		}
		results = append(results, res)
	}

	return results, nil
}
