# GitHub Telegram Control Center

A production Telegram bot that turns Telegram into a remote control center for
your GitHub organizations: browse repos, commits, PRs, issues, releases and
GitHub Actions; run, cancel and rerun workflows; download logs and artifacts;
receive push/PR/issue/release notifications; and perform audited, confirmed
write operations — all over an encrypted Telegram chat.

Built with Go, PostgreSQL, the GitHub App API and the Telegram Bot API.

## Features

- GitHub App authentication (JWT + installation tokens, never PATs)
- Browse organizations, repositories, files, branches, tags, commits, diffs
- PR & issue management with confirmations for destructive actions
- GitHub Actions: workflows, runs, jobs, logs, artifacts, cancel/rerun/trigger
- Webhook signature validation + delivery deduplication (async processing)
- Configurable push/PR/issue/release/workflow notifications per repository
- Inline keyboards, pagination, HTML escaping everywhere
- Audit log of every operation, permission-checked on both commands and callbacks
- Metrics (Prometheus), health/readiness endpoints, structured logs with secret redaction
- Multi-stage Docker image, Railway-ready, graceful shutdown

## Quick start

See [docs/setup.md](docs/setup.md). Minimal flow:

1. Create a GitHub App (docs/github-app.md) and a Telegram bot (docs/telegram.md).
2. Set environment variables (section below).
3. `go run ./cmd/bot` or `docker compose up`.

## Architecture

```
Telegram user ──▶ /webhooks/telegram ──▶ bot ──▶ github.Service ──▶ GitHub API
                                              │        │
                                              └──▶ store (PostgreSQL)
GitHub webhook ──▶ /webhooks/github ──▶ dispatcher ──▶ listener ──▶ notifier ──▶ Telegram
                                              │
                                              └──▶ store.RecordDelivery (dedup)
```

- `internal/config` – env-based configuration
- `internal/bot` – Telegram command/callback handlers, inline keyboards
- `internal/github` – GitHub App auth, API client, service layer, webhook verification
- `internal/events` – async webhook dispatcher + notification handlers
- `internal/notifications` – Telegram message sender with metrics
- `internal/database` – pgx pool, migrations, data access
- `internal/reconcile` – installation sync + session housekeeping
- `internal/server` – HTTP endpoints: `/health`, `/ready`, `/metrics`,
  `/webhooks/github`, `/webhooks/telegram`
- `internal/telegram` – Bot API client, escaping, pagination, keyboards
- `internal/logging`, `internal/metrics` – sanitized structured logs and Prometheus metrics

## Setup

Detailed guides:

- [docs/setup.md](docs/setup.md) – end-to-end setup
- [docs/github-app.md](docs/github-app.md) – GitHub App configuration
- [docs/telegram.md](docs/telegram.md) – Telegram bot configuration
- [docs/deployment.md](docs/deployment.md) – Docker / Railway deployment
- [docs/webhooks.md](docs/webhooks.md) – webhook configuration
- [docs/commands.md](docs/commands.md) – full command reference
- [docs/security.md](docs/security.md) – security model
- [docs/architecture.md](docs/architecture.md) – design notes
- [docs/troubleshooting.md](docs/troubleshooting.md) – common issues

## Environment variables

| Variable | Required | Description |
|---|---|---|
| `TELEGRAM_BOT_TOKEN` | yes | Token from BotFather |
| `TELEGRAM_WEBHOOK_SECRET` | no | Secret token for webhook auth |
| `TELEGRAM_UPDATE_MODE` | no | `webhook` (default) or `polling` |
| `DATABASE_URL` | yes | PostgreSQL connection string |
| `GITHUB_WEBHOOK_SECRET` | yes | Shared webhook HMAC secret |
| `GITHUB_APP_ID` | yes | GitHub App ID |
| `GITHUB_PRIVATE_KEY` | yes | App private key (PEM, `\n` allowed) |
| `AUTHORIZED_TELEGRAM_USERS` | yes | Comma-separated Telegram user IDs (`user` role) |
| `ADMIN_TELEGRAM_USERS` | no | Comma-separated Telegram user IDs (`admin` role) |
| `WEBHOOK_BASE_URL` | no | Public base URL, e.g. `https://bot.example.com` |
| `PORT` | no | HTTP port (default `8080`) |
| `ENVIRONMENT` | no | `development` / `production` |
| `LOG_LEVEL` | no | `debug` / `info` / `warn` / `error` (default `info`) |
| `LOG_FORMAT` | no | `json` (default) / `text` |
| `CACHE_TTL` | no | Cache TTL, e.g. `60s` |
| `RECONCILIATION_INTERVAL` | no | Installation sync interval (default `10m`) |

Secrets are never logged: keys containing `token`, `secret`, `password`,
`authorization`, `private_key`, `database_url` and any PEM/JWT-looking values
are redacted. See [docs/security.md](docs/security.md).

## Docker

```
docker build -t gh-control-center .
docker run -p 8080:8080 --env-file .env gh-control-center
# or
docker compose up
```

The image runs as a non-root user, includes a healthcheck against `/health`,
and takes the port from `PORT`. See [docs/deployment.md](docs/deployment.md).

## Railway

`railway up` deploys using the included `Dockerfile` and `railway.toml`, which
performs a health check on `/health` and exposes the `PORT` HTTP service.
Attach a PostgreSQL service and set `DATABASE_URL` via the Railway dashboard.

## Webhooks

- GitHub: `POST /webhooks/github` — signatures verified with `GITHUB_WEBHOOK_SECRET`.
- Telegram: `POST /webhooks/telegram` — requires `TELEGRAM_UPDATE_MODE=webhook`
  and a public HTTPS `WEBHOOK_BASE_URL`.

## Command reference

`/start /status /install /org /orgs /repo /switch /repos /whoami /account`
`/set /files /tree /branches /tags /commits /commit /compare /search`
`/prs /pr /merge /comment /issues /issue /open /workflows /runs /run /jobs`
`/log /artifacts /artifact /deploy /release /branch /tag /file /notify`
`/sub /unsub /list /audit /users`

See [docs/commands.md](docs/commands.md) for syntax and examples.

## Development

```
go build ./...
go vet ./...
go test ./...
```

## License

MIT