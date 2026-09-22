// Command bot is the entrypoint of the GitHub Telegram Control Center.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/control-center/github-telegram-control-center/internal/bot"
	"github.com/control-center/github-telegram-control-center/internal/config"
	"github.com/control-center/github-telegram-control-center/internal/database"
	"github.com/control-center/github-telegram-control-center/internal/events"
	"github.com/control-center/github-telegram-control-center/internal/github"
	"github.com/control-center/github-telegram-control-center/internal/logging"
	"github.com/control-center/github-telegram-control-center/internal/metrics"
	"github.com/control-center/github-telegram-control-center/internal/notifications"
	"github.com/control-center/github-telegram-control-center/internal/reconcile"
	"github.com/control-center/github-telegram-control-center/internal/server"
	"github.com/control-center/github-telegram-control-center/internal/telegram"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log := logging.New(cfg.LogLevel, cfg.LogFormat)
	log.Info("starting github telegram control center",
		"env", cfg.Environment,
		"port", cfg.Port,
		"update_mode", cfg.TelegramUpdateMode,
		"authorized_users", len(cfg.AuthorizedUsers),
		"webhook_base_url", cfg.WebhookBaseURL,
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()
	log.Info("database connected")

	store := database.New(pool)
	if err := database.Migrate(ctx, pool); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	log.Info("database migrations applied")

	auth := github.NewAuthManager(github.AppConfig{
		AppID:      cfg.GitHubAppID,
		PrivateKey: cfg.GitHubPrivateKey,
	})
	gh := github.NewService(auth)
	gh.SetRateLimitCallback(func(rl github.RateLimitInfo) {
		if rl.Resource == "core" {
			metrics.GitHubRateLimitRemain.Set(int64(rl.Remaining))
		}
	})

	tg := telegram.NewClient(cfg.TelegramBotToken)
	notifier := notifications.New(tg, store, log)

	listener := events.NewListener(store, gh, notifier, log)
	dispatch := events.NewDispatcher(listener, 8, log)
	dispatch.Start()

	b := bot.New(cfg, store, tg, notifier, gh, dispatch, log)

	srv := server.New(cfg, store, tg, b, dispatch, log)
	if err := srv.Start(ctx); err != nil {
		return err
	}

	worker := reconcile.New(store, gh, cfg.ReconciliationInterval, log)
	go worker.Run(ctx)

	if cfg.TelegramUpdateMode != "webhook" {
		go pollUpdates(ctx, b, tg, log)
	}

	<-ctx.Done()
	log.Info("shutdown signal received, draining...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dispatch.Shutdown(shutdownCtx)
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Warn("http server shutdown error", "error", err)
	}
	pool.Close()
	log.Info("shutdown complete")
	return nil
}

func pollUpdates(ctx context.Context, b *bot.Bot, tg *telegram.Client, log *slog.Logger) {
	log.Info("telegram long-polling started")
	var offset int64
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		upds, err := tg.GetUpdates(ctx, offset)
		if err != nil {
			if errors.Is(err, context.Canceled) || ctx.Err() != nil {
				return
			}
			log.Warn("get updates failed", "error", err)
			time.Sleep(2 * time.Second)
			continue
		}
		for _, u := range upds {
			if u.UpdateID >= offset {
				offset = u.UpdateID + 1
			}
			b.HandleUpdate(ctx, &u)
		}
	}
}

// metricsSet kept unused placeholder removed.
