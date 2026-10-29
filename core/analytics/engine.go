package analytics

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"github.com/senthilnasa/mailvex/core/domain"
	"github.com/senthilnasa/mailvex/core/repository"
)

type Engine struct {
	repo repository.Repository
}

func NewEngine(repo repository.Repository) *Engine {
	return &Engine{
		repo: repo,
	}
}

// ClassifyProvider maps an email or destination domain to a standard mailbox provider (§19.1)
func ClassifyProvider(emailOrDomain string) string {
	s := strings.ToLower(strings.TrimSpace(emailOrDomain))
	atIdx := strings.LastIndex(s, "@")
	dom := s
	if atIdx != -1 {
		dom = s[atIdx+1:]
	}

	switch {
	case dom == "gmail.com" || dom == "googlemail.com" || strings.HasSuffix(dom, ".google.com"):
		return "google"
	case dom == "outlook.com" || dom == "hotmail.com" || dom == "live.com" || dom == "msn.com" || strings.HasSuffix(dom, ".onmicrosoft.com") || strings.HasSuffix(dom, ".outlook.com"):
		return "microsoft"
	case dom == "yahoo.com" || dom == "ymail.com" || dom == "myyahoo.com" || dom == "aol.com" || strings.HasSuffix(dom, ".yahoodns.net"):
		return "yahoo"
	case dom == "icloud.com" || dom == "me.com" || dom == "mac.com":
		return "apple"
	default:
		return "other"
	}
}

// RecordEvent records an analytics event into pre-aggregated rollups (§19.1)
func (e *Engine) RecordEvent(ctx context.Context, tenantID, eventType string, streamID, domainID *string, recipientOrProvider, tag string, latencySec float64, occurredAt time.Time) error {
	provider := recipientOrProvider
	if strings.Contains(recipientOrProvider, "@") || strings.Contains(recipientOrProvider, ".") {
		provider = ClassifyProvider(recipientOrProvider)
	}
	return e.repo.RecordAnalyticsEvent(ctx, tenantID, eventType, streamID, domainID, provider, tag, latencySec, occurredAt)
}

func (e *Engine) GetOverview(ctx context.Context, tenantID string, filter domain.AnalyticsFilter) (*domain.AnalyticsSummary, error) {
	return e.repo.QueryAnalyticsOverview(ctx, tenantID, filter)
}

func (e *Engine) GetTimeSeries(ctx context.Context, tenantID string, interval string, filter domain.AnalyticsFilter) ([]*domain.TimeSeriesPoint, error) {
	return e.repo.QueryAnalyticsTimeSeries(ctx, tenantID, interval, filter)
}

func (e *Engine) GetProviderBreakdown(ctx context.Context, tenantID string, filter domain.AnalyticsFilter) ([]*domain.ProviderBreakdown, error) {
	return e.repo.QueryProviderBreakdown(ctx, tenantID, filter)
}

func (e *Engine) GetLatencyHistogram(ctx context.Context, tenantID string, filter domain.AnalyticsFilter) (*domain.LatencyHistogram, error) {
	return e.repo.QueryLatencyHistogram(ctx, tenantID, filter)
}

// GenerateCSVExport generates standard CSV analytics export (§19.3)
func (e *Engine) GenerateCSVExport(ctx context.Context, tenantID string, filter domain.AnalyticsFilter) ([]byte, error) {
	points, err := e.repo.QueryAnalyticsTimeSeries(ctx, tenantID, "1h", filter)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	// Write CSV Header
	header := []string{
		"timestamp",
		"sent",
		"delivered",
		"bounced",
		"complained",
		"delivery_rate_pct",
		"complaint_rate_pct",
	}
	if err := w.Write(header); err != nil {
		return nil, err
	}

	for _, p := range points {
		row := []string{
			p.Timestamp,
			fmt.Sprintf("%d", p.Sent),
			fmt.Sprintf("%d", p.Delivered),
			fmt.Sprintf("%d", p.Bounced),
			fmt.Sprintf("%d", p.Complained),
			fmt.Sprintf("%.2f%%", p.DeliveryRate*100),
			fmt.Sprintf("%.4f%%", p.ComplaintRate*100),
		}
		if err := w.Write(row); err != nil {
			return nil, err
		}
	}

	w.Flush()
	return buf.Bytes(), nil
}
