package metrics

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
)

// MetricType represents a Prometheus metric type
type MetricType string

const (
	TypeCounter   MetricType = "counter"
	TypeGauge     MetricType = "gauge"
	TypeHistogram MetricType = "histogram"
)

// MetricRegistry manages thread-safe Prometheus metrics
type MetricRegistry struct {
	mu         sync.RWMutex
	counters   map[string]*Counter
	gauges     map[string]*Gauge
	histograms map[string]*Histogram
}

// Global registry instance
var DefaultRegistry = NewMetricRegistry()

func NewMetricRegistry() *MetricRegistry {
	return &MetricRegistry{
		counters:   make(map[string]*Counter),
		gauges:     make(map[string]*Gauge),
		histograms: make(map[string]*Histogram),
	}
}

// Counter is a monotonically increasing cumulative metric
type Counter struct {
	mu     sync.Mutex
	help   string
	values map[string]float64
}

func (r *MetricRegistry) NewCounter(name, help string) *Counter {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, exists := r.counters[name]; exists {
		return c
	}
	c := &Counter{help: help, values: make(map[string]float64)}
	r.counters[name] = c
	return c
}

func (c *Counter) Inc(labels map[string]string) {
	c.Add(labels, 1)
}

func (c *Counter) Add(labels map[string]string, val float64) {
	if val < 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	k := formatLabels(labels)
	c.values[k] += val
}

func (c *Counter) Get(labels map[string]string) float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.values[formatLabels(labels)]
}

// Gauge represents a numerical value that can arbitrarily go up and down
type Gauge struct {
	mu     sync.Mutex
	help   string
	values map[string]float64
}

func (r *MetricRegistry) NewGauge(name, help string) *Gauge {
	r.mu.Lock()
	defer r.mu.Unlock()
	if g, exists := r.gauges[name]; exists {
		return g
	}
	g := &Gauge{help: help, values: make(map[string]float64)}
	r.gauges[name] = g
	return g
}

func (g *Gauge) Set(labels map[string]string, val float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	k := formatLabels(labels)
	g.values[k] = val
}

func (g *Gauge) Add(labels map[string]string, val float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	k := formatLabels(labels)
	g.values[k] += val
}

func (g *Gauge) Get(labels map[string]string) float64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.values[formatLabels(labels)]
}

// Histogram samples observations and counts them in configurable buckets
type Histogram struct {
	mu      sync.Mutex
	help    string
	buckets []float64
	counts  map[string]map[float64]uint64
	sums    map[string]float64
	totals  map[string]uint64
}

func (r *MetricRegistry) NewHistogram(name, help string, buckets []float64) *Histogram {
	r.mu.Lock()
	defer r.mu.Unlock()
	if h, exists := r.histograms[name]; exists {
		return h
	}
	if len(buckets) == 0 {
		buckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
	}
	sort.Float64s(buckets)
	h := &Histogram{
		help:    help,
		buckets: buckets,
		counts:  make(map[string]map[float64]uint64),
		sums:    make(map[string]float64),
		totals:  make(map[string]uint64),
	}
	r.histograms[name] = h
	return h
}

func (h *Histogram) Observe(labels map[string]string, val float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	k := formatLabels(labels)
	if h.counts[k] == nil {
		h.counts[k] = make(map[float64]uint64)
	}
	for _, b := range h.buckets {
		if val <= b {
			h.counts[k][b]++
		}
	}
	h.sums[k] += val
	h.totals[k]++
}

func formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, fmt.Sprintf(`%s="%s"`, k, labels[k]))
	}
	return strings.Join(pairs, ",")
}

