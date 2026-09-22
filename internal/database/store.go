package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	Pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{Pool: pool}
}

type User struct {
	ID             int64
	TelegramUserID int64
	GitHubUserID   *int64
	GitHubUsername *string
	Role           string
	Timezone       string
	DefaultOrg     *string
	DefaultRepo    *string
	DefaultBranch  *string
	PageSize       int
}

func (s *Store) EnsureUser(ctx context.Context, telegramUserID int64, role string) (*User, error) {
	if role == "" {
		role = "user"
	}
	var u User
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO users (telegram_user_id, role)
		VALUES ($1, $2)
		ON CONFLICT (telegram_user_id) DO UPDATE SET updated_at = now()
		RETURNING id, telegram_user_id, github_user_id, github_username, role, timezone,
		          default_org, default_repo, default_branch, page_size
	`, telegramUserID, role).Scan(
		&u.ID, &u.TelegramUserID, &u.GitHubUserID, &u.GitHubUsername, &u.Role,
		&u.Timezone, &u.DefaultOrg, &u.DefaultRepo, &u.DefaultBranch, &u.PageSize,
	)
	if err != nil {
		return nil, fmt.Errorf("ensure user: %w", err)
	}
	return &u, nil
}

func (s *Store) GetUser(ctx context.Context, telegramUserID int64) (*User, error) {
	var u User
	err := s.Pool.QueryRow(ctx, `
		SELECT id, telegram_user_id, github_user_id, github_username, role, timezone,
		       default_org, default_repo, default_branch, page_size
		FROM users WHERE telegram_user_id = $1
	`, telegramUserID).Scan(
		&u.ID, &u.TelegramUserID, &u.GitHubUserID, &u.GitHubUsername, &u.Role,
		&u.Timezone, &u.DefaultOrg, &u.DefaultRepo, &u.DefaultBranch, &u.PageSize,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) UpdateUserPrefs(ctx context.Context, telegramUserID int64, timezone, defaultOrg, defaultRepo, defaultBranch *string, pageSize *int) error {
	_, err := s.Pool.Exec(ctx, `
		UPDATE users SET
			timezone = COALESCE($2, timezone),
			default_org = COALESCE($3, default_org),
			default_repo = COALESCE($4, default_repo),
			default_branch = COALESCE($5, default_branch),
			page_size = COALESCE($6, page_size),
			updated_at = now()
		WHERE telegram_user_id = $1
	`, telegramUserID, timezone, defaultOrg, defaultRepo, defaultBranch, pageSize)
	return err
}

func (s *Store) LinkGitHubUser(ctx context.Context, telegramUserID, githubUserID int64, username string) error {
	_, err := s.Pool.Exec(ctx, `
		UPDATE users SET github_user_id=$2, github_username=$3, updated_at=now()
		WHERE telegram_user_id=$1
	`, telegramUserID, githubUserID, username)
	return err
}

func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, telegram_user_id, github_user_id, github_username, role, timezone,
		       default_org, default_repo, default_branch, page_size
		FROM users ORDER BY telegram_user_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.TelegramUserID, &u.GitHubUserID, &u.GitHubUsername, &u.Role,
			&u.Timezone, &u.DefaultOrg, &u.DefaultRepo, &u.DefaultBranch, &u.PageSize); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

type Installation struct {
	ID             int64
	InstallationID int64
	AccountID      *int64
	AccountLogin   *string
	AccountType    *string
	Permissions    map[string]any
	Selection      *string
}

func (s *Store) UpsertInstallation(ctx context.Context, installationID, accountID int64, login, accountType, selection string, permissions map[string]any) error {
	if permissions == nil {
		permissions = map[string]any{}
	}
	permJSON, err := json.Marshal(permissions)
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, `
		INSERT INTO github_installations (installation_id, account_id, account_login, account_type, permissions, repositories_selection)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (installation_id) DO UPDATE SET
			account_id=EXCLUDED.account_id,
			account_login=EXCLUDED.account_login,
			account_type=EXCLUDED.account_type,
			permissions=EXCLUDED.permissions,
			repositories_selection=EXCLUDED.repositories_selection,
			updated_at=now()
	`, installationID, accountID, login, accountType, permJSON, selection)
	return err
}

func (s *Store) ListInstallations(ctx context.Context) ([]Installation, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, installation_id, account_id, account_login, account_type, permissions, repositories_selection
		FROM github_installations ORDER BY installation_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Installation
	for rows.Next() {
		var i Installation
		var permJSON []byte
		if err := rows.Scan(&i.ID, &i.InstallationID, &i.AccountID, &i.AccountLogin, &i.AccountType, &permJSON, &i.Selection); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(permJSON, &i.Permissions)
		out = append(out, i)
	}
	return out, rows.Err()
}

func (s *Store) DeleteInstallation(ctx context.Context, installationID int64) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM github_installations WHERE installation_id=$1`, installationID)
	return err
}

