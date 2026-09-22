# Troubleshooting

## Startup

### "TELEGRAM_BOT_TOKEN is required" / other missing env
Set every required variable (see setup.md). Only `*_TELEGRAM_WEBHOOK_SECRET`,
`REDIS_URL`, `LOG_*`, `PORT`, `ENVIRONMENT`, `CACHE_TTL`,
`RECONCILIATION_INTERVAL`, `WEBHOOK_BASE_URL` are optional.

### "connect database: ..." 
- PostgreSQL reachable from the bot host? For Docker use the compose
  `DATABASE_URL`; for Railway use the "PostgreSQL" service connection string.
- Auth/ssl: try `?sslmode=disable` for local, always `sslmode=require` or
  Railway default for remote.

### "set telegram webhook: ..."
- `WEBHOOK_BASE_URL` must be a public HTTPS URL.
- Telegram rejects webhooks with self-signed/invalid TLS certs.

### 401 on `/webhooks/github`
- `GITHUB_WEBHOOK_SECRET` differs from the one configured in the GitHub App.
- Compare `sha256=` of the raw request body against
  `X-Hub-Signature-256`; any reformatting (pretty-print) breaks the signature.

## GitHub API

### 403 / 404 from service methods
- App not installed on that account, or the app lacks permissions (see
  github-app.md). Reinstall the app; the reconcile worker refreshes
  installations every `RECONCILIATION_INTERVAL`.
- Installation tokens are installation-scoped: an org not yet installed
  cannot be listed until `/install` shows it.

### Rate limited (403)
The HTTP client honors `Retry-After` and backoff; watch
`github_rate_limit_remaining` (Prometheus) and the GitHub response headers.

### private key "no PEM block found"
- `GITHUB_PRIVATE_KEY` empty or not a PEM (must be `-----BEGIN ... KEY-----`).
- Single-line env value with `\n` sequences is supported; if you escaped them
  twice (`\\n`) the PEM gets encoded `\n` literally → decode error. Use real
  newlines or a single `\n`.

## Telegram

### Bot silent / no replies
- Bot receives updates? In webhook mode check the server registers webhook at
  startup; test `curl https://api.telegram.org/bot<TOKEN>/getWebhookInfo`.
- User not in `AUTHORIZED_TELEGRAM_USERS` → ignored silently. Check the id.
- Long polling selected but server in webhook mode → set
  `TELEGRAM_UPDATE_MODE=polling`.

### "Unknown command"
Run `/help` for the menu. Commands must start with `/`.

## Notifications not arriving

- `/list` must show the repo subscription.
- `/notify` toggles per-event (e.g. `workflow_started`) may be off.
- `workflow_run` notifications only fire for events GitHub actually sends to
  the subscribed repo's installation; webhook deliveries are visible through
  `webhook_deliveries`/logs.

## Migrations

### Schema version conflicts
Migrations are embedded and tracked in `schema_migrations`, applied in
transactional per-file steps. If a migration fails, the whole file rolls back.
Do not edit applied migrations; add a new `00X_*.sql`.

## Metrics / observability

- `/metrics` needs no auth — put it behind a private network or reverse-proxy
  ACL in production.
- Log level: `LOG_LEVEL=debug` shows more (never secrets).