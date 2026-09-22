# PRD — GitHub Telegram Control Center

## 1. Product Overview

Build a production-ready Telegram bot that acts as a complete remote GitHub management and monitoring console.

The system must allow an authorized Telegram user to manage and monitor GitHub accounts, organizations, repositories, branches, commits, diffs, pull requests, issues, releases, GitHub Actions workflows, workflow runs, jobs, logs, artifacts, deployments, webhooks, and repository contents directly from Telegram.

The primary goal is:

> Turn Telegram into a secure, interactive, real-time GitHub control center.

The application must be designed for 24/7 deployment on platforms such as Railway, Oracle Cloud, or another Linux container/VPS environment.

## 2. Core Principles

1. Use GitHub App authentication rather than relying on a personal access token as the primary authentication mechanism.
2. Follow GitHub least-privilege permission principles.
3. Use GitHub Webhooks for realtime events.
4. Use GitHub REST API and GraphQL API where appropriate.
5. Use polling only as fallback/reconciliation.
6. Never expose GitHub private keys, tokens, webhook secrets, or Telegram bot tokens to Telegram users.
7. All destructive operations require explicit confirmation.
8. All GitHub write operations must generate audit logs.
9. Telegram UI must be interactive using inline keyboards wherever possible.
10. Long GitHub responses must be paginated and split safely for Telegram.
11. The bot must remain usable on mobile.
12. The backend must be stateless where possible.
13. Configuration must be environment-variable based.
14. The application must be Docker-ready.
15. The codebase must be modular and maintainable.

## 3. Target Architecture

```text
                         Telegram
                            │
                            ▼
                    ┌───────────────┐
                    │ Telegram Bot  │
                    │   Interface   │
                    └───────┬───────┘
                            │
                            ▼
                  ┌───────────────────┐
                  │ Application API   │
                  │ / Command Router  │
                  └─────────┬─────────┘
                            │
             ┌──────────────┼──────────────┐
             ▼              ▼              ▼
        GitHub Service   Auth Service   Event Service
             │              │              │
             └──────────────┼──────────────┘
                            │
                     GitHub App Auth
                            │
                            ▼
                    GitHub REST API
                    GitHub GraphQL
                            │
                            ▼
                    GitHub Webhooks
                            │
                            ▼
                    Webhook Receiver
                            │
                            ▼
                     Event Processor
                            │
                            ▼
                         Telegram
```

Recommended infrastructure:

- Application: Go
- Telegram: Telegram Bot API
- GitHub: GitHub App, REST API, GraphQL API, Webhooks
- Database: PostgreSQL
- Cache/Queue: Redis optional
- Reverse Proxy: Caddy or Nginx
- Deployment: Docker, Railway / Oracle Cloud / VPS

For the first implementation, PostgreSQL is required and Redis may remain optional.

## 4. Authentication Model

### 4.1 GitHub App

Create a GitHub App.

Required concepts:

- GitHub App ID
- GitHub Private Key
- GitHub Webhook Secret
- GitHub App Installation ID

The backend must generate installation access tokens when communicating with GitHub.

Never permanently store short-lived installation tokens unless required for caching. Tokens should be refreshed automatically.

## 5. Telegram Authentication

The bot must implement an allowlist.

Environment variable:

```text
AUTHORIZED_TELEGRAM_USERS
```

Example:

```text
AUTHORIZED_TELEGRAM_USERS=123456789,987654321
```

Unauthenticated users must receive:

```text
⛔ Unauthorized

This bot is private.
```

No GitHub information may be exposed before authorization.

## 6. Main Telegram Navigation

The `/start` command must show:

```text
┌──────────────────────────────┐
│ GitHub Control Center        │
├──────────────────────────────┤
│ 👤 Account                   │
│ 🏢 Organizations             │
│ 📦 Repositories              │
│ ⚙️ GitHub Actions            │
│ 🔀 Pull Requests             │
│ 🐛 Issues                    │
│ 📝 Commits                   │
│ 🌿 Branches                  │
│ 🏷 Releases                  │
│ 🔔 Monitoring                │
│ 📊 Dashboard                 │
│ ⚙️ Settings                  │
└──────────────────────────────┘
```

All buttons must use callback queries.

## 7. Account Features

Command:

```text
/account
```

Show:

- GitHub username
- display name
- avatar
- profile URL
- public repositories
- followers
- following
- organizations
- account type
- GitHub App installation status

Actions:

- Organizations
- Repositories
- Activity
- Notifications
- Profile

## 8. Organization Features

Command:

```text
/orgs
```

Show all organizations accessible through the GitHub App.

