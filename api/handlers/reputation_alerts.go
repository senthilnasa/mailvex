package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/senthilnasa/mailvex/api/middleware"
	"github.com/senthilnasa/mailvex/core/alerts"
	"github.com/senthilnasa/mailvex/core/domain"
)

// GET /v1/reputation/overview
func (a *API) GetReputationOverviewHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"Tenant ID missing from context"}}`))
		return
	}

	overview, err := a.repo.GetReputationOverview(r.Context(), tenantID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"internal_error","message":"%s"}}`, err.Error())))
		return
	}

	_ = json.NewEncoder(w).Encode(overview)
}

// GET /v1/dnsbl/providers
func (a *API) ListDNSBLProvidersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	providers, err := a.repo.ListDNSBLProviders(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"internal_error","message":"%s"}}`, err.Error())))
		return
	}
	if providers == nil {
		providers = []*domain.DNSBLProvider{}
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"data": providers})
}

type CheckDNSBLRequest struct {
	IPAddressID string `json:"ip_address_id"`
	IP          string `json:"ip"`
}

// POST /v1/dnsbl/check
func (a *API) CheckDNSBLHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req CheckDNSBLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"invalid_json","message":"Invalid request body"}}`))
		return
	}

	req.IP = strings.TrimSpace(req.IP)
	if req.IP == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"missing_ip","message":"IP address is required"}}`))
		return
	}
	if req.IPAddressID == "" {
		req.IPAddressID = "ip_" + req.IP
	}

	results, err := a.dnsbl.CheckIPAllProviders(r.Context(), req.IPAddressID, req.IP)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"check_failed","message":"%s"}}`, err.Error())))
		return
	}
	if results == nil {
		results = []*domain.DNSBLResult{}
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"data": results})
}

// GET /v1/dnsbl/results
func (a *API) ListDNSBLResultsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ipAddressID := r.URL.Query().Get("ip_address_id")
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}

	results, err := a.repo.ListDNSBLResults(r.Context(), ipAddressID, limit)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"internal_error","message":"%s"}}`, err.Error())))
		return
	}
	if results == nil {
		results = []*domain.DNSBLResult{}
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"data": results})
}

// GET /v1/warmup/plans
func (a *API) ListWarmupPlansHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	plans, err := a.repo.ListWarmupPlans(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"internal_error","message":"%s"}}`, err.Error())))
		return
	}
	if plans == nil {
		plans = []*domain.WarmupPlan{}
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"data": plans})
}

// GET /v1/warmup/plans/{id}
func (a *API) GetWarmupPlanHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id := r.PathValue("id")
	plan, err := a.repo.GetWarmupPlan(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"not_found","message":"Warmup plan not found"}}`))
		return
	}
	_ = json.NewEncoder(w).Encode(plan)
}

// GET /v1/warmup/ips
func (a *API) ListIPWarmupStatesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	states, err := a.repo.ListIPWarmupStates(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"internal_error","message":"%s"}}`, err.Error())))
		return
	}
	if states == nil {
		states = []*domain.IPWarmupState{}
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"data": states})
}

type EnrollIPWarmupRequest struct {
	IPAddressID string `json:"ip_address_id"`
	IP          string `json:"ip"`
	PlanID      string `json:"plan_id"`
}

// POST /v1/warmup/ips
func (a *API) EnrollIPWarmupHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req EnrollIPWarmupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"invalid_json","message":"Invalid request body"}}`))
		return
	}

	if req.IPAddressID == "" || req.IP == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"missing_fields","message":"ip_address_id and ip are required"}}`))
		return
	}
	if req.PlanID == "" {
		req.PlanID = "wplan_default"
	}

	state, err := a.warmup.EnrollIP(r.Context(), req.IPAddressID, req.IP, req.PlanID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"enroll_failed","message":"%s"}}`, err.Error())))
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(state)
}

type AdvanceIPWarmupRequest struct {
	Force bool `json:"force"`
}

// POST /v1/warmup/ips/{id}/advance
func (a *API) AdvanceIPWarmupHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id := r.PathValue("id")
	var req AdvanceIPWarmupRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	bounceRate := 0.01
	complaintRate := 0.000
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID != "" {
		if overview, err := a.repo.GetReputationOverview(r.Context(), tenantID); err == nil {
			bounceRate = overview.BounceRate
			complaintRate = overview.ComplaintRate
		}
	}

	if err := a.warmup.AdvanceDay(r.Context(), id, bounceRate, complaintRate, req.Force); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"advance_failed","message":"%s"}}`, err.Error())))
		return
	}

	state, _ := a.repo.GetIPWarmupState(r.Context(), id)
	_ = json.NewEncoder(w).Encode(state)
}

type HoldIPWarmupRequest struct {
	Reason string `json:"reason"`
}

