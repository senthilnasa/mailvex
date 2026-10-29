package handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/mailvex/mailvex/api/middleware"
	"github.com/mailvex/mailvex/core"
	"github.com/mailvex/mailvex/core/domain"
	"github.com/mailvex/mailvex/core/template"
)

type SendEmailRequest struct {
	From              string                 `json:"from"`
	To                []string               `json:"to"`
	Cc                []string               `json:"cc,omitempty"`
	Bcc               []string               `json:"bcc,omitempty"`
	ReplyTo           string                 `json:"reply_to,omitempty"`
	Subject           string                 `json:"subject"`
	Text              string                 `json:"text,omitempty"`
	HTML              string                 `json:"html,omitempty"`
	Attachments       []AttachmentItem       `json:"attachments,omitempty"`
	Headers           map[string]string      `json:"headers,omitempty"`
	Tags              []string               `json:"tags,omitempty"`
	Metadata          map[string]string      `json:"metadata,omitempty"`
	TemplateID        string                 `json:"template_id,omitempty"`
	TemplateVariables map[string]interface{} `json:"template_variables,omitempty"`
	Stream            string                 `json:"stream,omitempty"`
	TrackOpens        *bool                  `json:"track_opens,omitempty"`
	TrackClicks       *bool                  `json:"track_clicks,omitempty"`
	ScheduledAt       *time.Time             `json:"scheduled_at,omitempty"`
	UnsubscribeURL    string                 `json:"unsubscribe_url,omitempty"`
}

type AttachmentItem struct {
	Filename    string  `json:"filename"`
	ContentType string  `json:"content_type"`
	Content     string  `json:"content"` // base64
	ContentID   *string `json:"content_id,omitempty"`
}

type BatchRecipientOverride struct {
	To                string                 `json:"to"`
	TemplateVariables map[string]interface{} `json:"template_variables,omitempty"`
	Metadata          map[string]string      `json:"metadata,omitempty"`
}

type SendBatchEmailRequest struct {
	SendEmailRequest
	Recipients []BatchRecipientOverride `json:"recipients"`
}

func containsCRLF(s string) bool {
	return strings.Contains(s, "\r") || strings.Contains(s, "\n")
}

func validateCustomHeaders(headers map[string]string) error {
	if len(headers) > 20 {
		return fmt.Errorf("custom headers count exceeds maximum of 20")
	}

	prohibited := map[string]bool{
		"from":                   true,
		"to":                     true,
		"cc":                     true,
		"bcc":                    true,
		"received":               true,
		"return-path":            true,
		"authentication-results": true,
		"message-id":             true,
	}

	for k, v := range headers {
		if containsCRLF(k) || containsCRLF(v) {
			return fmt.Errorf("header_injection: CRLF detected in header %s", k)
		}
		lowerK := strings.ToLower(k)
		if prohibited[lowerK] || strings.HasPrefix(lowerK, "dkim-") {
			return fmt.Errorf("header_not_allowed: header %s is reserved", k)
		}
		if !strings.HasPrefix(lowerK, "x-") && lowerK != "list-id" && lowerK != "precedence" {
			return fmt.Errorf("header_not_allowed: header %s must be X-* or List-Id or Precedence", k)
		}
	}
	return nil
}

