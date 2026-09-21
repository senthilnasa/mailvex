package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mailvex/mailvex/core/crypto"
	"github.com/mailvex/mailvex/core/dns"
	"github.com/mailvex/mailvex/core/domain"
)

func TestDomainsAndDNSLifecycle(t *testing.T) {
	server, repo := setupTestServer(t)
	stubResolver := dns.NewStubResolver()
	server.SetDNSEngine(dns.NewEngine(stubResolver))

	// 1. Setup tenant and user
	orgID := crypto.NewULID(crypto.PrefixOrg)
	_ = repo.CreateOrganization(t.Context(), &domain.Organization{
		ID: orgID, Name: "Test Org", Slug: "test-org", Status: "active",
	})
	tenantID := crypto.NewULID(crypto.PrefixTenant)
	_ = repo.CreateTenant(t.Context(), &domain.Tenant{
		ID: tenantID, OrganizationID: orgID, Name: "Test Tenant", Slug: "test-tenant", Status: "active",
	})
	userID := crypto.NewULID(crypto.PrefixUser)
	_ = repo.CreateUser(t.Context(), &domain.User{
		ID: userID, OrganizationID: orgID, Email: "user@test.org", IsPlatformAdmin: true,
	})
	_ = repo.AddTenantMembership(t.Context(), &domain.TenantMembership{
		ID: crypto.NewULID(crypto.PrefixMembership), TenantID: tenantID, UserID: userID, RoleID: "role_tenant_owner",
	})

	// Wrap tenant key so DKIM generation can encrypt private keys
	dek, _ := crypto.NewEnvelopeEncryption([]byte("01234567890123456789012345678901"))
	d, _ := dek.GenerateDEK()
	encD, _ := dek.EncryptDEK(d)
	_ = repo.SaveTenantKey(t.Context(), &domain.TenantKey{
		ID: crypto.NewULID(crypto.PrefixAPIKey), TenantID: tenantID, KeyVersion: 1, EncryptedDEK: encD,
	})

	// 2. Add Domain via API: POST /v1/domains
	addDomainReq := map[string]string{
		"name": "acme-mail.com",
		"type": "public",
	}
	body, _ := json.Marshal(addDomainReq)
	req := httptest.NewRequest("POST", "/v1/domains", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-Tenant-ID", tenantID)
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 on add domain, got %d body=%s", w.Code, w.Body.String())
	}

	var addResp map[string]any
	_ = json.NewDecoder(w.Body).Decode(&addResp)
	domData := addResp["domain"].(map[string]any)
	domainID := domData["id"].(string)
	ownershipToken := domData["ownership_token"].(string)
	dkimData := addResp["dkim"].(map[string]any)
	dkimPubKey := dkimData["public_key"].(string)
	dnsRecords := addResp["dns_records"].([]any)

	if len(dnsRecords) < 5 {
		t.Fatalf("expected at least 5 DNS records generated, got %d", len(dnsRecords))
	}

	// 3. Populate Stub DNS Resolver
	stubResolver.TXTRecords["_mailvex-verify.acme-mail.com"] = []string{
		fmt.Sprintf("mailvex-verify=%s", ownershipToken),
	}
	stubResolver.TXTRecords["mvx._domainkey.acme-mail.com"] = []string{
		fmt.Sprintf("v=DKIM1; k=rsa; p=%s", dkimPubKey),
	}
	stubResolver.TXTRecords["acme-mail.com"] = []string{
		"v=spf1 include:mail.example.com -all",
	}
	stubResolver.TXTRecords["_dmarc.acme-mail.com"] = []string{
		"v=DMARC1; p=none; rua=mailto:dmarc-reports@mail.example.com",
	}

	// 4. Verify Domain via API: POST /v1/domains/{id}/verify
	req = httptest.NewRequest("POST", fmt.Sprintf("/v1/domains/%s/verify", domainID), nil)
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-Tenant-ID", tenantID)
	w = httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on domain verify, got %d body=%s", w.Code, w.Body.String())
	}

	var verifyResp map[string]any
	_ = json.NewDecoder(w.Body).Decode(&verifyResp)
	updatedDom := verifyResp["domain"].(map[string]any)
	if updatedDom["status"] != "verified" {
		t.Fatalf("expected domain status to be 'verified', got '%s'", updatedDom["status"])
	}

	// 5. Test Global Uniqueness Conflict (§10.2): Tenant 2 adding the same domain must fail
	tenant2ID := crypto.NewULID(crypto.PrefixTenant)
	_ = repo.CreateTenant(t.Context(), &domain.Tenant{
		ID: tenant2ID, OrganizationID: orgID, Name: "Tenant 2", Slug: "tenant-2", Status: "active",
	})
	_ = repo.AddTenantMembership(t.Context(), &domain.TenantMembership{
		ID: crypto.NewULID(crypto.PrefixMembership), TenantID: tenant2ID, UserID: userID, RoleID: "role_tenant_owner",
	})

	conflictReq := httptest.NewRequest("POST", "/v1/domains", bytes.NewReader(body))
	conflictReq.Header.Set("Content-Type", "application/json")
	conflictReq.Header.Set("X-User-ID", userID)
	conflictReq.Header.Set("X-Tenant-ID", tenant2ID)
	conflictRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(conflictRec, conflictReq)

	if conflictRec.Code != http.StatusConflict {
		t.Fatalf("expected 409 conflict when second tenant claims verified domain, got %d body=%s", conflictRec.Code, conflictRec.Body.String())
	}

	// 6. Test Degraded Flow (§10.1): Simulate DNS failure and verify degraded status
	stubResolver.TXTRecords["_mailvex-verify.acme-mail.com"] = []string{"mailvex-verify=invalid"}
	req = httptest.NewRequest("POST", fmt.Sprintf("/v1/domains/%s/verify", domainID), nil)
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-Tenant-ID", tenantID)
	w = httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	_ = json.NewDecoder(w.Body).Decode(&verifyResp)
	degradedDom := verifyResp["domain"].(map[string]any)
	if degradedDom["status"] != "degraded" {
		t.Fatalf("expected domain to transition to 'degraded', got '%s'", degradedDom["status"])
	}

	// 7. Test DKIM Rotation Flow (§10.5): POST /v1/domains/{id}/dkim/rotate
	req = httptest.NewRequest("POST", fmt.Sprintf("/v1/domains/%s/dkim/rotate", domainID), nil)
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-Tenant-ID", tenantID)
	w = httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on dkim rotate, got %d body=%s", w.Code, w.Body.String())
	}

	// 8. Test Local Domain (§28.1): Skips public DNS verification, immediately verified
	localReq := map[string]string{
		"name": "home.lan",
		"type": "local",
	}
	localBody, _ := json.Marshal(localReq)
	req = httptest.NewRequest("POST", "/v1/domains", bytes.NewReader(localBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-Tenant-ID", tenantID)
	w = httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 on local domain add, got %d", w.Code)
	}
	var localResp map[string]any
	_ = json.NewDecoder(w.Body).Decode(&localResp)
	localDom := localResp["domain"].(map[string]any)
	if localDom["status"] != "verified" {
		t.Fatalf("expected local domain to be immediately 'verified', got '%s'", localDom["status"])
	}

	// 9. Test Message Streams: Default streams and create custom stream
	_ = repo.CreateStream(t.Context(), &domain.MessageStream{
		ID: crypto.NewULID(crypto.PrefixStream), TenantID: tenantID, Name: "transactional", Type: "transactional", IsDefault: true, CreatedAt: time.Now(),
	})
	_ = repo.CreateStream(t.Context(), &domain.MessageStream{
		ID: crypto.NewULID(crypto.PrefixStream), TenantID: tenantID, Name: "broadcast", Type: "broadcast", IsDefault: false, CreatedAt: time.Now(),
	})

	req = httptest.NewRequest("GET", "/v1/streams", nil)
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-Tenant-ID", tenantID)
	w = httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on GET /v1/streams, got %d", w.Code)
	}
	var streamResp map[string]any
	_ = json.NewDecoder(w.Body).Decode(&streamResp)
	streams := streamResp["data"].([]any)
	if len(streams) < 2 {
		t.Fatalf("expected at least 2 streams, got %d", len(streams))
	}
}

