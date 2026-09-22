package bot

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/control-center/github-telegram-control-center/internal/database"
	"github.com/control-center/github-telegram-control-center/internal/github"
	"github.com/control-center/github-telegram-control-center/internal/metrics"
	"github.com/control-center/github-telegram-control-center/internal/notifications"
	"github.com/control-center/github-telegram-control-center/internal/telegram"
)

func withMarkup(m *telegram.InlineKeyboardMarkup) notifications.SendOptions {
	return notifications.SendOptions{ReplyMarkup: m}
}

func randomID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// audit records an action for the given telegram user.
func (b *Bot) auditUser(ctx context.Context, tgID int64, operation, target string, ok bool, errMsg string) {
	entry := database.AuditEntry{TelegramUserID: &tgID, Operation: operation, Target: target, Success: ok}
	if errMsg != "" {
		entry.Error = errMsg
	}
	if err := b.store.Audit(ctx, entry); err != nil {
		b.log.Error("audit write failed", "err", err)
	}
}

// audit records an action from a message.
func (b *Bot) audit(ctx context.Context, msg *telegram.Message, operation, target string, ok bool, errMsg string) {
	b.auditUser(ctx, msg.From.ID, operation, target, ok, errMsg)
}

// ---- Notifications & subscriptions ----

func (b *Bot) cmdNotify(ctx context.Context, msg *telegram.Message) error {
	prefs, err := b.store.GetNotifPrefs(ctx, msg.From.ID)
	if err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString("<b>🔔 Notification preferences</b>\n")
	sb.WriteString(fmt.Sprintf("Workflow started: %s\n", boolOn(prefs.WorkflowStarted)))
	sb.WriteString(fmt.Sprintf("Workflow success: %s\n", boolOn(prefs.WorkflowSuccess)))
	sb.WriteString(fmt.Sprintf("Workflow failure: %s\n", boolOn(prefs.WorkflowFailure)))
	sb.WriteString(fmt.Sprintf("Workflow cancelled: %s\n", boolOn(prefs.WorkflowCancelled)))
	sb.WriteString(fmt.Sprintf("Push events: %s\n", boolOn(prefs.PushEvents)))
	sb.WriteString(fmt.Sprintf("PR events: %s\n", boolOn(prefs.PREvents)))
	sb.WriteString(fmt.Sprintf("Issue events: %s\n", boolOn(prefs.IssueEvents)))
	sb.WriteString(fmt.Sprintf("Release events: %s\n", boolOn(prefs.ReleaseEvents)))
	sb.WriteString(fmt.Sprintf("Verbose jobs: %s\n", boolOn(prefs.VerboseJobs)))
	kb := telegram.Markup(
		telegram.Row(
			telegram.CbButton(tof("Workflow started", prefs.WorkflowStarted), "notif:workflow_started"),
			telegram.CbButton(tof("Workflow success", prefs.WorkflowSuccess), "notif:workflow_success"),
		),
		telegram.Row(
			telegram.CbButton(tof("Workflow failure", prefs.WorkflowFailure), "notif:workflow_failure"),
			telegram.CbButton(tof("Cancelled", prefs.WorkflowCancelled), "notif:workflow_cancelled"),
		),
		telegram.Row(
			telegram.CbButton(tof("Push", prefs.PushEvents), "notif:push_events"),
			telegram.CbButton(tof("PR", prefs.PREvents), "notif:pr_events"),
		),
		telegram.Row(
			telegram.CbButton(tof("Issues", prefs.IssueEvents), "notif:issue_events"),
			telegram.CbButton(tof("Releases", prefs.ReleaseEvents), "notif:release_events"),
		),
		telegram.Row(telegram.CbButton(tof("Verbose jobs", prefs.VerboseJobs), "notif:verbose_jobs")),
	)
	return b.notifier.Send(ctx, msg.Chat.ID, sb.String(), withMarkup(kb))
}

func boolOn(v bool) string {
	if v {
		return "✅"
	}
	return "⬜"
}

func tof(label string, v bool) string {
	if v {
		return "✅ " + label
	}
	return "⬜ " + label
}

