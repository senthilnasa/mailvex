package auth

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/mailvex/mailvex/core/domain"
	"github.com/mailvex/mailvex/core/repository"
)

func TestOIDCAuthURLAndCallback(t *testing.T) {
	repo := repository.NewMemoryRepository()
	engine := NewSSOEngine(repo)

	provider := &domain.SSOProvider{
		ID:           "sso_google",
		OrgID:        "org_default",
		Name:         "Google Workspace",
		ProviderType: "oidc",
		IssuerURL:    "https://accounts.google.com",
		ClientID:     "client-12345.apps.googleusercontent.com",
		SPACSURL:     "https://mailvex.local/v1/auth/sso/oidc/callback",
		AllowedDomains: []string{"acme.com", "@partner.org"},
		DefaultRole:  "developer",
		IsActive:     true,
	}

	// 1. Generate Auth URL
	authURL, err := engine.GenerateOIDCAuthURL(provider, "state-xyz", "nonce-123")
	if err != nil {
		t.Fatalf("GenerateOIDCAuthURL failed: %v", err)
	}
	if !strings.Contains(authURL, "client_id=client-12345") || !strings.Contains(authURL, "state=state-xyz") {
		t.Errorf("Unexpected authURL: %s", authURL)
	}

	// 2. Process Callback
	ctx := context.Background()
	identity, err := engine.ProcessOIDCCallback(ctx, provider, "sso:alice@acme.com:Alice Smith", "state-xyz")
	if err != nil {
		t.Fatalf("ProcessOIDCCallback failed: %v", err)
	}
	if identity.Email != "alice@acme.com" || identity.FullName != "Alice Smith" {
		t.Errorf("Unexpected identity: %+v", identity)
	}

	// 3. JIT Provisioning (allowed domain)
	user, mem, err := engine.ProvisionJITUser(ctx, repo, identity, provider)
	if err != nil {
		t.Fatalf("ProvisionJITUser failed: %v", err)
	}
	if user.Email != "alice@acme.com" {
		t.Errorf("User mismatch: %+v", user)
	}
	if mem != nil && mem.RoleID != "developer" {
		t.Errorf("Expected role developer, got %s", mem.RoleID)
	}

	// 4. JIT Provisioning with unauthorized domain
	unauthIdentity := &SSOIdentity{
		Email: "attacker@evil.com",
	}
	_, _, err = engine.ProvisionJITUser(ctx, repo, unauthIdentity, provider)
	if err != ErrDomainNotAllowed {
		t.Fatalf("Expected ErrDomainNotAllowed, got %v", err)
	}
}

func TestSAMLMetadataAndResponse(t *testing.T) {
	repo := repository.NewMemoryRepository()
	engine := NewSSOEngine(repo)

	provider := &domain.SSOProvider{
		ID:             "sso_okta",
		OrgID:          "org_default",
		Name:           "Okta SAML",
		ProviderType:   "saml",
		SPEntityID:     "https://mailvex.local/saml/sp",
		SPACSURL:       "https://mailvex.local/v1/auth/sso/saml/acs",
		IDPMetadataURL: "https://okta.com/app/sso",
		AllowedDomains: []string{"acme.com"},
		DefaultRole:    "viewer",
		IsActive:       true,
	}

	// 1. Generate SP Metadata
	metaXML, err := engine.GenerateSAMLMetadata(provider)
	if err != nil {
		t.Fatalf("GenerateSAMLMetadata failed: %v", err)
	}
	if !strings.Contains(metaXML, "https://mailvex.local/saml/sp") || !strings.Contains(metaXML, "AssertionConsumerService") {
		t.Errorf("Metadata missing entityID: %s", metaXML)
	}

	// 2. Generate AuthnRequest
	reqB64, err := engine.GenerateSAMLAuthnRequest(provider, "req_123")
	if err != nil {
		t.Fatalf("GenerateSAMLAuthnRequest failed: %v", err)
	}
	if reqB64 == "" {
		t.Fatal("Expected non-empty base64 AuthnRequest")
	}

	// 3. Process SAML Response
	rawResponseXML := `<?xml version="1.0"?>
<Response xmlns="urn:oasis:names:tc:SAML:2.0:protocol">
    <Assertion xmlns="urn:oasis:names:tc:SAML:2.0:assertion">
        <Subject>
            <NameID>bob@acme.com</NameID>
        </Subject>
        <AttributeStatement>
            <Attribute Name="http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress">
                <AttributeValue>bob@acme.com</AttributeValue>
            </Attribute>
            <Attribute Name="displayName">
                <AttributeValue>Bob Jones</AttributeValue>
            </Attribute>
            <Attribute Name="Role">
                <AttributeValue>Engineering</AttributeValue>
            </Attribute>
        </AttributeStatement>
    </Assertion>
</Response>`
	encodedResponse := base64.StdEncoding.EncodeToString([]byte(rawResponseXML))

	ctx := context.Background()
	identity, err := engine.ProcessSAMLResponse(ctx, provider, encodedResponse)
	if err != nil {
		t.Fatalf("ProcessSAMLResponse failed: %v", err)
	}
	if identity.Email != "bob@acme.com" || identity.FullName != "Bob Jones" {
		t.Errorf("Unexpected SAML identity: %+v", identity)
	}

	// 4. JIT User Provisioning
	user, _, err := engine.ProvisionJITUser(ctx, repo, identity, provider)
	if err != nil {
		t.Fatalf("ProvisionJITUser failed for SAML: %v", err)
	}
	if user.Email != "bob@acme.com" {
		t.Errorf("Expected bob@acme.com, got %s", user.Email)
	}
}

