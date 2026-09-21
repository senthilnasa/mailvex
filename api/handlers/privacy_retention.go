package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mailvex/mailvex/api/middleware"
	"github.com/mailvex/mailvex/core/domain"
)

type PrivacyEmailRequest struct {
	Email string `json:"email"`
}

// POST /v1/privacy/erasure
func (a *API) PrivacyErasureHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"Tenant ID missing from context"}}`))
		return
	}

	var req PrivacyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"invalid_json","message":"Invalid request body"}}`))
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"missing_email","message":"Email address is required"}}`))
		return
	}

	res, err := a.privacy.RequestErasure(r.Context(), tenantID, req.Email)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"erasure_failed","message":"%s"}}`, err.Error())))
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

// POST /v1/privacy/export
func (a *API) PrivacyExportHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"Tenant ID missing from context"}}`))
		return
	}

	var req PrivacyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"invalid_json","message":"Invalid request body"}}`))
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"missing_email","message":"Email address is required"}}`))
		return
	}

	res, err := a.privacy.RequestExport(r.Context(), tenantID, req.Email)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"export_failed","message":"%s"}}`, err.Error())))
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

// GET /v1/privacy/requests
func (a *API) ListPrivacyRequestsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"Tenant ID missing from context"}}`))
		return
	}

	reqs, err := a.privacy.ListRequests(r.Context(), tenantID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"internal_error","message":"%s"}}`, err.Error())))
		return
	}

	if reqs == nil {
		reqs = []*domain.PrivacyRequest{}
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": reqs,
	})
}

// GET /v1/retention
func (a *API) GetRetentionPolicyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"Tenant ID missing from context"}}`))
		return
	}

	pol, err := a.retention.GetPolicy(r.Context(), tenantID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"internal_error","message":"%s"}}`, err.Error())))
		return
	}

	_ = json.NewEncoder(w).Encode(pol)
}

// PUT /v1/retention
func (a *API) UpdateRetentionPolicyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"Tenant ID missing from context"}}`))
		return
	}

	var pol domain.RetentionPolicy
	if err := json.NewDecoder(r.Body).Decode(&pol); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"invalid_json","message":"Invalid request body"}}`))
		return
	}

	pol.TenantID = tenantID
	if err := a.retention.UpdatePolicy(r.Context(), &pol); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"bad_request","message":"%s"}}`, err.Error())))
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(pol)
}

// POST /v1/retention/purge
func (a *API) ExecuteRetentionPurgeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"Tenant ID missing from context"}}`))
		return
	}

	res, err := a.retention.ExecutePurge(r.Context(), tenantID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"internal_error","message":"%s"}}`, err.Error())))
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}
