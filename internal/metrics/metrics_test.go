package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCounter(t *testing.T) {
	c := &Counter{}
	c.Inc()
	c.Inc()
	c.Add(3)
	if got := c.Value(); got != 5 {
		t.Errorf("Counter.Value = %d; want 5", got)
	}
}

func TestGauge(t *testing.T) {
	g := &Gauge{}
	g.Set(42)
	if got := g.Value(); got != 42 {
		t.Errorf("Gauge.Value = %d; want 42", got)
	}
	g.Set(7)
	if got := g.Value(); got != 7 {
		t.Errorf("Gauge.Value = %d; want 7", got)
	}
}

func TestHistogramObserve(t *testing.T) {
	h := NewHistogram()
	for _, d := range []time.Duration{10 * time.Millisecond, 100 * time.Millisecond, 2 * time.Second} {
		h.Observe(d)
	}
	if got := h.Count(); got != 3 {
		t.Errorf("Count = %d; want 3", got)
	}
	if sum := h.Sum(); sum <= 0 {
		t.Errorf("Sum = %v; want positive", sum)
	}
}

func TestHandlerExposesAllMetrics(t *testing.T) {
	TelegramUpdatesTotal.Inc()
	GitHubRateLimitRemain.Set(4999)
	WebhooksReceivedTotal.Add(3)
	CommandDuration.Observe(120 * time.Millisecond)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/plain") {
		t.Errorf("Content-Type = %q", ct)
	}
	body := rec.Body.String()
	want := []string{
		"telegram_updates_total",
		"telegram_errors_total",
		"github_api_requests_total",
		"github_api_errors_total",
		"github_rate_limit_remaining",
		"webhooks_received_total",
		"webhooks_processed_total",
		"webhooks_failed_total",
		"notifications_sent_total",
		"notifications_failed_total",
		"workflow_events_total",
		"command_execution_time",
	}
	for _, name := range want {
		if !strings.Contains(body, name) {
			t.Errorf("metrics output missing %q\n%s", name, body)
		}
	}
	if !strings.Contains(body, "4999") {
		t.Errorf("rate limit gauge value missing:\n%s", body)
	}
}