func TestImpersonationLifecycleAndMasking(t *testing.T) {
	repo := repository.NewMemoryRepository()
	ctx := context.Background()

	// Setup organization and tenant
	_ = repo.CreateOrganization(ctx, &domain.Organization{ID: "org_default", Name: "Test Org", Slug: "test-org", Status: "active"})
	_ = repo.CreateTenant(ctx, &domain.Tenant{ID: "tenant_target", OrganizationID: "org_default", Name: "Customer Corp", Slug: "customer-corp", Environment: "live", Status: "active"})
	_ = repo.CreateUser(ctx, &domain.User{ID: "user_admin", OrganizationID: "org_default", Email: "admin@mailvex.io", Status: "active"})

	mgr := NewImpersonationManager(repo)

	// 1. Reason is required
	_, err := mgr.StartImpersonation(ctx, "user_admin", "tenant_target", "", "read_only", "127.0.0.1", "curl/8.0", false)
	if err != ErrImpersonationReasonRequired {
		t.Fatalf("Expected ErrImpersonationReasonRequired, got %v", err)
	}

	// 2. Start valid impersonation
	session, err := mgr.StartImpersonation(ctx, "user_admin", "tenant_target", "Debugging delivery latency", "read_only", "127.0.0.1", "Mozilla/5.0", true)
	if err != nil {
		t.Fatalf("StartImpersonation failed: %v", err)
	}
	if session.ActorUserID != "user_admin" || session.TargetTenantID != "tenant_target" || session.EndedAt != nil {
		t.Errorf("Unexpected session state: %+v", session)
	}

	// 3. Verify audit log was recorded
	logs, err := repo.ListAuditLogs(ctx, &session.TargetTenantID, 10)
	if err != nil || len(logs) == 0 {
		t.Fatalf("Audit log not found for impersonation")
	}
	if logs[0].Action != "admin.impersonation.start" {
		t.Errorf("Expected audit action admin.impersonation.start, got %s", logs[0].Action)
	}

	// 4. Test Support-role body masking (§7.4)
	originalBody := "Confidential customer invoice content with private financial figures"

	// Non-support view
	normalView := MaskMessageBody(originalBody, false)
	if normalView != originalBody {
		t.Errorf("Normal view should not be masked")
	}

	// Support view (masked)
	supportView := MaskMessageBody(originalBody, true)
	if supportView != MaskedSupportNotice {
		t.Errorf("Support view should be masked with notice, got: %s", supportView)
	}

	// 5. Exit impersonation
	err = mgr.ExitImpersonation(ctx, session.ID, "user_admin")
	if err != nil {
		t.Fatalf("ExitImpersonation failed: %v", err)
	}

	active, err := repo.GetActiveImpersonationSession(ctx, "user_admin", "tenant_target")
	if err == nil && active != nil {
		t.Errorf("Session should not be active after exit")
	}
}
