# Architecture

Source: internal/telegram, internal/github, internal/events, internal/database.

## Components

```
┌────────────┐   POST /webhooks/telegram   ┌─────────────────────┐
│ Telegram   │ ───────────────────────────▶│ internal/server      │
│ Bot API    │                             │  · /health           │
└────────────┘                             │  · /ready            │
                                           │  · /metrics          │
┌────────────┐   POST /webhooks/github     │  · /webhooks/github  │
│ GitHub     │ ───────────────────────────▶│  · /webhooks/telegram│
│ App        │                             └──────┬──────────────┘
└────────────┘                                    │
                                                  ▼
                                   ┌─────────────────────────┐
                                   │ internal/events         │
                                   │  Dispatcher (async)     │
                                   │  Listener → handlers    │
                                   └────────────┬────────────┘
                                                 │
                            RecordDelivery (dedup)│  sendToSubscribers
                                                 ▼
                                   ┌─────────────────────────┐
                                   │ internal/notifications  │
                                   │  Notifier → Telegram    │
                                   └─────────────────────────┘

User commands ──▶ internal/bot ──▶ internal/github.Service ──▶ GitHub API
                                        │
                                        └──▶ internal/database (PostgreSQL)
```

## Key design decisions

### GitHub App auth, not PATs
`internal/github/auth.go` signs a short-lived RS256 JWT (9 minutes) with the
app private key, exchanges it for per-installation access tokens
(`POST /app/installations/{id}/access_tokens`), and caches tokens until 60s
before expiry. Token cache can be invalidated or cleared.

### Webhook verification + dedup
`internal/github/webhooks.go` verifies the `X-Hub-Signature-256` HMAC over the
raw body in constant time. `events.Listener.Process` records every delivery ID
in `webhook_deliveries` (unique) and skips duplicates, and marks deliveries
processed/failed with retry up to 5 attempts.

### Async event processing
The HTTP webhook handler only validates and enqueues. A buffered channel
(shard capacity `workers*16`) with `workers` goroutines processes events, so a
slow GitHub/Telegram response never blocks webhook delivery. Overload drops
with a `webhooks_failed_total` increment instead of blocking.

### Service-layer abstraction
`internal/github.Service` exposes typed methods per PRD requirements
(GetCommit, CreatePRComment, RunWorkflow, ...). The bot and event handlers
never touch the raw HTTP API or the installation-token machinery.

### Confirmation for destructive ops
Every destructive operation (merge PR, close PR/issue, cancel/rerun run,
delete branch/tag/release/file, run workflow) requires `✅`/`❌` confirmation.
The confirmation token is a crypto-random session id stored in
`sessions` with a 10-minute TTL and bound to the requesting user.

### Audit
Every write operation records an `audit_logs` row (user, repository,
organization, operation, target, success, error, GitHub request id).
Admins can inspect with `/audit`.

### Escaping
All user-supplied content rendered into Telegram messages is escaped
(`EscapeHTML` for HTML parse mode). Long output is split/paginated with
word-boundary awareness (`internal/telegram/pagination.go`).

### No secrets in logs
`slog` handlers redact any attribute whose key contains a secret marker and
any value shaped like a PEM key or JWT (`internal/logging`).

### Reconciliation
A worker periodically lists the app installations from GitHub and upserts/
prunes the `github_installations` table, and deletes expired sessions.

## Shutdown order
1. Stop accepting new jobs (dispatcher receives close)
2. Drain in-flight events (dispatcher `WaitGroup` with timeout)
3. Shut down the HTTP server
4. Close the database pool