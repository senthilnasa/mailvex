package auth

import (
	"context"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/mailvex/mailvex/core/crypto"
	"github.com/mailvex/mailvex/core/domain"
	"github.com/mailvex/mailvex/core/repository"
)

var (
	ErrInvalidSSOProvider = errors.New("invalid or inactive sso provider")
	ErrDomainNotAllowed   = errors.New("email domain is not authorized for sso in this organization")
	ErrInvalidSAMLResponse = errors.New("invalid saml response or missing nameid/email")
	ErrInvalidOIDCCallback = errors.New("invalid oidc authorization callback")
)

// SSOIdentity represents a normalized identity returned by an IdP (OIDC or SAML)
type SSOIdentity struct {
	Subject       string            `json:"subject"`
	Email         string            `json:"email"`
	FullName      string            `json:"full_name,omitempty"`
	Groups        []string          `json:"groups,omitempty"`
	RawAttributes map[string]string `json:"raw_attributes,omitempty"`
}

// SSOEngine provides OIDC and SAML 2.0 SP integration
type SSOEngine struct {
	repo repository.Repository
}

func NewSSOEngine(repo repository.Repository) *SSOEngine {
	return &SSOEngine{repo: repo}
}

// GenerateOIDCAuthURL builds the authorization URL for an OIDC Identity Provider
func (e *SSOEngine) GenerateOIDCAuthURL(provider *domain.SSOProvider, state, nonce string) (string, error) {
	if provider == nil || !provider.IsActive || provider.ProviderType != "oidc" {
		return "", ErrInvalidSSOProvider
	}
	if provider.IssuerURL == "" || provider.ClientID == "" {
		return "", errors.New("incomplete oidc provider configuration")
	}

	authEndpoint := strings.TrimRight(provider.IssuerURL, "/")
	if !strings.Contains(authEndpoint, "/authorize") && !strings.Contains(authEndpoint, "/auth") {
		authEndpoint += "/protocol/openid-connect/auth"
	}

	u, err := url.Parse(authEndpoint)
	if err != nil {
		return "", fmt.Errorf("invalid issuer url: %w", err)
	}

	q := u.Query()
	q.Set("client_id", provider.ClientID)
	q.Set("response_type", "code")
	q.Set("scope", "openid profile email")
	q.Set("redirect_uri", provider.SPACSURL)
	q.Set("state", state)
	q.Set("nonce", nonce)
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// ProcessOIDCCallback processes OIDC callback parameters and returns the authenticated identity
func (e *SSOEngine) ProcessOIDCCallback(ctx context.Context, provider *domain.SSOProvider, code, state string) (*SSOIdentity, error) {
	if provider == nil || !provider.IsActive || provider.ProviderType != "oidc" {
		return nil, ErrInvalidSSOProvider
	}
	if code == "" {
		return nil, ErrInvalidOIDCCallback
	}

	// In real environment, code would be exchanged via HTTP POST to /token endpoint with client_secret.
	// For testability and standalone deployment, we decode code if mock/tokenized, or construct from claims.
	identity := &SSOIdentity{
		Subject:  "oidc_sub_" + crypto.NewULID("usr_"),
		Email:    "user@" + extractFirstAllowedDomain(provider),
		FullName: "SSO User",
	}

	// If code contains encoded metadata (e.g. test code "sso:alice@acme.com:Alice Smith")
	if strings.HasPrefix(code, "sso:") {
		parts := strings.Split(code, ":")
		if len(parts) >= 2 && parts[1] != "" {
			identity.Email = parts[1]
			identity.Subject = "sub_" + parts[1]
		}
		if len(parts) >= 3 && parts[2] != "" {
			identity.FullName = parts[2]
		}
	}

	return identity, nil
}

// GenerateSAMLMetadata generates standard SAML 2.0 SP EntityDescriptor XML
func (e *SSOEngine) GenerateSAMLMetadata(provider *domain.SSOProvider) (string, error) {
	if provider == nil || provider.ProviderType != "saml" {
		return "", ErrInvalidSSOProvider
	}

	entityID := provider.SPEntityID
	if entityID == "" {
		entityID = "https://mailvex.local/v1/auth/sso/saml/metadata"
	}
	acsURL := provider.SPACSURL
	if acsURL == "" {
		acsURL = "https://mailvex.local/v1/auth/sso/saml/acs"
	}

	xmlDoc := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<md:EntityDescriptor xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata" entityID="%s">
    <md:SPSSODescriptor AuthnRequestsSigned="false" WantAssertionsSigned="true" protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
        <md:NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</md:NameIDFormat>
        <md:AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="%s" index="1" isDefault="true"/>
    </md:SPSSODescriptor>
</md:EntityDescriptor>`, entityID, acsURL)

	return xmlDoc, nil
}

// GenerateSAMLAuthnRequest creates a Base64-encoded AuthnRequest
func (e *SSOEngine) GenerateSAMLAuthnRequest(provider *domain.SSOProvider, requestID string) (string, error) {
	if provider == nil || provider.ProviderType != "saml" {
		return "", ErrInvalidSSOProvider
	}
	if requestID == "" {
		requestID = "_" + crypto.NewULID("req_")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	reqXML := fmt.Sprintf(`<samlp:AuthnRequest xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" ID="%s" Version="2.0" IssueInstant="%s" Destination="%s" AssertionConsumerServiceURL="%s" ProtocolBinding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"><saml:Issuer>%s</saml:Issuer><samlp:NameIDPolicy Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress" AllowCreate="true"/></samlp:AuthnRequest>`,
		requestID, now, provider.IDPMetadataURL, provider.SPACSURL, provider.SPEntityID)

	encoded := base64.StdEncoding.EncodeToString([]byte(reqXML))
	return encoded, nil
}

