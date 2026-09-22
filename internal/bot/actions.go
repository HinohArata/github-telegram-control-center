package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/control-center/github-telegram-control-center/internal/github"
	"github.com/control-center/github-telegram-control-center/internal/telegram"
)

// cmdWorkflows lists workflows.
func (b *Bot) cmdWorkflows(ctx context.Context, msg *telegram.Message, args string) error {
	owner, repo, err := b.resolveRepo(ctx, msg.From.ID, args)
	if err != nil {
		return err
	}
	return b.renderWorkflows(ctx, msg.From.ID, msg.Chat.ID, owner, repo, 0)
}

func (b *Bot) renderWorkflows(ctx context.Context, tgID, chatID int64, owner, repo string, page int) error {
	installID, _, err := b.pickInstallation(ctx, tgID, "")
	if err != nil {
		return err
	}
	wfs, hasMore, err := b.gh.GetWorkflows(ctx, installID, owner, repo, github.ListOptions{Page: page, PerPage: perPageDefault})
	if err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>⚙ Workflows · %s</b>\n", trimRepo(owner, repo)))
	for _, w := range wfs {
		state := "●"
		if w.State == "active" {
			state = "▶️"
		} else if w.State == "disabled" {
			state = "⛔"
		}
		sb.WriteString(fmt.Sprintf("%s %d · %s\n", state, w.ID, telegram.EscapeH(brief(w.Name, 50))))
	}
	if len(wfs) == 0 {
		sb.WriteString("(none)")
	}
	kb := telegram.Markup()
	var row []telegram.InlineKeyboardButton
	for _, w := range wfs {
		row = append(row, telegram.CbButton(fmt.Sprintf("#%d", w.ID), fmt.Sprintf("rerun:%s/%s:%d", owner, repo, w.ID)))
	}
	kb.InlineKeyboard = append(kb.InlineKeyboard, row)
	if page > 0 || hasMore {
		paginated(kb, fmt.Sprintf("workflows:%s/%s", owner, repo), page, hasMore)
	}
	return b.notifier.Send(ctx, chatID, sb.String(), withMarkup(kb))
}

// cbWorkflows paginates workflows.
func (b *Bot) cbWorkflows(ctx context.Context, cq *telegram.CallbackQuery, chatID int64, mid int, data string) {
	page := parsePageArg(data, 0)
	ref := stripPage(strings.TrimPrefix(data, ":"))
	if ref == "" {
		return
	}
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) != 2 {
		return
	}
	installID, _, err := b.pickInstallation(ctx, cq.From.ID, "")
	if err != nil {
		return
	}
	wfs, hasMore, err := b.gh.GetWorkflows(ctx, installID, parts[0], parts[1], github.ListOptions{Page: page, PerPage: perPageDefault})
	if err != nil {
		return
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>⚙ Workflows · %s</b>\n", trimRepo(parts[0], parts[1])))
	for _, w := range wfs {
		sb.WriteString(fmt.Sprintf("#%d · %s\n", w.ID, telegram.EscapeH(brief(w.Name, 50))))
	}
	kb := telegram.Markup()
	paginated(kb, fmt.Sprintf("workflows:%s/%s", parts[0], parts[1]), page, hasMore)
	_ = b.notifier.Edit(ctx, chatID, mid, sb.String(), kb)
}

