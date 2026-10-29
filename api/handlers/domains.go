package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/senthilnasa/mailvex/api/middleware"
	"github.com/senthilnasa/mailvex/core/crypto"
	"github.com/senthilnasa/mailvex/core/dns"
	"github.com/senthilnasa/mailvex/core/domain"
)

type CreateDomainRequest struct {
	Name string `json:"name"`
	Type string `json:"type"` // "public" or "local"
}

// CreateDomainHandler adds a sending domain and generates DKIM and DNS records
func (a *API) CreateDomainHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	user, _ := r.Context().Value(middleware.UserKey).(*domain.User)

	var req CreateDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"code":"invalid_request","message":"Malformed JSON"}}`, http.StatusBadRequest)
		return
	}

	req.Name = strings.TrimSpace(strings.ToLower(req.Name))
	if req.Name == "" || !strings.Contains(req.Name, ".") {
		http.Error(w, `{"error":{"code":"invalid_domain","message":"Invalid domain name"}}`, http.StatusBadRequest)
		return
	}

	if req.Type == "" {
		req.Type = "public"
	}

	// 1. Generate ownership token
	tokenBytes := make([]byte, 16)
	_, _ = rand.Read(tokenBytes)
	ownershipToken := hex.EncodeToString(tokenBytes)

	domID := crypto.NewULID(crypto.PrefixDomain)
	dom := &domain.Domain{
		ID:             domID,
		TenantID:       tenantID,
		Name:           req.Name,
		Type:           req.Type,
		Status:         "pending",
		OwnershipToken: ownershipToken,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	if err := a.repo.CreateDomain(r.Context(), dom); err != nil {
		if err.Error() == "domain_claimed_by_another_tenant" {
			http.Error(w, `{"error":{"code":"domain_claimed_by_another_tenant","message":"Domain is already claimed and verified by another tenant"}}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"error":{"code":"domain_create_failed","message":"`+err.Error()+`"}}`, http.StatusConflict)
		return
	}

	// 2. Generate RSA 2048 DKIM key
	dkimPair, err := crypto.GenerateDKIMKey(2048)
	if err != nil {
		http.Error(w, `{"error":{"code":"dkim_gen_failed","message":"DKIM generation failed: `+err.Error()+`"}}`, http.StatusInternalServerError)
		return
	}

	// Encrypt private key with tenant DEK
	activeTenantKey, err := a.repo.GetActiveTenantKey(r.Context(), tenantID)
	var encPrivKey string
	if err == nil && activeTenantKey != nil && a.encryption != nil {
		dek, err := a.encryption.DecryptDEK(activeTenantKey.EncryptedDEK)
		if err == nil {
			encPrivKey, _ = crypto.EncryptSecret(dek, []byte(dkimPair.PrivateKeyPEM))
		}
	}
	if encPrivKey == "" {
		encPrivKey = dkimPair.PrivateKeyPEM
	}

	dkimKeyID := crypto.NewULID(crypto.PrefixAPIKey)
	dkimKey := &domain.DKIMKey{
		ID:                  dkimKeyID,
		DomainID:            domID,
		TenantID:            tenantID,
		Selector:            "mvx",
		Algorithm:           "rsa2048",
		EncryptedPrivateKey: encPrivKey,
		PublicKey:           dkimPair.PublicKeyB64,
		Status:              "pending",
		CreatedAt:           time.Now().UTC(),
	}
	_ = a.repo.SaveDKIMKey(r.Context(), dkimKey)

	// 3. Generate default DNS records
	platformHost := "mail.example.com"
	records := []*domain.DNSRecord{
		{
			ID:            crypto.NewULID("dns_"),
			DomainID:      domID,
			RecordType:    "TXT",
			Name:          fmt.Sprintf("_mailvex-verify.%s", dom.Name),
			ExpectedValue: fmt.Sprintf("mailvex-verify=%s", ownershipToken),
			Status:        "pending",
			CreatedAt:     time.Now().UTC(),
		},
		{
			ID:            crypto.NewULID("dns_"),
			DomainID:      domID,
			RecordType:    "TXT",
			Name:          dom.Name,
			ExpectedValue: fmt.Sprintf("v=spf1 include:%s -all", platformHost),
			Status:        "pending",
			CreatedAt:     time.Now().UTC(),
		},
		{
			ID:            crypto.NewULID("dns_"),
			DomainID:      domID,
			RecordType:    "TXT",
			Name:          fmt.Sprintf("mvx._domainkey.%s", dom.Name),
			ExpectedValue: fmt.Sprintf("v=DKIM1; k=rsa; p=%s", dkimPair.PublicKeyB64),
			Status:        "pending",
			CreatedAt:     time.Now().UTC(),
		},
		{
			ID:            crypto.NewULID("dns_"),
			DomainID:      domID,
			RecordType:    "TXT",
			Name:          fmt.Sprintf("_dmarc.%s", dom.Name),
			ExpectedValue: fmt.Sprintf("v=DMARC1; p=none; rua=mailto:dmarc-reports@%s", platformHost),
			Status:        "pending",
			CreatedAt:     time.Now().UTC(),
		},
		{
			ID:            crypto.NewULID("dns_"),
			DomainID:      domID,
			RecordType:    "CNAME",
			Name:          fmt.Sprintf("bounce.%s", dom.Name),
			ExpectedValue: fmt.Sprintf("bounce.%s", platformHost),
			Status:        "pending",
			CreatedAt:     time.Now().UTC(),
		},
		{
			ID:            crypto.NewULID("dns_"),
			DomainID:      domID,
			RecordType:    "CNAME",
			Name:          fmt.Sprintf("track.%s", dom.Name),
			ExpectedValue: fmt.Sprintf("track.%s", platformHost),
			Status:        "pending",
			CreatedAt:     time.Now().UTC(),
		},
	}
	_ = a.repo.SaveDNSRecords(r.Context(), domID, records)

	// Audit Log
	_ = a.audit.Log(r.Context(), &domain.AuditLog{
		TenantID:     &tenantID,
		ActorUserID:  &user.ID,
		Action:       "domain.created",
		ResourceType: "domain",
		ResourceID:   domID,
		AfterState:   `{"name":"` + dom.Name + `","type":"` + dom.Type + `"}`,
	})

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"domain":      dom,
		"dkim":        dkimKey,
		"dns_records": records,
	})
}

// ListDomainsHandler returns domains belonging to the tenant
func (a *API) ListDomainsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)

	domains, err := a.repo.ListDomains(r.Context(), tenantID)
	if err != nil {
		http.Error(w, `{"error":{"code":"internal_error","message":"`+err.Error()+`"}}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": domains,
	})
}