func (b *Bot) cbNotif(ctx context.Context, cq *telegram.CallbackQuery, chatID int64, mid int, data string) {
	field := strings.TrimSpace(data)
	if field == "view" {
		_ = b.notifier.Send(ctx, chatID, "Use /notify to view preferences.")
		return
	}
	if err := b.store.UpdateNotifPrefs(ctx, cq.From.ID, field, true); err != nil {
		b.log.Warn("update notif pref", "err", err)
		return
	}
	prefs, err := b.store.GetNotifPrefs(ctx, cq.From.ID)
	if err != nil {
		return
	}
	var sb strings.Builder
	sb.WriteString("<b>🔔 Notification preferences</b>\n")
	sb.WriteString(fmt.Sprintf("Workflow started: %s\n", boolOn(prefs.WorkflowStarted)))
	sb.WriteString(fmt.Sprintf("Workflow success: %s\n", boolOn(prefs.WorkflowSuccess)))
	sb.WriteString(fmt.Sprintf("Workflow failure: %s\n", boolOn(prefs.WorkflowFailure)))
	sb.WriteString(fmt.Sprintf("Workflow cancelled: %s\n", boolOn(prefs.WorkflowCancelled)))
	sb.WriteString(fmt.Sprintf("Push events: %s\n", boolOn(prefs.PushEvents)))
	sb.WriteString(fmt.Sprintf("PR events: %s\n", boolOn(prefs.PREvents)))
	sb.WriteString(fmt.Sprintf("Issue events: %s\n", boolOn(prefs.IssueEvents)))
	sb.WriteString(fmt.Sprintf("Release events: %s\n", boolOn(prefs.ReleaseEvents)))
	sb.WriteString(fmt.Sprintf("Verbose jobs: %s\n", boolOn(prefs.VerboseJobs)))
	kb := telegram.Markup(
		telegram.Row(
			telegram.CbButton(tof("Workflow started", prefs.WorkflowStarted), "notif:workflow_started"),
			telegram.CbButton(tof("Success", prefs.WorkflowSuccess), "notif:workflow_success"),
		),
		telegram.Row(
			telegram.CbButton(tof("Failure", prefs.WorkflowFailure), "notif:workflow_failure"),
			telegram.CbButton(tof("Cancelled", prefs.WorkflowCancelled), "notif:workflow_cancelled"),
		),
		telegram.Row(
			telegram.CbButton(tof("Push", prefs.PushEvents), "notif:push_events"),
			telegram.CbButton(tof("PR", prefs.PREvents), "notif:pr_events"),
			telegram.CbButton(tof("Issues", prefs.IssueEvents), "notif:issue_events"),
			telegram.CbButton(tof("Releases", prefs.ReleaseEvents), "notif:release_events"),
		),
		telegram.Row(telegram.CbButton(tof("Verbose jobs", prefs.VerboseJobs), "notif:verbose_jobs")),
	)
	_ = b.notifier.Edit(ctx, chatID, mid, sb.String(), kb)
}

// cmdSub subscribes to repo events.
func (b *Bot) cmdSub(ctx context.Context, msg *telegram.Message, args string) error {
	fields := strings.Fields(args)
	if len(fields) == 0 {
		return fmt.Errorf("usage: /sub <owner/repo> [event] [branch]")
	}
	repo := fields[0]
	event := "push"
	if len(fields) > 1 {
		event = fields[1]
	}
	var branch *string
	if len(fields) > 2 {
		bf := fields[2]
		branch = &bf
	}
	if err := b.store.AddRepositorySubscription(ctx, msg.From.ID, repo, event, branch); err != nil {
		return err
	}
	b.audit(ctx, msg, "subscribe", fmt.Sprintf("%s %s", repo, event), true, "")
	return b.notifier.Send(ctx, msg.Chat.ID,
		fmt.Sprintf("✅ Subscribed to <code>%s</code> (%s).", telegram.EscapeH(repo), telegram.EscapeH(event)))
}

// cmdUnsub removes a subscription.
func (b *Bot) cmdUnsub(ctx context.Context, msg *telegram.Message, args string) error {
	repo := strings.TrimSpace(args)
	if repo == "" {
		return fmt.Errorf("usage: /unsub <owner/repo> [event]")
	}
	event := "push"
	if f := strings.Fields(repo); len(f) > 1 {
		repo, event = f[0], f[1]
	}
	_ = event
	// store removes by repo and event
	if err := b.store.RemoveRepositorySubscription(ctx, msg.From.ID, repo); err != nil {
		return err
	}
	b.audit(ctx, msg, "unsubscribe", repo, true, "")
	return b.notifier.Send(ctx, msg.Chat.ID, fmt.Sprintf("⏹ Unsubscribed from <code>%s</code>.", telegram.EscapeH(repo)))
}