// cmdRuns lists workflow runs. Args: optional "workflowID|branch".
func (b *Bot) cmdRuns(ctx context.Context, msg *telegram.Message, args string) error {
	owner, repo, err := b.resolveRepo(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	var wfID *int64
	branch := ""
	if args != "" {
		if id, err := strconv.ParseInt(strings.TrimPrefix(args, "#"), 10, 64); err == nil {
			wfID = &id
		} else {
			branch = strings.TrimSpace(args)
		}
	}
	return b.renderRuns(ctx, msg.From.ID, msg.Chat.ID, owner, repo, wfID, branch, 0)
}

func (b *Bot) renderRuns(ctx context.Context, tgID, chatID int64, owner, repo string, wfID *int64, branch string, page int) error {
	installID, _, err := b.pickInstallation(ctx, tgID, "")
	if err != nil {
		return err
	}
	runs, hasMore, err := b.gh.GetWorkflowRuns(ctx, installID, owner, repo, wfID, branch, github.ListOptions{Page: page, PerPage: 10})
	if err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>🔁 Workflow runs · %s</b>\n", trimRepo(owner, repo)))
	if branch != "" {
		sb.WriteString(fmt.Sprintf("Branch: %s\n", telegram.EscapeH(branch)))
	}
	for _, r := range runs {
		icon := runIcon(r.Status, r.Conclusion)
		sb.WriteString(fmt.Sprintf("%s #%d <code>%s</code> %s %s\n", icon, r.RunNumber, telegram.EscapeH(brief(r.Name, 30)), r.HeadBranch, r.Conclusion))
	}
	if len(runs) == 0 {
		sb.WriteString("(no runs)")
	}
	kb := telegram.Markup()
	var row []telegram.InlineKeyboardButton
	for _, r := range runs {
		row = append(row, telegram.CbButton(fmt.Sprintf("#%d", r.RunNumber), fmt.Sprintf("run:%s/%s:%d", owner, repo, r.ID)))
	}
	kb.InlineKeyboard = append(kb.InlineKeyboard, row)
	if page > 0 || hasMore {
		paginated(kb, fmt.Sprintf("runs:%s/%s", owner, repo), page, hasMore)
	}
	return b.notifier.Send(ctx, chatID, sb.String(), withMarkup(kb))
}

func runIcon(status, conclusion string) string {
	switch status {
	case "completed":
		switch conclusion {
		case "success", "neutral":
			return "✅"
		case "failure":
			return "❌"
		case "cancelled":
			return "⏹"
		case "timed_out":
			return "⏱"
		}
		return "🏁"
	case "queued", "pending":
		return "⏳"
	case "in_progress":
		return "🔃"
	case "action_required":
		return "📋"
	}
	return "▪️"
}

// cbRuns paginates runs.
func (b *Bot) cbRuns(ctx context.Context, cq *telegram.CallbackQuery, chatID int64, mid int, data string) {
	page := parsePageArg(data, 0)
	ref := stripPage(strings.TrimPrefix(data, ":"))
	if ref == "" {
		return
	}
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) != 2 {
		return
	}
	installID, _, err := b.pickInstallation(ctx, cq.From.ID, "")
	if err != nil {
		return
	}
	runs, hasMore, err := b.gh.GetWorkflowRuns(ctx, installID, parts[0], parts[1], nil, "", github.ListOptions{Page: page, PerPage: 10})
	if err != nil {
		return
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>🔁 Workflow runs · %s</b>\n", trimRepo(parts[0], parts[1])))
	for _, r := range runs {
		sb.WriteString(fmt.Sprintf("%s #%d %s\n", runIcon(r.Status, r.Conclusion), r.RunNumber, telegram.EscapeH(brief(r.Name, 40))))
	}
	kb := telegram.Markup()
	var row []telegram.InlineKeyboardButton
	for _, r := range runs {
		row = append(row, telegram.CbButton(fmt.Sprintf("#%d", r.RunNumber), fmt.Sprintf("run:%s/%s:%d", parts[0], parts[1], r.ID)))
	}
	kb.InlineKeyboard = append(kb.InlineKeyboard, row)
	paginated(kb, fmt.Sprintf("runs:%s/%s", parts[0], parts[1]), page, hasMore)
	_ = b.notifier.Edit(ctx, chatID, mid, sb.String(), kb)
}

// cmdRun shows run detail.
func (b *Bot) cmdRun(ctx context.Context, msg *telegram.Message, args string) error {
	owner, repo, err := b.resolveRepo(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	runIDstr := strings.TrimPrefix(strings.TrimSpace(args), "#")
	runID, err := strconv.ParseInt(runIDstr, 10, 64)
	if err != nil {
		return fmt.Errorf("usage: /run <run-id>")
	}
	return b.renderRun(ctx, msg.From.ID, msg.Chat.ID, owner, repo, runID)
}

func (b *Bot) renderRun(ctx context.Context, tgID, chatID int64, owner, repo string, runID int64) error {
	installID, _, err := b.pickInstallation(ctx, tgID, "")
	if err != nil {
		return err
	}
	r, err := b.gh.GetWorkflowRun(ctx, installID, owner, repo, runID)
	if err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s <b>Run #%d</b>\n", runIcon(r.Status, r.Conclusion), r.RunNumber))
	sb.WriteString(fmt.Sprintf("<i>%s</i>\n", telegram.EscapeH(brief(r.DisplayTitle, 60))))
	sb.WriteString(fmt.Sprintf("Workflow: %d\n", r.WorkflowID))
	sb.WriteString(fmt.Sprintf("Branch: %s · SHA: %s\n", telegram.EscapeH(r.HeadBranch), telegram.EscapeH(shortSHA(r.HeadSHA))))
	sb.WriteString(fmt.Sprintf("Event: %s · Status: %s", telegram.EscapeH(r.Event), telegram.EscapeH(r.Status)))
	if r.Conclusion != "" {
		sb.WriteString(fmt.Sprintf(" · Conclusion: %s", telegram.EscapeH(r.Conclusion)))
	}
	sb.WriteString("\n")
	kb := telegram.Markup(
		telegram.Row(
			telegram.CbButton("📋 Jobs", fmt.Sprintf("runjobs:%s/%s:%d", owner, repo, r.ID)),
			telegram.CbButton("📜 Logs", fmt.Sprintf("runlog:%s/%s:%d", owner, repo, r.ID)),
			telegram.UrlButton("GitHub", r.HTMLURL),
		),
	)
	if r.Status == "in_progress" || r.Status == "queued" || r.Status == "pending" {
		kb.InlineKeyboard = append(kb.InlineKeyboard, telegram.Row(
			telegram.CbButton("⏹ Cancel", fmt.Sprintf("cancelrun:%s/%s:%d", owner, repo, r.ID)),
		))
	}
	if r.Status == "completed" && r.Conclusion != "success" {
		kb.InlineKeyboard = append(kb.InlineKeyboard, telegram.Row(
			telegram.CbButton("🔁 Rerun", fmt.Sprintf("rerunrun:%s/%s:%d", owner, repo, r.ID)),
		))
	}
	return b.notifier.Send(ctx, chatID, sb.String(), withMarkup(kb))
}

// cbRunDetail inline keyboard redirects.
func (b *Bot) jumpTo(ctx context.Context, cq *telegram.CallbackQuery, chatID int64, mid int, op string) {
	parts := strings.SplitN(strings.TrimPrefix(cq.Data, op+":"), ":", 2)
	if len(parts) != 2 {
		return
	}
	ownerrepo := strings.SplitN(parts[0], "/", 2)
	if len(ownerrepo) != 2 {
		return
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return
	}
	switch op {
	case "runjobs":
		installID, _, err := b.pickInstallation(ctx, cq.From.ID, "")
		if err != nil {
			return
		}
		jobs, err := b.gh.GetWorkflowRunJobs(ctx, installID, ownerrepo[0], ownerrepo[1], id)
		if err != nil {
			return
		}
		b.renderJobs(ctx, chatID, mid, ownerrepo[0], ownerrepo[1], id, jobs)
	}
}

// cmdJobs lists jobs for a run.
func (b *Bot) cmdJobs(ctx context.Context, msg *telegram.Message, args string) error {
	owner, repo, err := b.resolveRepo(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	runID, err := strconv.ParseInt(strings.TrimPrefix(strings.TrimSpace(args), "#"), 10, 64)
	if err != nil {
		return fmt.Errorf("usage: /jobs <run-id>")
	}
	installID, _, err := b.pickInstallation(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	jobs, err := b.gh.GetWorkflowRunJobs(ctx, installID, owner, repo, runID)
	if err != nil {
		return err
	}
	return b.renderJobs(ctx, msg.Chat.ID, 0, owner, repo, runID, jobs)
}

func (b *Bot) renderJobs(ctx context.Context, chatID int64, mid int, owner, repo string, runID int64, jobs []github.WorkflowJob) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>📋 Jobs (run #%d)</b>\n", runID))
	var row []telegram.InlineKeyboardButton
	for _, j := range jobs {
		sb.WriteString(fmt.Sprintf("%s %s [%s]\n", runIcon(j.Status, j.Conclusion), telegram.EscapeH(brief(j.Name, 50)), telegram.EscapeH(j.Conclusion)))
		row = append(row, telegram.CbButton(fmt.Sprintf("📜 %s", brief(j.Name, 10)), fmt.Sprintf("joblog:%s/%s:%d", owner, repo, j.ID)))
	}
	kb := telegram.Markup()
	if len(row) > 0 {
		kb.InlineKeyboard = append(kb.InlineKeyboard, row)
	}
	if mid > 0 {
		return b.notifier.Edit(ctx, chatID, mid, sb.String(), kb)
	}
	return b.notifier.Send(ctx, chatID, sb.String(), withMarkup(kb))
}

// cmdLog fetches run or job logs.
func (b *Bot) cmdLog(ctx context.Context, msg *telegram.Message, args string) error {
	owner, repo, err := b.resolveRepo(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	arg := strings.TrimSpace(args)
	if arg == "" {
		return fmt.Errorf("usage: /log <run-id|job-id>")
	}
	installID, _, err := b.pickInstallation(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	id, err := strconv.ParseInt(strings.TrimPrefix(arg, "#"), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid id")
	}
	var logData []byte
	var name string
	// try job logs first, fall back to run logs
	if ij, err := b.gh.GetJobLogs(ctx, installID, owner, repo, id); err == nil {
		logData = ij
		name = fmt.Sprintf("job-%d", id)
	} else if ir, err := b.gh.GetRunLogs(ctx, installID, owner, repo, id); err == nil {
		logData = ir
		name = fmt.Sprintf("run-%d", id)
	} else {
		return fmt.Errorf("could not fetch logs for id %d", id)
	}
	if len(logData) > 1_000_000 {
		logData = logData[:1_000_000]
	}
	return b.notifier.SendDocument(ctx, msg.Chat.ID, name+".log", logData, fmt.Sprintf("Logs for %s", name))
}

// cmdArtifacts lists artifacts.
func (b *Bot) cmdArtifacts(ctx context.Context, msg *telegram.Message, args string) error {
	owner, repo, err := b.resolveRepo(ctx, msg.From.ID, args)
	if err != nil {
		return err
	}
	return b.renderArtifacts(ctx, msg.From.ID, msg.Chat.ID, owner, repo)
}

func (b *Bot) renderArtifacts(ctx context.Context, tgID, chatID int64, owner, repo string) error {
	installID, _, err := b.pickInstallation(ctx, tgID, "")
	if err != nil {
		return err
	}
	arts, hasMore, err := b.gh.GetArtifacts(ctx, installID, owner, repo, nil, github.ListOptions{Page: 0, PerPage: perPageDefault})
	if err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>📦 Artifacts · %s</b>\n", trimRepo(owner, repo)))
	var row []telegram.InlineKeyboardButton
	for _, a := range arts {
		sb.WriteString(fmt.Sprintf("• %s (%d KB)\n", telegram.EscapeH(a.Name), a.SizeInBytes/1024))
		row = append(row, telegram.CbButton(fmt.Sprintf("⬇ %s", brief(a.Name, 12)), fmt.Sprintf("artifact:%s/%s:%d", owner, repo, a.ID)))
	}
	if len(arts) == 0 {
		sb.WriteString("(none)")
	}
	kb := telegram.Markup()
	if len(row) > 0 {
		kb.InlineKeyboard = append(kb.InlineKeyboard, row)
	}
	paginated(kb, fmt.Sprintf("arts:%s/%s", owner, repo), 0, hasMore)
	return b.notifier.Send(ctx, chatID, sb.String(), withMarkup(kb))
}

// cmdArtifact downloads an artifact.
func (b *Bot) cmdArtifact(ctx context.Context, msg *telegram.Message, args string) error {
	owner, repo, err := b.resolveRepo(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	artID, err := strconv.ParseInt(strings.TrimSpace(args), 10, 64)
	if err != nil {
		return fmt.Errorf("usage: /artifact <artifact-id>")
	}
	installID, _, err := b.pickInstallation(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	data, err := b.gh.DownloadArtifact(ctx, installID, owner, repo, artID)
	if err != nil {
		return err
	}
	return b.notifier.SendDocument(ctx, msg.Chat.ID, fmt.Sprintf("artifact-%d.zip", artID), data, "Artifact")
}

// handleActions handles "acts:" callback actions.
func (b *Bot) handleActions(ctx context.Context, cq *telegram.CallbackQuery, chatID int64, mid int, data string) {
	// acts:<action>:<owner/repo>:<num|name>
	parts := strings.SplitN(strings.TrimPrefix(data, "acts:"), ":", 3)
	if len(parts) < 2 {
		return
	}
	action := parts[0]
	ownerrepo := strings.SplitN(parts[1], "/", 2)
	if len(ownerrepo) != 2 {
		if action == "account" {
			b.cmdAccountCallback(ctx, cq, chatID, mid)
			return
		}
		if action == "ci" {
			b.cmdWorkflowsFromCallback(ctx, cq, chatID, mid)
			return
		}
		return
	}
	num, _ := strconv.ParseInt(parts[2], 10, 64)
	switch action {
	case "prmerge":
		b.startConfirm(ctx, cq, "merge_pr",
			fmt.Sprintf("Merge PR #%d in %s?", num, trimRepo(ownerrepo[0], ownerrepo[1])),
			map[string]any{"owner": ownerrepo[0], "repo": ownerrepo[1], "num": num})
	case "prcomment", "comment":
		b.sendCommentPrompt(ctx, cq, ownerrepo[0], ownerrepo[1], num)
	case "prclose":
		b.startConfirm(ctx, cq, "close_pr",
			fmt.Sprintf("Close PR #%d in %s?", num, trimRepo(ownerrepo[0], ownerrepo[1])),
			map[string]any{"owner": ownerrepo[0], "repo": ownerrepo[1], "num": num})
	case "issueclose":
		b.startConfirm(ctx, cq, "close_issue",
			fmt.Sprintf("Close issue #%d in %s?", num, trimRepo(ownerrepo[0], ownerrepo[1])),
			map[string]any{"owner": ownerrepo[0], "repo": ownerrepo[1], "num": num})
	case "cancelrun":
		b.startConfirm(ctx, cq, "cancel_run",
			fmt.Sprintf("Cancel run #%d in %s?", num, trimRepo(ownerrepo[0], ownerrepo[1])),
			map[string]any{"owner": ownerrepo[0], "repo": ownerrepo[1], "num": num})
	case "rerunrun":
		b.startConfirm(ctx, cq, "rerun_run",
			fmt.Sprintf("Rerun run #%d in %s?", num, trimRepo(ownerrepo[0], ownerrepo[1])),
			map[string]any{"owner": ownerrepo[0], "repo": ownerrepo[1], "num": num})
	case "deletebranch":
		b.startConfirm(ctx, cq, "delete_branch",
			fmt.Sprintf("Delete branch <code>%s</code> in %s?", telegram.EscapeH(parts[2]), trimRepo(ownerrepo[0], ownerrepo[1])),
			map[string]any{"owner": ownerrepo[0], "repo": ownerrepo[1], "name": parts[2]})
	case "deletetag":
		b.startConfirm(ctx, cq, "delete_tag",
			fmt.Sprintf("Delete tag <code>%s</code> in %s?", telegram.EscapeH(parts[2]), trimRepo(ownerrepo[0], ownerrepo[1])),
			map[string]any{"owner": ownerrepo[0], "repo": ownerrepo[1], "name": parts[2]})
	case "deletefile":
		b.startConfirm(ctx, cq, "delete_file",
			fmt.Sprintf("Delete file <code>%s</code> in %s?", telegram.EscapeH(parts[2]), trimRepo(ownerrepo[0], ownerrepo[1])),
			map[string]any{"owner": ownerrepo[0], "repo": ownerrepo[1], "path": parts[2]})
	}
}

func (b *Bot) cmdWorkflowsFromCallback(ctx context.Context, cq *telegram.CallbackQuery, chatID int64, mid int) {
	owner, repo, err := b.resolveRepo(ctx, cq.From.ID, "")
	if err != nil {
		return
	}
	installID, _, err := b.pickInstallation(ctx, cq.From.ID, "")
	if err != nil {
		return
	}
	wfs, hasMore, err := b.gh.GetWorkflows(ctx, installID, owner, repo, github.ListOptions{Page: 0, PerPage: perPageDefault})
	if err != nil {
		return
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>⚙ Workflows · %s</b>\n", trimRepo(owner, repo)))
	for _, w := range wfs {
		sb.WriteString(fmt.Sprintf("#%d · %s\n", w.ID, telegram.EscapeH(brief(w.Name, 50))))
	}
	kb := telegram.Markup()
	var row []telegram.InlineKeyboardButton
	for _, w := range wfs {
		row = append(row, telegram.CbButton(fmt.Sprintf("#%d", w.ID), fmt.Sprintf("rerun:%s/%s:%d", owner, repo, w.ID)))
	}
	kb.InlineKeyboard = append(kb.InlineKeyboard, row)
	paginated(kb, fmt.Sprintf("workflows:%s/%s", owner, repo), 0, hasMore)
	if mid > 0 {
		_ = b.notifier.Edit(ctx, chatID, mid, sb.String(), kb)
	} else {
		_ = b.notifier.Send(ctx, chatID, sb.String(), withMarkup(kb))
	}
}

func (b *Bot) cmdAccountCallback(ctx context.Context, cq *telegram.CallbackQuery, chatID int64, mid int) {
	user, err := b.resolveUser(ctx, cq.From.ID)
	if err != nil {
		return
	}
	var sb strings.Builder
	sb.WriteString("<b>Account</b>\n")
	sb.WriteString(fmt.Sprintf("Telegram: %s\n", telegram.EscapeH(tgName(cq.From))))
	sb.WriteString(fmt.Sprintf("Role: %s\n", telegram.EscapeH(roleOf(cq.From.ID, b))))
	if user.GitHubUsername != nil {
		sb.WriteString(fmt.Sprintf("GitHub: @%s\n", telegram.EscapeH(*user.GitHubUsername)))
	}
	_ = telegram.BackButton("menu")
	if mid > 0 {
		_ = b.notifier.Edit(ctx, chatID, mid, sb.String(), telegram.Markup(telegram.Row(telegram.BackButton("menu"))))
	} else {
		_ = b.notifier.Send(ctx, chatID, sb.String(), withMarkup(telegram.Markup(telegram.Row(telegram.BackButton("menu")))))
	}
}

func roleOf(tgID int64, b *Bot) string {
	if b.cfg.AdminUsers[tgID] {
		return "admin"
	}
	r, _ := b.cfg.RoleFor(tgID)
	return r
}

func (b *Bot) sendCommentPrompt(ctx context.Context, cq *telegram.CallbackQuery, owner, repo string, num int64) {
	b.notifier.Send(ctx, cq.Message.Chat.ID, fmt.Sprintf(
		"Reply to this message with the comment text to post on #%d in %s.\nThen tap 🔒 Confirm.",
		num, trimRepo(owner, repo)))
}