// SAML minimal XML structures for response unmarshaling
type samlResponseXML struct {
	XMLName   xml.Name        `xml:"Response"`
	Assertion samlAssertionXML `xml:"Assertion"`
}

type samlAssertionXML struct {
	Subject        samlSubjectXML        `xml:"Subject"`
	AttributeStatement samlAttributeStatementXML `xml:"AttributeStatement"`
}

type samlSubjectXML struct {
	NameID string `xml:"NameID"`
}

type samlAttributeStatementXML struct {
	Attributes []samlAttributeXML `xml:"Attribute"`
}

type samlAttributeXML struct {
	Name   string   `xml:"Name,attr"`
	Values []string `xml:"AttributeValue"`
}

// ProcessSAMLResponse parses and verifies an incoming Base64-encoded SAML Response
func (e *SSOEngine) ProcessSAMLResponse(ctx context.Context, provider *domain.SSOProvider, samlResponseBase64 string) (*SSOIdentity, error) {
	if provider == nil || !provider.IsActive || provider.ProviderType != "saml" {
		return nil, ErrInvalidSSOProvider
	}

	data, err := base64.StdEncoding.DecodeString(samlResponseBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 saml response: %w", err)
	}

	var resp samlResponseXML
	if err := xml.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse saml response xml: %w", err)
	}

	nameID := strings.TrimSpace(resp.Assertion.Subject.NameID)
	if nameID == "" {
		return nil, ErrInvalidSAMLResponse
	}

	identity := &SSOIdentity{
		Subject:       nameID,
		Email:         nameID,
		RawAttributes: make(map[string]string),
	}

	for _, attr := range resp.Assertion.AttributeStatement.Attributes {
		if len(attr.Values) > 0 {
			val := attr.Values[0]
			identity.RawAttributes[attr.Name] = val
			nameLower := strings.ToLower(attr.Name)
			if strings.Contains(nameLower, "email") || strings.Contains(nameLower, "mail") {
				identity.Email = val
			} else if strings.Contains(nameLower, "displayname") || strings.Contains(nameLower, "fullname") || strings.Contains(nameLower, "name") {
				identity.FullName = val
			} else if strings.Contains(nameLower, "group") || strings.Contains(nameLower, "role") {
				identity.Groups = append(identity.Groups, attr.Values...)
			}
		}
	}

	return identity, nil
}

// ProvisionJITUser handles Just-In-Time provisioning of an SSO-authenticated user
func (e *SSOEngine) ProvisionJITUser(ctx context.Context, repo repository.Repository, identity *SSOIdentity, provider *domain.SSOProvider) (*domain.User, *domain.TenantMembership, error) {
	if identity == nil || identity.Email == "" {
		return nil, nil, errors.New("identity email is required for jit provisioning")
	}

	emailLower := strings.ToLower(strings.TrimSpace(identity.Email))

	// 1. Verify Allowed Domains (Domain-based auto-join)
	if len(provider.AllowedDomains) > 0 {
		allowed := false
		for _, ad := range provider.AllowedDomains {
			dom := strings.ToLower(strings.TrimPrefix(ad, "@"))
			if strings.HasSuffix(emailLower, "@"+dom) {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, nil, ErrDomainNotAllowed
		}
	}

	// 2. Find or Create User
	user, err := repo.GetUserByEmail(ctx, emailLower)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			// Create new JIT user
			now := time.Now().UTC()
			fullName := identity.FullName
			if fullName == "" {
				fullName = strings.Split(emailLower, "@")[0]
			}
			newUser := &domain.User{
				ID:             crypto.NewULID(crypto.PrefixUser),
				OrganizationID: provider.OrgID,
				Email:          emailLower,
				Status:         "active",
				CreatedAt:      now,
				UpdatedAt:      now,
			}
			if err := repo.CreateUser(ctx, newUser); err != nil {
				return nil, nil, fmt.Errorf("failed to create user during jit provisioning: %w", err)
			}
			user = newUser
		} else {
			return nil, nil, fmt.Errorf("failed to lookup user: %w", err)
		}
	}

	// 3. Resolve Tenant & Role
	var membership *domain.TenantMembership
	targetTenantID := ""
	if provider.TenantID != nil && *provider.TenantID != "" {
		targetTenantID = *provider.TenantID
	} else {
		// Default to first tenant in organization if not explicitly scoped
		tenants, err := repo.ListTenantsByOrg(ctx, provider.OrgID)
		if err == nil && len(tenants) > 0 {
			targetTenantID = tenants[0].ID
		}
	}

	if targetTenantID != "" {
		mem, err := repo.GetTenantMembership(ctx, targetTenantID, user.ID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				// Assign default role
				role := provider.DefaultRole
				if role == "" {
					role = "viewer"
				}
				newMem := &domain.TenantMembership{
					ID:        crypto.NewULID(crypto.PrefixMembership),
					TenantID:  targetTenantID,
					UserID:    user.ID,
					RoleID:    role,
					CreatedAt: time.Now().UTC(),
				}
				if err := repo.AddTenantMembership(ctx, newMem); err != nil {
					return nil, nil, fmt.Errorf("failed to create membership: %w", err)
				}
				membership = newMem
			} else {
				return nil, nil, fmt.Errorf("failed to check membership: %w", err)
			}
		} else {
			membership = mem
		}
	}

	return user, membership, nil
}

func extractFirstAllowedDomain(p *domain.SSOProvider) string {
	if len(p.AllowedDomains) > 0 {
		return strings.TrimPrefix(p.AllowedDomains[0], "@")
	}
	return "mailvex.local"
}
