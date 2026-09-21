package alerts

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/mailvex/mailvex/core/domain"
	"github.com/mailvex/mailvex/core/repository"
)

type mockHTTPClient struct {
	lastReq  *http.Request
	lastBody []byte
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	m.lastReq = req
	if req.Body != nil {
		m.lastBody, _ = io.ReadAll(req.Body)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader([]byte(`{"ok":true}`))),
	}, nil
}

func TestDispatcherChannels(t *testing.T) {
	mockClient := &mockHTTPClient{}
	dispatcher := NewDispatcher(mockClient)
	ctx := context.Background()

	payload := &AlertPayload{
		IncidentID:     "inc_1",
		RuleID:         "arule_1",
		RuleName:       "High Bounce Rate Alert",
		MetricType:     "bounce_rate",
		Severity:       "critical",
		Status:         "firing",
		TriggeredValue: 0.082,
		Threshold:      0.05,
		Summary:        "Bounce rate reached 8.2%",
		Timestamp:      time.Now().UTC(),
	}

	// 1. Slack Channel
	slackCh := &domain.NotificationChannel{
		ID:        "nchan_slack",
		Name:      "DevOps Slack",
		Type:      "slack",
		Config:    `{"url":"https://hooks.slack.com/services/test"}`,
		IsEnabled: true,
	}
	if err := dispatcher.Dispatch(ctx, slackCh, payload); err != nil {
		t.Fatalf("Slack dispatch failed: %v", err)
	}
	if mockClient.lastReq.URL.String() != "https://hooks.slack.com/services/test" {
		t.Errorf("unexpected Slack URL: %s", mockClient.lastReq.URL.String())
	}

	// 2. Generic Webhook with HMAC signature
	webhookCh := &domain.NotificationChannel{
		ID:        "nchan_wh",
		Name:      "Custom Webhook",
		Type:      "generic_webhook",
		Config:    `{"url":"https://api.example.com/alerts","secret":"topsecret"}`,
		IsEnabled: true,
	}
	if err := dispatcher.Dispatch(ctx, webhookCh, payload); err != nil {
		t.Fatalf("Generic webhook dispatch failed: %v", err)
	}
	if mockClient.lastReq.Header.Get("X-Mailvex-Signature") == "" {
		t.Errorf("expected X-Mailvex-Signature header on signed webhook")
	}

	// 3. Teams Channel
	teamsCh := &domain.NotificationChannel{
		ID:        "nchan_teams",
		Name:      "Teams Ops",
		Type:      "microsoft_teams",
		Config:    `{"url":"https://outlook.office.com/webhook/test"}`,
		IsEnabled: true,
	}
	if err := dispatcher.Dispatch(ctx, teamsCh, payload); err != nil {
		t.Fatalf("Teams dispatch failed: %v", err)
	}
}

func TestAlertRuleLifecycleAndAutoResolve(t *testing.T) {
	repo := repository.NewMemoryRepository()
	mockClient := &mockHTTPClient{}
	dispatcher := NewDispatcher(mockClient)
	engine := NewEngine(repo, dispatcher)
	ctx := context.Background()

	tenantID := "tenant_alerts_1"
	rule := &domain.AlertRule{
		ID:                 "arule_bounce",
		TenantID:           &tenantID,
		Name:               "High Bounce Rate",
		MetricType:         "bounce_rate",
		ComparisonOperator: ">",
		Threshold:          0.05,
		CooldownMinutes:    60,
		Severity:           "warning",
		IsEnabled:          true,
	}
	_ = repo.CreateAlertRule(ctx, rule)

	// Configure channel
	_ = repo.CreateNotificationChannel(ctx, &domain.NotificationChannel{
		ID:        "nchan_slack_1",
		TenantID:  &tenantID,
		Name:      "Slack Channel",
		Type:      "slack",
		Config:    `{"url":"https://hooks.slack.com/services/alert"}`,
		IsEnabled: true,
	})

	// 1. Evaluate with 0.02 (under threshold) -> no incident
	inc, err := engine.EvaluateRule(ctx, rule, 0.02)
	if err != nil || inc != nil {
		t.Fatalf("expected no incident under threshold, got %v", inc)
	}

	// 2. Evaluate with 0.07 (over threshold) -> fires incident
	incFired, err := engine.EvaluateRule(ctx, rule, 0.07)
	if err != nil || incFired == nil {
		t.Fatalf("expected incident to fire, got error: %v", err)
	}
	if incFired.Status != "firing" {
		t.Errorf("expected firing status, got %s", incFired.Status)
	}

	// Verify in-app notification was created
	notifs, _ := repo.ListAlertNotifications(ctx, &tenantID, true, 10)
	if len(notifs) != 1 {
		t.Errorf("expected 1 unread notification, got %d", len(notifs))
	}

	// 3. Evaluate with 0.08 immediately -> cooldown prevents duplicate dispatch
	mockClient.lastReq = nil
	incDup, _ := engine.EvaluateRule(ctx, rule, 0.08)
	if incDup == nil || incDup.ID != incFired.ID {
		t.Errorf("expected existing incident returned during cooldown")
	}

	// 4. Metric normalizes to 0.03 -> auto-resolves incident
	incResolved, err := engine.EvaluateRule(ctx, rule, 0.03)
	if err != nil {
		t.Fatalf("EvaluateRule error: %v", err)
	}
	if incResolved != nil {
		t.Errorf("expected nil incident returned on resolution")
	}

	// Check incident status in repo
	storedInc, _ := repo.GetAlertIncident(ctx, &tenantID, incFired.ID)
	if storedInc.Status != "resolved" {
		t.Errorf("expected incident to be marked resolved, got %s", storedInc.Status)
	}
}