Selecting an organization:

- Repositories
- Members
- Teams
- Actions
- Issues
- Pull Requests
- Releases
- Activity
- Settings

Organization overview:

- name
- description
- avatar
- repository count
- members
- public/private status where available
- organization URL

## 9. Repository Explorer

Command:

```text
/repos
```

Repository dashboard:

```text
📦 OWNER/REPOSITORY

⭐ Stars
🍴 Forks
👁 Watchers

Default branch:
main

Last commit:
abcdef1

Updated:
2 minutes ago
```

Buttons:

- Files
- Branches
- Commits
- Diff
- Actions
- PRs
- Issues
- Releases
- Tags
- Deployments
- Webhooks
- Settings

## 10. Repository File Browser

Support:

```text
/files OWNER/REPO
```

Directory navigation and file viewing.

Selecting a file displays:

- filename
- path
- size
- SHA
- language
- content

Large files must be paginated.

Example:

```text
📄 BoardConfig.mk

Lines 1-100 / 483

[⬅️ Previous]
[Next ➡️]
[Search]
[Raw]
[GitHub]
```

Do not send huge files as a single Telegram message.

## 11. File Search

Implement repository search.

Example:

```text
/search OWNER/REPO "TARGET_KERNEL_VERSION"
```

Return matching files and line context.

## 12. Commit History

Command:

```text
/commits OWNER/REPO
```

Support:

- default branch
- selected branch
- selected path
- pagination
- author filter
- date range

Commit detail must show:

- SHA
- author
- committer
- date
- message
- files changed
- additions
- deletions
- diff link/action

## 13. Commit Diff

Command:

```text
/commit OWNER/REPO SHA
```

Show:

- commit metadata
- changed files
- additions
- deletions
- patch/diff

Large diffs must be paginated.

## 14. Compare / Diff

Implement:

```text
/compare OWNER/REPO BASE HEAD
```

Support branches, tags, and SHAs.

Show:

- files changed
- additions
- deletions
- commits
- changed files
- diff

## 15. Branch Management

Command:

```text
/branches OWNER/REPO
```

Show:

- branch name
- protected status
- latest commit
- ahead/behind where available

Actions:

- Create Branch
- Delete Branch
- Compare
- Open

Deleting a branch requires confirmation.

## 16. Tag Management

Command:

```text
/tags OWNER/REPO
```

Support:

- list tags
- inspect tag
- create tag
- delete tag

Delete requires confirmation.

## 17. Pull Requests

Command:

```text
/prs OWNER/REPO
```

Filters:

- Open
- Closed
- Merged
- Author
- Assignee
- Label
- Branch

PR detail must show:

- author
- base/head
- additions/deletions
- changed files
- checks

Actions:

- Commits
- Files
- Diff
- Checks
- Comments
- Approve
- Comment
- Merge
- Close
- Reopen

Merge requires explicit confirmation.

## 18. Issues

Command:

```text
/issues OWNER/REPO
```

Support:

- list issues
- labels
- milestones
- assignees
- comments
- create issue
- edit issue
- close
- reopen

Commands:

```text
/issues create
/issues close NUMBER
/issues reopen NUMBER
/issues comment NUMBER
```

## 19. Releases

Command:

```text
/releases OWNER/REPO
```

Show:

- version
- tag
- title
- author
- creation date
- published date
- prerelease status
- draft status
- assets

Actions:

- Create Release
- Edit Release
- Delete Release
- Assets

Delete requires confirmation.

## 20. GitHub Actions

This is a core feature.

Command:

```text
/actions OWNER/REPO
```

Show workflows and recent runs.

Example:

```text
⚙️ GitHub Actions

Build ROM
Build Kernel
Release
Lint
Tests
```

Workflow runs:

```text
#182 🟢 success
#181 🔴 failure
#180 🟢 success
#179 🟡 running
```

## 21. Workflow Run Details

Show:

- workflow
- run number
- branch
- commit SHA
- status
- conclusion
- started time
- finished time
- duration

Actions:

- Jobs
- Logs
- Artifacts
- Commit
- Open GitHub
- Re-run

## 22. Workflow Job Monitoring

Show every job and duration.

Example:

```text
Jobs

🟢 sync
12m 31s

🟢 lunch
8s

🟢 build
31m 42s

🟢 sign
2m 14s

🟢 upload
1m 55s
```

## 23. Workflow Logs

Implement:

- retrieve job logs
- large-log handling
- pagination
- search inside logs
- failed-step detection
- optional `.txt` export

Example:

```text
🔴 Build failed

Job:
build

Step:
ninja

Error matches:
ninja: error:
FAILED:
error:
```