// SendEmailHandler handles POST /v1/email
func (a *API) SendEmailHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	reqID, _ := r.Context().Value(middleware.RequestIDKey).(string)

	var apiKeyID string
	if k, ok := r.Context().Value(middleware.APIKeyKey).(*domain.APIKey); ok && k != nil {
		apiKeyID = k.ID
	}

	// 1. Read request body bytes for idempotency hashing
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":{"code":"invalid_body","message":"Failed to read request body"}}`, http.StatusBadRequest)
		return
	}
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		idempotencyKey = r.Header.Get("X-Idempotency-Key")
	}
	var scopedIdemKey string
	if idempotencyKey != "" {
		scopedIdemKey = fmt.Sprintf("%s:%s", apiKeyID, idempotencyKey)
		hasher := sha256.New()
		hasher.Write(bodyBytes)
		reqHash := hex.EncodeToString(hasher.Sum(nil))

		existing, err := a.repo.GetIdempotencyRecord(r.Context(), tenantID, scopedIdemKey)
		if err == nil && existing != nil {
			if existing.Status == "in_progress" {
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"error":{"code":"idempotency_in_progress","message":"A request with this idempotency key is currently processing"}}`))
				return
			}
			if existing.RequestHash != reqHash {
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"error":{"code":"idempotency_key_reused","message":"Idempotency key reused with differing request payload"}}`))
				return
			}
			// Return cached original response
			w.WriteHeader(existing.StatusCode)
			_, _ = w.Write(existing.ResponseBody)
			return
		}

		// Save in-progress idempotency record
		_ = a.repo.SetIdempotencyRecord(r.Context(), &domain.IdempotencyRecord{
			ID:          fmt.Sprintf("%s:%s", tenantID, scopedIdemKey),
			TenantID:    tenantID,
			RequestHash: reqHash,
			Status:      "in_progress",
			CreatedAt:   time.Now().UTC(),
			ExpiresAt:   time.Now().UTC().Add(24 * time.Hour),
		})
	}

	// 2. Decode payload
	var req SendEmailRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		http.Error(w, `{"error":{"code":"invalid_json","message":"Malformed JSON request payload"}}`, http.StatusBadRequest)
		return
	}

	// 3. Header injection & CRLF checks
	if containsCRLF(req.From) || containsCRLF(req.ReplyTo) || containsCRLF(req.Subject) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"header_injection","message":"CRLF characters detected in header or envelope field"}}`))
		return
	}
	for _, addr := range append(append(req.To, req.Cc...), req.Bcc...) {
		if containsCRLF(addr) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"code":"header_injection","message":"CRLF characters detected in recipient address"}}`))
			return
		}
	}

	// 4. Validate custom headers
	if err := validateCustomHeaders(req.Headers); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		if strings.HasPrefix(err.Error(), "header_injection") {
			_, _ = w.Write([]byte(`{"error":{"code":"header_injection","message":"CRLF characters detected in custom headers"}}`))
		} else {
			_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"header_not_allowed","message":"%s"}}`, err.Error())))
		}
		return
	}

	// 5. Recipient limits
	totalRecipients := len(req.To) + len(req.Cc) + len(req.Bcc)
	if totalRecipients == 0 {
		http.Error(w, `{"error":{"code":"no_recipients","message":"At least one recipient is required"}}`, http.StatusBadRequest)
		return
	}
	if totalRecipients > 50 {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"recipient_limit_exceeded","message":"Total recipients (to + cc + bcc) cannot exceed 50"}}`))
		return
	}

	// 6. Attachments check (max 20 MB total)
	var totalAttachmentBytes int64
	for _, att := range req.Attachments {
		decoded, err := base64.StdEncoding.DecodeString(att.Content)
		if err != nil {
			http.Error(w, `{"error":{"code":"invalid_attachment","message":"Attachment content must be valid base64"}}`, http.StatusBadRequest)
			return
		}
		totalAttachmentBytes += int64(len(decoded))
	}
	if totalAttachmentBytes > 20*1024*1024 {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"attachment_limit_exceeded","message":"Total attachments size cannot exceed 20 MB"}}`))
		return
	}

	// 7. Parse sender domain & verify domain ownership in tenant
	parsedFrom, err := mail.ParseAddress(req.From)
	if err != nil {
		http.Error(w, `{"error":{"code":"invalid_from","message":"Invalid sender email address"}}`, http.StatusBadRequest)
		return
	}
	fromParts := strings.Split(parsedFrom.Address, "@")
	if len(fromParts) != 2 {
		http.Error(w, `{"error":{"code":"invalid_from","message":"Invalid sender email address format"}}`, http.StatusBadRequest)
		return
	}
	fromDomain := fromParts[1]

	domainObj, err := a.repo.GetDomainByName(r.Context(), fromDomain)
	if err != nil || domainObj.TenantID != tenantID || (domainObj.Status != "verified" && domainObj.Status != "degraded") {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"domain_not_verified","message":"The sending domain has not been verified."}}`))
		return
	}

	// 8. Template rendering if template_id specified
	subject := req.Subject
	htmlBody := req.HTML
	textBody := req.Text

	if req.TemplateID != "" {
		tmplVer, err := a.repo.GetPublishedTemplateVersion(r.Context(), tenantID, req.TemplateID)
		if err != nil {
			http.Error(w, `{"error":{"code":"template_not_found","message":"Published template version not found"}}`, http.StatusNotFound)
			return
		}

		s, err := template.Render(tmplVer.Subject, req.TemplateVariables, template.Options{AutoEscape: false, StrictVariables: false})
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":{"code":"template_render_error","message":"%s"}}`, err.Error()), http.StatusBadRequest)
			return
		}
		h, err := template.Render(tmplVer.HTML, req.TemplateVariables, template.Options{AutoEscape: true, StrictVariables: false})
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":{"code":"template_render_error","message":"%s"}}`, err.Error()), http.StatusBadRequest)
			return
		}
		t, err := template.Render(tmplVer.Text, req.TemplateVariables, template.Options{AutoEscape: false, StrictVariables: false})
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":{"code":"template_render_error","message":"%s"}}`, err.Error()), http.StatusBadRequest)
			return
		}

		subject = s
		htmlBody = h
		textBody = t
	}

	// 9. Environment determination
	environment := "live"
	if k, ok := r.Context().Value(middleware.APIKeyKey).(*domain.APIKey); ok && k != nil && k.Environment == "test" {
		environment = "development"
	}
	if ctxEnv, ok := r.Context().Value(middleware.EnvironmentKey).(string); ok && ctxEnv == "test" {
		environment = "development"
	}

	// 10. Prepare and execute Ingestion
	stream := req.Stream
	if stream == "" {
		stream = "transactional"
	}

	var apiKeyIDPtr *string
	if apiKeyID != "" {
		apiKeyIDPtr = &apiKeyID
	}

	var replyToPtr *string
	if req.ReplyTo != "" {
		replyToPtr = &req.ReplyTo
	}

	ingestReq := core.IngestRequest{
		TenantID:    tenantID,
		Source:      "api",
		APIKeyID:    apiKeyIDPtr,
		FromAddress: parsedFrom.Address,
		FromName:    parsedFrom.Name,
		ReplyTo:     replyToPtr,
		To:          req.To,
		Cc:          req.Cc,
		Bcc:         req.Bcc,
		Subject:     subject,
		BodyHTML:    htmlBody,
		BodyText:    textBody,
		StreamName:  stream,
		Tags:        req.Tags,
		Metadata:    req.Metadata,
		Environment: environment,
	}

	msgRes, err := a.ingest.Ingest(r.Context(), &ingestReq)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":{"code":"ingestion_failed","message":"%s"}}`, err.Error()), http.StatusInternalServerError)
		return
	}

	// 11. Format response
	respObj := map[string]interface{}{
		"id":          msgRes.MessageID,
		"status":      "queued",
		"environment": environment,
		"request_id":  reqID,
	}
	respBytes, _ := json.Marshal(respObj)

	// 12. Save completed idempotency record if key was supplied
	if scopedIdemKey != "" {
		hasher := sha256.New()
		hasher.Write(bodyBytes)
		reqHash := hex.EncodeToString(hasher.Sum(nil))

		_ = a.repo.SetIdempotencyRecord(r.Context(), &domain.IdempotencyRecord{
			ID:           fmt.Sprintf("%s:%s", tenantID, scopedIdemKey),
			TenantID:     tenantID,
			RequestHash:  reqHash,
			Status:       "completed",
			StatusCode:   http.StatusAccepted,
			ResponseBody: respBytes,
			CreatedAt:    time.Now().UTC(),
			ExpiresAt:    time.Now().UTC().Add(24 * time.Hour),
		})
	}

	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write(respBytes)
}