// Predefined production metrics
var (
	MessagesQueued = DefaultRegistry.NewCounter(
		"mailvex_messages_queued_total",
		"Total count of messages enqueued for delivery",
	)
	MessagesDelivered = DefaultRegistry.NewCounter(
		"mailvex_messages_delivered_total",
		"Total count of successfully delivered messages",
	)
	MessagesBounced = DefaultRegistry.NewCounter(
		"mailvex_messages_bounced_total",
		"Total count of bounced messages",
	)
	MessagesFailed = DefaultRegistry.NewCounter(
		"mailvex_messages_failed_total",
		"Total count of permanently failed messages",
	)
	QueueDepth = DefaultRegistry.NewGauge(
		"mailvex_queue_depth",
		"Current number of pending messages waiting in the outbox queue",
	)
	BounceRate = DefaultRegistry.NewGauge(
		"mailvex_bounce_rate",
		"Rolling 7-day bounce rate fraction (0.0 to 1.0)",
	)
	ComplaintRate = DefaultRegistry.NewGauge(
		"mailvex_complaint_rate",
		"Rolling 7-day complaint rate fraction (0.0 to 1.0)",
	)
	DeliveryDuration = DefaultRegistry.NewHistogram(
		"mailvex_delivery_duration_seconds",
		"End-to-end delivery latency in seconds",
		[]float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60},
	)
	SMTPTransactionDuration = DefaultRegistry.NewHistogram(
		"mailvex_smtp_transaction_seconds",
		"Duration of outbound SMTP dialogue with remote MTA",
		[]float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	)
)

// RenderPrometheusText outputs metrics in standard Prometheus exposition format
func (r *MetricRegistry) RenderPrometheusText() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var sb strings.Builder

	// Write Counters
	counterNames := make([]string, 0, len(r.counters))
	for k := range r.counters {
		counterNames = append(counterNames, k)
	}
	sort.Strings(counterNames)

	for _, name := range counterNames {
		c := r.counters[name]
		c.mu.Lock()
		sb.WriteString(fmt.Sprintf("# HELP %s %s\n", name, c.help))
		sb.WriteString(fmt.Sprintf("# TYPE %s counter\n", name))
		for l, val := range c.values {
			if l != "" {
				sb.WriteString(fmt.Sprintf("%s{%s} %g\n", name, l, val))
			} else {
				sb.WriteString(fmt.Sprintf("%s %g\n", name, val))
			}
		}
		c.mu.Unlock()
	}

	// Write Gauges
	gaugeNames := make([]string, 0, len(r.gauges))
	for k := range r.gauges {
		gaugeNames = append(gaugeNames, k)
	}
	sort.Strings(gaugeNames)

	for _, name := range gaugeNames {
		g := r.gauges[name]
		g.mu.Lock()
		sb.WriteString(fmt.Sprintf("# HELP %s %s\n", name, g.help))
		sb.WriteString(fmt.Sprintf("# TYPE %s gauge\n", name))
		for l, val := range g.values {
			if l != "" {
				sb.WriteString(fmt.Sprintf("%s{%s} %g\n", name, l, val))
			} else {
				sb.WriteString(fmt.Sprintf("%s %g\n", name, val))
			}
		}
		g.mu.Unlock()
	}

	// Write Histograms
	histogramNames := make([]string, 0, len(r.histograms))
	for k := range r.histograms {
		histogramNames = append(histogramNames, k)
	}
	sort.Strings(histogramNames)

	for _, name := range histogramNames {
		h := r.histograms[name]
		h.mu.Lock()
		sb.WriteString(fmt.Sprintf("# HELP %s %s\n", name, h.help))
		sb.WriteString(fmt.Sprintf("# TYPE %s histogram\n", name))
		for l := range h.totals {
			for _, b := range h.buckets {
				var lbl string
				if l != "" {
					lbl = fmt.Sprintf(`%s,le="%g"`, l, b)
				} else {
					lbl = fmt.Sprintf(`le="%g"`, b)
				}
				sb.WriteString(fmt.Sprintf("%s_bucket{%s} %d\n", name, lbl, h.counts[l][b]))
			}
			// +Inf bucket
			var infLbl string
			if l != "" {
				infLbl = fmt.Sprintf(`%s,le="+Inf"`, l)
			} else {
				infLbl = `le="+Inf"`
			}
			sb.WriteString(fmt.Sprintf("%s_bucket{%s} %d\n", name, infLbl, h.totals[l]))
			if l != "" {
				sb.WriteString(fmt.Sprintf("%s_sum{%s} %g\n", name, l, h.sums[l]))
				sb.WriteString(fmt.Sprintf("%s_count{%s} %d\n", name, l, h.totals[l]))
			} else {
				sb.WriteString(fmt.Sprintf("%s_sum %g\n", name, h.sums[l]))
				sb.WriteString(fmt.Sprintf("%s_count %d\n", name, h.totals[l]))
			}
		}
		h.mu.Unlock()
	}

	return sb.String()
}

// Handler returns an http.Handler serving metrics at /metrics
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(DefaultRegistry.RenderPrometheusText()))
	})
}
