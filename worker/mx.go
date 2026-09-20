package worker

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

type MXRecord struct {
	Host     string
	Pref     uint16
	Provider string
}

type MXResolver interface {
	Resolve(ctx context.Context, domain string) ([]*MXRecord, error)
}

type DNSMXResolver struct {
	mu    sync.RWMutex
	cache map[string]cachedMX
	ttl   time.Duration
}

type cachedMX struct {
	records   []*MXRecord
	expiresAt time.Time
}

func NewDNSMXResolver(ttl time.Duration) *DNSMXResolver {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	return &DNSMXResolver{
		cache: make(map[string]cachedMX),
		ttl:   ttl,
	}
}

func (r *DNSMXResolver) Resolve(ctx context.Context, domain string) ([]*MXRecord, error) {
	normDomain := strings.ToLower(strings.TrimSpace(domain))
	if normDomain == "" {
		return nil, fmt.Errorf("empty domain")
	}

	r.mu.RLock()
	entry, found := r.cache[normDomain]
	r.mu.RUnlock()

	if found && time.Now().Before(entry.expiresAt) {
		return entry.records, nil
	}

	// Resolve via net.LookupMX
	mxs, err := net.LookupMX(normDomain)
	if err != nil || len(mxs) == 0 {
		// RFC 5321 fallback: if no MX record exists, treat domain as an implicit MX with preference 0
		rec := &MXRecord{
			Host:     normDomain,
			Pref:     0,
			Provider: ClassifyProvider(normDomain),
		}
		records := []*MXRecord{rec}
		r.cacheResult(normDomain, records)
		return records, nil
	}

	// Check RFC 7505 NULL MX: a single MX of "." with preference 0 indicates the domain does not accept email
	if len(mxs) == 1 && (mxs[0].Host == "." || mxs[0].Host == "") && mxs[0].Pref == 0 {
		return nil, fmt.Errorf("domain %s publishes RFC 7505 Null MX record (rejects all email)", normDomain)
	}

	var records []*MXRecord
	for _, mx := range mxs {
		h := strings.TrimSuffix(mx.Host, ".")
		records = append(records, &MXRecord{
			Host:     h,
			Pref:     mx.Pref,
			Provider: ClassifyProvider(h),
		})
	}

	r.cacheResult(normDomain, records)
	return records, nil
}

func (r *DNSMXResolver) cacheResult(domain string, records []*MXRecord) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cache[domain] = cachedMX{
		records:   records,
		expiresAt: time.Now().Add(r.ttl),
	}
}

// ClassifyProvider maps MX hostname to provider category (§21.1)
func ClassifyProvider(mxHost string) string {
	h := strings.ToLower(mxHost)
	if strings.HasSuffix(h, ".google.com") || strings.HasSuffix(h, ".googlemail.com") || h == "aspmx.l.google.com" {
		return "google"
	}
	if strings.HasSuffix(h, ".mail.protection.outlook.com") || strings.HasSuffix(h, ".outlook.com") || strings.HasSuffix(h, ".office365.com") {
		return "microsoft"
	}
	if strings.HasSuffix(h, ".yahoodns.net") || strings.HasSuffix(h, ".yahoo.com") {
		return "yahoo"
	}
	if strings.HasSuffix(h, ".icloud.com") || strings.HasSuffix(h, ".apple.com") {
		return "apple"
	}
	if strings.HasSuffix(h, ".mail.ru") {
		return "mailru"
	}
	if strings.HasSuffix(h, ".qq.com") {
		return "qq"
	}
	return "custom"
}
