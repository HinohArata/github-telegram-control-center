package events

import (
	"context"
	"fmt"
	"strings"

	"github.com/control-center/github-telegram-control-center/internal/telegram"
)

func shortSHA(s string) string {
	if len(s) > 7 {
		return s[:7]
	}
	return s
}

func brief(msg string) string {
	if len(msg) > 80 {
		return msg[:80] + "…"
	}
	return msg
}

type repoCommon struct {
	FullName string `json:"full_name"`
	HTMLURL  string `json:"html_url"`
}

type pushEvent struct {
	Ref        string `json:"ref"`
	Before     string `json:"before"`
	After      string `json:"after"`
	Pusher     struct{ Login string } `json:"pusher"`
	Repository repoCommon
	HeadCommit struct {
		ID      string `json:"id"`
		Message string `json:"message"`
		URL     string `json:"url"`
	} `json:"head_commit"`
	Commits []struct {
		ID      string `json:"id"`
		Message string `json:"message"`
		URL     string `json:"url"`
		Actor   struct{ Login string } `json:"author"`
	} `json:"commits"`
}

func (l *Listener) handlePush(ctx context.Context, job *Job) error {
	var e pushEvent
	if err := parseBody(job.Body, &e); err != nil {
		return err
	}
	if e.Repository.FullName == "" {
		return nil
	}
	branch := strings.TrimPrefix(e.Ref, "refs/heads/")
	var b strings.Builder
	b.WriteString("🔨 <b>Push</b> → " + telegram.EscapeHTML(e.Repository.FullName) + "\n")
	b.WriteString("🌿 Branch: " + telegram.EscapeHTML(branch) + "\n")
	if e.Pusher.Login != "" {
		b.WriteString("👤 Pusher: " + telegram.EscapeHTML(e.Pusher.Login) + "\n")
	}
	if len(e.Commits) > 0 {
		for i, c := range e.Commits {
			if i >= 5 {
				b.WriteString(fmt.Sprintf("… and %d more commits\n", len(e.Commits)-5))
				break
			}
			b.WriteString(fmt.Sprintf("   <code>%s</code> %s\n", shortSHA(c.ID), telegram.EscapeHTML(brief(strings.Split(c.Message, "\n")[0]))))
		}
	} else if e.HeadCommit.ID != "" {
		b.WriteString(fmt.Sprintf("   <code>%s</code> %s\n", shortSHA(e.HeadCommit.ID), telegram.EscapeHTML(brief(strings.Split(e.HeadCommit.Message, "\n")[0]))))
	}
	b.WriteString("🔗 " + e.Repository.HTMLURL)
	l.sendToSubscribers(ctx, e.Repository.FullName, "push", b.String(), nil, branch)
	return nil
}

type pullRequestEvent struct {
	Action     string `json:"action"`
	Number     int    `json:"number"`
	Repository repoCommon
	PullRequest struct {
		Title   string `json:"title"`
		HTMLURL string `json:"html_url"`
		User    struct{ Login string } `json:"user"`
		Body    string `json:"body"`
	} `json:"pull_request"`
}

