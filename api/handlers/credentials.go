package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/senthilnasa/mailvex/api/middleware"
	"github.com/senthilnasa/mailvex/core/crypto"
	"github.com/senthilnasa/mailvex/core/domain"
)

type CreateSMTPCredRequest struct {
	Username       string   `json:"username"`
	AllowedDomains []string `json:"allowed_domains,omitempty"`
	AllowedCIDRs   []string `json:"allowed_cidrs,omitempty"`
	RateLimit      int      `json:"rate_limit,omitempty"`
}

// CreateSMTPCredentialHandler generates a new SMTP credential with displayed-once password
func (a *API) CreateSMTPCredentialHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	user, _ := r.Context().Value(middleware.UserKey).(*domain.User)

	var req CreateSMTPCredRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.Username == "" {
		req.Username = "smtp_" + crypto.NewULID("")[5:17]
	}
	if req.RateLimit <= 0 {
		req.RateLimit = 100
	}

	// Generate 16-byte random password
	pwdBytes := make([]byte, 16)
	_, _ = rand.Read(pwdBytes)
	plainPassword := hex.EncodeToString(pwdBytes)

	hash, err := crypto.HashPassword(plainPassword)
	if err != nil {
		http.Error(w, `{"error":{"code":"hashing_failed","message":"Argon2id hashing failed"}}`, http.StatusInternalServerError)
		return
	}

	credID := crypto.NewULID(crypto.PrefixSMTPCred)
	cred := &domain.SMTPCredential{
		ID:             credID,
		TenantID:       tenantID,
		Username:       req.Username,
		PasswordHash:   hash,
		AllowedDomains: req.AllowedDomains,
		AllowedCIDRs:   req.AllowedCIDRs,
		RateLimit:      req.RateLimit,
		CreatedAt:      time.Now().UTC(),
	}

	if err := a.repo.CreateSMTPCredential(r.Context(), cred); err != nil {
		http.Error(w, `{"error":{"code":"cred_create_failed","message":"`+err.Error()+`"}}`, http.StatusConflict)
		return
	}

	_ = a.audit.Log(r.Context(), &domain.AuditLog{
		TenantID:     &tenantID,
		ActorUserID:  &user.ID,
		Action:       "smtp_credentials.created",
		ResourceType: "smtp_credential",
		ResourceID:   credID,
		AfterState:   `{"username":"` + req.Username + `"}`,
	})

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"credential": cred,
		"password":   plainPassword, // Displayed once
	})
}

// ListSMTPCredentialsHandler lists credentials for tenant
func (a *API) ListSMTPCredentialsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)

	creds, err := a.repo.ListSMTPCredentials(r.Context(), tenantID)
	if err != nil {
		http.Error(w, `{"error":{"code":"internal_error","message":"`+err.Error()+`"}}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": creds,
	})
}

type CreateDeviceRequest struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"` // camera, printer, nas, etc.
	ExpectedIP  string   `json:"expected_ip"`
	AllowedFrom []string `json:"allowed_from,omitempty"`
}

// CreateDeviceHandler registers an IoT device
func (a *API) CreateDeviceHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)

	var req CreateDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"code":"invalid_request","message":"Malformed JSON"}}`, http.StatusBadRequest)
		return
	}

	devID := crypto.NewULID("dev_")
	dev := &domain.Device{
		ID:          devID,
		TenantID:    tenantID,
		Name:        req.Name,
		Type:        req.Type,
		ExpectedIP:  req.ExpectedIP,
		AllowedFrom: req.AllowedFrom,
		CreatedAt:   time.Now().UTC(),
	}

	if err := a.repo.CreateDevice(r.Context(), dev); err != nil {
		http.Error(w, `{"error":{"code":"device_create_failed","message":"`+err.Error()+`"}}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(dev)
}

// ListDevicesHandler lists registered LAN devices
func (a *API) ListDevicesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)

	devices, err := a.repo.ListDevices(r.Context(), tenantID)
	if err != nil {
		http.Error(w, `{"error":{"code":"internal_error","message":"`+err.Error()+`"}}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": devices,
	})
}

type AddTrustedNetworkRequest struct {
	CIDR        string `json:"cidr"`
	Description string `json:"description"`
}

// AddTrustedNetworkHandler adds a subnet CIDR to trusted networks
func (a *API) AddTrustedNetworkHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)

	var req AddTrustedNetworkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"code":"invalid_request","message":"Malformed JSON"}}`, http.StatusBadRequest)
		return
	}

	tn := &domain.TrustedNetwork{
		ID:          crypto.NewULID("tn_"),
		TenantID:    tenantID,
		CIDR:        req.CIDR,
		Description: req.Description,
		CreatedAt:   time.Now().UTC(),
	}

	if err := a.repo.AddTrustedNetwork(r.Context(), tn); err != nil {
		http.Error(w, `{"error":{"code":"trusted_net_failed","message":"`+err.Error()+`"}}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(tn)
}
