package metrics

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Counter struct {
	v atomic.Int64
}

func (c *Counter) Inc()         { c.v.Add(1) }
func (c *Counter) Add(n int64)  { c.v.Add(n) }
func (c *Counter) Value() int64 { return c.v.Load() }

type Gauge struct {
	v atomic.Int64
}

func (g *Gauge) Set(n int64)  { g.v.Store(n) }
func (g *Gauge) Value() int64 { return g.v.Load() }

type Histogram struct {
	mu      sync.Mutex
	count   int64
	sum     time.Duration
	buckets map[float64]int64
}

var defaultBuckets = []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60}

func NewHistogram() *Histogram {
	b := make(map[float64]int64, len(defaultBuckets))
	for _, bb := range defaultBuckets {
		b[bb] = 0
	}
	return &Histogram{buckets: b}
}

func (h *Histogram) Observe(d time.Duration) {
	seconds := d.Seconds()
	h.mu.Lock()
	defer h.mu.Unlock()
	h.count++
	h.sum += d
	for bb := range h.buckets {
		if seconds <= bb {
			h.buckets[bb]++
		}
	}
}

func (h *Histogram) Count() int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.count
}

func (h *Histogram) Sum() time.Duration {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.sum
}

var (
	TelegramUpdatesTotal     = &Counter{}
	TelegramErrorsTotal      = &Counter{}
	GitHubAPIRequestsTotal   = &Counter{}
	GitHubAPIErrorsTotal     = &Counter{}
	GitHubRateLimitRemain    = &Gauge{}
	WebhooksReceivedTotal    = &Counter{}
	WebhooksProcessedTotal   = &Counter{}
	WebhooksFailedTotal      = &Counter{}
	NotificationsSentTotal   = &Counter{}
	NotificationsFailedTotal = &Counter{}
	WorkflowEventsTotal      = &Counter{}
	CommandDuration          = NewHistogram()
)

func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		lines := []string{}
		add := func(name, typ string, value float64, labels string) {
			if labels == "" {
				lines = append(lines, fmt.Sprintf("%s %s %g", name, typ, value))
			} else {
				lines = append(lines, fmt.Sprintf("%s{%s} %s %g", name, labels, typ, value))
			}
		}
		add("telegram_updates_total", "counter", float64(TelegramUpdatesTotal.Value()), "")
		add("telegram_errors_total", "counter", float64(TelegramErrorsTotal.Value()), "")
		add("github_api_requests_total", "counter", float64(GitHubAPIRequestsTotal.Value()), "")
		add("github_api_errors_total", "counter", float64(GitHubAPIErrorsTotal.Value()), "")
		add("github_rate_limit_remaining", "gauge", float64(GitHubRateLimitRemain.Value()), "")
		add("webhooks_received_total", "counter", float64(WebhooksReceivedTotal.Value()), "")
		add("webhooks_processed_total", "counter", float64(WebhooksProcessedTotal.Value()), "")
		add("webhooks_failed_total", "counter", float64(WebhooksFailedTotal.Value()), "")
		add("notifications_sent_total", "counter", float64(NotificationsSentTotal.Value()), "")
		add("notifications_failed_total", "counter", float64(NotificationsFailedTotal.Value()), "")
		add("workflow_events_total", "counter", float64(WorkflowEventsTotal.Value()), "")

		CommandDuration.mu.Lock()
		count, sum := CommandDuration.count, CommandDuration.sum
		buckets := make([]float64, 0, len(CommandDuration.buckets))
		for k := range CommandDuration.buckets {
			buckets = append(buckets, k)
		}
		sort.Float64s(buckets)
		upper := int64(0)
		for _, b := range buckets {
			upper = CommandDuration.buckets[b]
			lines = append(lines, fmt.Sprintf("command_execution_time_bucket{le=\"%g\"} %d", b, upper))
		}
		lines = append(lines, fmt.Sprintf("command_execution_time_bucket{le=\"+Inf\"} %d", count))
		lines = append(lines, fmt.Sprintf("command_execution_time_sum %g", sum.Seconds()))
		lines = append(lines, fmt.Sprintf("command_execution_time_count %d", count))
		CommandDuration.mu.Unlock()

		fmt.Fprint(w, strings.Join(lines, "\n")+"\n")
	})
}
