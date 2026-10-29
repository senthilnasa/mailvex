package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/senthilnasa/mailvex/api/middleware"
	"github.com/senthilnasa/mailvex/core/crypto"
	"github.com/senthilnasa/mailvex/core/domain"
	"github.com/senthilnasa/mailvex/core/template"
)

type CreateTemplateRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Subject     string   `json:"subject"`
	HTML        string   `json:"html,omitempty"`
	Text        string   `json:"text,omitempty"`
	Variables   []string `json:"variables,omitempty"`
	PublishNow  bool     `json:"publish_now"`
}

// CreateTemplateHandler creates a new template and initial version 1
func (a *API) CreateTemplateHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)

	var req CreateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"code":"invalid_request","message":"Malformed JSON"}}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, `{"error":{"code":"validation_failed","message":"Template name is required"}}`, http.StatusBadRequest)
		return
	}

	tmplID := crypto.NewULID(crypto.PrefixTemplate)
	verID := crypto.NewULID(crypto.PrefixTemplateVer)

	tmpl := &domain.Template{
		ID:          tmplID,
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	ver := &domain.TemplateVersion{
		ID:          verID,
		TemplateID:  tmplID,
		TenantID:    tenantID,
		Version:     1,
		Subject:     req.Subject,
		HTML:        req.HTML,
		Text:        req.Text,
		Variables:   req.Variables,
		IsPublished: req.PublishNow,
		CreatedAt:   time.Now().UTC(),
	}

	if err := a.repo.CreateTemplate(r.Context(), tmpl, ver); err != nil {
		http.Error(w, `{"error":{"code":"internal_error","message":"`+err.Error()+`"}}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"template": tmpl,
		"version":  ver,
	})
}

// ListTemplatesHandler lists all templates for active tenant
func (a *API) ListTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)

	tmpls, err := a.repo.ListTemplates(r.Context(), tenantID)
	if err != nil {
		http.Error(w, `{"error":{"code":"internal_error","message":"`+err.Error()+`"}}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data": tmpls,
	})
}

// GetTemplateHandler retrieves template and published version
func (a *API) GetTemplateHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	id := r.PathValue("id")

	tmpl, err := a.repo.GetTemplate(r.Context(), tenantID, id)
	if err != nil {
		http.Error(w, `{"error":{"code":"template_not_found","message":"Template not found"}}`, http.StatusNotFound)
		return
	}

	pubVer, _ := a.repo.GetPublishedTemplateVersion(r.Context(), tenantID, id)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"template":          tmpl,
		"published_version": pubVer,
	})
}

type RenderTemplateRequest struct {
	Variables map[string]interface{} `json:"variables"`
}

// RenderTemplateHandler renders template preview with test variables
func (a *API) RenderTemplateHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	id := r.PathValue("id")

	pubVer, err := a.repo.GetPublishedTemplateVersion(r.Context(), tenantID, id)
	if err != nil {
		http.Error(w, `{"error":{"code":"template_not_found","message":"Published template version not found"}}`, http.StatusNotFound)
		return
	}

	var req RenderTemplateRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	subj, err := template.Render(pubVer.Subject, req.Variables, template.Options{AutoEscape: false, StrictVariables: false})
	if err != nil {
		http.Error(w, `{"error":{"code":"render_error","message":"`+err.Error()+`"}}`, http.StatusBadRequest)
		return
	}

	htmlOut, err := template.Render(pubVer.HTML, req.Variables, template.Options{AutoEscape: true, StrictVariables: false})
	if err != nil {
		http.Error(w, `{"error":{"code":"render_error","message":"`+err.Error()+`"}}`, http.StatusBadRequest)
		return
	}

	textOut, err := template.Render(pubVer.Text, req.Variables, template.Options{AutoEscape: false, StrictVariables: false})
	if err != nil {
		http.Error(w, `{"error":{"code":"render_error","message":"`+err.Error()+`"}}`, http.StatusBadRequest)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"subject": subj,
		"html":    htmlOut,
		"text":    textOut,
	})
}