## 24. Workflow Control

Support:

- Run workflow
- Cancel run
- Re-run failed jobs
- Re-run all jobs

Run workflow UI must support branch and workflow inputs.

Example:

```text
Workflow:
Build ROM

Branch:
[afl-16]

Inputs:

DEVICE:
[surya]

RELEASETYPE:
[userdebug]

GMS_VARIANT:
[vanilla]

[🚀 Run Workflow]
```

Confirmation is required before execution.

## 25. Artifacts

Show workflow artifacts:

- name
- size
- expiration where available
- download/open action

Do not attempt to send files larger than Telegram limits. For large artifacts, provide a secure download URL when possible.

## 26. Deployments

Implement deployment visibility:

- environment
- deployment status
- commit
- creator
- timestamps
- deployment URL where available

## 27. Realtime Webhook System

GitHub Webhooks are mandatory.

Endpoint:

```text
POST /webhooks/github
```

Validate:

- `X-Hub-Signature-256`
- `X-GitHub-Event`
- `X-GitHub-Delivery`

Reject invalid signatures.

Store GitHub delivery IDs to prevent duplicate processing.

## 28. Realtime Events

Support at minimum, subject to GitHub App permissions:

```text
push
pull_request
pull_request_review
issues
issue_comment
workflow_run
workflow_job
check_run
release
deployment
deployment_status
create
delete
branch_protection_rule
repository
repository_import
repository_vulnerability_alert
security_advisory
star
fork
```

## 29. Realtime Workflow Notifications

When a workflow starts:

```text
⚙️ Workflow Started

AfterlifeOS/rom

Build ROM #182

Branch:
afl-16

Commit:
abc1234
```

When completed:

```text
🟢 Workflow Completed

AfterlifeOS/rom

Build ROM #182

Result:
SUCCESS

Duration:
47m 21s
```

Failure:

```text
🔴 Workflow Failed

Build ROM #183

Failed job:
build

Failed step:
ninja

[View Logs]
```

## 30. Realtime Push Notifications

Example:

```text
📤 New Push

AfterlifeOS/device_xiaomi_surya

Branch:
afl-16

Commits:
3

Latest:
abc1234

Fix init configuration

Author:
HinohArata

[View Commit]
```

## 31. Realtime Pull Request Notifications

Support events:

- opened
- closed
- reopened
- merged
- updated
- review submitted

## 32. Monitoring / Subscriptions

Command:

```text
/monitor
```

Allow subscriptions by:

- repository
- organization
- branch
- workflow
- event type

Subscriptions must be persisted in PostgreSQL.

## 33. Monitoring Filters

Example:

```text
Monitor:
AfterlifeOS/rom

Only branch:
afl-16

Only workflows:
Build ROM
Build Kernel
```

## 34. Notification Preferences

Allow configuration:

```text
Notify workflow started: ON
Notify workflow success: ON
Notify workflow failure: ON
Notify push: OFF
Notify PR: ON
Notify issues: OFF
Notify releases: ON
```

## 35. Dashboard

Command:

```text
/dashboard
```

Display:

- repositories
- organizations
- active workflows
- failed workflows
- open PRs
- open issues
- recent activity

## 36. Global Search

Command:

```text
/search QUERY
```

Search:

- repositories
- commits
- issues
- PRs
- code
- users
- organizations

Categorize results.

## 37. Repository Creation

Optional write feature:

```text
/repo create
```

Fields:

- name
- description
- visibility
- initialize README

Confirmation required.

## 38. Repository File Editing

Support:

- edit file
- create file
- delete file

Flow:

```text
Repository
→ Files
→ File
→ Edit
```

Commit form:

- replacement content
- commit message
- branch

Confirmation required.

## 39. File Creation

Fields:

- path
- content
- branch
- commit message

Confirmation required.

## 40. File Deletion

Support deletion through GitHub API.

Must require:

1. explicit action
2. confirmation
3. repository/path display
4. branch display
5. commit message

## 41. Commit Creation

All file write operations must create normal GitHub commits.

Record:

- repository
- branch
- author
- commit SHA
- timestamp
- Telegram user ID
- operation

## 42. Webhook Security

Validate:

```text
X-Hub-Signature-256
X-GitHub-Event
X-GitHub-Delivery
```

Invalid signature:

```text
HTTP 401
```

Duplicate delivery:

```text
HTTP 200
already processed
```

## 43. Rate Limit Management

Implement GitHub API rate-limit tracking:

- limit
- remaining
- reset timestamp

Warn when rate limit is low.

Cache read-heavy data.

## 44. Error Handling