// SendEmailBatchHandler handles POST /v1/email/batch (up to 500 recipients)
func (a *API) SendEmailBatchHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	reqID, _ := r.Context().Value(middleware.RequestIDKey).(string)

	var apiKeyID string
	if k, ok := r.Context().Value(middleware.APIKeyKey).(*domain.APIKey); ok && k != nil {
		apiKeyID = k.ID
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":{"code":"invalid_body","message":"Failed to read request body"}}`, http.StatusBadRequest)
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		idempotencyKey = r.Header.Get("X-Idempotency-Key")
	}
	var scopedIdemKey string
	if idempotencyKey != "" {
		scopedIdemKey = fmt.Sprintf("%s:%s", apiKeyID, idempotencyKey)
		hasher := sha256.New()
		hasher.Write(bodyBytes)
		reqHash := hex.EncodeToString(hasher.Sum(nil))

		existing, err := a.repo.GetIdempotencyRecord(r.Context(), tenantID, scopedIdemKey)
		if err == nil && existing != nil {
			if existing.Status == "in_progress" {
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"error":{"code":"idempotency_in_progress","message":"Batch request currently processing"}}`))
				return
			}
			if existing.RequestHash != reqHash {
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"error":{"code":"idempotency_key_reused","message":"Idempotency key reused with differing batch payload"}}`))
				return
			}
			w.WriteHeader(existing.StatusCode)
			_, _ = w.Write(existing.ResponseBody)
			return
		}

		_ = a.repo.SetIdempotencyRecord(r.Context(), &domain.IdempotencyRecord{
			ID:          fmt.Sprintf("%s:%s", tenantID, scopedIdemKey),
			TenantID:    tenantID,
			RequestHash: reqHash,
			Status:      "in_progress",
			CreatedAt:   time.Now().UTC(),
			ExpiresAt:   time.Now().UTC().Add(24 * time.Hour),
		})
	}

	var req SendBatchEmailRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		http.Error(w, `{"error":{"code":"invalid_json","message":"Malformed JSON request"}}`, http.StatusBadRequest)
		return
	}

	if len(req.Recipients) == 0 {
		http.Error(w, `{"error":{"code":"no_recipients","message":"At least one recipient is required"}}`, http.StatusBadRequest)
		return
	}
	if len(req.Recipients) > 500 {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"batch_limit_exceeded","message":"Batch items cannot exceed 500"}}`))
		return
	}

	// Validate sender domain
	parsedFrom, err := mail.ParseAddress(req.From)
	if err != nil {
		http.Error(w, `{"error":{"code":"invalid_from","message":"Invalid sender email address"}}`, http.StatusBadRequest)
		return
	}
	fromParts := strings.Split(parsedFrom.Address, "@")
	if len(fromParts) != 2 {
		http.Error(w, `{"error":{"code":"invalid_from","message":"Invalid sender email address format"}}`, http.StatusBadRequest)
		return
	}
	fromDomain := fromParts[1]

	domainObj, err := a.repo.GetDomainByName(r.Context(), fromDomain)
	if err != nil || domainObj.TenantID != tenantID || (domainObj.Status != "verified" && domainObj.Status != "degraded") {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"domain_not_verified","message":"The sending domain has not been verified."}}`))
		return
	}

	environment := "live"
	if k, ok := r.Context().Value(middleware.APIKeyKey).(*domain.APIKey); ok && k != nil && k.Environment == "test" {
		environment = "development"
	}
	if ctxEnv, ok := r.Context().Value(middleware.EnvironmentKey).(string); ok && ctxEnv == "test" {
		environment = "development"
	}

	var apiKeyIDPtr *string
	if apiKeyID != "" {
		apiKeyIDPtr = &apiKeyID
	}

	stream := req.Stream
	if stream == "" {
		stream = "transactional"
	}

	type BatchItemResult struct {
		ID     string `json:"id,omitempty"`
		Status string `json:"status,omitempty"`
		Error  string `json:"error,omitempty"`
	}

	results := make([]BatchItemResult, len(req.Recipients))

	for i, rcpt := range req.Recipients {
		if containsCRLF(rcpt.To) {
			results[i] = BatchItemResult{Error: "header_injection: recipient address contains CRLF"}
			continue
		}

		// Merge metadata
		mergedMeta := make(map[string]string)
		for k, v := range req.Metadata {
			mergedMeta[k] = v
		}
		for k, v := range rcpt.Metadata {
			mergedMeta[k] = v
		}

		// Merge template variables
		mergedVars := make(map[string]interface{})
		for k, v := range req.TemplateVariables {
			mergedVars[k] = v
		}
		for k, v := range rcpt.TemplateVariables {
			mergedVars[k] = v
		}

		subject := req.Subject
		htmlBody := req.HTML
		textBody := req.Text

		if req.TemplateID != "" {
			tmplVer, err := a.repo.GetPublishedTemplateVersion(r.Context(), tenantID, req.TemplateID)
			if err == nil && tmplVer != nil {
				subject, _ = template.Render(tmplVer.Subject, mergedVars, template.Options{AutoEscape: false})
				htmlBody, _ = template.Render(tmplVer.HTML, mergedVars, template.Options{AutoEscape: true})
				textBody, _ = template.Render(tmplVer.Text, mergedVars, template.Options{AutoEscape: false})
			}
		}

		var replyToPtr *string
		if req.ReplyTo != "" {
			replyToPtr = &req.ReplyTo
		}

		ingestReq := core.IngestRequest{
			TenantID:    tenantID,
			Source:      "api",
			APIKeyID:    apiKeyIDPtr,
			FromAddress: parsedFrom.Address,
			FromName:    parsedFrom.Name,
			ReplyTo:     replyToPtr,
			To:          []string{rcpt.To},
			Subject:     subject,
			BodyHTML:    htmlBody,
			BodyText:    textBody,
			StreamName:  stream,
			Tags:        req.Tags,
			Metadata:    mergedMeta,
			Environment: environment,
		}

		msgRes, err := a.ingest.Ingest(r.Context(), &ingestReq)
		if err != nil {
			results[i] = BatchItemResult{Error: err.Error()}
		} else {
			results[i] = BatchItemResult{ID: msgRes.MessageID, Status: "queued"}
		}
	}

	respObj := map[string]interface{}{
		"data":        results,
		"request_id":  reqID,
		"environment": environment,
	}
	respBytes, _ := json.Marshal(respObj)

	if scopedIdemKey != "" {
		hasher := sha256.New()
		hasher.Write(bodyBytes)
		reqHash := hex.EncodeToString(hasher.Sum(nil))

		_ = a.repo.SetIdempotencyRecord(r.Context(), &domain.IdempotencyRecord{
			ID:           fmt.Sprintf("%s:%s", tenantID, scopedIdemKey),
			TenantID:     tenantID,
			RequestHash:  reqHash,
			Status:       "completed",
			StatusCode:   http.StatusOK,
			ResponseBody: respBytes,
			CreatedAt:    time.Now().UTC(),
			ExpiresAt:    time.Now().UTC().Add(24 * time.Hour),
		})
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
}
