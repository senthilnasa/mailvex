package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/senthilnasa/mailvex/core/audit"
	"github.com/senthilnasa/mailvex/core/crypto"
	"github.com/senthilnasa/mailvex/core/domain"
	"github.com/senthilnasa/mailvex/core/repository"
)

func setupTestServerForSSO(t *testing.T) (*Server, repository.Repository, string, string, string) {
	repo := repository.NewMemoryRepository()
	auditSvc := audit.NewService(repo)
	kek := bytes.Repeat([]byte{0x07}, 32)
	enc, _ := crypto.NewEnvelopeEncryption(kek)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	srv := NewServer(repo, auditSvc, enc, logger)

	ctx := context.Background()
	orgID := "org_default"
	tenantID := "tenant_default"
	userID := "user_admin"

	_ = repo.CreateOrganization(ctx, &domain.Organization{ID: orgID, Name: "Default Org", Slug: "default-org", Status: "active"})
	_ = repo.CreateTenant(ctx, &domain.Tenant{ID: tenantID, OrganizationID: orgID, Name: "Default Tenant", Slug: "default-tenant", Environment: "live", Status: "active"})
	_ = repo.CreateUser(ctx, &domain.User{ID: userID, OrganizationID: orgID, Email: "admin@mailvex.io", Status: "active", IsPlatformAdmin: true})

	return srv, repo, orgID, tenantID, userID
}

func TestSSOEndpoints(t *testing.T) {
	srv, _, orgID, tenantID, userID := setupTestServerForSSO(t)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// 1. Create SSO Provider (OIDC)
	createOIDCBody := map[string]interface{}{
		"org_id":          orgID,
		"name":            "Entra ID OIDC",
		"provider_type":   "oidc",
		"issuer_url":      "https://login.microsoftonline.com/common/v2.0",
		"client_id":       "entra-client-id",
		"sp_acs_url":      "https://mailvex.local/v1/auth/sso/oidc/callback",
		"allowed_domains": []string{"acme.com"},
		"default_role":    "developer",
		"enforce_sso":     false,
	}
	bodyBytes, _ := json.Marshal(createOIDCBody)
	req, _ := http.NewRequest("POST", ts.URL+"/v1/auth/sso/providers", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-Tenant-ID", tenantID)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Create SSO Provider failed: %v", err)
	}
	if res.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(res.Body)
		t.Fatalf("Expected 201 Created, got %d: %s", res.StatusCode, string(respBody))
	}

	var createdRes struct {
		Data domain.SSOProvider `json:"data"`
	}
	_ = json.NewDecoder(res.Body).Decode(&createdRes)
	providerID := createdRes.Data.ID

	// 2. List SSO Providers
	req, _ = http.NewRequest("GET", ts.URL+"/v1/auth/sso/providers?org_id="+orgID, nil)
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-Tenant-ID", tenantID)
	res, err = http.DefaultClient.Do(req)
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("List SSO Providers failed: %v", err)
	}

	// 3. Create SAML Provider
	createSAMLBody := map[string]interface{}{
		"org_id":          orgID,
		"name":            "Okta SAML 2.0",
		"provider_type":   "saml",
		"sp_entity_id":    "https://mailvex.local/saml/sp",
		"sp_acs_url":      "https://mailvex.local/v1/auth/sso/saml/acs",
		"allowed_domains": []string{"acme.com"},
		"default_role":    "viewer",
	}
	bodyBytes, _ = json.Marshal(createSAMLBody)
	req, _ = http.NewRequest("POST", ts.URL+"/v1/auth/sso/providers", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-Tenant-ID", tenantID)
	res, err = http.DefaultClient.Do(req)
	if err != nil || res.StatusCode != http.StatusCreated {
		t.Fatalf("Create SAML Provider failed: %v", err)
	}

	var samlRes struct {
		Data domain.SSOProvider `json:"data"`
	}
	_ = json.NewDecoder(res.Body).Decode(&samlRes)
	samlID := samlRes.Data.ID

	// 4. Download SAML SP Metadata (Public endpoint)
	req, _ = http.NewRequest("GET", ts.URL+"/v1/auth/sso/saml/metadata?provider_id="+samlID, nil)
	res, err = http.DefaultClient.Do(req)
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("Get SAML Metadata failed: %v", err)
	}
	metaBytes, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(metaBytes), "EntityDescriptor") {
		t.Errorf("Expected EntityDescriptor XML, got: %s", string(metaBytes))
	}

	// 5. OIDC Callback Endpoint
	oidcCallbackBody := map[string]string{
		"provider_id": providerID,
		"code":        "sso:sam@acme.com:Sam Winchester",
		"state":       "state-123",
	}
	bodyBytes, _ = json.Marshal(oidcCallbackBody)
	req, _ = http.NewRequest("POST", ts.URL+"/v1/auth/sso/oidc/callback", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	res, err = http.DefaultClient.Do(req)
	if err != nil || res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("OIDC Callback failed: %d %s", res.StatusCode, string(b))
	}

	// 6. SAML ACS Endpoint
	rawSAML := `<?xml version="1.0"?><Response xmlns="urn:oasis:names:tc:SAML:2.0:protocol"><Assertion xmlns="urn:oasis:names:tc:SAML:2.0:assertion"><Subject><NameID>dean@acme.com</NameID></Subject><AttributeStatement><Attribute Name="Email"><AttributeValue>dean@acme.com</AttributeValue></Attribute></AttributeStatement></Assertion></Response>`
	samlACSBody := map[string]string{
		"provider_id":   samlID,
		"saml_response": base64.StdEncoding.EncodeToString([]byte(rawSAML)),
	}
	bodyBytes, _ = json.Marshal(samlACSBody)
	req, _ = http.NewRequest("POST", ts.URL+"/v1/auth/sso/saml/acs", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	res, err = http.DefaultClient.Do(req)
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("SAML ACS failed: %v", err)
	}
}

