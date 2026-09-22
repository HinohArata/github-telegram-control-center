package logging

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func capture(t *testing.T, level string) (*bytes.Buffer, *slog.Logger) {
	t.Helper()
	var buf bytes.Buffer
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
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
	return &buf, slog.New(slog.NewTextHandler(&buf, opts))
}

func logString(buf *bytes.Buffer) string { return buf.String() }

func TestRedactsNamedSecrets(t *testing.T) {
	buf, log := capture(t, "debug")
	log.Info("test", "token", "abc123", "authorization", "Bearer xyz", "database_url", "postgres://u:p@h/db")

	out := logString(buf)
	for _, secret := range []string{"abc123", "Bearer xyz", "postgres://u:p@h/db"} {
		if strings.Contains(out, secret) {
			t.Errorf("log leaked secret %q:\n%s", secret, out)
		}
	}
	if c := strings.Count(out, "[REDACTED]"); c < 3 {
		t.Errorf("expected 3 redactions, got %d:\n%s", c, out)
	}
}

func TestRedactsPEMAndJWT(t *testing.T) {
	buf, log := capture(t, "debug")
	log.Info("test",
		"cert", "-----BEGIN PRIVATE KEY-----\nMIIE\n-----END PRIVATE KEY-----\n",
		"jwt", "eyJhbGciOiJSUzI1NiJ9.eyJpc3MiOiIxMjMifQ.signature",
	)
	out := logString(buf)
	if strings.Contains(out, "BEGIN") || strings.Contains(out, "eyJhbGci") || strings.Contains(out, "signature") {
		t.Errorf("log leaked PEM/JWT material:\n%s", out)
	}
}

func TestKeepsBenignValues(t *testing.T) {
	buf, log := capture(t, "debug")
	log.Info("test", "repo", "owner/name", "event", "push", "installation_id", 42)
	out := logString(buf)
	for _, want := range []string{"owner/name", "push", "42"} {
		if !strings.Contains(out, want) {
			t.Errorf("log lost benign value %q:\n%s", want, out)
		}
	}
}

func TestWithAndFromContext(t *testing.T) {
	_, log := capture(t, "debug")
	ctx := WithLogger(context.Background(), log)
	if got := FromContext(ctx); got != log {
		t.Error("FromContext did not return the stored logger")
	}
	if got := FromContext(context.Background()); got == log {
		t.Error("FromContext on empty context must return default logger")
	}
}

func TestMiddleware(t *testing.T) {
	buf, log := capture(t, "debug")

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("ok"))
	})
	req := httptest.NewRequest(http.MethodPost, "/webhooks/telegram?x=1", nil)
	req.RemoteAddr = "203.0.113.9:12345"
	rec := httptest.NewRecorder()

	Middleware(log, inner).ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d; want 201", rec.Code)
	}
	out := logString(buf)
	for _, want := range []string{"POST", "/webhooks/telegram", "203.0.113.9", "201"} {
		if !strings.Contains(out, want) {
			t.Errorf("request log missing %q:\n%s", want, out)
		}
	}
}