Never expose raw stack traces to Telegram.

Use clear user-facing errors and detailed internal structured logs.

## 45. Audit Logging

Every write operation must be logged.

Fields:

```text
id
telegram_user_id
github_user
repository
organization
operation
target
timestamp
success
error
github_request_id
```

Commands:

```text
/audit
```

Admin-only.

## 46. Database Schema

Minimum tables:

```text
users
github_installations
organizations
repositories
repository_subscriptions
workflow_subscriptions
notification_preferences
webhook_deliveries
audit_logs
github_rate_limits
sessions
```

Recommended fields:

### users

```text
id
telegram_user_id
github_user_id
github_username
is_admin
created_at
updated_at
```

### github_installations

```text
id
installation_id
account_id
account_login
account_type
permissions
created_at
updated_at
```

### repositories

```text
id
installation_id
github_repository_id
owner
name
full_name
default_branch
private
archived
created_at
updated_at
```

### subscriptions

```text
id
telegram_user_id
repository_id
event_type
branch_filter
workflow_filter
enabled
created_at
```

### webhook_deliveries

```text
id
delivery_id
event_type
repository
payload_hash
received_at
processed_at
status
```

## 47. Caching

Cache:

- repository metadata
- organization metadata
- branch lists
- workflow lists
- recent commits
- rate-limit information

Do not cache sensitive credentials.

Cache TTL must be configurable.

## 48. Telegram Message Management

Prefer one dashboard message with inline keyboard and edit it during navigation instead of spamming new messages.

## 49. Pagination

Every large list must support pagination.

Example:

```text
Commits 1-20

[⬅️] [1/15] [➡️]
```

Do not encode excessive data directly inside callback data.

## 50. Telegram Limits

Handle:

- message length
- callback query length
- file upload limits
- Markdown/HTML escaping
- rate limits

All GitHub-generated text must be escaped.

## 51. Markdown / HTML Safety

Implement:

```text
escapeTelegramMarkdown()
escapeTelegramHTML()
```

to prevent malformed messages and formatting injection.

## 52. Permission Model

Roles:

```text
ADMIN
USER
VIEWER
```

VIEWER can read and monitor.

USER can perform permitted write operations.

ADMIN can manage Telegram users, monitoring, installations, audit logs, and application settings.

## 53. Destructive Operation Confirmation

Confirmation required for:

```text
Delete repository
Delete branch
Delete tag
Delete release
Delete file
Merge PR
Cancel workflow
Rerun workflow
Create repository
Create release
Edit file
```

Highly destructive actions should require textual confirmation.

## 54. Background Workers

Long-running operations must not block Telegram update handling.

Use background jobs for:

- webhook processing
- GitHub API operations
- artifact processing
- large log retrieval
- notification delivery
- reconciliation

## 55. Webhook Event Queue

Recommended flow:

```text
GitHub
 ↓
Webhook HTTP Handler
 ↓
Validate Signature
 ↓
Persist Delivery
 ↓
Queue Event
 ↓
Worker
 ↓
Determine Subscribers
 ↓
Generate Notification
 ↓
Telegram
```

Webhook HTTP handler must return quickly.

## 56. Event Deduplication

Use GitHub delivery ID.

If delivery ID already exists, do not process the event again.

## 57. Reconciliation

Implement periodic reconciliation every configurable 5–15 minutes to:

- recover missed webhooks
- detect stale workflow states
- repair subscription state
- verify repository metadata

Do not use aggressive polling.

## 58. Health Checks

Expose:

```text
GET /health
GET /ready
```

Example:

```json
{
  "status": "ok",
  "database": "ok",
  "github": "ok",
  "telegram": "ok"
}
```

## 59. Metrics

Track:

```text
telegram_updates_total
telegram_errors_total
github_api_requests_total
github_api_errors_total
github_rate_limit_remaining
webhooks_received_total
webhooks_processed_total
webhooks_failed_total
notifications_sent_total
notifications_failed_total
workflow_events_total
command_execution_time
```

## 60. Logging

Use structured JSON logs.

Never log:

- Telegram bot token
- GitHub private key
- GitHub installation token
- webhook secret
- database password

## 61. Environment Variables

Required:

```text
TELEGRAM_BOT_TOKEN

GITHUB_APP_ID
GITHUB_PRIVATE_KEY
GITHUB_WEBHOOK_SECRET

DATABASE_URL

AUTHORIZED_TELEGRAM_USERS
```

Optional:

```text
REDIS_URL
LOG_LEVEL
LOG_FORMAT
WEBHOOK_BASE_URL
CACHE_TTL
RECONCILIATION_INTERVAL
ENVIRONMENT
```

