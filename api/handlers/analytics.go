package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/senthilnasa/mailvex/api/middleware"
	"github.com/senthilnasa/mailvex/core/crypto"
	"github.com/senthilnasa/mailvex/core/domain"
)

type CreateExportRequest struct {
	Format      string  `json:"format"` // csv
	Range       string  `json:"range,omitempty"`
	From        *string `json:"from,omitempty"`
	To          *string `json:"to,omitempty"`
	StreamID    *string `json:"stream_id,omitempty"`
	DomainID    *string `json:"domain_id,omitempty"`
	Provider    *string `json:"provider,omitempty"`
	CampaignTag *string `json:"campaign_tag,omitempty"`
}

func parseAnalyticsFilter(r *http.Request) domain.AnalyticsFilter {
	filter := domain.AnalyticsFilter{}
	q := r.URL.Query()

	if fromStr := q.Get("from"); fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			filter.From = &t
		}
	}
	if toStr := q.Get("to"); toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			filter.To = &t
		}
	}
	if filter.From == nil {
		rangeParam := q.Get("range")
		now := time.Now().UTC()
		switch rangeParam {
		case "24h":
			from := now.Add(-24 * time.Hour)
			filter.From = &from
			filter.To = &now
		case "7d":
			from := now.Add(-7 * 24 * time.Hour)
			filter.From = &from
			filter.To = &now
		case "30d":
			from := now.Add(-30 * 24 * time.Hour)
			filter.From = &from
			filter.To = &now
		}
	}
	if sID := q.Get("stream_id"); sID != "" {
		filter.StreamID = &sID
	}
	if dID := q.Get("domain_id"); dID != "" {
		filter.DomainID = &dID
	}
	if prov := q.Get("provider"); prov != "" {
		filter.Provider = &prov
	}
	if tag := q.Get("tag"); tag != "" {
		filter.CampaignTag = &tag
	}

	return filter
}

// GetAnalyticsOverviewHandler returns aggregate counters and deliverability rates (§19.2)
func (a *API) GetAnalyticsOverviewHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "unauthorized"}})
		return
	}

	filter := parseAnalyticsFilter(r)
	summary, err := a.analytics.GetOverview(r.Context(), tenantID, filter)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": err.Error()}})
		return
	}

	_ = json.NewEncoder(w).Encode(summary)
}

// GetAnalyticsTimeSeriesHandler returns time-series analytics buckets (§19.2)
func (a *API) GetAnalyticsTimeSeriesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "unauthorized"}})
		return
	}

	interval := r.URL.Query().Get("interval")
	if interval == "" {
		interval = "1h"
	}

	filter := parseAnalyticsFilter(r)
	points, err := a.analytics.GetTimeSeries(r.Context(), tenantID, interval, filter)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": err.Error()}})
		return
	}
	if points == nil {
		points = []*domain.TimeSeriesPoint{}
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"data": points})
}

// GetAnalyticsProvidersHandler returns per-provider deliverability and complaint breakdown (§19.2)
func (a *API) GetAnalyticsProvidersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "unauthorized"}})
		return
	}

	filter := parseAnalyticsFilter(r)
	breakdown, err := a.analytics.GetProviderBreakdown(r.Context(), tenantID, filter)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": err.Error()}})
		return
	}
	if breakdown == nil {
		breakdown = []*domain.ProviderBreakdown{}
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"data": breakdown})
}

// GetAnalyticsLatencyHandler returns delivery speed histogram distribution (§19.2)
func (a *API) GetAnalyticsLatencyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "unauthorized"}})
		return
	}

	filter := parseAnalyticsFilter(r)
	hist, err := a.analytics.GetLatencyHistogram(r.Context(), tenantID, filter)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": err.Error()}})
		return
	}

	_ = json.NewEncoder(w).Encode(hist)
}

// CreateAnalyticsExportHandler creates and runs a CSV export generation (§19.3)
func (a *API) CreateAnalyticsExportHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "unauthorized"}})
		return
	}

	var req CreateExportRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	filter := domain.AnalyticsFilter{
		StreamID:    req.StreamID,
		DomainID:    req.DomainID,
		Provider:    req.Provider,
		CampaignTag: req.CampaignTag,
	}
	if req.From != nil {
		if t, err := time.Parse(time.RFC3339, *req.From); err == nil {
			filter.From = &t
		}
	}
	if req.To != nil {
		if t, err := time.Parse(time.RFC3339, *req.To); err == nil {
			filter.To = &t
		}
	}
	if filter.From == nil && req.Range != "" {
		now := time.Now().UTC()
		switch req.Range {
		case "24h":
			from := now.Add(-24 * time.Hour)
			filter.From = &from
			filter.To = &now
		case "7d":
			from := now.Add(-7 * 24 * time.Hour)
			filter.From = &from
			filter.To = &now
		case "30d":
			from := now.Add(-30 * 24 * time.Hour)
			filter.From = &from
			filter.To = &now
		}
	}

	exportID := crypto.NewULID("exp_")
	export := &domain.AnalyticsExport{
		ID:        exportID,
		TenantID:  tenantID,
		Status:    "pending",
		Format:    "csv",
		CreatedAt: time.Now().UTC(),
	}

	if err := a.repo.CreateAnalyticsExport(r.Context(), export); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": err.Error()}})
		return
	}

	// Generate CSV data
	csvData, err := a.analytics.GenerateCSVExport(r.Context(), tenantID, filter)
	if err != nil {
		_ = a.repo.UpdateAnalyticsExport(r.Context(), tenantID, exportID, "failed", nil, nil)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "export generation failed"}})
		return
	}

	downloadURL := fmt.Sprintf("/v1/analytics/exports/%s/download", exportID)
	_ = a.repo.UpdateAnalyticsExport(r.Context(), tenantID, exportID, "completed", &downloadURL, csvData)

	exp, _ := a.repo.GetAnalyticsExport(r.Context(), tenantID, exportID)
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(exp)
}

// ListAnalyticsExportsHandler lists generated exports for tenant
func (a *API) ListAnalyticsExportsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "unauthorized"}})
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	exports, err := a.repo.ListAnalyticsExports(r.Context(), tenantID, limit)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": err.Error()}})
		return
	}
	if exports == nil {
		exports = []*domain.AnalyticsExport{}
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"data": exports})
}

// DownloadAnalyticsExportHandler downloads the exported CSV file
func (a *API) DownloadAnalyticsExportHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "missing export id", http.StatusBadRequest)
		return
	}

	exp, err := a.repo.GetAnalyticsExport(r.Context(), tenantID, id)
	if err != nil || exp.Status != "completed" || len(exp.FileData) == 0 {
		http.Error(w, "export not found or not ready", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"mailvex-analytics-%s.csv\"", exp.ID))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(exp.FileData)
}
