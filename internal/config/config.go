package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment string
	Port        int

	TelegramBotToken      string
	TelegramWebhookSecret string
	AuthorizedUsers       map[int64]string
	AdminUsers            map[int64]bool

	GitHubAppID         int64
	GitHubPrivateKey    []byte
	GitHubWebhookSecret string

	DatabaseURL string

	RedisURL               string
	LogLevel               string
	LogFormat              string
	WebhookBaseURL         string
	CacheTTL               time.Duration
	ReconciliationInterval time.Duration

	TelegramUpdateMode string
}

func Load() (*Config, error) {
	cfg := &Config{
		Environment:            getEnv("ENVIRONMENT", "development"),
		Port:                   getIntEnv("PORT", 8080),
		TelegramBotToken:       os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramWebhookSecret:  os.Getenv("TELEGRAM_WEBHOOK_SECRET"),
		GitHubWebhookSecret:    os.Getenv("GITHUB_WEBHOOK_SECRET"),
		DatabaseURL:            os.Getenv("DATABASE_URL"),
		RedisURL:               os.Getenv("REDIS_URL"),
		LogLevel:               getEnv("LOG_LEVEL", "info"),
		LogFormat:              getEnv("LOG_FORMAT", "json"),
		WebhookBaseURL:         strings.TrimSuffix(os.Getenv("WEBHOOK_BASE_URL"), "/"),
		CacheTTL:               getDurationEnv("CACHE_TTL", 60*time.Second),
		ReconciliationInterval: getDurationEnv("RECONCILIATION_INTERVAL", 10*time.Minute),
		TelegramUpdateMode:     getEnv("TELEGRAM_UPDATE_MODE", "webhook"),
	}

	cfg.AuthorizedUsers = map[int64]string{}
	for _, part := range strings.Split(os.Getenv("AUTHORIZED_TELEGRAM_USERS"), ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid AUTHORIZED_TELEGRAM_USERS entry %q: %w", part, err)
		}
		cfg.AuthorizedUsers[id] = "user"
	}
	for _, part := range strings.Split(os.Getenv("ADMIN_TELEGRAM_USERS"), ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid ADMIN_TELEGRAM_USERS entry %q: %w", part, err)
		}
		cfg.AuthorizedUsers[id] = "admin"
	}

	if cfg.TelegramBotToken == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN is required")
	}
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.GitHubWebhookSecret == "" {
		return nil, fmt.Errorf("GITHUB_WEBHOOK_SECRET is required")
	}

	appID := os.Getenv("GITHUB_APP_ID")
	if appID == "" {
		return nil, fmt.Errorf("GITHUB_APP_ID is required")
	}
	id, err := strconv.ParseInt(appID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid GITHUB_APP_ID: %w", err)
	}
	cfg.GitHubAppID = id

	key := os.Getenv("GITHUB_PRIVATE_KEY")
	if key == "" {
		return nil, fmt.Errorf("GITHUB_PRIVATE_KEY is required")
	}
	cfg.GitHubPrivateKey = []byte(strings.ReplaceAll(key, "\\n", "\n"))

	if len(cfg.AuthorizedUsers) == 0 {
		return nil, fmt.Errorf("AUTHORIZED_TELEGRAM_USERS is required")
	}

	return cfg, nil
}

func (c *Config) RoleFor(telegramUserID int64) (string, bool) {
	role, ok := c.AuthorizedUsers[telegramUserID]
	return role, ok
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getIntEnv(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