## 62. Docker

Create production multi-stage Dockerfile.

Requirements:

- minimal runtime image
- non-root user
- healthcheck
- no secrets inside image
- configurable port

## 63. Railway Deployment

Provide:

```text
Dockerfile
railway.toml
```

Support:

- environment variables
- PostgreSQL connection
- public webhook URL
- health check
- automatic deployment from GitHub
- graceful shutdown

Application must not depend on local persistent filesystem storage.

## 64. Graceful Shutdown

Handle SIGTERM and SIGINT.

Shutdown order:

1. stop accepting new jobs
2. finish current safe operations
3. close webhook workers
4. close Telegram connection
5. close database
6. exit

## 65. Telegram Update Mode

Use Telegram webhook when a public HTTPS endpoint is available.

Recommended:

```text
POST /webhooks/telegram
POST /webhooks/github
```

## 66. Configuration Commands

Admin-only:

```text
/settings
/github/status
/github/installations
/monitor/list
/users
/audit
/health
```

## 67. GitHub Installation Management

Admin should inspect:

- Installation ID
- Account
- Account type
- Repository access
- Permissions
- Status

Gracefully handle:

- GitHub App uninstalled
- repository removed from installation
- permission changes
- organization access revoked

## 68. Failure Recovery

GitHub API failure:

- retry with exponential backoff

Telegram API failure:

- retry

Webhook processing failure:

- persist failure
- retry asynchronously

Avoid infinite retry loops.

## 69. Security Requirements

Mandatory:

- HTTPS
- GitHub webhook signature validation
- Telegram user authorization
- encrypted secrets
- least privilege GitHub permissions
- SQL parameterization
- input validation
- output escaping
- audit logs
- destructive action confirmation
- rate limiting
- request timeouts
- retry limits
- no secret logging

## 70. API Client Abstraction

Do not scatter GitHub API calls throughout Telegram handlers.

Create a GitHub service layer with methods for:

```text
GetUser()
GetOrganizations()
GetRepositories()
GetRepository()
GetContents()
GetFile()
GetCommits()
GetCommit()
CompareRefs()
GetBranches()
CreateBranch()
DeleteBranch()
GetPullRequests()
GetPullRequest()
MergePullRequest()
GetIssues()
CreateIssue()
GetReleases()
CreateRelease()
GetWorkflows()
GetWorkflowRuns()
GetWorkflowJobs()
GetWorkflowLogs()
RunWorkflow()
CancelWorkflow()
RerunWorkflow()
GetArtifacts()
GetDeployments()
```

Telegram handlers should call services rather than raw HTTP.

## 71. Telegram Handler Architecture

Separate:

```text
commands
callbacks
messages
middleware
authorization
formatters
keyboards
pagination
```

## 72. GitHub Service Architecture

Recommended:

```text
/internal/github/

auth/
client/
repositories/
organizations/
contents/
commits/
branches/
tags/
pullrequests/
issues/
actions/
releases/
deployments/
webhooks/
search/
```

## 73. Event Architecture

Recommended:

```text
/internal/events/

dispatcher/
handlers/
workflow/
repository/
push/
pullrequest/
issues/
release/
deployment/
```

## 74. Testing

### Unit tests

Test:

- GitHub authentication
- Telegram authorization
- Markdown escaping
- HTML escaping
- pagination
- webhook signature validation
- event deduplication
- permission checks
- confirmation flow

### Integration tests

Test:

- PostgreSQL
- GitHub API mock
- Telegram API mock
- webhook processing

### End-to-end

Test:

```text
Telegram command
→ GitHub API
→ response

GitHub webhook
→ event processor
→ Telegram notification
```

## 75. Webhook Test Cases

Test:

- valid signature
- invalid signature
- missing signature
- duplicate delivery
- unknown event
- malformed JSON
- repository deleted
- workflow completed
- workflow failed
- workflow cancelled
- PR opened
- PR merged
- push received
- release published

## 76. Actions Test Cases

Test:

- list workflows
- list runs
- running run
- successful run
- failed run
- cancelled run
- run workflow
- cancel workflow
- rerun workflow
- retrieve logs
- retrieve artifacts

## 77. UX Requirements

The bot must feel like an application rather than a CLI.

Use:

- inline keyboards
- breadcrumbs
- pagination
- back buttons
- refresh buttons
- contextual actions
- status icons
- compact summaries

## 78. Status Icons

Standardize:

```text
🟢 success
🔴 failure
🟡 running
⚪ cancelled
🔵 queued
⚠️ warning
⛔ unauthorized
❌ error
```

