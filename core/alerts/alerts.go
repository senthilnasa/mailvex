package alerts

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/senthilnasa/mailvex/core/crypto"
	"github.com/senthilnasa/mailvex/core/domain"
	"github.com/senthilnasa/mailvex/core/repository"
)

// HTTPClient interface for mockable external dispatch
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Dispatcher struct {
	client HTTPClient
}

func NewDispatcher(client HTTPClient) *Dispatcher {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &Dispatcher{client: client}
}

type AlertPayload struct {
	IncidentID     string    `json:"incident_id"`
	RuleID         string    `json:"rule_id"`
	RuleName       string    `json:"rule_name"`
	TenantID       *string   `json:"tenant_id,omitempty"`
	MetricType     string    `json:"metric_type"`
	Severity       string    `json:"severity"`
	Status         string    `json:"status"` // firing, resolved
	TriggeredValue float64   `json:"triggered_value"`
	Threshold      float64   `json:"threshold"`
	Summary        string    `json:"summary"`
	Timestamp      time.Time `json:"timestamp"`
}

func (d *Dispatcher) Dispatch(ctx context.Context, channel *domain.NotificationChannel, payload *AlertPayload) error {
	if !channel.IsEnabled {
		return nil
	}

	var configMap map[string]string
	if channel.Config != "" {
		_ = json.Unmarshal([]byte(channel.Config), &configMap)
	}
	if configMap == nil {
		configMap = make(map[string]string)
	}

	targetURL := configMap["url"]

	switch channel.Type {
	case "slack":
		if targetURL == "" {
			return fmt.Errorf("missing slack webhook url")
		}
		color := "#e11d48" // red for firing
		if payload.Status == "resolved" {
			color = "#10b981" // green
		}
		slackMsg := map[string]any{
			"text": fmt.Sprintf("[%s] %s: %s", strings.ToUpper(payload.Severity), payload.RuleName, payload.Summary),
			"attachments": []map[string]any{
				{
					"color": color,
					"fields": []map[string]any{
						{"title": "Metric", "value": payload.MetricType, "short": true},
						{"title": "Current Value", "value": fmt.Sprintf("%.4f", payload.TriggeredValue), "short": true},
						{"title": "Threshold", "value": fmt.Sprintf("%.4f", payload.Threshold), "short": true},
						{"title": "Status", "value": strings.ToUpper(payload.Status), "short": true},
					},
				},
			},
		}
		body, _ := json.Marshal(slackMsg)
		req, _ := http.NewRequestWithContext(ctx, "POST", targetURL, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := d.client.Do(req)
		if err != nil {
			return err
		}
		_ = resp.Body.Close()

	case "microsoft_teams":
		if targetURL == "" {
			return fmt.Errorf("missing teams webhook url")
		}
		teamsCard := map[string]any{
			"@type":      "MessageCard",
			"@context":   "http://schema.org/extensions",
			"summary":    payload.Summary,
			"themeColor": "E81123",
			"title":      fmt.Sprintf("[%s] %s", strings.ToUpper(payload.Severity), payload.RuleName),
			"text":       payload.Summary,
		}
		if payload.Status == "resolved" {
			teamsCard["themeColor"] = "107C41"
		}
		body, _ := json.Marshal(teamsCard)
		req, _ := http.NewRequestWithContext(ctx, "POST", targetURL, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := d.client.Do(req)
		if err != nil {
			return err
		}
		_ = resp.Body.Close()

	case "pagerduty":
		routingKey := configMap["routing_key"]
		pdEventAction := "trigger"
		if payload.Status == "resolved" {
			pdEventAction = "resolve"
		}
		pdPayload := map[string]any{
			"routing_key":  routingKey,
			"event_action": pdEventAction,
			"dedup_key":    payload.RuleID,
			"payload": map[string]any{
				"summary":  payload.Summary,
				"severity": payload.Severity,
				"source":   "mailvex-alert-engine",
			},
		}
		pdURL := targetURL
		if pdURL == "" {
			pdURL = "https://events.pagerduty.com/v2/enqueue"
		}
		body, _ := json.Marshal(pdPayload)
		req, _ := http.NewRequestWithContext(ctx, "POST", pdURL, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := d.client.Do(req)
		if err != nil {
			return err
		}
		_ = resp.Body.Close()

	case "generic_webhook":
		if targetURL == "" {
			return fmt.Errorf("missing webhook url")
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequestWithContext(ctx, "POST", targetURL, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		secret := configMap["secret"]
		if secret != "" {
			ts := fmt.Sprintf("%d", time.Now().Unix())
			mac := hmac.New(sha256.New, []byte(secret))
			mac.Write([]byte(ts))
			mac.Write([]byte("."))
			mac.Write(body)
			sig := hex.EncodeToString(mac.Sum(nil))
			req.Header.Set("X-Mailvex-Timestamp", ts)
			req.Header.Set("X-Mailvex-Signature", fmt.Sprintf("t=%s,v1=%s", ts, sig))
		}
		resp, err := d.client.Do(req)
		if err != nil {
			return err
		}
		_ = resp.Body.Close()

	case "email":
		// Simulated email delivery or recorded in system logs
		return nil
	}

	return nil
}

// Engine evaluates alert rules and dispatches incident notifications
type Engine struct {
	mu         sync.RWMutex
	repo       repository.Repository
	dispatcher *Dispatcher
}

func NewEngine(repo repository.Repository, dispatcher *Dispatcher) *Engine {
	if dispatcher == nil {
		dispatcher = NewDispatcher(nil)
	}
	return &Engine{
		repo:       repo,
		dispatcher: dispatcher,
	}
}

// EvaluateRule evaluates a single alert rule against current metrics
func (e *Engine) EvaluateRule(ctx context.Context, rule *domain.AlertRule, metricValue float64) (*domain.AlertIncident, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !rule.IsEnabled {
		return nil, nil
	}

	triggered := false
	switch rule.ComparisonOperator {
	case ">":
		triggered = metricValue > rule.Threshold
	case ">=":
		triggered = metricValue >= rule.Threshold
	case "<":
		triggered = metricValue < rule.Threshold
	case "<=":
		triggered = metricValue <= rule.Threshold
	case "==":
		triggered = metricValue == rule.Threshold
	default:
		triggered = metricValue > rule.Threshold
	}

	// Fetch existing incidents for this rule
	incidents, err := e.repo.ListAlertIncidents(ctx, rule.TenantID, "firing", 10)
	if err != nil {
		return nil, err
	}

	var activeIncident *domain.AlertIncident
	for _, inc := range incidents {
		if inc.RuleID == rule.ID {
			activeIncident = inc
			break
		}
	}

	now := time.Now().UTC()

	if triggered {
		// If already firing, check cooldown
		if activeIncident != nil {
			cooldownDuration := time.Duration(rule.CooldownMinutes) * time.Minute
			if cooldownDuration <= 0 {
				cooldownDuration = 60 * time.Minute
			}
			if now.Sub(activeIncident.FiredAt) < cooldownDuration {
				// Cooldown active, don't spam
				return activeIncident, nil
			}
		}

		// Fire new incident
		summary := fmt.Sprintf("Alert [%s]: %s is %.4f (threshold %s %.4f)", rule.Name, rule.MetricType, metricValue, rule.ComparisonOperator, rule.Threshold)
		incident := &domain.AlertIncident{
			ID:             crypto.NewULID(crypto.PrefixAlertIncident),
			RuleID:         rule.ID,
			TenantID:       rule.TenantID,
			MetricType:     rule.MetricType,
			Severity:       rule.Severity,
			Status:         "firing",
			TriggeredValue: metricValue,
			Threshold:      rule.Threshold,
			Summary:        summary,
			FiredAt:        now,
		}

		if err := e.repo.RecordAlertIncident(ctx, incident); err != nil {
			return nil, err
		}

		// Create in-app notification
		_ = e.repo.CreateAlertNotification(ctx, &domain.AlertNotification{
			ID:         crypto.NewULID(crypto.PrefixAlertNotif),
			TenantID:   rule.TenantID,
			IncidentID: incident.ID,
			Title:      fmt.Sprintf("[%s] %s", strings.ToUpper(rule.Severity), rule.Name),
			Message:    summary,
			Severity:   rule.Severity,
			IsRead:     false,
			CreatedAt:  now,
		})

		// Dispatch to notification channels
		e.dispatchToChannels(ctx, rule, incident, "firing")

		return incident, nil
	}

	// Not triggered: check if we should auto-resolve an active firing incident
	if activeIncident != nil {
		_ = e.repo.UpdateAlertIncidentStatus(ctx, rule.TenantID, activeIncident.ID, "resolved")
		activeIncident.Status = "resolved"
		activeIncident.ResolvedAt = &now

		resolveSummary := fmt.Sprintf("RESOLVED [%s]: %s returned to normal (value %.4f, threshold %s %.4f)", rule.Name, rule.MetricType, metricValue, rule.ComparisonOperator, rule.Threshold)
		_ = e.repo.CreateAlertNotification(ctx, &domain.AlertNotification{
			ID:         crypto.NewULID(crypto.PrefixAlertNotif),
			TenantID:   rule.TenantID,
			IncidentID: activeIncident.ID,
			Title:      fmt.Sprintf("[RESOLVED] %s", rule.Name),
			Message:    resolveSummary,
			Severity:   "info",
			IsRead:     false,
			CreatedAt:  now,
		})

		e.dispatchToChannels(ctx, rule, activeIncident, "resolved")
	}

	return nil, nil
}

func (e *Engine) dispatchToChannels(ctx context.Context, rule *domain.AlertRule, incident *domain.AlertIncident, status string) {
	payload := &AlertPayload{
		IncidentID:     incident.ID,
		RuleID:         rule.ID,
		RuleName:       rule.Name,
		TenantID:       rule.TenantID,
		MetricType:     rule.MetricType,
		Severity:       rule.Severity,
		Status:         status,
		TriggeredValue: incident.TriggeredValue,
		Threshold:      incident.Threshold,
		Summary:        incident.Summary,
		Timestamp:      time.Now().UTC(),
	}

	// Find channels configured for this tenant/platform
	channels, err := e.repo.ListNotificationChannels(ctx, rule.TenantID)
	if err != nil {
		return
	}

	for _, ch := range channels {
		// If rule defines specific channels, filter
		if len(rule.Channels) > 0 {
			matched := false
			for _, chID := range rule.Channels {
				if chID == ch.ID {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		_ = e.dispatcher.Dispatch(ctx, ch, payload)
	}
}

// EvaluateAllRules evaluates all alert rules for a tenant using their current reputation metrics
func (e *Engine) EvaluateAllRules(ctx context.Context, tenantID string) error {
	overview, err := e.repo.GetReputationOverview(ctx, tenantID)
	if err != nil {
		return err
	}

	tid := tenantID
	rules, err := e.repo.ListAlertRules(ctx, &tid)
	if err != nil {
		return err
	}

	for _, rule := range rules {
		var val float64
		switch rule.MetricType {
		case "bounce_rate":
			val = overview.BounceRate
		case "complaint_rate":
			val = overview.ComplaintRate
		case "dnsbl_listed":
			val = float64(overview.DNSBLListingsCount)
		case "warmup_held":
			val = 0
		case "delivery_rate":
			val = overview.DeliveryRate
		default:
			continue
		}

		_, _ = e.EvaluateRule(ctx, rule, val)
	}

	return nil
}