func (s *Store) UpsertRepository(ctx context.Context, githubRepoID int64, owner, name, fullName, defaultBranch, htmlURL string, private, archived bool, installationID *int64, pushedAt *time.Time) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO repositories (installation_id, github_repository_id, owner, name, full_name, default_branch, private, archived, html_url, pushed_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (full_name) DO UPDATE SET
			installation_id=EXCLUDED.installation_id,
			default_branch=EXCLUDED.default_branch,
			private=EXCLUDED.private,
			archived=EXCLUDED.archived,
			html_url=EXCLUDED.html_url,
			pushed_at=EXCLUDED.pushed_at,
			updated_at=now()
	`, installationID, githubRepoID, owner, name, fullName, defaultBranch, private, archived, htmlURL, pushedAt)
	return err
}

type Subscription struct {
	ID             int64
	TelegramUserID int64
	Repository     string
	EventType      string
	BranchFilter   *string
	WorkflowFilter *string
	Enabled        bool
	CreatedAt      time.Time
}

func (s *Store) AddRepositorySubscription(ctx context.Context, telegramUserID int64, repo, eventType string, branchFilter *string) error {
	if eventType == "" {
		eventType = "*"
	}
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO repository_subscriptions (telegram_user_id, repository_full_name, event_type, branch_filter)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (telegram_user_id, repository_full_name, event_type)
		DO UPDATE SET branch_filter=EXCLUDED.branch_filter, enabled=true
	`, telegramUserID, repo, eventType, branchFilter)
	return err
}

func (s *Store) RemoveRepositorySubscription(ctx context.Context, telegramUserID int64, repo string) error {
	_, err := s.Pool.Exec(ctx, `
		DELETE FROM repository_subscriptions WHERE telegram_user_id=$1 AND repository_full_name=$2
	`, telegramUserID, repo)
	return err
}

