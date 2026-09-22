package logging

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

type ctxKey struct{}

var secretKeys = []string{
	"token", "secret", "password", "authorization", "private_key",
	"bot_token", "installation_token", "access_token", "database_url",
}

func New(level, format string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: lvl,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			k := strings.ToLower(a.Key)
			for _, s := range secretKeys {
				if strings.Contains(k, s) {
					return slog.String(a.Key, "[REDACTED]")
				}
			}
			if a.Value.Kind() == slog.KindString {
				if looksLikeSecret(a.Value.String()) {
					return slog.String(a.Key, "[REDACTED]")
				}
			}
			return slog.Attr{Key: a.Key, Value: a.Value}
		},
	}

	var handler slog.Handler
	if strings.ToLower(format) == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}
	return slog.New(handler)
}

func looksLikeSecret(v string) bool {
	if strings.HasPrefix(v, "-----BEGIN") {
		return true
	}
	if len(v) > 40 && strings.Count(v, ".") == 2 && !strings.ContainsAny(v, " \n") {
		return true
	}
	return false
}

func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

type statusWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(p []byte) (int, error) {
	n, err := w.ResponseWriter.Write(p)
	w.bytes += n
	return n, err
}

// Middleware logs each HTTP request. Webhook bodies are never logged, so
// payloads (which may contain tokens or source) cannot leak.
func Middleware(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)

		log.Log(r.Context(), slog.LevelInfo, "http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"bytes", sw.bytes,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote", strings.Split(r.RemoteAddr, ":")[0],
			"user_agent", r.UserAgent(),
		)
	})
}
