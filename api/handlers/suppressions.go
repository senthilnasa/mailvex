package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mailvex/mailvex/api/middleware"
	"github.com/mailvex/mailvex/core/crypto"
	"github.com/mailvex/mailvex/core/domain"
)

type CreateSuppressionRequest struct {
	Address  string  `json:"address"`
	Type     string  `json:"type"` // hard_bounce, complaint, unsubscribe, manual, domain_block
	StreamID *string `json:"stream_id,omitempty"`
	Reason   *string `json:"reason,omitempty"`
}

func (a *API) ListSuppressionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"Tenant ID missing from context"}}`))
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	supps, err := a.repo.ListSuppressions(r.Context(), tenantID, limit)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"internal_error","message":"%s"}}`, err.Error())))
		return
	}

	if supps == nil {
		supps = []*domain.Suppression{}
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": supps,
	})
}

func (a *API) CreateSuppressionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"Tenant ID missing from context"}}`))
		return
	}

	var req CreateSuppressionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"invalid_json","message":"Invalid request body"}}`))
		return
	}

	req.Address = strings.TrimSpace(strings.ToLower(req.Address))
	if req.Address == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"missing_address","message":"Address is required"}}`))
		return
	}

	if req.Type == "" {
		req.Type = "manual"
	}

	dom := ""
	parts := strings.Split(req.Address, "@")
	if len(parts) == 2 {
		dom = parts[1]
	} else if req.Type == "domain_block" {
		dom = req.Address
	}

	supp := &domain.Suppression{
		ID:            crypto.NewULID("supp_"),
		TenantID:      tenantID,
		StreamID:      req.StreamID,
		Address:       req.Address,
		AddressDomain: dom,
		Type:          req.Type,
		Reason:        req.Reason,
		Source:        "api",
		CreatedAt:     time.Now().UTC(),
	}

	if err := a.repo.AddSuppression(r.Context(), supp); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"internal_error","message":"%s"}}`, err.Error())))
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(supp)
}

func (a *API) DeleteSuppressionHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"Tenant ID missing from context"}}`))
		return
	}

	id := r.PathValue("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"invalid_id","message":"Suppression ID required"}}`))
		return
	}

	if err := a.repo.DeleteSuppression(r.Context(), tenantID, id); err != nil {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"not_found","message":"Suppression not found"}}`))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
