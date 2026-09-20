package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/mailvex/mailvex/api/middleware"
	"github.com/mailvex/mailvex/core/crypto"
	"github.com/mailvex/mailvex/core/domain"
)

type CreateAPIKeyRequest struct {
	Name           string     `json:"name"`
	Environment    string     `json:"environment"` // live or test
	Scopes         []string   `json:"scopes"`
	AllowedDomains []string   `json:"allowed_domains,omitempty"`
	AllowedCIDRs   []string   `json:"allowed_cidrs,omitempty"`
	StreamID       *string    `json:"stream_id,omitempty"`
	RateLimit      int        `json:"rate_limit,omitempty"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
}

// CreateAPIKeyHandler generates a new API key with the token returned once
func (a *API) CreateAPIKeyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)

	var req CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"code":"invalid_request","message":"Malformed JSON"}}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, `{"error":{"code":"validation_failed","message":"Key name is required"}}`, http.StatusBadRequest)
		return
	}

	env := req.Environment
	if env != "test" {
		env = "live"
	}

	scopes := req.Scopes
	if len(scopes) == 0 {
		scopes = []string{"send.email", "messages.read"}
	}

	rateLimit := req.RateLimit
	if rateLimit <= 0 {
		rateLimit = 100
	}

	rawKey, keyHash, prefix, lastFour, err := crypto.GenerateAPIKey(env)
	if err != nil {
		http.Error(w, `{"error":{"code":"keygen_failed","message":"`+err.Error()+`"}}`, http.StatusInternalServerError)
		return
	}

	keyID := crypto.NewULID(crypto.PrefixAPIKey)
	apiKey := &domain.APIKey{
		ID:             keyID,
		TenantID:       tenantID,
		Name:           req.Name,
		Prefix:         prefix,
		KeyHash:        keyHash,
		LastFour:       lastFour,
		Environment:    env,
		Scopes:         scopes,
		AllowedDomains: req.AllowedDomains,
		AllowedCIDRs:   req.AllowedCIDRs,
		StreamID:       req.StreamID,
		RateLimit:      rateLimit,
		ExpiresAt:      req.ExpiresAt,
		CreatedAt:      time.Now().UTC(),
	}

	if err := a.repo.CreateAPIKey(r.Context(), apiKey); err != nil {
		http.Error(w, `{"error":{"code":"db_error","message":"`+err.Error()+`"}}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"token":       rawKey, // returned once!
		"id":          apiKey.ID,
		"name":        apiKey.Name,
		"prefix":      apiKey.Prefix,
		"last_four":   apiKey.LastFour,
		"environment": apiKey.Environment,
		"scopes":      apiKey.Scopes,
		"created_at":  apiKey.CreatedAt,
	})
}

// ListAPIKeysHandler lists active API keys for active tenant
func (a *API) ListAPIKeysHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)

	keys, err := a.repo.ListAPIKeys(r.Context(), tenantID)
	if err != nil {
		http.Error(w, `{"error":{"code":"internal_error","message":"`+err.Error()+`"}}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data": keys,
	})
}

// RevokeAPIKeyHandler revokes an API key immediately
func (a *API) RevokeAPIKeyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	id := r.PathValue("id")

	if err := a.repo.RevokeAPIKey(r.Context(), tenantID, id); err != nil {
		http.Error(w, `{"error":{"code":"revoke_failed","message":"`+err.Error()+`"}}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     id,
		"status": "revoked",
	})
}