func (s *Store) ListRepositorySubscriptions(ctx context.Context, telegramUserID int64) ([]Subscription, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, telegram_user_id, repository_full_name, event_type, branch_filter, enabled, created_at
		FROM repository_subscriptions WHERE telegram_user_id=$1 ORDER BY repository_full_name
	`, telegramUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Subscription
	for rows.Next() {
		var sub Subscription
		if err := rows.Scan(&sub.ID, &sub.TelegramUserID, &sub.Repository, &sub.EventType, &sub.BranchFilter, &sub.Enabled, &sub.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, sub)
	}
	return out, rows.Err()
}

type NotificationTarget struct {
	TelegramUserID int64
	BranchFilter   *string
}

func (s *Store) ListNotificationSubscribers(ctx context.Context, repo, eventType string) ([]NotificationTarget, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT DISTINCT sub.telegram_user_id, sub.branch_filter
		FROM repository_subscriptions sub
		JOIN users u ON u.telegram_user_id = sub.telegram_user_id
		WHERE sub.repository_full_name = $1
		  AND sub.enabled = true
		  AND (sub.event_type = '*' OR sub.event_type = $2)
	`, repo, eventType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []NotificationTarget
	for rows.Next() {
		var t NotificationTarget
		if err := rows.Scan(&t.TelegramUserID, &t.BranchFilter); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

type WorkflowTarget struct {
	TelegramUserID int64
	WorkflowFilter *string
	BranchFilter   *string
}

func (s *Store) ListWorkflowSubscribers(ctx context.Context, repo string) ([]WorkflowTarget, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT sub.telegram_user_id, sub.workflow_filter, sub.branch_filter
		FROM workflow_subscriptions sub
		JOIN users u ON u.telegram_user_id = sub.telegram_user_id
		WHERE sub.repository_full_name = $1 AND sub.enabled = true
	`, repo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WorkflowTarget
	for rows.Next() {
		var t WorkflowTarget
		var wf, bf *string
		if err := rows.Scan(&t.TelegramUserID, &wf, &bf); err != nil {
			return nil, err
		}
		if wf != nil && *wf != "" {
			t.WorkflowFilter = wf
		}
		if bf != nil && *bf != "" {
			t.BranchFilter = bf
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) AddWorkflowSubscription(ctx context.Context, telegramUserID int64, repo, workflowFilter, branchFilter *string) error {
	var wf any
	if workflowFilter != nil {
		wf = *workflowFilter
	}
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO workflow_subscriptions (telegram_user_id, repository_full_name, workflow_filter, branch_filter)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (telegram_user_id, repository_full_name, workflow_filter)
		DO UPDATE SET branch_filter=EXCLUDED.branch_filter, enabled=true
	`, telegramUserID, repo, wf, branchFilter)
	return err
}

func (s *Store) ListWorkflowSubscriptions(ctx context.Context, telegramUserID int64) ([]Subscription, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, telegram_user_id, repository_full_name, COALESCE(workflow_filter,''), branch_filter, enabled, created_at
		FROM workflow_subscriptions WHERE telegram_user_id=$1 ORDER BY repository_full_name
	`, telegramUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Subscription
	for rows.Next() {
		var sub Subscription
		if err := rows.Scan(&sub.ID, &sub.TelegramUserID, &sub.Repository, &sub.WorkflowFilter, &sub.BranchFilter, &sub.Enabled, &sub.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, sub)
	}
	return out, rows.Err()
}

type NotifPrefs struct {
	TelegramUserID    int64
	WorkflowStarted   bool
	WorkflowSuccess   bool
	WorkflowFailure   bool
	WorkflowCancelled bool
	PushEvents        bool
	PREvents          bool
	IssueEvents       bool
	ReleaseEvents     bool
	VerboseJobs       bool
}

func (s *Store) GetNotifPrefs(ctx context.Context, telegramUserID int64) (*NotifPrefs, error) {
	p := &NotifPrefs{TelegramUserID: telegramUserID, WorkflowStarted: true, WorkflowSuccess: true, WorkflowFailure: true, WorkflowCancelled: true, PREvents: true, ReleaseEvents: true}
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO notification_preferences (telegram_user_id)
		VALUES ($1)
		ON CONFLICT (telegram_user_id) DO UPDATE SET updated_at = notification_preferences.updated_at
		RETURNING workflow_started, workflow_success, workflow_failure, workflow_cancelled,
		          push_events, pr_events, issue_events, release_events, verbose_jobs
	`, telegramUserID).Scan(
		&p.WorkflowStarted, &p.WorkflowSuccess, &p.WorkflowFailure, &p.WorkflowCancelled,
		&p.PushEvents, &p.PREvents, &p.IssueEvents, &p.ReleaseEvents, &p.VerboseJobs,
	)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Store) UpdateNotifPrefs(ctx context.Context, telegramUserID int64, field string, value bool) error {
	allowed := map[string]bool{
		"workflow_started": true, "workflow_success": true, "workflow_failure": true,
		"workflow_cancelled": true, "push_events": true, "pr_events": true,
		"issue_events": true, "release_events": true, "verbose_jobs": true,
	}
	if !allowed[field] {
		return fmt.Errorf("invalid preference field %q", field)
	}
	_, err := s.Pool.Exec(ctx, fmt.Sprintf(
		`UPDATE notification_preferences SET %s=$2, updated_at=now() WHERE telegram_user_id=$1`,
		field,
	), telegramUserID, value)
	return err
}

func (s *Store) RecordDelivery(ctx context.Context, deliveryID, eventType, repository, payloadHash string) (inserted bool, err error) {
	err = s.Pool.QueryRow(ctx, `
		INSERT INTO webhook_deliveries (delivery_id, event_type, repository, payload_hash)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (delivery_id) DO NOTHING
		RETURNING true
	`, deliveryID, eventType, repository, payloadHash).Scan(&inserted)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) MarkDeliveryProcessed(ctx context.Context, deliveryID string) error {
	_, err := s.Pool.Exec(ctx, `
		UPDATE webhook_deliveries SET status='processed', processed_at=now() WHERE delivery_id=$1
	`, deliveryID)
	return err
}

func (s *Store) MarkDeliveryFailed(ctx context.Context, deliveryID string, attempts int, lastErr string) error {
	status := "failed"
	if attempts < 5 {
		status = "pending"
	}
	_, err := s.Pool.Exec(ctx, `
		UPDATE webhook_deliveries SET status=$2, attempts=$3, last_error=$4 WHERE delivery_id=$1
	`, deliveryID, status, attempts, lastErr)
	return err
}