// cmdList shows subscriptions.
func (b *Bot) cmdList(ctx context.Context, msg *telegram.Message) error {
	subs, err := b.store.ListRepositorySubscriptions(ctx, msg.From.ID)
	if err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString("<b>📌 Subscriptions</b>\n")
	if len(subs) == 0 {
		sb.WriteString("(none)")
	}
	for _, s := range subs {
		bf := ""
		if s.BranchFilter != nil && *s.BranchFilter != "" {
			bf = " (" + *s.BranchFilter + ")"
		}
		sb.WriteString(fmt.Sprintf("• <code>%s</code> — %s%s\n", telegram.EscapeH(s.Repository), telegram.EscapeH(s.EventType), telegram.EscapeH(bf)))
	}
	return b.notifier.Send(ctx, msg.Chat.ID, sb.String())
}

// cmdAudit shows recent audit entries.
func (b *Bot) cmdAudit(ctx context.Context, msg *telegram.Message, args string) error {
	limit := 10
	if args != "" {
		if n, err := strconv.Atoi(args); err == nil && n >= 1 && n <= 100 {
			limit = n
		}
	}
	rows, err := b.store.ListAudit(ctx, limit)
	if err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString("<b>📜 Audit log</b>\n")
	for _, r := range rows {
		mark := "✅"
		if !r.Success {
			mark = "❌"
		}
		sb.WriteString(fmt.Sprintf("%s <b>%s</b> — %s (by %d)\n", mark, telegram.EscapeH(r.Operation), telegram.EscapeH(r.Target), r.TelegramUserID))
	}
	if len(rows) == 0 {
		sb.WriteString("(none)")
	}
	return b.notifier.Send(ctx, msg.Chat.ID, sb.String())
}

// cmdSet updates user prefs (page size, timezone).
func (b *Bot) cmdSet(ctx context.Context, msg *telegram.Message, args string) error {
	fields := strings.Fields(args)
	if len(fields) == 0 {
		return fmt.Errorf("usage: /set page <n> | timezone <TZ> | repo <owner/repo> | org <name>")
	}
	switch fields[0] {
	case "page":
		n, err := strconv.Atoi(fields[1])
		if err != nil || n < 5 {
			return fmt.Errorf("page size must be >= 5")
		}
		u, err := b.resolveUser(ctx, msg.From.ID)
		if err != nil {
			return err
		}
		ps := n
		if err := b.store.UpdateUserPrefs(ctx, msg.From.ID, nil, u.DefaultOrg, u.DefaultRepo, nil, &ps); err != nil {
			return err
		}
		return b.notifier.Send(ctx, msg.Chat.ID, fmt.Sprintf("✅ Page size = %d", n))
	case "timezone", "tz":
		u, err := b.resolveUser(ctx, msg.From.ID)
		if err != nil {
			return err
		}
		tz := fields[1]
		if err := b.store.UpdateUserPrefs(ctx, msg.From.ID, &tz, u.DefaultOrg, u.DefaultRepo, nil, nil); err != nil {
			return err
		}
		return b.notifier.Send(ctx, msg.Chat.ID, fmt.Sprintf("✅ Timezone saved: %s", telegram.EscapeH(tz)))
	case "repo":
		return b.cmdRepo(ctx, msg, strings.Join(fields[1:], " "))
	case "org":
		return b.cmdOrg(ctx, msg, strings.Join(fields[1:], " "))
	}
	return fmt.Errorf("unknown setting")
}

// cmdUsers lists telegram users (admin only).
func (b *Bot) cmdUsers(ctx context.Context, msg *telegram.Message) error {
	users, err := b.store.ListUsers(ctx)
	if err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString("<b>👥 Users</b>\n")
	for _, u := range users {
		role := "user"
		if u.Role == "admin" {
			role = "admin"
		}
		gh := ""
		if u.GitHubUsername != nil {
			gh = " @" + *u.GitHubUsername
		}
		sb.WriteString(fmt.Sprintf("• %d — %s%s\n", u.TelegramUserID, telegram.EscapeH(role), telegram.EscapeH(gh)))
	}
	return b.notifier.Send(ctx, msg.Chat.ID, sb.String())
}

// ---- Write operations ----

