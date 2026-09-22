package events

import (
	"context"
	"log/slog"
	"sync"

	"github.com/control-center/github-telegram-control-center/internal/metrics"
)

type Job struct {
	DeliveryID string
	EventType  string
	Repository string
	Body       []byte
}

type Dispatcher struct {
	jobs     chan Job
	workers  int
	wg       sync.WaitGroup
	listener *Listener
	log      *slog.Logger
}

func NewDispatcher(l *Listener, workers int, log *slog.Logger) *Dispatcher {
	if workers <= 0 {
		workers = 4
	}
	return &Dispatcher{
		jobs:     make(chan Job, workers*16),
		workers:  workers,
		listener: l,
		log:      log,
	}
}

func (d *Dispatcher) Start() {
	for i := 0; i < d.workers; i++ {
		d.wg.Add(1)
		go func() {
			defer d.wg.Done()
			for job := range d.jobs {
				ctx := context.Background()
				if err := d.listener.Process(ctx, &job); err != nil {
					metrics.WebhooksFailedTotal.Inc()
					d.log.Error("webhook job failed",
						"delivery_id", job.DeliveryID, "event", job.EventType, "err", err)
				} else {
					metrics.WebhooksProcessedTotal.Inc()
				}
			}
		}()
	}
}

func (d *Dispatcher) Dispatch(job Job) {
	select {
	case d.jobs <- job:
	default:
		d.log.Warn("webhook queue full, dropping delivery", "delivery_id", job.DeliveryID)
		metrics.WebhooksFailedTotal.Inc()
	}
}

func (d *Dispatcher) Shutdown(ctx context.Context) {
	close(d.jobs)
	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		d.log.Warn("webhook worker shutdown timed out")
	}
}
