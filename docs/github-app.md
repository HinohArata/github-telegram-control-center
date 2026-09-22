# GitHub App configuration

The bot authenticates as a GitHub App. It never uses a personal access token.

## Create the app

https://github.com/settings/apps/new

- **GitHub App name**: e.g. `control-center`
- **Homepage URL**: your public URL (can be any reachable HTTPS page)
- **Webhook URL**: `https://your-host/webhooks/github` (optional but required
  for notifications)
- **Webhook secret**: random value, use it as `GITHUB_WEBHOOK_SECRET`
- **Permissions** (Repository permissions):
  - Actions: Read and write
  - Administration: Read and write
  - Checks: Read (and write for check runs)
  - Contents: Read and write
  - Deployments: Read and write
  - Issues: Read and write
  - Metadata: Read (mandatory)
  - Pull requests: Read and write
- **Subscribe to events**: push, pull_request, issues, issue_comment,
  workflow_run, workflow_job, release, create, delete, repository

## Install it

Install the app on the organizations/personal accounts you manage. The bot
shows installed accounts via `/install` and `/orgs`.

## Private key

Settings → Developer settings → GitHub Apps → your app → **Generate a private
key**. Store the `.pem` contents in `GITHUB_PRIVATE_KEY`.

Two PEM formats work: `RSA PRIVATE KEY` (PKCS#1) and `PRIVATE KEY` (PKCS#8).
The bot parses either. `\n` sequences in the env value are converted to real
newlines.

## App permissions the bot uses

| Feature | Requires |
|---|---|
| Browse repos / files / commits / branches | Metadata, Contents (read) |
| PR / issues management | Pull requests / Issues (read+write) |
| Workflows, runs, cancel, rerun, trigger | Actions (read+write) |
| Job/run logs and artifacts | Actions (read) |
| Deployments | Deployments (read) |
| Notifications | Webhooks → events above |

Grant only what you need. The bot redacts secrets and refuses non-confirmed
write operations, but GitHub App permissions are the real boundary of what a
compromised token can do.