func TestPlatformAdminImpersonationEndpoints(t *testing.T) {
	srv, repo, orgID, _, userID := setupTestServerForSSO(t)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	ctx := context.Background()
	_ = repo.CreateTenant(ctx, &domain.Tenant{ID: "tenant_target", OrganizationID: orgID, Name: "Customer Tenant", Slug: "customer-tenant", Environment: "live", Status: "active"})

	// 1. Start Impersonation via API
	impReqBody := map[string]interface{}{
		"target_tenant_id":   "tenant_target",
		"reason":             "Customer reported slow webhook deliveries",
		"access_type":        "read_only",
		"is_support_masking": true,
	}
	bodyBytes, _ := json.Marshal(impReqBody)
	req, _ := http.NewRequest("POST", ts.URL+"/v1/admin/impersonate", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-Tenant-ID", "tenant_target")

	res, err := http.DefaultClient.Do(req)
	if err != nil || res.StatusCode != http.StatusCreated {
		t.Fatalf("Start Impersonation failed: %v", err)
	}

	var sessionRes struct {
		Session domain.ImpersonationSession `json:"session"`
	}
	_ = json.NewDecoder(res.Body).Decode(&sessionRes)
	sessionID := sessionRes.Session.ID

	// 2. Perform Impersonated Request using X-Impersonate-Tenant-ID header
	req, _ = http.NewRequest("GET", ts.URL+"/v1/domains", nil)
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-Impersonate-Tenant-ID", "tenant_target")
	req.Header.Set("X-Impersonate-Reason", "Auditing active domains")
	res, err = http.DefaultClient.Do(req)
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("Impersonated request failed: %v", err)
	}
	if res.Header.Get("X-Impersonating") != "true" {
		t.Errorf("Expected X-Impersonating header in response")
	}

	// 3. List Impersonation Sessions
	req, _ = http.NewRequest("GET", ts.URL+"/v1/admin/impersonate/sessions", nil)
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-Tenant-ID", "tenant_target")
	res, err = http.DefaultClient.Do(req)
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("List Impersonation Sessions failed: %v", err)
	}

	// 4. Exit Impersonation
	exitBody := map[string]string{
		"session_id": sessionID,
	}
	bodyBytes, _ = json.Marshal(exitBody)
	req, _ = http.NewRequest("POST", ts.URL+"/v1/admin/impersonate/exit", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-Tenant-ID", "tenant_target")
	res, err = http.DefaultClient.Do(req)
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("Exit Impersonation failed: %v", err)
	}
}

func TestComplianceAndHealthEndpoints(t *testing.T) {
	srv, repo, _, tenantID, userID := setupTestServerForSSO(t)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	ctx := context.Background()
	_ = repo.CreateDomain(ctx, &domain.Domain{
		ID:       "dom_test",
		TenantID: tenantID,
		Name:     "example.com",
		Status:   "verified",
	})

	// 1. Compliance Check
	req, _ := http.NewRequest("GET", ts.URL+"/v1/compliance/check?domain=example.com", nil)
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-Tenant-ID", tenantID)
	res, err := http.DefaultClient.Do(req)
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("Compliance check failed: %v", err)
	}

	var compRes struct {
		Data domain.ComplianceCheck `json:"data"`
	}
	_ = json.NewDecoder(res.Body).Decode(&compRes)
	if compRes.Data.DomainName != "example.com" {
		t.Errorf("Expected domain example.com, got %s", compRes.Data.DomainName)
	}

	// 2. FCrDNS Check
	fcrdnsBody := map[string]string{
		"ip_address": "127.0.0.1",
	}
	bodyBytes, _ := json.Marshal(fcrdnsBody)
	req, _ = http.NewRequest("POST", ts.URL+"/v1/health/fcrdns", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-Tenant-ID", tenantID)
	res, err = http.DefaultClient.Do(req)
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("FCrDNS check failed: %v", err)
	}
}