// cmdBranch creates or deletes a branch.
func (b *Bot) cmdBranch(ctx context.Context, msg *telegram.Message, args string) error {
	action, rest, _ := strings.Cut(strings.TrimSpace(args), " ")
	owner, repo, err := b.resolveRepo(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	installID, _, err := b.pickInstallation(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	switch action {
	case "create":
		name, from, _ := strings.Cut(rest, " ")
		name, from = strings.TrimSpace(name), strings.TrimSpace(from)
		if name == "" {
			return fmt.Errorf("usage: /branch create <name> [from-ref]")
		}
		if from == "" {
			defaultBranch, derr := b.defaultBranchSHA(ctx, installID, owner, repo)
			if derr != nil {
				return derr
			}
			from = defaultBranch
		}
		if err := b.gh.CreateBranch(ctx, installID, owner, repo, name, from); err != nil {
			return err
		}
		b.audit(ctx, msg, "create_branch", fmt.Sprintf("%s %s", trimRepo(owner, repo), name), true, "")
		return b.notifier.Send(ctx, msg.Chat.ID, fmt.Sprintf("✅ Branch <code>%s</code> created.", telegram.EscapeH(name)))
	case "delete":
		name := strings.TrimSpace(rest)
		if name == "" {
			return fmt.Errorf("usage: /branch delete <name>")
		}
		return b.confirmAndRun(ctx, "", msg, "delete_branch",
			fmt.Sprintf("Delete branch <code>%s</code> in %s?", telegram.EscapeH(name), trimRepo(owner, repo)),
			map[string]any{"owner": owner, "repo": repo, "name": name})
	}
	return fmt.Errorf("usage: /branch create|delete <name>")
}

// cmdTag creates or deletes a tag.
func (b *Bot) cmdTag(ctx context.Context, msg *telegram.Message, args string) error {
	action, rest, _ := strings.Cut(strings.TrimSpace(args), " ")
	owner, repo, err := b.resolveRepo(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	installID, _, err := b.pickInstallation(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	switch action {
	case "create":
		name, sha, _ := strings.Cut(rest, " ")
		name, sha = strings.TrimSpace(name), strings.TrimSpace(sha)
		if name == "" {
			return fmt.Errorf("usage: /tag create <name> [commit-sha]")
		}
		if sha == "" {
			sha, err = b.defaultBranchSHA(ctx, installID, owner, repo)
			if err != nil {
				return err
			}
		}
		msgTxt := fmt.Sprintf("Release %s", name)
		if err := b.gh.CreateTag(ctx, installID, owner, repo, name, sha, msgTxt); err != nil {
			return err
		}
		b.audit(ctx, msg, "create_tag", fmt.Sprintf("%s %s", trimRepo(owner, repo), name), true, "")
		return b.notifier.Send(ctx, msg.Chat.ID, fmt.Sprintf("✅ Tag <code>%s</code> created.", telegram.EscapeH(name)))
	case "delete":
		name := strings.TrimSpace(rest)
		if name == "" {
			return fmt.Errorf("usage: /tag delete <name>")
		}
		return b.confirmAndRun(ctx, "", msg, "delete_tag",
			fmt.Sprintf("Delete tag <code>%s</code> in %s?", telegram.EscapeH(name), trimRepo(owner, repo)),
			map[string]any{"owner": owner, "repo": repo, "name": name})
	}
	return fmt.Errorf("usage: /tag create|delete <name>")
}

// cmdRelease creates or deletes a release.
func (b *Bot) cmdRelease(ctx context.Context, msg *telegram.Message, args string) error {
	action, rest, _ := strings.Cut(strings.TrimSpace(args), " ")
	owner, repo, err := b.resolveRepo(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	installID, _, err := b.pickInstallation(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	switch action {
	case "create":
		fields := strings.Fields(rest)
		if len(fields) == 0 {
			return fmt.Errorf("usage: /release create <tag> [name]")
		}
		tag := fields[0]
		name := tag
		if len(fields) > 1 {
			name = strings.Join(fields[1:], " ")
		}
		rel, err := b.gh.CreateRelease(ctx, installID, owner, repo, tag, "HEAD", name, "", false, false)
		if err != nil {
			return err
		}
		b.audit(ctx, msg, "create_release", fmt.Sprintf("%s %s", trimRepo(owner, repo), tag), true, "")
		return b.notifier.Send(ctx, msg.Chat.ID, fmt.Sprintf("✅ Release <a href=\"%s\">%s</a> created.", telegram.EscapeH(rel.HTMLURL), telegram.EscapeH(tag)))
	case "delete":
		tag := strings.TrimSpace(rest)
		if tag == "" {
			return fmt.Errorf("usage: /release delete <tag>")
		}
		rel, err := b.gh.GetReleaseByTag(ctx, installID, owner, repo, tag)
		if err != nil {
			return err
		}
		return b.confirmAndRun(ctx, "", msg, "delete_release",
			fmt.Sprintf("Delete release <code>%s</code> in %s?", telegram.EscapeH(tag), trimRepo(owner, repo)),
			map[string]any{"owner": owner, "repo": repo, "num": rel.ID})
	}
	return fmt.Errorf("usage: /release create|delete <tag>")
}

// cmdFile reads, writes, or deletes a file.
func (b *Bot) cmdFile(ctx context.Context, msg *telegram.Message, args string) error {
	action, rest, _ := strings.Cut(strings.TrimSpace(args), " ")
	owner, repo, err := b.resolveRepo(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	installID, _, err := b.pickInstallation(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	switch action {
	case "read":
		path := strings.TrimSpace(rest)
		if path == "" {
			return fmt.Errorf("usage: /file read <path>")
		}
		return b.renderFile(ctx, msg.From.ID, msg.Chat.ID, owner, repo, path)
	case "write", "create":
		path, content, _ := strings.Cut(rest, " ")
		path = strings.TrimSpace(path)
		if path == "" || content == "" {
			return fmt.Errorf("usage: /file write <path> <content>")
		}
		sha := ""
		if c, err := b.gh.GetContents(ctx, installID, owner, repo, path, ""); err == nil {
			sha = c.SHA
		}
		if err := b.gh.CreateOrUpdateFile(ctx, installID, owner, repo, path, fmt.Sprintf("Update %s via Telegram", path), content, "HEAD", sha); err != nil {
			return err
		}
		b.audit(ctx, msg, "write_file", fmt.Sprintf("%s %s", trimRepo(owner, repo), path), true, "")
		return b.notifier.Send(ctx, msg.Chat.ID, fmt.Sprintf("✅ Wrote <code>%s</code>.", telegram.EscapeH(path)))
	case "delete":
		path := strings.TrimSpace(rest)
		if path == "" {
			return fmt.Errorf("usage: /file delete <path>")
		}
		if _, err := b.gh.GetContents(ctx, installID, owner, repo, path, ""); err != nil {
			return err
		}
		return b.confirmAndRun(ctx, "", msg, "delete_file",
			fmt.Sprintf("Delete file <code>%s</code> in %s?", telegram.EscapeH(path), trimRepo(owner, repo)),
			map[string]any{"owner": owner, "repo": repo, "path": path})
	}
	return fmt.Errorf("usage: /file read|write|delete <path> [content]")
}

func (b *Bot) renderFile(ctx context.Context, tgID, chatID int64, owner, repo, path string) error {
	installID, _, err := b.pickInstallation(ctx, tgID, "")
	if err != nil {
		return err
	}
	content, err := b.gh.GetContents(ctx, installID, owner, repo, path, "")
	if err != nil {
		return err
	}
	if content.Type == "file" {
		data, err := decodeBase64(content.Content)
		if err != nil {
			return err
		}
		if len(data) > 4000 {
			data = data[:4000]
		}
		return b.notifier.Send(ctx, chatID,
			fmt.Sprintf("<b>%s</b>\n<pre>%s</pre>", telegram.EscapeH(path), telegram.EscapeH(string(data))))
	}
	return b.notifier.Send(ctx, chatID, fmt.Sprintf("<code>%s</code> is a %s.", telegram.EscapeH(path), telegram.EscapeH(content.Type)))
}

// defaultBranchSHA returns the default branch's SHA for creating branches/tags.
func (b *Bot) defaultBranchSHA(ctx context.Context, installID int64, owner, repo string) (string, error) {
	br, err := b.gh.GetBranch(ctx, installID, owner, repo, "HEAD")
	if err != nil {
		// fall back to listing branches
		bs, _, err2 := b.gh.GetBranches(ctx, installID, owner, repo, github.ListOptions{Page: 0, PerPage: 1})
		if err2 != nil || len(bs) == 0 {
			return "", fmt.Errorf("could not resolve default branch: %v", err)
		}
		return bs[0].Commit.SHA, nil
	}
	return br.Commit.SHA, nil
}

// confirmAndRun shows an inline confirm for a write op (used from command handlers).
// When writing from command handlers we reuse the same confirm mechanism for consistency.
func (b *Bot) confirmAndRun(ctx context.Context, _ string, msg *telegram.Message, kind, question string, payload map[string]any) error {
	// command-driven confirmations are posted as a new message.
	sid := randomID()
	if err := b.store.CreateSession(ctx, sid, msg.From.ID, "confirm:"+kind, payload, confirmTTL); err != nil {
		return err
	}
	kb := telegram.Markup(
		telegram.Row(
			telegram.CbButton("✅ Confirm", "confirm:"+sid+":yes"),
			telegram.CbButton("❌ Cancel", "confirm:"+sid+":no"),
		),
	)
	return b.notifier.Send(ctx, msg.Chat.ID, question, withMarkup(kb))
}

func decodeBase64(s string) ([]byte, error) {
	return decodeBase64Std(s)
}

func decodeBase64Std(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// log helper shim kept minimal.
var _ = slog.LevelInfo
var _ = metrics.TelegramErrorsTotal
var _ = telegram.MaxMessageLen
