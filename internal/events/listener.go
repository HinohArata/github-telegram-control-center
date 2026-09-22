package events

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/control-center/github-telegram-control-center/internal/database"
	"github.com/control-center/github-telegram-control-center/internal/github"
	"github.com/control-center/github-telegram-control-center/internal/metrics"
	"github.com/control-center/github-telegram-control-center/internal/notifications"
	"github.com/control-center/github-telegram-control-center/internal/telegram"
)

type Listener struct {
	store    *database.Store
	gh       *github.Service
	notifier *notifications.Notifier
	log      *slog.Logger
}

func NewListener(store *database.Store, gh *github.Service, notifier *notifications.Notifier, log *slog.Logger) *Listener {
	return &Listener{store: store, gh: gh, notifier: notifier, log: log}
}

func hashBody(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

var handlers = map[string]func(*Listener, context.Context, *Job) error{
	"push":           (*Listener).handlePush,
	"pull_request":   (*Listener).handlePullRequest,
	"issues":         (*Listener).handleIssues,
	"issue_comment":  (*Listener).handleIssueComment,
	"workflow_run":   (*Listener).handleWorkflowRun,
	"release":        (*Listener).handleRelease,
	"create":         (*Listener).handleCreate,
	"delete":         (*Listener).handleDelete,
	"repository":     (*Listener).handleRepository,
	"workflow_job":   (*Listener).handleWorkflowJob,
}

func (l *Listener) Process(ctx context.Context, job *Job) error {
	inserted, err := l.store.RecordDelivery(ctx, job.DeliveryID, job.EventType, job.Repository, hashBody(job.Body))
	if err != nil {
		return fmt.Errorf("record delivery: %w", err)
	}
	if !inserted {
		l.log.Debug("duplicate delivery, skipping", "delivery_id", job.DeliveryID)
		return nil
	}
	metrics.WebhooksReceivedTotal.Inc()
	handler, ok := handlers[job.EventType]
	if !ok {
		metrics.WebhooksProcessedTotal.Inc()
		return l.store.MarkDeliveryProcessed(ctx, job.DeliveryID)
	}
	if err := handler(l, ctx, job); err != nil {
		metrics.WebhooksFailedTotal.Inc()
		_ = l.store.MarkDeliveryFailed(ctx, job.DeliveryID, 0, err.Error())
		l.log.Error("handle webhook event failed",
			"delivery_id", job.DeliveryID, "event", job.EventType, "err", err)
		return err
	}
	metrics.WorkflowEventsTotal.Inc()
	return l.store.MarkDeliveryProcessed(ctx, job.DeliveryID)
}

func (l *Listener) sendToSubscribers(ctx context.Context, repo, eventType string, text string, markup *telegram.InlineKeyboardMarkup, branchMatch string) {
	targets, err := l.store.ListNotificationSubscribers(ctx, repo, eventType)
	if err != nil {
		l.log.Error("list notification subscribers failed", "repo", repo, "err", err)
		return
	}
	for _, t := range targets {
		if branchMatch != "" && t.BranchFilter != nil && *t.BranchFilter != "" {
			if !branchMatches(*t.BranchFilter, branchMatch) {
				continue
			}
		}
		if err := l.notifier.Send(ctx, t.TelegramUserID, text); err != nil {
			l.log.Error("notify subscriber failed", "telegram_user_id", t.TelegramUserID, "err", err)
		}
	}
}

func branchMatches(pattern, branch string) bool {
	if pattern == "*" || pattern == "**" {
		return true
	}
	branch = strings.TrimPrefix(branch, "refs/heads/")
	if strings.HasSuffix(pattern, "/**") {
		return strings.HasPrefix(branch, strings.TrimSuffix(pattern, "/**"))
	}
	if strings.Contains(pattern, "*") {
		return wildcardMatch(pattern, branch)
	}
	return pattern == branch
}

func wildcardMatch(pattern, s string) bool {
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return pattern == s
	}
	idx := 0
	for i, p := range parts {
		if p == "" {
			continue
		}
		if i == len(parts)-1 {
			return strings.HasSuffix(s, p)
		}
		pos := strings.Index(s[idx:], p)
		if pos < 0 {
			return false
		}
		idx += pos + len(p)
	}
	return true
}

func parseBody(body []byte, out any) error {
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("parse webhook body: %w", err)
	}
	return nil
}