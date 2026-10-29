package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/senthilnasa/mailvex/api/middleware"
	"github.com/senthilnasa/mailvex/core/crypto"
	"github.com/senthilnasa/mailvex/core/domain"
)

// ListSSOProvidersHandler lists configured SSO providers for an organization
func (a *API) ListSSOProvidersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	orgID := r.URL.Query().Get("org_id")

	providers, err := a.repo.ListSSOProviders(r.Context(), orgID)
	if err != nil {
		http.Error(w, `{"error":{"code":"internal_error","message":"`+err.Error()+`"}}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data": providers,
	})
}

// CreateSSOProviderHandler registers a new OIDC or SAML 2.0 Identity Provider
func (a *API) CreateSSOProviderHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		OrgID          string   `json:"org_id"`
		TenantID       *string  `json:"tenant_id,omitempty"`
		Name           string   `json:"name"`
		ProviderType   string   `json:"provider_type"` // oidc, saml
		IssuerURL      string   `json:"issuer_url,omitempty"`
		ClientID       string   `json:"client_id,omitempty"`
		ClientSecret   string   `json:"client_secret,omitempty"`
		IDPMetadataURL string   `json:"idp_metadata_url,omitempty"`
		IDPMetadataXML string   `json:"idp_metadata_xml,omitempty"`
		SPEntityID     string   `json:"sp_entity_id,omitempty"`
		SPACSURL       string   `json:"sp_acs_url,omitempty"`
		AllowedDomains []string `json:"allowed_domains"`
		DefaultRole    string   `json:"default_role"`
		EnforceSSO     bool     `json:"enforce_sso"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"code":"invalid_json","message":"Malformed request body"}}`, http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.OrgID) == "" {
		http.Error(w, `{"error":{"code":"missing_fields","message":"Name and org_id are required"}}`, http.StatusBadRequest)
		return
	}

	req.ProviderType = strings.ToLower(strings.TrimSpace(req.ProviderType))
	if req.ProviderType != "oidc" && req.ProviderType != "saml" {
		http.Error(w, `{"error":{"code":"invalid_provider_type","message":"provider_type must be 'oidc' or 'saml'"}}`, http.StatusBadRequest)
		return
	}

	if req.DefaultRole == "" {
		req.DefaultRole = "viewer"
	}

	provider := &domain.SSOProvider{
		ID:             crypto.NewULID(crypto.PrefixSSOProvider),
		OrgID:          req.OrgID,
		TenantID:       req.TenantID,
		Name:           req.Name,
		ProviderType:   req.ProviderType,
		IssuerURL:      req.IssuerURL,
		ClientID:       req.ClientID,
		IDPMetadataURL: req.IDPMetadataURL,
		IDPMetadataXML: req.IDPMetadataXML,
		SPEntityID:     req.SPEntityID,
		SPACSURL:       req.SPACSURL,
		AllowedDomains: req.AllowedDomains,
		DefaultRole:    req.DefaultRole,
		EnforceSSO:     req.EnforceSSO,
		IsActive:       true,
	}

	if err := a.repo.CreateSSOProvider(r.Context(), provider); err != nil {
		http.Error(w, `{"error":{"code":"creation_failed","message":"`+err.Error()+`"}}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data": provider,
	})
}

// GetSSOProviderHandler fetches a single SSO provider configuration
func (a *API) GetSSOProviderHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id := r.PathValue("id")

	p, err := a.repo.GetSSOProvider(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":{"code":"not_found","message":"SSO provider not found"}}`, http.StatusNotFound)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data": p,
	})
}

// DeleteSSOProviderHandler removes an SSO provider
func (a *API) DeleteSSOProviderHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id := r.PathValue("id")

	if err := a.repo.DeleteSSOProvider(r.Context(), id); err != nil {
		http.Error(w, `{"error":{"code":"not_found","message":"SSO provider not found"}}`, http.StatusNotFound)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"id":      id,
	})
}

// GetSAMLMetadataHandler returns standard SP EntityDescriptor XML
func (a *API) GetSAMLMetadataHandler(w http.ResponseWriter, r *http.Request) {
	providerID := r.URL.Query().Get("provider_id")
	var provider *domain.SSOProvider
	if providerID != "" {
		provider, _ = a.repo.GetSSOProvider(r.Context(), providerID)
	}
	if provider == nil {
		provider = &domain.SSOProvider{
			ProviderType: "saml",
			SPEntityID:   "https://mailvex.local/v1/auth/sso/saml/metadata",
			SPACSURL:     "https://mailvex.local/v1/auth/sso/saml/acs",
		}
	}

	metaXML, err := a.ssoEngine.GenerateSAMLMetadata(provider)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/samlmetadata+xml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(metaXML))
}

// PostSAMLACSHandler processes incoming SAML responses from IdP
func (a *API) PostSAMLACSHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		ProviderID   string `json:"provider_id"`
		SAMLResponse string `json:"saml_response"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"code":"invalid_json","message":"Invalid request body"}}`, http.StatusBadRequest)
		return
	}

	provider, err := a.repo.GetSSOProvider(r.Context(), req.ProviderID)
	if err != nil || provider == nil {
		http.Error(w, `{"error":{"code":"provider_not_found","message":"SAML provider not found"}}`, http.StatusNotFound)
		return
	}

	identity, err := a.ssoEngine.ProcessSAMLResponse(r.Context(), provider, req.SAMLResponse)
	if err != nil {
		http.Error(w, `{"error":{"code":"saml_verification_failed","message":"`+err.Error()+`"}}`, http.StatusUnauthorized)
		return
	}

	user, mem, err := a.ssoEngine.ProvisionJITUser(r.Context(), a.repo, identity, provider)
	if err != nil {
		http.Error(w, `{"error":{"code":"jit_provision_failed","message":"`+err.Error()+`"}}`, http.StatusForbidden)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"user":       user,
		"membership": mem,
		"identity":   identity,
	})
}

