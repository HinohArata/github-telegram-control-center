# Webhooks

## GitHub webhook

**URL**: `POST /webhooks/github`

- Method: `POST` only; otherwise `405`.
- Body limit 10 MiB; larger → `413`.
- Requires valid `X-Hub-Signature-256` → else `401` and a warning in logs.
- Responds `202 {"status":"accepted"}` once the event is enqueued.

The event is processed asynchronously:

1. `RecordDelivery` inserts with a unique `delivery_id`; duplicates are
   dropped (idempotent).
2. The listener routes by `X-GitHub-Event` to a handler.
3. Handlers decide recipients from `repository_subscriptions` and per-user
   `notification_preferences`, then notify via the notifier.

Delivery processing state is tracked (`pending / processing / processed /
failed`) with a retry budget of 5 attempts; inspect via the database or the
logging of `webhooks_failed_total`.

### Supported events

`push`, `pull_request`, `issues`, `issue_comment`, `workflow_run`,
`workflow_job`, `release`, `create`, `delete`, `repository`.

Unknown events are acknowledged and marked processed (no crash, no retry
storm).

## Telegram webhook

**URL**: `POST /webhooks/telegram`

- Only active when `TELEGRAM_UPDATE_MODE=webhook` and `WEBHOOK_BASE_URL` is
  set; server registers it via the Bot API `setWebhook` at startup.
- Requires `X-Telegram-Bot-Api-Secret-Token` to match
  `TELEGRAM_WEBHOOK_SECRET` when configured.
- Responds `200` immediately; the update is handled in a goroutine.

## Registering webhooks manually

GitHub: Settings → Developer settings → your App → Webhook. URL:
`https://your-host/webhooks/github`, content type `application/json`,
secret = `GITHUB_WEBHOOK_SECRET`.

Telegram: none needed — the app calls `setWebhook` at startup.

## Local testing

GitHub webhooks require a public URL. Options:

- `ssh -R 80:localhost:8080 npx localtunnel` then set that as the webhook URL.
- Or run `TELEGRAM_UPDATE_MODE=polling` and trigger GitHub events manually
  (curl a signed request to `localhost:8080/webhooks/github`):

```bash
SECRET=GITHUB_WEBHOOK_SECRET
BODY='{"action":"opened","number":1,"repository":{"full_name":"owner/repo"}}'
SIG="sha256=$(printf %s "$BODY" | openssl dgst -sha256 -hmac "$SECRET" -hex | awk '{print $2}')"
curl -X POST localhost:8080/webhooks/github \
  -H "X-GitHub-Event: push" \
  -H "X-GitHub-Delivery: test-1" \
  -H "X-Hub-Signature-256: $SIG" \
  -H "Content-Type: application/json" \
  --data-binary "$BODY"
```