// GetDomainHandler returns domain details, active DKIM key, and DNS records
func (a *API) GetDomainHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	domainID := r.PathValue("id")

	dom, err := a.repo.GetDomain(r.Context(), tenantID, domainID)
	if err != nil {
		http.Error(w, `{"error":{"code":"not_found","message":"Domain not found"}}`, http.StatusNotFound)
		return
	}

	dkim, _ := a.repo.GetActiveDKIMKey(r.Context(), domainID)
	records, _ := a.repo.GetDNSRecords(r.Context(), domainID)

	_ = json.NewEncoder(w).Encode(map[string]any{
		"domain":      dom,
		"dkim":        dkim,
		"dns_records": records,
	})
}

// VerifyDomainHandler executes DNS verification checks against configured resolver
func (a *API) VerifyDomainHandler(w http.ResponseWriter, r *http.Request, dnsEngine *dns.Engine) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	domainID := r.PathValue("id")

	dom, err := a.repo.GetDomain(r.Context(), tenantID, domainID)
	if err != nil {
		http.Error(w, `{"error":{"code":"not_found","message":"Domain not found"}}`, http.StatusNotFound)
		return
	}

	dkim, _ := a.repo.GetActiveDKIMKey(r.Context(), domainID)
	platformHost := "mail.example.com"

	if dnsEngine == nil {
		dnsEngine = dns.NewEngine(dns.NewSystemResolver())
	}
	results, allPassed := dnsEngine.VerifyDomain(r.Context(), dom, dkim, platformHost)

	// Update records
	var updatedRecords []*domain.DNSRecord
	for _, res := range results {
		now := time.Now().UTC()
		rec := &domain.DNSRecord{
			ID:            crypto.NewULID("dns_"),
			DomainID:      domainID,
			RecordType:    res.RecordType,
			Name:          res.Name,
			ExpectedValue: res.ExpectedValue,
			Status:        res.Status,
			LastCheckedAt: &now,
			CreatedAt:     now,
		}
		if res.ObservedValue != "" {
			rec.ObservedValue = &res.ObservedValue
		}
		if res.ErrorMessage != "" {
			rec.ErrorMessage = &res.ErrorMessage
		}
		updatedRecords = append(updatedRecords, rec)
	}
	_ = a.repo.SaveDNSRecords(r.Context(), domainID, updatedRecords)

	now := time.Now().UTC()
	dom.LastCheckedAt = &now

	// Check ownership status
	ownershipPassed := false
	if r.URL.Query().Get("force") == "true" {
		user, _ := r.Context().Value(middleware.UserKey).(*domain.User)
		if user != nil && user.IsPlatformAdmin {
			ownershipPassed = true
		}
	}
	if !ownershipPassed {
		for _, res := range results {
			if strings.HasPrefix(res.Name, "_mailvex-verify") && res.Status == "pass" {
				ownershipPassed = true
				break
			}
		}
	}

	if ownershipPassed {
		dom.Status = "verified"
		dom.VerifiedAt = &now
		dom.DegradedAt = nil
		if dkim != nil {
			_ = a.repo.UpdateDKIMKeyStatus(r.Context(), dkim.ID, "active")
		}
	} else {
		// Evaluate degraded / suspended flow (§10.1)
		newStatus, _ := dns.CheckDegradedStatus(dom, false, 72)
		dom.Status = newStatus
	}

	_ = a.repo.UpdateDomain(r.Context(), dom)

	_ = json.NewEncoder(w).Encode(map[string]any{
		"domain":      dom,
		"all_passed":  allPassed,
		"dns_results": updatedRecords,
	})
}

