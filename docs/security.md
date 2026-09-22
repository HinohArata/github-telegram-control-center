# Security model

## Authentication

- **GitHub**: the bot uses a GitHub App. The app private key signs a 9-minute
  RS256 JWT; the JWT is exchanged for per-installation access tokens that are
  cached until 60s before expiry. No personal access tokens, ever.
- **Telegram**: the bot replies only to users in `AUTHORIZED_TELEGRAM_USERS`
  (role `user`) and `ADMIN_TELEGRAM_USERS` (role `admin`). The check runs on
  commands **and** on every inline button press.
- **Webhooks (Telegram)**: `X-Telegram-Bot-Api-Secret-Token` must match
  `TELEGRAM_WEBHOOK_SECRET`.
- **Webhooks (GitHub)**: `X-Hub-Signature-256` HMAC-SHA256 verified over the
  raw body in constant time using `GITHUB_WEBHOOK_SECRET`.

## Authorization

Two roles: `admin` (everything, including `/users` and `/audit`) and `user`.
Role is derived from `cfg.RoleFor(telegramUserID)`. Server-side, every write
operation passes through the store/session confirmation flow — the bot never
trusts a client-side claim.

## Confirmations for destructive operations

`merge`, `close` (PR/issue), `cancel`/`rerun` (workflow run), `branch delete`,
`tag delete`, `release delete`, `file delete`, `workflow run trigger` require
a `✅/❌` confirmation. The confirmation binds:

- a crypto-random session id (`crypto/rand`, hex)
- the requesting Telegram user id
- a TTL of 10 minutes (`sessions.expires_at`)

A confirmation pressed by a different user, after expiry, or replayed is
rejected. Expired sessions are purged by the reconcile worker.

## Audit log

Every auditable operation writes a row to `audit_logs`: user, GitHub user,
repository, organization, operation, target, success, GitHub request id, and
error. Inspect via `/audit` (admin). Audit writes never contain secrets.

## Secret handling

- Configuration is environment-only; secrets are never written by the app.
- slog redacts attribute values whose key contains `token`, `secret`,
  `password`, `authorization`, `private_key` or `database_url` (case
  insensitive) and any value that looks like a PEM key or a JWT.
- Webhook bodies are stored only as `webhook_deliveries` metadata; payload
  content is not logged.

## Webhook deduplication and overload

- Each delivery is recorded by unique `delivery_id`; duplicates are dropped.
- Deliveries fail with up to 5 attempts against external services.
- The webhook handler never blocks on the goroutine queue; under overload the
  event is dropped and counted (`webhooks_failed_total`), never retried with
  a stale signature.

## Rate limiting

GitHub rate-limit headers are tracked per resource and exposed as a Prometheus
gauge (`github_rate_limit_remaining`). The HTTP client retries transient
5xx/401/403/429 with backoff and honors `Retry-After`.

## Known boundary / report

All write operations are gated behind GitHub App permissions. Grant the app
the minimum permissions needed (see github-app.md). If a private key leaks,
rotate it in the GitHub App settings and update `GITHUB_PRIVATE_KEY`.