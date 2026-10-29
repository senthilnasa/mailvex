package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/senthilnasa/mailvex/api/middleware"
	"github.com/senthilnasa/mailvex/core/crypto"
	"github.com/senthilnasa/mailvex/core/domain"
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

func (a *API) ImportSuppressionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"Tenant ID missing from context"}}`))
		return
	}

	contentType := r.Header.Get("Content-Type")
	filename := "import.csv"
	var entries []*domain.Suppression

	if strings.HasPrefix(contentType, "multipart/form-data") {
		// Parse multipart form
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"bad_request","message":"Failed to parse multipart form: %s"}}`, err.Error())))
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"code":"bad_request","message":"Form file 'file' is required"}}`))
			return
		}
		defer file.Close()
		if header != nil && header.Filename != "" {
			filename = header.Filename
		}

		parsed, err := parseCSVSuppressions(file)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"csv_parse_error","message":"%s"}}`, err.Error())))
			return
		}
		entries = parsed
	} else if strings.Contains(contentType, "text/csv") || strings.Contains(contentType, "application/csv") {
		parsed, err := parseCSVSuppressions(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"csv_parse_error","message":"%s"}}`, err.Error())))
			return
		}
		entries = parsed
	} else {
		// Assume JSON
		var payload struct {
			Filename string `json:"filename"`
			Entries  []struct {
				Address  string  `json:"address"`
				Type     string  `json:"type"`
				Reason   *string `json:"reason,omitempty"`
				StreamID *string `json:"stream_id,omitempty"`
			} `json:"entries"`
		}

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"code":"bad_request","message":"Failed to read request body"}}`))
			return
		}

		// Try structured payload first
		if err := json.Unmarshal(bodyBytes, &payload); err == nil && len(payload.Entries) > 0 {
			if payload.Filename != "" {
				filename = payload.Filename
			}
			for _, item := range payload.Entries {
				suppType := item.Type
				if suppType == "" {
					suppType = "manual"
				}
				entries = append(entries, &domain.Suppression{
					Address:  item.Address,
					Type:     suppType,
					Reason:   item.Reason,
					StreamID: item.StreamID,
					Source:   "import",
				})
			}
		} else {
			// Try bare list of entries
			var rawList []struct {
				Address  string  `json:"address"`
				Type     string  `json:"type"`
				Reason   *string `json:"reason,omitempty"`
				StreamID *string `json:"stream_id,omitempty"`
			}
			if err := json.Unmarshal(bodyBytes, &rawList); err == nil {
				for _, item := range rawList {
					suppType := item.Type
					if suppType == "" {
						suppType = "manual"
					}
					entries = append(entries, &domain.Suppression{
						Address:  item.Address,
						Type:     suppType,
						Reason:   item.Reason,
						StreamID: item.StreamID,
						Source:   "import",
					})
				}
			} else {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":{"code":"invalid_format","message":"Expected CSV or JSON entries"}}`))
				return
			}
		}
	}

	result, err := a.repo.ImportSuppressions(r.Context(), tenantID, filename, entries)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"internal_error","message":"%s"}}`, err.Error())))
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result)
}

func parseCSVSuppressions(r io.Reader) ([]*domain.Suppression, error) {
	reader := csv.NewReader(r)
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []*domain.Suppression{}, nil
	}

	emailIdx := 0
	typeIdx := -1
	reasonIdx := -1
	streamIdx := -1

	startRow := 0
	firstRow := rows[0]
	isHeader := false
	for i, col := range firstRow {
		colLower := strings.ToLower(strings.TrimSpace(col))
		if colLower == "email" || colLower == "address" || colLower == "recipient" {
			emailIdx = i
			isHeader = true
		} else if colLower == "type" || colLower == "classification" {
			typeIdx = i
			isHeader = true
		} else if colLower == "reason" {
			reasonIdx = i
			isHeader = true
		} else if colLower == "stream_id" || colLower == "stream" {
			streamIdx = i
			isHeader = true
		}
	}

	if isHeader {
		startRow = 1
	}

	var result []*domain.Suppression
	for i := startRow; i < len(rows); i++ {
		row := rows[i]
		if len(row) == 0 || (len(row) == 1 && strings.TrimSpace(row[0]) == "") {
			continue
		}
		if emailIdx >= len(row) {
			continue
		}
		addr := strings.TrimSpace(row[emailIdx])
		if addr == "" {
			continue
		}

		suppType := "manual"
		if typeIdx >= 0 && typeIdx < len(row) && strings.TrimSpace(row[typeIdx]) != "" {
			suppType = strings.TrimSpace(row[typeIdx])
		}

		var reason *string
		if reasonIdx >= 0 && reasonIdx < len(row) && strings.TrimSpace(row[reasonIdx]) != "" {
			rVal := strings.TrimSpace(row[reasonIdx])
			reason = &rVal
		}

		var streamID *string
		if streamIdx >= 0 && streamIdx < len(row) && strings.TrimSpace(row[streamIdx]) != "" {
			sVal := strings.TrimSpace(row[streamIdx])
			streamID = &sVal
		}

		result = append(result, &domain.Suppression{
			Address:  addr,
			Type:     suppType,
			Reason:   reason,
			StreamID: streamID,
			Source:   "import",
		})
	}
	return result, nil
}

func (a *API) ExportSuppressionsHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"Tenant ID missing from context"}}`))
		return
	}

	var streamID *string
	if s := r.URL.Query().Get("stream_id"); s != "" {
		streamID = &s
	}

	format := strings.ToLower(r.URL.Query().Get("format"))
	if format == "" {
		format = "csv"
	}

	supps, err := a.repo.ExportSuppressions(r.Context(), tenantID, streamID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"internal_error","message":"%s"}}`, err.Error())))
		return
	}

	if format == "json" {
		w.Header().Set("Content-Type", "application/json")
		if supps == nil {
			supps = []*domain.Suppression{}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": supps,
		})
		return
	}

	// Default CSV export
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="suppressions.csv"`)
	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"id", "address", "type", "reason", "stream_id", "source", "created_at"})

	for _, s := range supps {
		reason := ""
		if s.Reason != nil {
			reason = *s.Reason
		}
		stID := ""
		if s.StreamID != nil {
			stID = *s.StreamID
		}
		_ = writer.Write([]string{
			s.ID,
			s.Address,
			s.Type,
			reason,
			stID,
			s.Source,
			s.CreatedAt.Format(time.RFC3339),
		})
	}
	writer.Flush()
}