## 79. Realtime Dashboard

Dashboard must update based on incoming events.

Example:

```text
🟡 Build ROM #183

AfterlifeOS/rom
afl-16

Running:
17m 22s

Jobs:
✓ sync
✓ lunch
🟡 build
○ sign
○ upload
```

After completion:

```text
🟢 Build ROM #183

Completed:
48m 11s

Jobs:
✓ sync
✓ lunch
✓ build
✓ sign
✓ upload
```

## 80. Notification Throttling

Prevent notification spam.

Default:

```text
workflow started
workflow failed
workflow completed
workflow cancelled
```

Optional verbose mode:

```text
job started
job completed
check completed
```

## 81. User Preferences

Each Telegram user may configure:

- timezone
- notification verbosity
- default organization
- default repository
- default branch
- default page size

Default timezone:

```text
Asia/Jakarta
```

## 82. Command Reference

Implement at minimum:

```text
/start
/help

/account

/orgs
/org

/repos
/repo

/files
/file
/search

/commits
/commit
/compare

/branches
/tags

/prs
/pr

/issues
/issue

/releases
/release

/actions
/workflow
/run

/artifacts

/deployments

/monitor
/dashboard

/settings

/audit
/health
```

Commands may be supplemented by inline buttons.

## 83. Help System

Implement:

```text
/help
/help actions
```

Categories:

- Repository
- Git
- Actions
- Pull Requests
- Issues
- Monitoring
- Administration

## 84. Implementation Phases

### Phase 1 — Foundation

Tasks:

- [ ] Initialize repository
- [ ] Setup Go module
- [ ] Setup project structure
- [ ] Implement configuration system
- [ ] Implement structured logging
- [ ] Implement graceful shutdown
- [ ] Implement health endpoints
- [ ] Implement Dockerfile
- [ ] Implement Railway configuration
- [ ] Setup CI

Acceptance: application builds and runs in Docker.

### Phase 2 — PostgreSQL

Tasks:

- [ ] Configure PostgreSQL
- [ ] Create migration system
- [ ] Create users table
- [ ] Create installations table
- [ ] Create repositories table
- [ ] Create subscriptions table
- [ ] Create notification preferences table
- [ ] Create webhook deliveries table
- [ ] Create audit logs table
- [ ] Create indexes
- [ ] Add database health check

Acceptance: database migrations execute automatically and safely.

### Phase 3 — Telegram

Tasks:

- [ ] Implement Telegram client
- [ ] Implement authorization middleware
- [ ] Implement /start
- [ ] Implement /help
- [ ] Implement inline keyboards
- [ ] Implement callback routing
- [ ] Implement pagination
- [ ] Implement Markdown/HTML escaping
- [ ] Implement error messages
- [ ] Implement Telegram webhook
- [ ] Implement graceful update handling

Acceptance: authorized Telegram users can interact with the bot.

### Phase 4 — GitHub App Authentication

Tasks:

- [ ] Implement GitHub App JWT generation
- [ ] Implement installation token generation
- [ ] Implement token caching
- [ ] Implement token expiration handling
- [ ] Implement installation discovery
- [ ] Implement permission inspection
- [ ] Implement GitHub API client
- [ ] Implement rate-limit tracking
- [ ] Implement retry/backoff

Acceptance: bot can securely access repositories through GitHub App authentication.

### Phase 5 — Account & Organization

Tasks:

- [ ] Account endpoint
- [ ] Organization listing
- [ ] Organization details
- [ ] Organization repositories
- [ ] Organization members
- [ ] Organization teams
- [ ] Organization dashboard

Acceptance: bot can navigate GitHub account and organizations.

### Phase 6 — Repository Explorer

Tasks:

- [ ] Repository listing
- [ ] Repository details
- [ ] Directory browser
- [ ] File viewer
- [ ] Large file pagination
- [ ] Code search
- [ ] Branch listing
- [ ] Tag listing

Acceptance: user can browse repository contents entirely from Telegram.

### Phase 7 — Git History

Tasks:

- [ ] Commit history
- [ ] Commit details
- [ ] Changed files
- [ ] Commit diff
- [ ] Compare refs
- [ ] Branch comparison
- [ ] Pagination
- [ ] Diff pagination

Acceptance: user can inspect Git history and diffs without opening GitHub.

### Phase 8 — Pull Requests & Issues

Tasks:

