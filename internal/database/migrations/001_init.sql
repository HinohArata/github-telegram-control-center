CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    telegram_user_id BIGINT NOT NULL UNIQUE,
    github_user_id BIGINT,
    github_username TEXT,
    role TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('admin','user','viewer')),
    timezone TEXT NOT NULL DEFAULT 'Asia/Jakarta',
    default_org TEXT,
    default_repo TEXT,
    default_branch TEXT,
    page_size INT NOT NULL DEFAULT 20,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS github_installations (
    id BIGSERIAL PRIMARY KEY,
    installation_id BIGINT NOT NULL UNIQUE,
    account_id BIGINT,
    account_login TEXT,
    account_type TEXT,
    permissions JSONB NOT NULL DEFAULT '{}',
    repositories_selection TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS organizations (
    id BIGSERIAL PRIMARY KEY,
    github_org_id BIGINT NOT NULL,
    login TEXT NOT NULL,
    avatar_url TEXT,
    description TEXT,
    repos_url TEXT,
    html_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (github_org_id)
);

CREATE TABLE IF NOT EXISTS repositories (
    id BIGSERIAL PRIMARY KEY,
    installation_id BIGINT,
    github_repository_id BIGINT NOT NULL,
    owner TEXT NOT NULL,
    name TEXT NOT NULL,
    full_name TEXT NOT NULL UNIQUE,
    default_branch TEXT,
    private BOOLEAN NOT NULL DEFAULT false,
    archived BOOLEAN NOT NULL DEFAULT false,
    html_url TEXT,
    pushed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_repositories_owner ON repositories (owner);
CREATE INDEX IF NOT EXISTS idx_repositories_full_name ON repositories (full_name);

CREATE TABLE IF NOT EXISTS repository_subscriptions (
    id BIGSERIAL PRIMARY KEY,
    telegram_user_id BIGINT NOT NULL,
    repository_full_name TEXT NOT NULL,
    event_type TEXT NOT NULL DEFAULT '*',
    branch_filter TEXT,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (telegram_user_id, repository_full_name, event_type)
);

CREATE TABLE IF NOT EXISTS workflow_subscriptions (
    id BIGSERIAL PRIMARY KEY,
    telegram_user_id BIGINT NOT NULL,
    repository_full_name TEXT NOT NULL,
    workflow_filter TEXT,
    branch_filter TEXT,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (telegram_user_id, repository_full_name, workflow_filter)
);

CREATE TABLE IF NOT EXISTS notification_preferences (
    id BIGSERIAL PRIMARY KEY,
    telegram_user_id BIGINT NOT NULL UNIQUE,
    workflow_started BOOLEAN NOT NULL DEFAULT true,
    workflow_success BOOLEAN NOT NULL DEFAULT true,
    workflow_failure BOOLEAN NOT NULL DEFAULT true,
    workflow_cancelled BOOLEAN NOT NULL DEFAULT true,
    push_events BOOLEAN NOT NULL DEFAULT false,
    pr_events BOOLEAN NOT NULL DEFAULT true,
    issue_events BOOLEAN NOT NULL DEFAULT false,
    release_events BOOLEAN NOT NULL DEFAULT true,
    verbose_jobs BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id BIGSERIAL PRIMARY KEY,
    delivery_id TEXT NOT NULL UNIQUE,
    event_type TEXT NOT NULL,
    repository TEXT,
    payload_hash TEXT,
    received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','processing','processed','failed')),
    attempts INT NOT NULL DEFAULT 0,
    last_error TEXT
);

CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_status ON webhook_deliveries (status, received_at);

CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGSERIAL PRIMARY KEY,
    telegram_user_id BIGINT,
    github_user TEXT,
    repository TEXT,
    organization TEXT,
    operation TEXT NOT NULL,
    target TEXT,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT now(),
    success BOOLEAN NOT NULL DEFAULT false,
    error TEXT,
    github_request_id TEXT
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs (timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user ON audit_logs (telegram_user_id);

CREATE TABLE IF NOT EXISTS github_rate_limits (
    id BIGSERIAL PRIMARY KEY,
    resource TEXT NOT NULL UNIQUE,
    limit_val INT NOT NULL,
    remaining INT NOT NULL,
    reset_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    telegram_user_id BIGINT NOT NULL,
    kind TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions (telegram_user_id, kind);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions (expires_at);