func (l *Listener) handlePullRequest(ctx context.Context, job *Job) error {
	var e pullRequestEvent
	if err := parseBody(job.Body, &e); err != nil {
		return err
	}
	if e.Repository.FullName == "" {
		return nil
	}
	icons := map[string]string{"opened": "🟢", "closed": "🔴", "reopened": "🔁", "merged": "🎉", "edited": "✏️", "synchronize": "⚡"}
	icon := icons[e.Action]
	if icon == "" {
		icon = "🐙"
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s <b>PR #%d</b> %s in %s\n", icon, e.Number, telegram.EscapeHTML(e.Action), telegram.EscapeHTML(e.Repository.FullName)))
	b.WriteString("📌 <a href=\"" + e.PullRequest.HTMLURL + "\">" + telegram.EscapeHTML(e.PullRequest.Title) + "</a>\n")
	if e.PullRequest.User.Login != "" {
		b.WriteString("👤 " + telegram.EscapeHTML(e.PullRequest.User.Login) + "\n")
	}
	if e.PullRequest.Body != "" && e.Action == "opened" {
		b.WriteString("💬 " + telegram.EscapeHTML(brief(e.PullRequest.Body)) + "\n")
	}
	l.sendToSubscribers(ctx, e.Repository.FullName, "pull_request", b.String(), nil, "")
	return nil
}

type issueEvent struct {
	Action     string `json:"action"`
	Issue      struct {
		Number   int    `json:"number"`
		Title    string `json:"title"`
		HTMLURL  string `json:"html_url"`
		User     struct{ Login string } `json:"user"`
	}
	Repository repoCommon
}

func (l *Listener) handleIssues(ctx context.Context, job *Job) error {
	var e issueEvent
	if err := parseBody(job.Body, &e); err != nil {
		return err
	}
	if e.Repository.FullName == "" {
		return nil
	}
	icon := "🐛"
	switch e.Action {
	case "opened":
		icon = "🆕"
	case "closed":
		icon = "✅"
	case "reopened":
		icon = "🔁"
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s <b>Issue #%d</b> %s in %s\n", icon, e.Issue.Number, telegram.EscapeHTML(e.Action), telegram.EscapeHTML(e.Repository.FullName)))
	b.WriteString("📌 <a href=\"" + e.Issue.HTMLURL + "\">" + telegram.EscapeHTML(e.Issue.Title) + "</a>\n")
	if e.Issue.User.Login != "" {
		b.WriteString("👤 " + telegram.EscapeHTML(e.Issue.User.Login) + "\n")
	}
	l.sendToSubscribers(ctx, e.Repository.FullName, "issues", b.String(), nil, "")
	return nil
}

type issueCommentEvent struct {
	Action     string `json:"action"`
	Repository repoCommon
	Issue struct {
		Number int `json:"number"`
		Title  string `json:"title"`
	}
	Comment struct {
		HTMLURL string `json:"html_url"`
		User    struct{ Login string } `json:"user"`
		Body    string `json:"body"`
	}
}

func (l *Listener) handleIssueComment(ctx context.Context, job *Job) error {
	var e issueCommentEvent
	if err := parseBody(job.Body, &e); err != nil {
		return err
	}
	if e.Repository.FullName == "" {
		return nil
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("💬 <b>Comment</b> %s on issue #%d in %s\n", telegram.EscapeHTML(e.Action), e.Issue.Number, telegram.EscapeHTML(e.Repository.FullName)))
	b.WriteString("📌 <a href=\"" + e.Comment.HTMLURL + "\">" + telegram.EscapeHTML(e.Issue.Title) + "</a>\n")
	if e.Comment.User.Login != "" {
		b.WriteString("👤 " + telegram.EscapeHTML(e.Comment.User.Login) + "\n")
	}
	if e.Comment.Body != "" {
		b.WriteString("💭 " + telegram.EscapeHTML(brief(e.Comment.Body)) + "\n")
	}
	l.sendToSubscribers(ctx, e.Repository.FullName, "issue_comment", b.String(), nil, "")
	return nil
}

type workflowRunEvent struct {
	Action     string `json:"action"`
	Repository repoCommon
	Sender     struct{ Login string }
	WorkflowRun struct {
		ID             int64  `json:"id"`
		Name           string `json:"name"`
		DisplayName    string `json:"display_title"`
		HeadBranch     string `json:"head_branch"`
		Status         string `json:"status"`
		Conclusion     string `json:"conclusion"`
		HTMLURL        string `json:"html_url"`
		RunNumber      int    `json:"run_number"`
		Event          string `json:"event"`
		RunAttempt     int    `json:"run_attempt"`
		WorkflowID     int64  `json:"workflow_id"`
	} `json:"workflow_run"`
}

func (l *Listener) handleWorkflowRun(ctx context.Context, job *Job) error {
	var e workflowRunEvent
	if err := parseBody(job.Body, &e); err != nil {
		return err
	}
	if e.Repository.FullName == "" {
		return nil
	}
	targets, err := l.store.ListWorkflowSubscribers(ctx, e.Repository.FullName)
	if err != nil {
		return err
	}
	icon := "⚙️"
	var conclusion string
	var prefField string
	switch e.Action {
	case "requested":
		icon = "▶️"
		conclusion = "started"
		prefField = "workflow_started"
	case "completed":
		prefField = "workflow_" + e.WorkflowRun.Conclusion
		switch e.WorkflowRun.Conclusion {
		case "success":
			icon = "✅"
			conclusion = "succeeded"
		case "failure":
			icon = "❌"
			conclusion = "failed"
		case "cancelled":
			icon = "🚫"
			conclusion = "cancelled"
		case "skipped":
			icon = "⏭️"
			conclusion = "skipped"
		default:
			icon = "🏁"
			conclusion = e.WorkflowRun.Conclusion
		}
	default:
		return nil
	}
	content := fmt.Sprintf("%s <b>Workflow %s</b> → %s\n", icon, telegram.EscapeHTML(conclusion), telegram.EscapeHTML(e.WorkflowRun.Name))
	content += "📦 <b>" + telegram.EscapeHTML(e.Repository.FullName) + "</b>\n"
	content += fmt.Sprintf("🌿 Branch: %s · <code>#%d</code>\n", telegram.EscapeHTML(e.WorkflowRun.HeadBranch), e.WorkflowRun.RunNumber)
	if e.WorkflowRun.Event != "" {
		content += "⚡ Trigger: " + telegram.EscapeHTML(e.WorkflowRun.Event) + "\n"
	}
	if e.Sender.Login != "" {
		content += "👤 " + telegram.EscapeHTML(e.Sender.Login) + "\n"
	}
	content += "🔗 <a href=\"" + e.WorkflowRun.HTMLURL + "\">View run</a>"
	for _, t := range targets {
		if t.WorkflowFilter != nil && *t.WorkflowFilter != "" {
			if !branchMatches(*t.WorkflowFilter, e.WorkflowRun.Name) && !strings.Contains(e.WorkflowRun.Name, *t.WorkflowFilter) {
				continue
			}
		}
		if t.BranchFilter != nil && *t.BranchFilter != "" {
			if !branchMatches(*t.BranchFilter, e.WorkflowRun.HeadBranch) {
				continue
			}
		}
		prefs, err := l.store.GetNotifPrefs(ctx, t.TelegramUserID)
		if err != nil {
			l.log.Error("get notif prefs failed", "telegram_user_id", t.TelegramUserID, "err", err)
			continue
		}
		switch prefField {
		case "workflow_started":
			if !prefs.WorkflowStarted {
				continue
			}
		case "workflow_success":
			if !prefs.WorkflowSuccess {
				continue
			}
		case "workflow_failure":
			if !prefs.WorkflowFailure {
				continue
			}
		case "workflow_cancelled":
			if !prefs.WorkflowCancelled {
				continue
			}
		default:
			if !prefs.WorkflowStarted {
				continue
			}
		}
		if err := l.notifier.Send(ctx, t.TelegramUserID, content); err != nil {
			l.log.Error("notify workflow subscriber failed", "telegram_user_id", t.TelegramUserID, "err", err)
		}
	}
	return nil
}

type releaseEvent struct {
	Action     string `json:"action"`
	Repository repoCommon
	Release struct {
		TagName   string `json:"tag_name"`
		Name      string `json:"name"`
		HTMLURL   string `json:"html_url"`
		Prerelease bool  `json:"prerelease"`
		Draft     bool  `json:"draft"`
		Author    struct{ Login string } `json:"author"`
	} `json:"release"`
}

func (l *Listener) handleRelease(ctx context.Context, job *Job) error {
	var e releaseEvent
	if err := parseBody(job.Body, &e); err != nil {
		return err
	}
	if e.Repository.FullName == "" {
		return nil
	}
	switch e.Action {
	case "published", "released", "created":
	default:
		return nil
	}
	name := e.Release.Name
	if name == "" {
		name = e.Release.TagName
	}
	var tag strings.Builder
	tag.WriteString("🏷️ <b>Release</b> " + telegram.EscapeHTML(e.Action) + "\n")
	tag.WriteString("📦 <a href=\"" + e.Release.HTMLURL + "\">" + telegram.EscapeHTML(name) + "</a>\n")
	tag.WriteString("🏷️ Tag: <code>" + telegram.EscapeHTML(e.Release.TagName) + "</code>\n")
	tag.WriteString("📁 " + telegram.EscapeHTML(e.Repository.FullName) + "\n")
	if e.Release.Prerelease {
		tag.WriteString("🧪 <i>prerelease</i>\n")
	}
	if e.Release.Author.Login != "" {
		tag.WriteString("👤 " + telegram.EscapeHTML(e.Release.Author.Login) + "\n")
	}
	l.sendToSubscribers(ctx, e.Repository.FullName, "release", tag.String(), nil, "")
	return nil
}

type refEvent struct {
	Ref        string `json:"ref"`
	RefType    string `json:"ref_type"`
	Repository repoCommon
	Sender     struct{ Login string } `json:"sender"`
}

func (l *Listener) handleCreate(ctx context.Context, job *Job) error {
	var e refEvent
	if err := parseBody(job.Body, &e); err != nil {
		return err
	}
	if e.Repository.FullName == "" {
		return nil
	}
	icon := "🌿"
	if e.RefType == "tag" {
		icon = "🏷️"
	}
	text := fmt.Sprintf("%s <b>Created %s</b> <code>%s</code> in %s\n🔗 %s",
		icon, telegram.EscapeHTML(e.RefType), telegram.EscapeHTML(e.Ref),
		telegram.EscapeHTML(e.Repository.FullName), e.Repository.HTMLURL)
	l.sendToSubscribers(ctx, e.Repository.FullName, "create", text, nil, "")
	return nil
}

func (l *Listener) handleDelete(ctx context.Context, job *Job) error {
	var e refEvent
	if err := parseBody(job.Body, &e); err != nil {
		return err
	}
	if e.Repository.FullName == "" {
		return nil
	}
	icon := "🗑️"
	text := fmt.Sprintf("%s <b>Deleted %s</b> <code>%s</code> in %s",
		icon, telegram.EscapeHTML(e.RefType), telegram.EscapeHTML(e.Ref),
		telegram.EscapeHTML(e.Repository.FullName))
	l.sendToSubscribers(ctx, e.Repository.FullName, "delete", text, nil, "")
	return nil
}

func (l *Listener) handleRepository(ctx context.Context, job *Job) error {
	var e struct {
		Action     string `json:"action"`
		Repository repoCommon
		Sender     struct{ Login string } `json:"sender"`
	}
	if err := parseBody(job.Body, &e); err != nil {
		return err
	}
	if e.Repository.FullName == "" {
		return nil
	}
	text := fmt.Sprintf("📁 <b>Repository %s</b>: %s\n🔗 %s",
		telegram.EscapeHTML(e.Action), telegram.EscapeHTML(e.Repository.FullName), e.Repository.HTMLURL)
	if e.Sender.Login != "" {
		text += "\n👤 " + telegram.EscapeHTML(e.Sender.Login)
	}
	l.sendToSubscribers(ctx, e.Repository.FullName, "repository", text, nil, "")
	return nil
}

type workflowJobEvent struct {
	Action     string `json:"action"`
	Repository repoCommon
	Sender     struct{ Login string }
	WorkflowJob struct {
		ID      int64  `json:"id"`
		RunID   int64  `json:"run_id"`
		Name    string `json:"name"`
		Status  string `json:"status"`
		Conclusion string `json:"conclusion"`
		HTMLURL string `json:"html_url"`
	} `json:"workflow_job"`
}

func (l *Listener) handleWorkflowJob(ctx context.Context, job *Job) error {
	var e workflowJobEvent
	if err := parseBody(job.Body, &e); err != nil {
		return err
	}
	if e.Repository.FullName == "" || e.Action != "completed" {
		return nil
	}
	icon := "🔧"
	switch e.WorkflowJob.Conclusion {
	case "success":
		icon = "✅"
	case "failure":
		icon = "❌"
	case "cancelled":
		icon = "🚫"
	}
	text := fmt.Sprintf("%s <b>Job</b> <code>%s</code> %s · run <code>#%d</code>\n📦 %s\n🔗 <a href=\"%s\">View</a>",
		icon, telegram.EscapeHTML(e.WorkflowJob.Name), telegram.EscapeHTML(e.WorkflowJob.Conclusion),
		e.WorkflowJob.RunID, telegram.EscapeHTML(e.Repository.FullName), e.WorkflowJob.HTMLURL)
	l.sendToSubscribers(ctx, e.Repository.FullName, "workflow_job", text, nil, "")
	return nil
}