// PostOIDCCallbackHandler processes OIDC code exchange & JIT provisioning
func (a *API) PostOIDCCallbackHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		ProviderID string `json:"provider_id"`
		Code       string `json:"code"`
		State      string `json:"state"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"code":"invalid_json","message":"Invalid request body"}}`, http.StatusBadRequest)
		return
	}

	provider, err := a.repo.GetSSOProvider(r.Context(), req.ProviderID)
	if err != nil || provider == nil {
		http.Error(w, `{"error":{"code":"provider_not_found","message":"OIDC provider not found"}}`, http.StatusNotFound)
		return
	}

	identity, err := a.ssoEngine.ProcessOIDCCallback(r.Context(), provider, req.Code, req.State)
	if err != nil {
		http.Error(w, `{"error":{"code":"oidc_exchange_failed","message":"`+err.Error()+`"}}`, http.StatusUnauthorized)
		return
	}

	user, mem, err := a.ssoEngine.ProvisionJITUser(r.Context(), a.repo, identity, provider)
	if err != nil {
		http.Error(w, `{"error":{"code":"jit_provision_failed","message":"`+err.Error()+`"}}`, http.StatusForbidden)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"user":       user,
		"membership": mem,
		"identity":   identity,
	})
}

// StartImpersonationHandler initiates a platform admin tenant impersonation session (§7.4)
func (a *API) StartImpersonationHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	user, ok := r.Context().Value(middleware.UserKey).(*domain.User)
	if !ok || user == nil || !user.IsPlatformAdmin {
		http.Error(w, `{"error":{"code":"forbidden","message":"Only platform administrators can start impersonation"}}`, http.StatusForbidden)
		return
	}

	var req struct {
		TargetTenantID   string `json:"target_tenant_id"`
		Reason           string `json:"reason"`
		AccessType       string `json:"access_type"` // read_only, read_write
		IsSupportMasking bool   `json:"is_support_masking"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"code":"invalid_json","message":"Invalid request body"}}`, http.StatusBadRequest)
		return
	}

	session, err := a.impersonationMgr.StartImpersonation(
		r.Context(),
		user.ID,
		req.TargetTenantID,
		req.Reason,
		req.AccessType,
		r.RemoteAddr,
		r.UserAgent(),
		req.IsSupportMasking,
	)
	if err != nil {
		http.Error(w, `{"error":{"code":"impersonation_failed","message":"`+err.Error()+`"}}`, http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"session": session,
	})
}

// ExitImpersonationHandler concludes an active impersonation session
func (a *API) ExitImpersonationHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	user, _ := r.Context().Value(middleware.UserKey).(*domain.User)
	userID := ""
	if user != nil {
		userID = user.ID
	}

	var req struct {
		SessionID string `json:"session_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.SessionID == "" {
		http.Error(w, `{"error":{"code":"session_required","message":"session_id is required"}}`, http.StatusBadRequest)
		return
	}

	if err := a.impersonationMgr.ExitImpersonation(r.Context(), req.SessionID, userID); err != nil {
		http.Error(w, `{"error":{"code":"exit_failed","message":"`+err.Error()+`"}}`, http.StatusNotFound)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"id":      req.SessionID,
	})
}

// ListImpersonationSessionsHandler returns audit log history of platform admin impersonations
func (a *API) ListImpersonationSessionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 {
		limit = 50
	}

	actorID := r.URL.Query().Get("actor_id")
	sessions, err := a.repo.ListImpersonationSessions(r.Context(), actorID, limit, offset)
	if err != nil {
		http.Error(w, `{"error":{"code":"internal_error","message":"`+err.Error()+`"}}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data": sessions,
	})
}

// GetComplianceCheckHandler runs or retrieves deliverability compliance check
func (a *API) GetComplianceCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)

	domainName := r.URL.Query().Get("domain")
	sendingIP := r.URL.Query().Get("ip")

	if domainName == "" {
		http.Error(w, `{"error":{"code":"domain_required","message":"domain query parameter is required"}}`, http.StatusBadRequest)
		return
	}

	check, err := a.complianceEval.EvaluateDomain(r.Context(), tenantID, domainName, sendingIP)
	if err != nil {
		http.Error(w, `{"error":{"code":"evaluation_failed","message":"`+err.Error()+`"}}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data": check,
	})
}

// PostFCrDNSCheckHandler checks forward-confirmed reverse DNS for an IP
func (a *API) PostFCrDNSCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		IPAddress string `json:"ip_address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.IPAddress == "" {
		http.Error(w, `{"error":{"code":"invalid_request","message":"ip_address is required"}}`, http.StatusBadRequest)
		return
	}

	result, err := a.fcrdnsEval.Verify(r.Context(), req.IPAddress)
	if err != nil {
		http.Error(w, `{"error":{"code":"check_failed","message":"`+err.Error()+`"}}`, http.StatusBadRequest)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data": result,
	})
}

// PostTLSCheckHandler checks remote host STARTTLS capability
func (a *API) PostTLSCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Host == "" {
		http.Error(w, `{"error":{"code":"invalid_request","message":"host is required"}}`, http.StatusBadRequest)
		return
	}

	result, err := a.tlsChecker.CheckHost(r.Context(), req.Host, req.Port)
	if err != nil {
		http.Error(w, `{"error":{"code":"tls_check_failed","message":"`+err.Error()+`"}}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data": result,
	})
}