// POST /v1/warmup/ips/{id}/hold
func (a *API) HoldIPWarmupHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id := r.PathValue("id")
	var req HoldIPWarmupRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Reason == "" {
		req.Reason = "Manual administrative hold"
	}

	if err := a.warmup.HoldWarmup(r.Context(), id, req.Reason); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"hold_failed","message":"%s"}}`, err.Error())))
		return
	}

	state, _ := a.repo.GetIPWarmupState(r.Context(), id)
	_ = json.NewEncoder(w).Encode(state)
}

// POST /v1/warmup/ips/{id}/resume
func (a *API) ResumeIPWarmupHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id := r.PathValue("id")
	if err := a.warmup.ResumeWarmup(r.Context(), id); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"resume_failed","message":"%s"}}`, err.Error())))
		return
	}

	state, _ := a.repo.GetIPWarmupState(r.Context(), id)
	_ = json.NewEncoder(w).Encode(state)
}

// --- Alert Rules Handlers ---

type CreateAlertRuleRequest struct {
	Name               string   `json:"name"`
	MetricType         string   `json:"metric_type"`
	ComparisonOperator string   `json:"comparison_operator"`
	Threshold          float64  `json:"threshold"`
	DurationMinutes    int      `json:"duration_minutes"`
	CooldownMinutes    int      `json:"cooldown_minutes"`
	Severity           string   `json:"severity"`
	IsEnabled          *bool    `json:"is_enabled,omitempty"`
	Channels           []string `json:"channels,omitempty"`
}

// GET /v1/alerts/rules
func (a *API) ListAlertRulesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	var tid *string
	if tenantID != "" {
		tid = &tenantID
	}

	rules, err := a.repo.ListAlertRules(r.Context(), tid)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"internal_error","message":"%s"}}`, err.Error())))
		return
	}
	if rules == nil {
		rules = []*domain.AlertRule{}
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"data": rules})
}

// POST /v1/alerts/rules
func (a *API) CreateAlertRuleHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	var tid *string
	if tenantID != "" {
		tid = &tenantID
	}

	var req CreateAlertRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"invalid_json","message":"Invalid request body"}}`))
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || req.MetricType == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"missing_fields","message":"Name and metric_type are required"}}`))
		return
	}

	if req.ComparisonOperator == "" {
		req.ComparisonOperator = ">"
	}
	if req.Severity == "" {
		req.Severity = "warning"
	}
	if req.CooldownMinutes <= 0 {
		req.CooldownMinutes = 60
	}
	if req.DurationMinutes <= 0 {
		req.DurationMinutes = 5
	}

	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	rule := &domain.AlertRule{
		TenantID:           tid,
		Name:               req.Name,
		MetricType:         req.MetricType,
		ComparisonOperator: req.ComparisonOperator,
		Threshold:          req.Threshold,
		DurationMinutes:    req.DurationMinutes,
		CooldownMinutes:    req.CooldownMinutes,
		Severity:           req.Severity,
		IsEnabled:          isEnabled,
		Channels:           req.Channels,
	}

	if err := a.repo.CreateAlertRule(r.Context(), rule); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"create_failed","message":"%s"}}`, err.Error())))
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(rule)
}

// GET /v1/alerts/rules/{id}
func (a *API) GetAlertRuleHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id := r.PathValue("id")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	var tid *string
	if tenantID != "" {
		tid = &tenantID
	}

	rule, err := a.repo.GetAlertRule(r.Context(), tid, id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"not_found","message":"Alert rule not found"}}`))
		return
	}
	_ = json.NewEncoder(w).Encode(rule)
}

// DELETE /v1/alerts/rules/{id}
func (a *API) DeleteAlertRuleHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	var tid *string
	if tenantID != "" {
		tid = &tenantID
	}

	if err := a.repo.DeleteAlertRule(r.Context(), tid, id); err != nil {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"not_found","message":"Alert rule not found"}}`))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /v1/alerts/evaluate
func (a *API) EvaluateAlertsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	if tenantID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"Tenant ID missing from context"}}`))
		return
	}

	if err := a.alerts.EvaluateAllRules(r.Context(), tenantID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"eval_failed","message":"%s"}}`, err.Error())))
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"evaluated": true})
}

// GET /v1/alerts/incidents
func (a *API) ListAlertIncidentsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	var tid *string
	if tenantID != "" {
		tid = &tenantID
	}

	status := r.URL.Query().Get("status")
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}

	incidents, err := a.repo.ListAlertIncidents(r.Context(), tid, status, limit)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"internal_error","message":"%s"}}`, err.Error())))
		return
	}
	if incidents == nil {
		incidents = []*domain.AlertIncident{}
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"data": incidents})
}

// POST /v1/alerts/incidents/{id}/acknowledge
func (a *API) AcknowledgeAlertIncidentHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id := r.PathValue("id")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	var tid *string
	if tenantID != "" {
		tid = &tenantID
	}

	if err := a.repo.UpdateAlertIncidentStatus(r.Context(), tid, id, "acknowledged"); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"update_failed","message":"%s"}}`, err.Error())))
		return
	}

	inc, _ := a.repo.GetAlertIncident(r.Context(), tid, id)
	_ = json.NewEncoder(w).Encode(inc)
}