- [ ] PR listing
- [ ] PR details
- [ ] PR commits
- [ ] PR diff
- [ ] PR checks
- [ ] PR comments
- [ ] PR approval
- [ ] PR merge
- [ ] Issue listing
- [ ] Issue details
- [ ] Create issue
- [ ] Comment
- [Close/reopen

Acceptance: user can manage PRs and issues through Telegram according to GitHub permissions.

### Phase 9 — GitHub Actions

Tasks:

- [ ] Workflow listing
- [ ] Workflow details
- [ ] Run listing
- [ ] Run details
- [ ] Job listing
- [ ] Job details
- [ ] Logs
- [ ] Log search
- [ ] Artifacts
- [ ] Run workflow
- [ ] Cancel workflow
- [ ] Rerun workflow
- [ ] Workflow inputs
- [ ] Branch selection

Acceptance: user can inspect and control GitHub Actions.

### Phase 10 — GitHub Webhooks

Tasks:

- [ ] Webhook HTTP endpoint
- [ ] Signature validation
- [ ] Delivery ID validation
- [ ] Event persistence
- [ ] Event deduplication
- [ ] Event queue
- [ ] Event dispatcher
- [ ] Push handler
- [ ] PR handler
- [ ] Issue handler
- [ ] Release handler
- [ ] Workflow run handler
- [ ] Workflow job handler
- [ ] Check run handler
- [ ] Deployment handler

Acceptance: GitHub events appear in Telegram without polling.

### Phase 11 — Monitoring

Tasks:

- [ ] Repository subscriptions
- [ ] Workflow subscriptions
- [ ] Branch filters
- [ ] Event filters
- [ ] Notification preferences
- [ ] Enable/disable monitoring
- [ ] Notification throttling
- [ ] Realtime workflow notifications
- [ ] Realtime PR notifications
- [ ] Realtime push notifications
- [ ] Realtime release notifications

Acceptance: user receives only configured GitHub notifications.

### Phase 12 — Write Operations

Tasks:

- [ ] Create repository
- [ ] Create branch
- [ ] Delete branch
- [ ] Create tag
- [ ] Delete tag
- [ ] Edit file
- [ ] Create file
- [ ] Delete file
- [ ] Create issue
- [ ] Edit issue
- [ ] Merge PR
- [ ] Create release
- [ ] Edit release
- [ ] Delete release
- [ ] Run workflow
- [ ] Cancel workflow
- [ ] Rerun workflow

Acceptance: every write operation validates GitHub permissions and requires confirmation where appropriate.

### Phase 13 — Security

Tasks:

- [ ] Telegram authorization
- [ ] Role-based authorization
- [ ] GitHub App least privilege
- [ ] Webhook signature verification
- [ ] Secret protection
- [ ] Input validation
- [ ] SQL parameterization
- [ ] Telegram output escaping
- [ ] Audit logs
- [ ] Destructive confirmation
- [ ] Rate limiting
- [ ] Request timeouts
- [ ] Retry limits
- [ ] Security tests

Acceptance: no secret or unauthorized GitHub data is exposed.

### Phase 14 — Reconciliation

Tasks:

- [ ] Periodic reconciliation worker
- [ ] Workflow state reconciliation
- [ ] Repository metadata reconciliation
- [ ] Missed webhook detection
- [ ] Failed event retry
- [ ] Dead-letter handling

Acceptance: temporary webhook/API failures do not permanently desynchronize monitoring state.

### Phase 15 — Production

Tasks:

- [ ] Production Docker image
- [ ] Railway deployment
- [ ] Environment variables
- [ ] PostgreSQL production setup
- [ ] HTTPS
- [ ] Telegram webhook
- [ ] GitHub webhook
- [ ] Health checks
- [ ] Structured logs
- [ ] Metrics
- [ ] Graceful shutdown
- [ ] Automatic restart
- [ ] CI/CD
- [ ] Database migrations
- [ ] Backup strategy

Acceptance: application can run continuously in production.

## 85. CI/CD

GitHub Actions must implement:

```text
lint
test
build
docker build
security scan
```

On main branch:

```text
test
→ build
→ docker build
→ publish image
```

Optional deployment:

```text
deploy Railway
```

## 86. GitHub Actions for the Bot

Create:

```text
.github/workflows/ci.yml
.github/workflows/docker.yml
.github/workflows/release.yml
```

CI runs on push and pull_request.

Release workflow runs on tags matching `v*`.

## 87. Documentation

Create:

```text
README.md

docs/
├── architecture.md
├── setup.md
├── github-app.md
├── telegram.md
├── deployment.md
├── security.md
├── commands.md
├── webhooks.md
└── troubleshooting.md
```

README must include:

- features
- architecture
- setup
- GitHub App configuration
- Telegram configuration
- environment variables
- Docker deployment
- Railway deployment
- webhook configuration
- security notes
- command reference

## 88. Definition of Done

The project is complete when:

1. Telegram bot runs continuously.
2. Authorized users can authenticate.
3. GitHub App authentication works.
4. Organizations can be viewed.
5. Repositories can be viewed.
6. Repository files can be browsed.
7. Files can be viewed safely.
8. Commits can be viewed.
9. Commit history works.
10. Commit diffs work.
11. Compare works.
12. Branches work.
13. Tags work.
14. PRs work.
15. Issues work.
16. Releases work.
17. GitHub Actions workflows work.
18. Workflow runs work.
19. Jobs work.
20. Logs work.
21. Artifacts work.
22. Workflow execution works.
23. Workflow cancellation works.
24. Workflow rerun works.
25. GitHub webhooks work.
26. Realtime workflow notifications work.
27. Realtime push notifications work.
28. Realtime PR notifications work.
29. Monitoring subscriptions work.
30. Notification preferences work.
31. Audit logging works.
32. Destructive confirmation works.
33. Rate-limit handling works.
34. Reconciliation works.
35. Database migrations work.
36. Docker deployment works.
37. Railway deployment works.
38. Health checks work.
39. CI passes.
40. No secrets are committed.
41. Security tests pass.
42. Documentation is complete.

## 89. Important Implementation Rules

DO NOT:

- use a GitHub PAT as the core authentication architecture
- hardcode secrets
- store GitHub installation tokens permanently without justification
- poll GitHub aggressively
- process webhook events synchronously when expensive
- trust Telegram callback data blindly
- execute destructive operations without confirmation
- expose GitHub API errors containing secrets
- log credentials
- send massive diffs/logs as one Telegram message
- couple Telegram handlers directly to GitHub HTTP requests
- put business logic inside Telegram callback handlers

DO:

- use GitHub App authentication
- use installation tokens
- validate webhook signatures
- deduplicate webhook deliveries
- use asynchronous event processing
- use pagination
- cache read-heavy operations
- track API rate limits
- implement retries with exponential backoff
- maintain audit logs
- use PostgreSQL
- use interfaces around GitHub services
- use structured logging
- implement health checks
- make all configuration environment-based
- keep Telegram UI compact
- make write operations permission-aware

## 90. Development Strategy

Implement incrementally.

Do NOT attempt to implement every feature in a single step.

Recommended order:

```text
Foundation
↓
Database
↓
Telegram
↓
GitHub App
↓
Repositories
↓
Commits/Diff
↓
PR/Issues
↓
Actions
↓
Webhooks
↓
Realtime Monitoring
↓
Write Operations
↓
Security Hardening
↓
Production
```

After every phase:

1. Run tests.
2. Build the application.
3. Verify the feature manually.
4. Update documentation.
5. Commit changes.
6. Do not proceed if the current phase is broken.

## 91. AI Builder Execution Instructions

The AI builder must behave as a senior backend engineer and implement the project incrementally.

Before coding:

1. Analyze the entire repository.
2. Create an implementation plan.
3. Identify required GitHub App permissions.
4. Identify required Telegram APIs.
5. Identify database schema.
6. Identify security risks.
7. Create task tracking.

During implementation:

1. Work phase-by-phase.
2. Keep changes modular.
3. Do not rewrite unrelated code.
4. Do not introduce unnecessary dependencies.
5. Prefer standard libraries when practical.
6. Verify API contracts against current official GitHub documentation when implementing GitHub integrations.
7. Add tests for every important service.
8. Run tests after each logical unit.
9. Never silently ignore errors.
10. Never fake API responses in production code.
11. Never hardcode credentials.
12. Never reduce security requirements to make implementation easier.

When an API capability is unavailable through GitHub App permissions, explicitly identify the required permission instead of implementing an insecure workaround.

## 92. Final Product Goal

The final product should feel like:

```text
Telegram
    +
GitHub App
    +
GitHub API
    +
GitHub Webhooks
    +
PostgreSQL
```

forming a complete:

# GitHub Control Center

From Telegram, the user should be able to:

- manage account
- browse organizations
- browse repositories
- browse files
- search code
- inspect commits
- compare branches
- inspect diffs
- manage branches
- manage tags
- manage pull requests
- manage issues
- manage releases
- monitor GitHub Actions
- monitor running workflows
- inspect logs
- manage artifacts
- run workflows
- cancel workflows
- rerun workflows
- subscribe to realtime events
- receive GitHub webhook notifications
- view dashboard
- maintain secure permissions
- audit write operations

The application must prioritize reliability, security, realtime event processing, low API usage, modularity, and production deployment readiness.
