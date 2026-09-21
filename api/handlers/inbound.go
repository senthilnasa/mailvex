package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/mailvex/mailvex/api/middleware"
	"github.com/mailvex/mailvex/core/domain"
)

type CreateInboundRouteRequest struct {
	Priority     int     `json:"priority"`
	MatchType    string  `json:"match_type"` // exact, wildcard, catch_all
	Pattern      string  `json:"pattern"`
	Action       string  `json:"action"` // webhook, store_only, forward, discard
	ActionTarget *string `json:"action_target,omitempty"`
	IsActive     *bool   `json:"is_active,omitempty"`
}

type UpdateInboundRouteRequest struct {
	Priority     *int    `json:"priority,omitempty"`
	MatchType    *string `json:"match_type,omitempty"`
	Pattern      *string `json:"pattern,omitempty"`
	Action       *string `json:"action,omitempty"`
	ActionTarget *string `json:"action_target,omitempty"`
	IsActive     *bool   `json:"is_active,omitempty"`
}

type TestInboundMatchRequest struct {
	Address string `json:"address"`
}

// POST /v1/inbound/routes
func (a *API) CreateInboundRouteHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"Tenant ID missing from context"}}`))
		return
	}

	var req CreateInboundRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"invalid_json","message":"Invalid request body"}}`))
		return
	}

	req.Pattern = strings.TrimSpace(req.Pattern)
	if req.Pattern == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"missing_pattern","message":"Route pattern is required"}}`))
		return
	}

	if req.Priority <= 0 {
		req.Priority = 10
	}
	if req.MatchType == "" {
		req.MatchType = "exact"
	}
	if req.Action == "" {
		req.Action = "webhook"
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	route := &domain.InboundRoute{
		TenantID:     tenantID,
		Priority:     req.Priority,
		MatchType:    req.MatchType,
		Pattern:      req.Pattern,
		Action:       req.Action,
		ActionTarget: req.ActionTarget,
		IsActive:     isActive,
	}

	if err := a.inbound.CreateRoute(r.Context(), route); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"create_failed","message":"%s"}}`, err.Error())))
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(route)
}

// GET /v1/inbound/routes
func (a *API) ListInboundRoutesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"Tenant ID missing from context"}}`))
		return
	}

	routes, err := a.inbound.ListRoutes(r.Context(), tenantID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"internal_error","message":"%s"}}`, err.Error())))
		return
	}

	if routes == nil {
		routes = []*domain.InboundRoute{}
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": routes,
	})
}

// GET /v1/inbound/routes/{id}
func (a *API) GetInboundRouteHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"Tenant ID missing from context"}}`))
		return
	}

	id := r.PathValue("id")
	route, err := a.inbound.GetRoute(r.Context(), tenantID, id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"not_found","message":"Route not found"}}`))
		return
	}

	_ = json.NewEncoder(w).Encode(route)
}

// PATCH /v1/inbound/routes/{id}
func (a *API) UpdateInboundRouteHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"Tenant ID missing from context"}}`))
		return
	}

	id := r.PathValue("id")
	route, err := a.inbound.GetRoute(r.Context(), tenantID, id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"not_found","message":"Route not found"}}`))
		return
	}

	var req UpdateInboundRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"invalid_json","message":"Invalid request body"}}`))
		return
	}

	if req.Priority != nil {
		route.Priority = *req.Priority
	}
	if req.MatchType != nil {
		route.MatchType = *req.MatchType
	}
	if req.Pattern != nil {
		route.Pattern = strings.TrimSpace(*req.Pattern)
	}
	if req.Action != nil {
		route.Action = *req.Action
	}
	if req.ActionTarget != nil {
		route.ActionTarget = req.ActionTarget
	}
	if req.IsActive != nil {
		route.IsActive = *req.IsActive
	}

	if err := a.inbound.UpdateRoute(r.Context(), route); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"update_failed","message":"%s"}}`, err.Error())))
		return
	}

	_ = json.NewEncoder(w).Encode(route)
}

// DELETE /v1/inbound/routes/{id}
func (a *API) DeleteInboundRouteHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"Tenant ID missing from context"}}`))
		return
	}

	id := r.PathValue("id")
	if err := a.inbound.DeleteRoute(r.Context(), tenantID, id); err != nil {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"not_found","message":"Route not found"}}`))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// POST /v1/inbound/routes/test
func (a *API) TestInboundRouteMatchHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"Tenant ID missing from context"}}`))
		return
	}

	var req TestInboundMatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"invalid_json","message":"Invalid request body"}}`))
		return
	}

	matchedRoute, err := a.inbound.TestRouteMatch(r.Context(), tenantID, req.Address)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"matched": false,
			"address": req.Address,
			"message": "No matching inbound route found",
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"matched": true,
		"address": req.Address,
		"route":   matchedRoute,
	})
}

// GET /v1/inbound/messages
func (a *API) ListInboundMessagesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"Tenant ID missing from context"}}`))
		return
	}

	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if p, err := strconv.Atoi(l); err == nil && p > 0 {
			limit = p
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if p, err := strconv.Atoi(o); err == nil && p >= 0 {
			offset = p
		}
	}

	messages, err := a.inbound.ListMessages(r.Context(), tenantID, limit, offset)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"internal_error","message":"%s"}}`, err.Error())))
		return
	}

	if messages == nil {
		messages = []*domain.InboundMessage{}
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": messages,
	})
}

// GET /v1/inbound/messages/{id}
func (a *API) GetInboundMessageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"Tenant ID missing from context"}}`))
		return
	}

	id := r.PathValue("id")
	msg, err := a.inbound.GetMessage(r.Context(), tenantID, id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"not_found","message":"Inbound message not found"}}`))
		return
	}

	_ = json.NewEncoder(w).Encode(msg)
}
