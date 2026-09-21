package analytics

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mailvex/mailvex/core/domain"
	"github.com/mailvex/mailvex/core/repository"
)

func TestAnalyticsEngineRollupsAndRates(t *testing.T) {
	repo := repository.NewMemoryRepository()
	engine := NewEngine(repo)
	ctx := context.Background()

	tenantID := "tenant_analytics_test"
	now := time.Now().UTC()

	// 1. Record 100 sent events
	for i := 0; i < 100; i++ {
		email := "user@gmail.com"
		if i%2 == 0 {
			email = "user@outlook.com"
		}
		_ = engine.RecordEvent(ctx, tenantID, "message.sent", nil, nil, email, "onboarding", 0, now)
	}

	// 2. Record 95 delivered events with varying latencies
	for i := 0; i < 95; i++ {
		email := "user@gmail.com"
		latency := 2.5 // under 5s
		if i >= 50 {
			email = "user@outlook.com"
			latency = 45.0 // under 1m
		}
		_ = engine.RecordEvent(ctx, tenantID, "message.delivered", nil, nil, email, "onboarding", latency, now)
	}

	// 3. Record 5 bounces (3 hard, 2 soft)
	_ = engine.RecordEvent(ctx, tenantID, "hard_bounce", nil, nil, "bad@gmail.com", "onboarding", 0, now)
	_ = engine.RecordEvent(ctx, tenantID, "hard_bounce", nil, nil, "bad2@gmail.com", "onboarding", 0, now)
	_ = engine.RecordEvent(ctx, tenantID, "hard_bounce", nil, nil, "bad3@outlook.com", "onboarding", 0, now)
	_ = engine.RecordEvent(ctx, tenantID, "soft_bounce", nil, nil, "full@outlook.com", "onboarding", 0, now)
	_ = engine.RecordEvent(ctx, tenantID, "soft_bounce", nil, nil, "full2@outlook.com", "onboarding", 0, now)

	// 4. Record 1 complaint
	_ = engine.RecordEvent(ctx, tenantID, "message.complained", nil, nil, "angry@gmail.com", "onboarding", 0, now)

	// Query Overview
	summary, err := engine.GetOverview(ctx, tenantID, domain.AnalyticsFilter{})
	if err != nil {
		t.Fatalf("unexpected error getting overview: %v", err)
	}

	if summary.Sent != 100 {
		t.Fatalf("expected 100 sent, got %d", summary.Sent)
	}
	if summary.Delivered != 95 {
		t.Fatalf("expected 95 delivered, got %d", summary.Delivered)
	}
	if summary.BouncedHard != 3 {
		t.Fatalf("expected 3 hard bounces, got %d", summary.BouncedHard)
	}
	if summary.BouncedSoft != 2 {
		t.Fatalf("expected 2 soft bounces, got %d", summary.BouncedSoft)
	}
	if summary.Complained != 1 {
		t.Fatalf("expected 1 complaint, got %d", summary.Complained)
	}

	// Verify Delivery Rate = 95 / 100 = 95.0%
	expectedDeliveryRate := 0.95
	if summary.DeliveryRate != expectedDeliveryRate {
		t.Fatalf("expected delivery rate %f, got %f", expectedDeliveryRate, summary.DeliveryRate)
	}

	// Verify Bounce Rate = 5 / 100 = 5.0%
	expectedBounceRate := 0.05
	if summary.BounceRate != expectedBounceRate {
		t.Fatalf("expected bounce rate %f, got %f", expectedBounceRate, summary.BounceRate)
	}

	// Verify Complaint Rate = 1 / 95 ~= 0.0105 (1.05%)
	if summary.ComplaintRate < 0.01 || summary.ComplaintRate > 0.011 {
		t.Fatalf("expected complaint rate ~0.0105, got %f", summary.ComplaintRate)
	}

	// Complaint warning should trigger (> 0.10%)
	if !summary.ComplaintWarning {
		t.Fatalf("expected ComplaintWarning to be true when complaint rate > 0.10%%")
	}

	// Query Latency Histogram
	hist, err := engine.GetLatencyHistogram(ctx, tenantID, domain.AnalyticsFilter{})
	if err != nil {
		t.Fatalf("unexpected error getting latency: %v", err)
	}

	if hist.Under5s != 50 {
		t.Fatalf("expected 50 events under 5s, got %d", hist.Under5s)
	}
	if hist.Under1m != 45 {
		t.Fatalf("expected 45 events under 1m, got %d", hist.Under1m)
	}

	// Query Provider Breakdown
	breakdown, err := engine.GetProviderBreakdown(ctx, tenantID, domain.AnalyticsFilter{})
	if err != nil {
		t.Fatalf("unexpected error getting provider breakdown: %v", err)
	}

	if len(breakdown) < 2 {
		t.Fatalf("expected at least 2 providers (google, microsoft), got %d", len(breakdown))
	}

	// Verify CSV Export Generation
	csvBytes, err := engine.GenerateCSVExport(ctx, tenantID, domain.AnalyticsFilter{})
	if err != nil {
		t.Fatalf("unexpected error generating CSV: %v", err)
	}

	csvStr := string(csvBytes)
	if !strings.Contains(csvStr, "timestamp,sent,delivered,bounced,complained") {
		t.Fatalf("expected CSV header in export, got: %s", csvStr)
	}
	if !strings.Contains(csvStr, "95.00%") {
		t.Fatalf("expected 95.00%% delivery rate in CSV output, got: %s", csvStr)
	}
}

func TestClassifyProvider(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"alice@gmail.com", "google"},
		{"bob@googlemail.com", "google"},
		{"carol@corp.google.com", "google"},
		{"dave@outlook.com", "microsoft"},
		{"eve@hotmail.com", "microsoft"},
		{"frank@live.com", "microsoft"},
		{"grace@company.onmicrosoft.com", "microsoft"},
		{"heidi@yahoo.com", "yahoo"},
		{"ivan@aol.com", "yahoo"},
		{"judy@icloud.com", "apple"},
		{"mallory@me.com", "apple"},
		{"peggy@unknown-domain.test", "other"},
		{"custom.org", "other"},
	}

	for _, tc := range tests {
		got := ClassifyProvider(tc.input)
		if got != tc.expected {
			t.Errorf("ClassifyProvider(%q) = %q, expected %q", tc.input, got, tc.expected)
		}
	}
}
