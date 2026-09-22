// Package server exposes HTTP endpoints: health, readiness, metrics and the
// GitHub + Telegram webhook receivers.
package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/control-center/github-telegram-control-center/internal/bot"
	"github.com/control-center/github-telegram-control-center/internal/config"
	"github.com/control-center/github-telegram-control-center/internal/database"
	"github.com/control-center/github-telegram-control-center/internal/events"
	"github.com/control-center/github-telegram-control-center/internal/github"
	"github.com/control-center/github-telegram-control-center/internal/logging"
	"github.com/control-center/github-telegram-control-center/internal/metrics"
	"github.com/control-center/github-telegram-control-center/internal/telegram"
)

const (
	maxBodyBytes = 10 << 20 // 10 MiB
	version      = "1.0.0"
)

var startTime = time.Now()

type Server struct {
	cfg      *config.Config
	store    *database.Store
	tg       *telegram.Client
	bot      *bot.Bot
	dispatch *events.Dispatcher
	log      *slog.Logger
	http     *http.Server
}

func New(cfg *config.Config, store *database.Store, tg *telegram.Client, b *bot.Bot,
	disp *events.Dispatcher, log *slog.Logger) *Server {
	return &Server{cfg: cfg, store: store, tg: tg, bot: b, dispatch: disp, log: log}
}

func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/ready", s.handleReady)
	mux.HandleFunc("/metrics", metrics.Handler().ServeHTTP)
	mux.HandleFunc("/webhooks/github", s.handleGitHubWebhook)
	mux.HandleFunc("/webhooks/telegram", s.handleTelegramWebhook)

	srv := &http.Server{
		Addr:              ":" + strconv.Itoa(s.cfg.Port),
		Handler:           logging.Middleware(s.log, recoverer(mux)),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	s.http = srv

	if s.cfg.TelegramUpdateMode == "webhook" && s.cfg.WebhookBaseURL != "" {
		hookURL := s.cfg.WebhookBaseURL + "/webhooks/telegram"
		if err := s.tg.SetWebhook(ctx, hookURL, s.cfg.TelegramWebhookSecret); err != nil {
			return errors.New("set telegram webhook: " + err.Error())
		}
		s.log.Info("telegram webhook registered", "url", hookURL)
	} else {
		s.log.Info("telegram running in polling mode")
	}

	s.log.Info("http server starting", "port", s.cfg.Port)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.log.Error("http server error", "error", err)
		}
	}()
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.http == nil {
		return nil
	}
	return s.http.Shutdown(ctx)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"version":  version,
		"env":      s.cfg.Environment,
		"uptime_s": int(time.Since(startTime).Seconds()),
	})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	dbOK := s.store.Ping(ctx) == nil
	status := http.StatusOK
	if !dbOK {
		status = http.StatusServiceUnavailable
	}
	writeJSON(w, status, map[string]any{
		"ready": dbOK,
		"checks": map[string]bool{
			"database": dbOK,
		},
	})
}

func (s *Server) handleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := readBody(r)
	if err != nil {
		s.log.Warn("github webhook body too large", "error", err)
		http.Error(w, "body too large", http.StatusRequestEntityTooLarge)
		return
	}
	req, err := github.ParseWebhookRequest(r, s.cfg.GitHubWebhookSecret, body)
	if err != nil {
		s.log.Warn("github webhook rejected", "error", err, "event", r.Header.Get("X-GitHub-Event"))
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}
	s.log.Debug("github webhook received", "delivery", req.DeliveryID, "event", req.EventType)

	s.dispatch.Dispatch(events.Job{
		DeliveryID: req.DeliveryID,
		EventType:  req.EventType,
		Body:       req.Body,
		Repository: repoFromPayload(body),
	})
	writeJSON(w, http.StatusAccepted, map[string]any{"status": "accepted"})
}

func (s *Server) handleTelegramWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.cfg.TelegramUpdateMode != "webhook" {
		http.Error(w, "webhook mode disabled", http.StatusNotFound)
		return
	}
	body, err := readBody(r)
	if err != nil {
		http.Error(w, "body too large", http.StatusRequestEntityTooLarge)
		return
	}
	if s.cfg.TelegramWebhookSecret != "" &&
		r.Header.Get("X-Telegram-Bot-Api-Secret-Token") != s.cfg.TelegramWebhookSecret {
		s.log.Warn("telegram webhook unauthorized")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var upd telegram.Update
	if err := json.Unmarshal(body, &upd); err != nil {
		s.log.Error("cannot decode telegram update", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if upd.UpdateID == 0 && upd.Message == nil && upd.CallbackQuery == nil {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ignored"})
		return
	}
	go s.bot.HandleUpdate(r.Context(), &upd)
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

// repoFromPayload extracts "owner/repo" from a webhook payload for logging and
// queue routing; the listener re-parses authoritative fields itself.
func repoFromPayload(body []byte) string {
	var det struct {
		Repo struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	}
	if json.Unmarshal(body, &det) == nil {
		return det.Repo.FullName
	}
	return ""
}

func readBody(r *http.Request) ([]byte, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, maxBodyBytes)
	var buf bytes.Buffer
	_, err := io.Copy(&buf, r.Body)
	return buf.Bytes(), err
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				metrics.TelegramErrorsTotal.Inc()
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
