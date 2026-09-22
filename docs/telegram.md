# Telegram configuration

## Create the bot

Message @BotFather in Telegram:

```
/newbot
```

Save the token (format `123456:AA...`); set it as `TELEGRAM_BOT_TOKEN`.

## Security: only your users

The bot ignores everyone not listed in `AUTHORIZED_TELEGRAM_USERS` /
`ADMIN_TELEGRAM_USERS`. Both commands and inline-callback presses are
re-checked against the user id — authorization is not only checked at command
time but also on every callback (so a leaked message cannot be operated on by
a stranger).

- Users in `AUTHORIZED_TELEGRAM_USERS` get role `user`.
- Users in `ADMIN_TELEGRAM_USERS` get role `admin` (can run `/users`,
  `/audit`, and everything else).
- Find your numeric Telegram user id via @userinfobot.

## Webhook mode

Telegram Bots normally receive updates via long polling. In production the bot
uses the webhook:

```
TELEGRAM_UPDATE_MODE=webhook
WEBHOOK_BASE_URL=https://bot.example.com
TELEGRAM_WEBHOOK_SECRET=random-value
```

On startup the server registers the webhook at
`https://bot.example.com/webhooks/telegram` and protects it with the
`X-Telegram-Bot-Api-Secret-Token` header.

Set `TELEGRAM_UPDATE_MODE=polling` during local development so no public URL
is needed.

## Notifications

Users opt in per repository:

- `/sub owner/repo` — all events
- `/sub owner/repo push` — a specific event (`push`, `pull_request`,
  `issues`, `release`, `workflow_run`, ...)
- `/sub owner/repo workflow_run main ci-*` — workflow/branch filters
- `/list` — show your subscriptions
- `/unsub owner/repo` — remove

Fine-grained toggles via `/notify`: workflow started/success/failure/cancelled,
push events, PR events, issue events, release events, verbose job output.