func TestDomainVerification_NilDNSEngineSafe(t *testing.T) {
	server, repo := setupTestServer(t)
	// Explicitly simulate nil resolver
	server.SetDNSEngine(dns.NewEngine(nil))

	orgID := crypto.NewULID(crypto.PrefixOrg)
	_ = repo.CreateOrganization(t.Context(), &domain.Organization{
		ID: orgID, Name: "DNS Safe Org", Slug: "dns-safe-org", Status: "active",
	})
	tenantID := crypto.NewULID(crypto.PrefixTenant)
	_ = repo.CreateTenant(t.Context(), &domain.Tenant{
		ID: tenantID, OrganizationID: orgID, Name: "Safe Tenant", Slug: "safe-tenant", Status: "active",
	})
	userID := crypto.NewULID(crypto.PrefixUser)
	_ = repo.CreateUser(t.Context(), &domain.User{
		ID: userID, OrganizationID: orgID, Email: "admin@dns-safe.org", IsPlatformAdmin: true,
	})
	_ = repo.AddTenantMembership(t.Context(), &domain.TenantMembership{
		ID: crypto.NewULID(crypto.PrefixMembership), TenantID: tenantID, UserID: userID, RoleID: "role_tenant_owner",
	})

	// Add domain
	domID := crypto.NewULID(crypto.PrefixDomain)
	_ = repo.CreateDomain(t.Context(), &domain.Domain{
		ID:             domID,
		TenantID:       tenantID,
		Name:           "my-homelab.local",
		Type:           "local",
		Status:         "pending",
		OwnershipToken: "token123",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	})

	// Verify domain: POST /v1/domains/{id}/verify
	req := httptest.NewRequest("POST", "/v1/domains/"+domID+"/verify", nil)
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-Tenant-ID", tenantID)
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on verify with nil-safe engine, got %d (body: %s)", w.Code, w.Body.String())
	}
}
