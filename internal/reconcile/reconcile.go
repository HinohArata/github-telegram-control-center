// Package reconcile periodically synchronizes GitHub App installations into
// the database and performs housekeeping (expired sessions, stuck deliveries).
package reconcile

import (
	"context"
	"log/slog"
	"time"

	"github.com/control-center/github-telegram-control-center/internal/database"
	"github.com/control-center/github-telegram-control-center/internal/github"
)

type Worker struct {
	store    *database.Store
	gh       *github.Service
	interval time.Duration
	log      *slog.Logger
}

func New(store *database.Store, gh *github.Service, interval time.Duration, log *slog.Logger) *Worker {
	if interval <= 0 {
		interval = 10 * time.Minute
	}
	return &Worker{store: store, gh: gh, interval: interval, log: log}
}

func (w *Worker) Run(ctx context.Context) {
	w.log.Info("reconcile worker started", "interval", w.interval.String())
	if err := w.reconcileInstallations(ctx); err != nil {
		w.log.Warn("initial installation reconciliation failed", "error", err)
	}
	if err := w.housekeeping(ctx); err != nil {
		w.log.Warn("initial housekeeping failed", "error", err)
	}

	t := time.NewTicker(w.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			w.log.Info("reconcile worker stopped")
			return
		case <-t.C:
			if err := w.reconcileInstallations(ctx); err != nil {
				w.log.Warn("installation reconciliation failed", "error", err)
			}
			if err := w.housekeeping(ctx); err != nil {
				w.log.Warn("housekeeping failed", "error", err)
			}
		}
	}
}

// reconcileInstallations syncs the set of app installations known to GitHub
// into the database and removes installations that were uninstalled.
func (w *Worker) reconcileInstallations(ctx context.Context) error {
	installs, err := w.gh.ListInstallations(ctx)
	if err != nil {
		return err
	}
	live := make(map[int64]struct{}, len(installs))
	for _, in := range installs {
		if in.ID == 0 {
			continue
		}
		live[in.ID] = struct{}{}
		var accountID int64
		if in.Account != nil {
			accountID = in.Account.ID
		}
		login, acctType := "", ""
		if in.Account != nil {
			login = in.Account.Login
			acctType = in.Account.Type
		}
		if err := w.store.UpsertInstallation(ctx, in.ID, accountID, login, acctType, "", nil); err != nil {
			w.log.Warn("upsert installation failed", "installation_id", in.ID, "error", err)
			continue
		}
		w.log.Debug("installation synced", "installation_id", in.ID, "login", login)
	}

	known, err := w.store.ListInstallations(ctx)
	if err != nil {
		return err
	}
	for _, k := range known {
		if _, ok := live[k.InstallationID]; ok {
			continue
		}
		if err := w.store.DeleteInstallation(ctx, k.InstallationID); err != nil {
			w.log.Warn("delete stale installation failed", "installation_id", k.InstallationID, "error", err)
			continue
		}
		w.log.Info("removed uninstalled installation", "installation_id", k.InstallationID)
	}
	return nil
}

// housekeeping purges expired sessions so stale confirmation tokens cannot be
// replayed.
func (w *Worker) housekeeping(ctx context.Context) error {
	if err := w.store.DeleteExpiredSessions(ctx); err != nil {
		return err
	}
	return nil
}
