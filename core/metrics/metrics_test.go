package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsRegistryAndRendering(t *testing.T) {
	reg := NewMetricRegistry()

	cnt := reg.NewCounter("test_counter", "Test counter help")
	cnt.Inc(map[string]string{"env": "prod"})
	cnt.Add(map[string]string{"env": "prod"}, 4)
	if val := cnt.Get(map[string]string{"env": "prod"}); val != 5 {
		t.Errorf("expected 5, got %f", val)
	}

	gauge := reg.NewGauge("test_gauge", "Test gauge help")
	gauge.Set(map[string]string{"queue": "inbound"}, 42)
	gauge.Add(map[string]string{"queue": "inbound"}, 8)
	if val := gauge.Get(map[string]string{"queue": "inbound"}); val != 50 {
		t.Errorf("expected 50, got %f", val)
	}

	hist := reg.NewHistogram("test_hist", "Test histogram help", []float64{0.1, 0.5, 1.0, 5.0})
	hist.Observe(map[string]string{"endpoint": "/send"}, 0.25)
	hist.Observe(map[string]string{"endpoint": "/send"}, 0.75)
	hist.Observe(map[string]string{"endpoint": "/send"}, 2.0)

	text := reg.RenderPrometheusText()

	// Verify Prometheus text output
	if !strings.Contains(text, "# TYPE test_counter counter") {
		t.Error("missing counter type line")
	}
	if !strings.Contains(text, `test_counter{env="prod"} 5`) {
		t.Errorf("missing formatted counter line in:\n%s", text)
	}
	if !strings.Contains(text, `test_gauge{queue="inbound"} 50`) {
		t.Errorf("missing formatted gauge line in:\n%s", text)
	}
	if !strings.Contains(text, `test_hist_count{endpoint="/send"} 3`) {
		t.Errorf("missing histogram count in:\n%s", text)
	}
}

func TestMetricsHTTPHandler(t *testing.T) {
	MessagesQueued.Inc(map[string]string{"tenant": "ten_1"})
	QueueDepth.Set(map[string]string{"tenant": "ten_1"}, 12)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/plain") {
		t.Errorf("expected text/plain content type, got %s", ct)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "mailvex_messages_queued_total") {
		t.Error("expected mailvex_messages_queued_total in /metrics output")
	}
	if !strings.Contains(body, "mailvex_queue_depth") {
		t.Error("expected mailvex_queue_depth in /metrics output")
	}
}
