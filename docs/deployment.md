# Deployment

The app is stateless (persistent state lives in PostgreSQL only) and runs as a
single binary that serves HTTP and polls/registers webhooks. It handles
`SIGTERM`/`SIGINT` with a graceful shutdown: stop accepting jobs → drain
in-flight events → close HTTP → close database.

## Docker

Production multi-stage image: `golang:1.27-alpine` builder → `alpine:3.21`
runtime, non-root user `app`, healthcheck on `/health`, port from `PORT`.

```bash
docker build -t gh-control-center .
docker run --rm -p 8080:8080 --env-file .env gh-control-center
```

Or with PostgreSQL:

```bash
docker compose up
# set WEBHOOK_BASE_URL first for webhook mode (see docker-compose.yml)
```

No secrets are baked into the image; pass them via environment.

## Railway

1. `railway up` (or connect the repo for auto-deploy on push).
2. Add a PostgreSQL plugin; copy its connection string to `DATABASE_URL`.
3. Set all required env vars (Railway dashboard).
4. Railway injects `PORT`; `railway.toml` configures the health check on
   `/health` and the HTTP service on `$PORT`.
5. Set `WEBHOOK_BASE_URL` to the generated public domain
   (`https://<service>.up.railway.app`) for webhook mode.

The service domain must be HTTPS for Telegram webhooks.

## Oracle Cloud / VPS

Run the Docker image behind any TLS-terminating proxy (Caddy, Nginx,
Traefik) that forwards `/webhooks/*` and `/health` to the container on
`PORT`. The reverse proxy must provide HTTPS; only the container port is
needed internally.

## Checks after deploy

- `GET /ready` → 200 once DB migration + ping succeed.
- `GET /health` → 200 always (liveness).
- Give Telegram a webhook: probably worked automatically; if you see an API
  error at startup check `WEBHOOK_BASE_URL` and `TELEGRAM_UPDATE_MODE`.

## Graceful shutdown

Signals: `SIGTERM`, `SIGINT`. Order:
1. stop accepting new jobs
2. finish current safe operations (drain dispatcher)
3. close HTTP server
4. close database pool