// RotateDKIMHandler creates a new DKIM key and initiates selector rotation flow (§10.5)
func (a *API) RotateDKIMHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	domainID := r.PathValue("id")
	user, _ := r.Context().Value(middleware.UserKey).(*domain.User)

	dom, err := a.repo.GetDomain(r.Context(), tenantID, domainID)
	if err != nil {
		http.Error(w, `{"error":{"code":"not_found","message":"Domain not found"}}`, http.StatusNotFound)
		return
	}

	// Mark active key as retiring
	activeKey, err := a.repo.GetActiveDKIMKey(r.Context(), domainID)
	if err == nil && activeKey != nil {
		_ = a.repo.UpdateDKIMKeyStatus(r.Context(), activeKey.ID, "retiring")
	}

	newSelector := fmt.Sprintf("mvx%d", time.Now().Unix()%10000)
	dkimPair, err := crypto.GenerateDKIMKey(2048)
	if err != nil {
		http.Error(w, `{"error":{"code":"dkim_gen_failed","message":"Failed to generate RSA key"}}`, http.StatusInternalServerError)
		return
	}

	activeTenantKey, _ := a.repo.GetActiveTenantKey(r.Context(), tenantID)
	var encPrivKey string
	if activeTenantKey != nil && a.encryption != nil {
		dek, err := a.encryption.DecryptDEK(activeTenantKey.EncryptedDEK)
		if err == nil {
			encPrivKey, _ = crypto.EncryptSecret(dek, []byte(dkimPair.PrivateKeyPEM))
		}
	}
	if encPrivKey == "" {
		encPrivKey = dkimPair.PrivateKeyPEM
	}

	newKey := &domain.DKIMKey{
		ID:                  crypto.NewULID(crypto.PrefixAPIKey),
		DomainID:            domainID,
		TenantID:            tenantID,
		Selector:            newSelector,
		Algorithm:           "rsa2048",
		EncryptedPrivateKey: encPrivKey,
		PublicKey:           dkimPair.PublicKeyB64,
		Status:              "pending",
		CreatedAt:           time.Now().UTC(),
	}
	_ = a.repo.SaveDKIMKey(r.Context(), newKey)

	_ = a.audit.Log(r.Context(), &domain.AuditLog{
		TenantID:     &tenantID,
		ActorUserID:  &user.ID,
		Action:       "domain.dkim.rotated",
		ResourceType: "domain",
		ResourceID:   dom.ID,
		AfterState:   `{"new_selector":"` + newSelector + `"}`,
	})

	_ = json.NewEncoder(w).Encode(map[string]any{
		"message": "DKIM rotation initiated. Create new DNS record and verify.",
		"new_key": newKey,
		"new_dns_record": map[string]string{
			"type":  "TXT",
			"name":  fmt.Sprintf("%s._domainkey.%s", newSelector, dom.Name),
			"value": fmt.Sprintf("v=DKIM1; k=rsa; p=%s", dkimPair.PublicKeyB64),
		},
	})
}

// ListStreamsHandler returns message streams for the tenant
func (a *API) ListStreamsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)

	streams, err := a.repo.ListStreams(r.Context(), tenantID)
	if err != nil {
		http.Error(w, `{"error":{"code":"internal_error","message":"`+err.Error()+`"}}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": streams,
	})
}

type CreateStreamRequest struct {
	Name string `json:"name"`
	Type string `json:"type"` // "transactional" or "broadcast"
}

// CreateStreamHandler adds a custom message stream
func (a *API) CreateStreamHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)

	var req CreateStreamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"code":"invalid_request","message":"Malformed JSON"}}`, http.StatusBadRequest)
		return
	}

	if req.Type != "transactional" && req.Type != "broadcast" {
		req.Type = "transactional"
	}

	streamID := crypto.NewULID(crypto.PrefixStream)
	stream := &domain.MessageStream{
		ID:        streamID,
		TenantID:  tenantID,
		Name:      req.Name,
		Type:      req.Type,
		IsDefault: false,
		CreatedAt: time.Now().UTC(),
	}

	if err := a.repo.CreateStream(r.Context(), stream); err != nil {
		http.Error(w, `{"error":{"code":"stream_create_failed","message":"`+err.Error()+`"}}`, http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(stream)
}
