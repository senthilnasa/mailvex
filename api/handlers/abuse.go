package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/senthilnasa/mailvex/api/middleware"
	"github.com/senthilnasa/mailvex/core/crypto"
	"github.com/senthilnasa/mailvex/core/domain"
)

type UpdateAbuseStatusRequest struct {
	Notes string `json:"notes"`
}

type IngestReportRequest struct {
	RawMIME           string `json:"raw_mime"`
	EnvelopeRecipient string `json:"envelope_recipient,omitempty"`
}

// ListAbuseReportsHandler returns abuse intake reports (§18.1)
func (a *API) ListAbuseReportsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	status := r.URL.Query().Get("status")
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	reports, err := a.repo.ListAbuseReports(r.Context(), status, limit)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": err.Error()}})
		return
	}
	if reports == nil {
		reports = []*domain.AbuseReport{}
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"data": reports})
}

// GetAbuseReportHandler returns a single abuse report
func (a *API) GetAbuseReportHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id := r.PathValue("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "missing report id"}})
		return
	}

	report, err := a.repo.GetAbuseReport(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "report not found"}})
		return
	}

	_ = json.NewEncoder(w).Encode(report)
}

// ResolveAbuseReportHandler marks an abuse report as resolved
func (a *API) ResolveAbuseReportHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id := r.PathValue("id")

	var req UpdateAbuseStatusRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	if err := a.repo.UpdateAbuseReportStatus(r.Context(), id, "resolved", &req.Notes); err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": err.Error()}})
		return
	}

	rep, _ := a.repo.GetAbuseReport(r.Context(), id)
	_ = json.NewEncoder(w).Encode(rep)
}

// EscalateAbuseReportHandler marks an abuse report as escalated
func (a *API) EscalateAbuseReportHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id := r.PathValue("id")

	var req UpdateAbuseStatusRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	if err := a.repo.UpdateAbuseReportStatus(r.Context(), id, "escalated", &req.Notes); err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": err.Error()}})
		return
	}

	rep, _ := a.repo.GetAbuseReport(r.Context(), id)
	_ = json.NewEncoder(w).Encode(rep)
}

// IngestDSNReportHandler processes an incoming DSN bounce report (§18.1)
func (a *API) IngestDSNReportHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ct := r.Header.Get("Content-Type")

	var rawMIME []byte
	var envRcpt string

	if strings.Contains(ct, "application/json") {
		var req IngestReportRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "invalid JSON"}})
			return
		}
		rawMIME = []byte(req.RawMIME)
		envRcpt = req.EnvelopeRecipient
	} else {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "failed to read body"}})
			return
		}
		rawMIME = body
		envRcpt = r.URL.Query().Get("recipient")
	}

	res, err := a.reports.ProcessInboundDSN(r.Context(), rawMIME, envRcpt)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": err.Error()}})
		return
	}

	_ = json.NewEncoder(w).Encode(res)
}

// IngestARFReportHandler processes an incoming RFC 5965 ARF feedback loop report (§18.1)
func (a *API) IngestARFReportHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ct := r.Header.Get("Content-Type")

	var rawMIME []byte
	if strings.Contains(ct, "application/json") {
		var req IngestReportRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "invalid JSON"}})
			return
		}
		rawMIME = []byte(req.RawMIME)
	} else {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "failed to read body"}})
			return
		}
		rawMIME = body
	}

	res, err := a.reports.ProcessInboundARF(r.Context(), rawMIME)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": err.Error()}})
		return
	}

	_ = json.NewEncoder(w).Encode(res)
}

// ListFBLReportsHandler returns FBL complaint reports for the tenant
func (a *API) ListFBLReportsHandler(w http.ResponseWriter, r *http.Request) {
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

	reports, err := a.repo.ListFBLReports(r.Context(), tenantID, limit)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": err.Error()}})
		return
	}
	if reports == nil {
		reports = []*domain.FBLReport{}
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"data": reports})
}

// ListPostmasterSnapshotsHandler returns Google Postmaster / SNDS metrics snapshots
func (a *API) ListPostmasterSnapshotsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	domainID := r.URL.Query().Get("domain_id")
	if domainID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "domain_id is required"}})
		return
	}

	limit := 30
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	snaps, err := a.repo.ListPostmasterSnapshots(r.Context(), domainID, limit)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": err.Error()}})
		return
	}
	if snaps == nil {
		snaps = []*domain.PostmasterSnapshot{}
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"data": snaps})
}

// SavePostmasterSnapshotHandler saves a new provider reputation snapshot
func (a *API) SavePostmasterSnapshotHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var snap domain.PostmasterSnapshot
	if err := json.NewDecoder(r.Body).Decode(&snap); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "invalid JSON"}})
		return
	}

	if snap.ID == "" {
		snap.ID = crypto.NewULID("pms_")
	}
	if snap.SampleDate == "" {
		snap.SampleDate = time.Now().UTC().Format("2006-01-02")
	}
	snap.CreatedAt = time.Now().UTC()

	if err := a.repo.SavePostmasterSnapshot(r.Context(), &snap); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": err.Error()}})
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(snap)
}
