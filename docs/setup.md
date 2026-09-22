# Setup

Prerequisites: Go 1.27+, PostgreSQL 15+, a GitHub App, a Telegram bot.

## 1. Create the components

- Follow [github-app.md](github-app.md) to create the GitHub App.
- Follow [telegram.md](telegram.md) to create the Telegram bot.

## 2. Database

```sql
CREATE USER bot WITH PASSWORD '...';
CREATE DATABASE control_center OWNER bot;
```

Migrations run automatically on startup (embedded in the binary, tracked in
`schema_migrations`). No manual migration step needed.

Connection string:

```
DATABASE_URL=postgres://bot:PASSWORD@localhost:5432/control_center?sslmode=disable
```

## 3. Environment

Copy this into a `.env` file (or your host's environment):

```bash
TELEGRAM_BOT_TOKEN=123456:ABC...          # from BotFather
TELEGRAM_WEBHOOK_SECRET=change-me
TELEGRAM_UPDATE_MODE=polling              # change to webhook in production

DATABASE_URL=postgres://bot:PASSWORD@localhost:5432/control_center?sslmode=disable

GITHUB_WEBHOOK_SECRET=change-me-too
GITHUB_APP_ID=123456
GITHUB_PRIVATE_KEY="-----BEGIN RSA PRIVATE KEY-----\n...\n-----END RSA PRIVATE KEY-----"
```

`GITHUB_PRIVATE_KEY` accepts literal `\n` sequences so it can live in a single
line. You may also use a real newline-separated PEM block.

Authorization:

```bash
AUTHORIZED_TELEGRAM_USERS=111111,222222    # these get role "user"
ADMIN_TELEGRAM_USERS=999999                # these get role "admin"
```

Find your Telegram user id by messaging @userinfobot.

## 4. Run

Development (long polling — no public URL needed):

```bash
export TELEGRAM_UPDATE_MODE=polling
go run ./cmd/bot
```

Then open Telegram and `/start` the bot. Use `/install` to list installation
commands and `/org <name>` to browse.

For webhook mode see [deployment.md](deployment.md) and [webhooks.md](webhooks.md).

## 5. Verify

- `/health` returns 200 with JSON.
- `/ready` returns 200 when the database is reachable.
- `/metrics` exposes Prometheus text metrics.
- In Telegram: `/install`, `/status`, `/repos` respond.