// POST /v1/alerts/incidents/{id}/resolve
func (a *API) ResolveAlertIncidentHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id := r.PathValue("id")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	var tid *string
	if tenantID != "" {
		tid = &tenantID
	}

	if err := a.repo.UpdateAlertIncidentStatus(r.Context(), tid, id, "resolved"); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"update_failed","message":"%s"}}`, err.Error())))
		return
	}

	inc, _ := a.repo.GetAlertIncident(r.Context(), tid, id)
	_ = json.NewEncoder(w).Encode(inc)
}

// --- Notification Channels Handlers ---

type CreateNotificationChannelRequest struct {
	Name      string `json:"name"`
	Type      string `json:"type"` // email, slack, generic_webhook, pagerduty, microsoft_teams
	Config    string `json:"config"`
	IsEnabled *bool  `json:"is_enabled,omitempty"`
}

// GET /v1/alerts/channels
func (a *API) ListNotificationChannelsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	var tid *string
	if tenantID != "" {
		tid = &tenantID
	}

	channels, err := a.repo.ListNotificationChannels(r.Context(), tid)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"internal_error","message":"%s"}}`, err.Error())))
		return
	}
	if channels == nil {
		channels = []*domain.NotificationChannel{}
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"data": channels})
}

// POST /v1/alerts/channels
func (a *API) CreateNotificationChannelHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	var tid *string
	if tenantID != "" {
		tid = &tenantID
	}

	var req CreateNotificationChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"invalid_json","message":"Invalid request body"}}`))
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || req.Type == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"missing_fields","message":"Name and type are required"}}`))
		return
	}

	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	channel := &domain.NotificationChannel{
		TenantID:  tid,
		Name:      req.Name,
		Type:      req.Type,
		Config:    req.Config,
		IsEnabled: isEnabled,
	}

	if err := a.repo.CreateNotificationChannel(r.Context(), channel); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"create_failed","message":"%s"}}`, err.Error())))
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(channel)
}

// DELETE /v1/alerts/channels/{id}
func (a *API) DeleteNotificationChannelHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	var tid *string
	if tenantID != "" {
		tid = &tenantID
	}

	if err := a.repo.DeleteNotificationChannel(r.Context(), tid, id); err != nil {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"not_found","message":"Notification channel not found"}}`))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /v1/alerts/channels/{id}/test
func (a *API) TestNotificationChannelHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id := r.PathValue("id")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	var tid *string
	if tenantID != "" {
		tid = &tenantID
	}

	channel, err := a.repo.GetNotificationChannel(r.Context(), tid, id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"not_found","message":"Notification channel not found"}}`))
		return
	}

	dispatcher := alerts.NewDispatcher(nil)
	testPayload := &alerts.AlertPayload{
		IncidentID:     "inc_test",
		RuleID:         "arule_test",
		RuleName:       "Test Notification Channel",
		TenantID:       tid,
		MetricType:     "test",
		Severity:       "info",
		Status:         "firing",
		TriggeredValue: 1.0,
		Threshold:      1.0,
		Summary:        fmt.Sprintf("This is a test notification from Mailvex to channel '%s'", channel.Name),
		Timestamp:      time.Now().UTC(),
	}

	if err := dispatcher.Dispatch(r.Context(), channel, testPayload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"dispatch_failed","message":"%s"}}`, err.Error())))
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": "Test notification dispatched successfully",
	})
}

// GET /v1/alerts/notifications
func (a *API) ListAlertNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	var tid *string
	if tenantID != "" {
		tid = &tenantID
	}

	unreadOnly := r.URL.Query().Get("unread") == "true"
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}

	notifs, err := a.repo.ListAlertNotifications(r.Context(), tid, unreadOnly, limit)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"internal_error","message":"%s"}}`, err.Error())))
		return
	}
	if notifs == nil {
		notifs = []*domain.AlertNotification{}
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"data": notifs})
}

// POST /v1/alerts/notifications/{id}/read
func (a *API) MarkAlertNotificationReadHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(string)
	var tid *string
	if tenantID != "" {
		tid = &tenantID
	}

	if err := a.repo.MarkAlertNotificationRead(r.Context(), tid, id); err != nil {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"not_found","message":"Notification not found"}}`))
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"success":true}`))
}
