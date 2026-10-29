package handlers

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/senthilnasa/mailvex/api/middleware"
	"github.com/senthilnasa/mailvex/core/crypto"
	"github.com/senthilnasa/mailvex/core/domain"
	"github.com/senthilnasa/mailvex/core/webhooks"
)

type CreateWebhookRequest struct {
	URL         string   `json:"url"`
	Description *string  `json:"description,omitempty"`
	Events      []string `json:"events"`
}

type UpdateWebhookRequest struct {
	URL         *string   `json:"url,omitempty"`
	Description *string   `json:"description,omitempty"`
	Events      *[]string `json:"events,omitempty"`
	Status      *string   `json:"status,omitempty"`
}

// CreateWebhookEndpointHandler handles POST /v1/webhooks
func (a *API) CreateWebhookEndpointHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "unauthorized"}})
		return
	}

	var req CreateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "malformed json body"}})
		return
	}

	trimmedURL := strings.TrimSpace(req.URL)
	parsedURL, err := url.Parse(trimmedURL)
	if err != nil || (parsedURL.Scheme != "https" && parsedURL.Scheme != "http") || parsedURL.Host == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "valid http/https destination url required"}})
		return
	}

	events := req.Events
	if len(events) == 0 {
		events = []string{"*"}
	}

	now := time.Now().UTC()
	endpoint := &domain.WebhookEndpoint{
		ID:                  crypto.NewULID("whe_"),
		TenantID:            tenantID,
		URL:                 trimmedURL,
		Description:         req.Description,
		Events:              events,
		Secret:              webhooks.GenerateSecret(),
		Status:              "active",
		ConsecutiveFailures: 0,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	if err := a.repo.CreateWebhookEndpoint(r.Context(), endpoint); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": err.Error()}})
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": endpoint,
	})
}

// ListWebhookEndpointsHandler handles GET /v1/webhooks
func (a *API) ListWebhookEndpointsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "unauthorized"}})
		return
	}

	endpoints, err := a.repo.ListWebhookEndpoints(r.Context(), tenantID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": err.Error()}})
		return
	}
	if endpoints == nil {
		endpoints = []*domain.WebhookEndpoint{}
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": endpoints,
	})
}

// GetWebhookEndpointHandler handles GET /v1/webhooks/{id}
func (a *API) GetWebhookEndpointHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "unauthorized"}})
		return
	}

	id := r.PathValue("id")
	endpoint, err := a.repo.GetWebhookEndpoint(r.Context(), tenantID, id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "webhook endpoint not found"}})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": endpoint,
	})
}

// UpdateWebhookEndpointHandler handles PATCH /v1/webhooks/{id}
func (a *API) UpdateWebhookEndpointHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "unauthorized"}})
		return
	}

	id := r.PathValue("id")
	endpoint, err := a.repo.GetWebhookEndpoint(r.Context(), tenantID, id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "webhook endpoint not found"}})
		return
	}

	var req UpdateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "malformed json request body"}})
		return
	}

	if req.URL != nil {
		trimmedURL := strings.TrimSpace(*req.URL)
		parsedURL, err := url.Parse(trimmedURL)
		if err != nil || (parsedURL.Scheme != "https" && parsedURL.Scheme != "http") || parsedURL.Host == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "valid http/https destination url required"}})
			return
		}
		endpoint.URL = trimmedURL
	}

	if req.Description != nil {
		endpoint.Description = req.Description
	}

	if req.Events != nil && len(*req.Events) > 0 {
		endpoint.Events = *req.Events
	}

	if req.Status != nil {
		status := strings.ToLower(*req.Status)
		if status != "active" && status != "disabled_manual" && status != "disabled_failing" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "status must be active or disabled_manual"}})
			return
		}
		endpoint.Status = status
		if status == "active" {
			endpoint.ConsecutiveFailures = 0
		}
	}

	if err := a.repo.UpdateWebhookEndpoint(r.Context(), endpoint); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": err.Error()}})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": endpoint,
	})
}

// DeleteWebhookEndpointHandler handles DELETE /v1/webhooks/{id}
func (a *API) DeleteWebhookEndpointHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "unauthorized"}})
		return
	}

	id := r.PathValue("id")
	if err := a.repo.DeleteWebhookEndpoint(r.Context(), tenantID, id); err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "webhook endpoint not found"}})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"deleted": true,
		"id":      id,
	})
}

// RotateWebhookSecretHandler handles POST /v1/webhooks/{id}/rotate-secret (§1.2: 24h overlap window)
func (a *API) RotateWebhookSecretHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "unauthorized"}})
		return
	}

	id := r.PathValue("id")
	newSecret := webhooks.GenerateSecret()
	overlapExpiry := time.Now().UTC().Add(24 * time.Hour)

	updated, err := a.repo.RotateWebhookSecret(r.Context(), tenantID, id, newSecret, overlapExpiry)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "webhook endpoint not found"}})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data":               updated,
		"overlap_expires_at": overlapExpiry.Format(time.RFC3339),
		"message":            "Secret rotated. Signatures will include both old and new secrets for 24 hours.",
	})
}

// TestWebhookEndpointHandler handles POST /v1/webhooks/{id}/test
func (a *API) TestWebhookEndpointHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "unauthorized"}})
		return
	}

	id := r.PathValue("id")
	del, err := a.webhooks.DispatchPing(r.Context(), tenantID, id)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": err.Error()}})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"delivery": del,
	})
}

// ListWebhookDeliveriesHandler handles GET /v1/webhooks/{id}/deliveries
func (a *API) ListWebhookDeliveriesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "unauthorized"}})
		return
	}

	id := r.PathValue("id")
	limit := 20
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 && val <= 100 {
			limit = val
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}

	deliveries, err := a.repo.ListWebhookDeliveries(r.Context(), tenantID, id, limit, offset)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": err.Error()}})
		return
	}
	if deliveries == nil {
		deliveries = []*domain.WebhookDelivery{}
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data":   deliveries,
		"limit":  limit,
		"offset": offset,
	})
}