func (s *Store) ListPendingDeliveries(ctx context.Context, limit int) ([]Delivery, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT delivery_id, event_type, COALESCE(repository,''), attempts
		FROM webhook_deliveries
		WHERE status='pending'
		ORDER BY received_at
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Delivery
	for rows.Next() {
		var d Delivery
		if err := rows.Scan(&d.DeliveryID, &d.EventType, &d.Repository, &d.Attempts); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

type Delivery struct {
	DeliveryID string
	EventType  string
	Repository string
	Attempts   int
}

type AuditEntry struct {
	TelegramUserID  *int64
	GitHubUser      string
	Repository      string
	Organization    string
	Operation       string
	Target          string
	Success         bool
	Error           string
	GitHubRequestID string
}

func (s *Store) Audit(ctx context.Context, e AuditEntry) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO audit_logs (telegram_user_id, github_user, repository, organization, operation, target, success, error, github_request_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,NULLIF($8,''),NULLIF($9,''))
	`, e.TelegramUserID, nullStr(e.GitHubUser), nullStr(e.Repository), nullStr(e.Organization),
		e.Operation, nullStr(e.Target), e.Success, e.Error, e.GitHubRequestID)
	return err
}

func (s *Store) ListAudit(ctx context.Context, limit int) ([]AuditRow, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, COALESCE(telegram_user_id,0), COALESCE(github_user,''), COALESCE(repository,''),
		       COALESCE(organization,''), operation, COALESCE(target,''), timestamp, success,
		       COALESCE(error,''), COALESCE(github_request_id,'')
		FROM audit_logs ORDER BY timestamp DESC LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AuditRow
	for rows.Next() {
		var r AuditRow
		if err := rows.Scan(&r.ID, &r.TelegramUserID, &r.GitHubUser, &r.Repository, &r.Organization,
			&r.Operation, &r.Target, &r.Timestamp, &r.Success, &r.Error, &r.GitHubRequestID); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type AuditRow struct {
	ID              int64
	TelegramUserID  int64
	GitHubUser      string
	Repository      string
	Organization    string
	Operation       string
	Target          string
	Timestamp       time.Time
	Success         bool
	Error           string
	GitHubRequestID string
}

func (s *Store) SaveRateLimit(ctx context.Context, resource string, limit, remaining int, resetAt time.Time) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO github_rate_limits (resource, limit_val, remaining, reset_at, updated_at)
		VALUES ($1,$2,$3,$4,now())
		ON CONFLICT (resource) DO UPDATE SET
			limit_val=EXCLUDED.limit_val,
			remaining=EXCLUDED.remaining,
			reset_at=EXCLUDED.reset_at,
			updated_at=now()
	`, resource, limit, remaining, resetAt)
	return err
}

func (s *Store) GetRateLimit(ctx context.Context, resource string) (limit, remaining int, resetAt time.Time, err error) {
	err = s.Pool.QueryRow(ctx, `
		SELECT limit_val, remaining, reset_at FROM github_rate_limits WHERE resource=$1
	`, resource).Scan(&limit, &remaining, &resetAt)
	return
}

func (s *Store) CreateSession(ctx context.Context, id string, telegramUserID int64, kind string, payload map[string]any, ttl time.Duration) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, `
		INSERT INTO sessions (id, telegram_user_id, kind, payload, expires_at)
		VALUES ($1,$2,$3,$4, now() + $5::interval)
		ON CONFLICT (id) DO UPDATE SET payload=EXCLUDED.payload, expires_at=EXCLUDED.expires_at, kind=EXCLUDED.kind
	`, id, telegramUserID, kind, body, fmt.Sprintf("%f seconds", ttl.Seconds()))
	return err
}

func (s *Store) GetSession(ctx context.Context, id string) (telegramUserID int64, kind string, payload map[string]any, err error) {
	var body []byte
	err = s.Pool.QueryRow(ctx, `
		SELECT telegram_user_id, kind, payload FROM sessions WHERE id=$1 AND expires_at > now()
	`, id).Scan(&telegramUserID, &kind, &body)
	if err != nil {
		return 0, "", nil, err
	}
	payload = map[string]any{}
	_ = json.Unmarshal(body, &payload)
	return
}

func (s *Store) DeleteSession(ctx context.Context, id string) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM sessions WHERE id=$1`, id)
	return err
}

func (s *Store) DeleteExpiredSessions(ctx context.Context) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM sessions WHERE expires_at <= now()`)
	return err
}

func (s *Store) Ping(ctx context.Context) error {
	return s.Pool.Ping(ctx)
}

func nullStr(v string) any {
	if v == "" {
		return nil
	}
	return v
}
