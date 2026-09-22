# Command reference

Auth: commands and inline-button presses both check the Telegram user id.
`/users` and `/audit` require the `admin` role (`ADMIN_TELEGRAM_USERS`).

## Getting started

| Command | Description |
|---|---|
| `/start`, `/help` | Main menu + command help |
| `/status` | Health, DB, GitHub installation counts |
| `/install` | List GitHub App installations |
| `/account` / `/whoami` | GitHub account / Telegram user info |
| `/set default_repo owner/repo` | Set default repository |
| `/set default_org NAME` | Set default organization |
| `/set page_size 100` | Set pagination size |
| `/set timezone Asia/Ho_Chi_Minh` | Set display timezone |

## Organization & repository

| Command | Description |
|---|---|
| `/orgs` | List installed accounts, switch account |
| `/org NAME` | Show organization details |
| `/repos` | List repositories |
| `/repo owner/repo` | Repository menu (nav buttons) |
| `/repo` | Repository menu for default repo |

## Exploring code & history

| Command | Description |
|---|---|
| `/files [path]` | Browse file tree / read file |
| `/branches` | List branches |
| `/tags` | List tags |
| `/commits [branch]` | Commit history |
| `/commit SHA` | Commit detail + diff |
| `/compare base..head` | Compare refs |
| `/search repos QUERY` | Search repositories |
| `/search issues QUERY` | Search issues |
| `/search code QUERY` | Search code (owner-scoped) |
| `/deploy` | Deployment listing |

## Pull requests & issues

| Command | Description |
|---|---|
| `/prs [state]` | List PRs (`open`/`closed`/`merged`) |
| `/pr NUMBER` | PR detail + diff stats |
| `/merge NUMBER [method]` | Merge PR (confirm) |
| `/comment pr NUMBER text` | Comment on PR |
| `/issues [state]` | List issues |
| `/issue NUMBER` | Issue detail |
| `/open TITLE` | Create issue (confirm) |
| `/comment issue NUMBER text` | Comment on issue |

## GitHub Actions

| Command | Description |
|---|---|
| `/workflows` | List workflows |
| `/runs` | List recent runs |
| `/run [NUMBER]` | Run detail + jobs |
| `/jobs` / `/jobs RUN` | Job list |
| `/log` | Download run/job logs (document) |
| `/artifacts` | List artifacts |
| `/artifact NAME` | Download artifact |
| `/run` (run menu buttons) | Cancel / rerun / trigger (confirm) |

## Releases, branches, tags, files (write ops)

| Command | Description |
|---|---|
| `/release` | List / view releases |
| `/release create TAG [--name N --body B]` | Create release (confirm) |
| `/branch create NAME` / `/branch delete NAME` | Branch ops (confirm) |
| `/tag create NAME SHA` / `/tag delete NAME` | Tag ops (confirm) |
| `/file PATH` | Read file |
| `/file write PATH --message M --content C` | Write/update file (confirm) |
| `/file delete PATH --message M` | Delete file (confirm) |

All destructive commands are confirmed via inline `✅/❌` and audited.

## Notifications

| Command | Description |
|---|---|
| `/notify` | Toggle notification preferences (inline) |
| `/sub owner/repo` | Subscribe to all events for a repo |
| `/sub owner/repo push` | Subscribe to one event type |
| `/sub owner/repo workflow_run main ci-*` | Workflow+branch filters |
| `/list` | List your subscriptions |
| `/unsub owner/repo` | Remove subscription |

## Administration

| Command | Description |
|---|---|
| `/list` | List users (alias) |
| `/users` | List authorized users (admin) |
| `/audit [limit]` | Recent audit entries (admin) |

## Pagination

Long lists render one page at a time with `⬅️ [n/N] ➡️` buttons. Jump with
`/repos` → page keyboard, or tap prev/next on any paginated list.