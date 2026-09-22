package config

import (
	"testing"
)

// validEnv is the minimal required environment for Load to succeed.
func setEnv(t *testing.T) {
	t.Helper()
	vars := map[string]string{
		"TELEGRAM_BOT_TOKEN":     "123:test-token",
		"DATABASE_URL":           "postgres://user:pass@localhost:5432/db",
		"GITHUB_WEBHOOK_SECRET":  "wh-secret",
		"GITHUB_APP_ID":          "123456",
		"GITHUB_PRIVATE_KEY":     "-----BEGIN RSA PRIVATE KEY-----\\nABCD\\n-----END RSA PRIVATE KEY-----",
		"AUTHORIZED_TELEGRAM_USERS": "111,222",
		"ADMIN_TELEGRAM_USERS":      "999",
	}
	for k, v := range vars {
		t.Setenv(k, v)
	}
}

func TestLoad(t *testing.T) {
	setEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.GitHubAppID != 123456 {
		t.Errorf("GitHubAppID = %d", cfg.GitHubAppID)
	}
	if got := string(cfg.GitHubPrivateKey); got != "-----BEGIN RSA PRIVATE KEY-----\nABCD\n-----END RSA PRIVATE KEY-----" {
		t.Errorf("private key \\n not expanded: %q", got)
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d; want 8080 default", cfg.Port)
	}
	if cfg.TelegramUpdateMode != "webhook" {
		t.Errorf("TelegramUpdateMode = %q", cfg.TelegramUpdateMode)
	}
	if cfg.WebhookBaseURL != "" {
		t.Errorf("WebhookBaseURL = %q; want empty", cfg.WebhookBaseURL)
	}
}

func TestLoadTrimsWebhookBaseURL(t *testing.T) {
	setEnv(t)
	t.Setenv("WEBHOOK_BASE_URL", "https://bot.example.com/")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.WebhookBaseURL != "https://bot.example.com" {
		t.Errorf("WebhookBaseURL = %q; want trailing slash trimmed", cfg.WebhookBaseURL)
	}
}

func TestLoadPortOverride(t *testing.T) {
	setEnv(t)
	t.Setenv("PORT", "9000")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != 9000 {
		t.Errorf("Port = %d; want 9000", cfg.Port)
	}
}

func TestRoleFor(t *testing.T) {
	setEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	cases := []struct {
		id       int64
		wantRole string
		wantOK   bool
	}{
		{111, "user", true},
		{222, "user", true},
		{999, "admin", true},
		{0, "", false},
		{333, "", false},
		{-1, "", false},
	}
	for _, c := range cases {
		role, ok := cfg.RoleFor(c.id)
		if ok != c.wantOK || role != c.wantRole {
			t.Errorf("RoleFor(%d) = (%q, %t); want (%q, %t)", c.id, role, ok, c.wantRole, c.wantOK)
		}
	}
}

func TestLoadRequiresSecrets(t *testing.T) {
	// No required variable set: Load must fail.
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("GITHUB_WEBHOOK_SECRET", "")
	t.Setenv("GITHUB_APP_ID", "")
	t.Setenv("GITHUB_PRIVATE_KEY", "")
	t.Setenv("AUTHORIZED_TELEGRAM_USERS", "")
	t.Setenv("ADMIN_TELEGRAM_USERS", "")
	if _, err := Load(); err == nil {
		t.Fatal("Load must fail without required env vars")
	}
}

func TestLoadRejectsInvalidUsers(t *testing.T) {
	setEnv(t)
	t.Setenv("AUTHORIZED_TELEGRAM_USERS", "not-a-number")
	if _, err := Load(); err == nil {
		t.Fatal("Load must reject non-numeric telegram user ids")
	}
}

func TestLoadRejectsInvalidAppID(t *testing.T) {
	setEnv(t)
	t.Setenv("GITHUB_APP_ID", "abc")
	if _, err := Load(); err == nil {
		t.Fatal("Load must reject non-numeric GITHUB_APP_ID")
	}
}
