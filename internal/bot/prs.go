package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/control-center/github-telegram-control-center/internal/github"
	"github.com/control-center/github-telegram-control-center/internal/telegram"
)

// cmdPullRequests lists PRs.
func (b *Bot) cmdPullRequests(ctx context.Context, msg *telegram.Message, args string) error {
	state, rest := splitSpace(args)
	if state == "" {
		state = "open"
	}
	owner, repo, err := b.resolveRepo(ctx, msg.From.ID, rest)
	if err != nil {
		return err
	}
	return b.renderPRs(ctx, msg.From.ID, msg.Chat.ID, owner, repo, state, 0)
}

func (b *Bot) renderPRs(ctx context.Context, tgID, chatID int64, owner, repo, state string, page int) error {
	installID, _, err := b.pickInstallation(ctx, tgID, "")
	if err != nil {
		return err
	}
	prs, hasMore, err := b.gh.GetPullRequests(ctx, installID, owner, repo, state, github.ListOptions{Page: page, PerPage: perPageDefault})
	if err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>🔀 PRs (%s) · %s</b>\n", state, trimRepo(owner, repo)))
	for _, pr := range prs {
		mark := "🔀"
		if pr.Merged {
			mark = "✅"
		}
		author := ""
		if pr.User.Login != "" {
			author = " @" + pr.User.Login
		}
		sb.WriteString(fmt.Sprintf("%s #%d %s <i>%s</i>\n", mark, pr.Number, telegram.EscapeH(brief(pr.Title, 50)), telegram.EscapeH(pr.State)))
		_ = author
	}
	if len(prs) == 0 {
		sb.WriteString("(none)")
	}
	kb := telegram.Markup()
	var row []telegram.InlineKeyboardButton
	for _, pr := range prs {
		shortTxt := fmt.Sprintf("#%d", pr.Number)
		row = append(row, telegram.CbButton(shortTxt, fmt.Sprintf("prdetail:%s/%s:%d", owner, repo, pr.Number)))
	}
	kb.InlineKeyboard = append(kb.InlineKeyboard, row)
	if page > 0 || hasMore {
		combined := fmt.Sprintf("prs:%s/%s:%s", owner, repo, state)
		kb.InlineKeyboard = append(kb.InlineKeyboard, telegram.PaginationButtons(combined, page, page+1))
	}
	return b.notifier.Send(ctx, chatID, sb.String(), withMarkup(kb))
}

// cbPRs paginates PR list.
func (b *Bot) cbPRs(ctx context.Context, cq *telegram.CallbackQuery, chatID int64, mid int, data string) {
	page := parsePageArg(data, 0)
	core := stripPage(data)
	parts := strings.SplitN(core, ":", 2)
	if len(parts) != 2 {
		return
	}
	ownerrepo := strings.SplitN(parts[0], "/", 2)
	if len(ownerrepo) != 2 {
		return
	}
	state := parts[1]
	if state == "" {
		state = "open"
	}
	installID, _, err := b.pickInstallation(ctx, cq.From.ID, "")
	if err != nil {
		return
	}
	prs, hasMore, err := b.gh.GetPullRequests(ctx, installID, ownerrepo[0], ownerrepo[1], state, github.ListOptions{Page: page, PerPage: perPageDefault})
	if err != nil {
		return
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>🔀 PRs (%s) · %s</b>\n", telegram.EscapeH(state), trimRepo(ownerrepo[0], ownerrepo[1])))
	for _, pr := range prs {
		sb.WriteString(fmt.Sprintf("#%d %s [%s]\n", pr.Number, telegram.EscapeH(brief(pr.Title, 50)), telegram.EscapeH(pr.State)))
	}
	kb := telegram.Markup()
	var row []telegram.InlineKeyboardButton
	for _, pr := range prs {
		row = append(row, telegram.CbButton(fmt.Sprintf("#%d", pr.Number), fmt.Sprintf("prdetail:%s/%s:%d", ownerrepo[0], ownerrepo[1], pr.Number)))
	}
	kb.InlineKeyboard = append(kb.InlineKeyboard, row)
	combined := fmt.Sprintf("prs:%s/%s:%s", ownerrepo[0], ownerrepo[1], state)
	paginated(kb, combined, page, hasMore)
	_ = b.notifier.Edit(ctx, chatID, mid, sb.String(), kb)
}

// cmdPullRequest shows PR detail.
func (b *Bot) cmdPullRequest(ctx context.Context, msg *telegram.Message, args string) error {
	owner, repo, err := b.resolveRepo(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	num, _ := splitSpace(args)
	number, err := strconv.ParseInt(strings.TrimPrefix(num, "#"), 10, 64)
	if err != nil {
		return fmt.Errorf("usage: /pr <number>")
	}
	installID, _, err := b.pickInstallation(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	pr, err := b.gh.GetPullRequest(ctx, installID, owner, repo, number)
	if err != nil {
		return err
	}
	text := formatPR(pr, owner, repo)
	kb := telegram.Markup(
		telegram.Row(
			telegram.UrlButton("Open on GitHub", pr.HTMLURL),
		),
		telegram.Row(
			telegram.CbButton("🔀 Merge", fmt.Sprintf("acts:prmerge:%s/%s:%d", owner, repo, number)),
			telegram.CbButton("💬 Comment", fmt.Sprintf("acts:prcomment:%s/%s:%d", owner, repo, number)),
		),
	)
	return b.notifier.Send(ctx, msg.Chat.ID, text, withMarkup(kb))
}

func formatPR(pr *github.PullRequest, owner, repo string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>#%d %s</b>\n", pr.Number, telegram.EscapeH(pr.Title)))
	sb.WriteString(fmt.Sprintf("State: %s", telegram.EscapeH(pr.State)))
	if pr.Merged {
		sb.WriteString(" · merged ✓")
	}
	sb.WriteString("\n")
	if pr.User.Login != "" {
		sb.WriteString(fmt.Sprintf("Author: @%s\n", telegram.EscapeH(pr.User.Login)))
	}
	if pr.Base.Ref != "" && pr.Head.Ref != "" {
		sb.WriteString(fmt.Sprintf("<code>%s</code> ← <code>%s</code>\n", telegram.EscapeH(pr.Base.Ref), telegram.EscapeH(pr.Head.Ref)))
	}
	sb.WriteString(fmt.Sprintf("Changes: +%d/-%d across %d files\n", pr.Additions, pr.Deletions, pr.ChangedFiles))
	if pr.Body != "" {
		sb.WriteString(fmt.Sprintf("\n%s\n", telegram.EscapeH(brief(pr.Body, 300))))
	}
	return sb.String()
}

// cmdMerge merges a PR.
func (b *Bot) cmdMerge(ctx context.Context, msg *telegram.Message, args string) error {
	owner, repo, err := b.resolveRepo(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	num, method, _ := strings.Cut(strings.TrimSpace(args), " ")
	if num == "" {
		return fmt.Errorf("usage: /merge <number> [squash|merge|rebase]")
	}
	number, err := strconv.ParseInt(strings.TrimPrefix(strings.TrimSpace(num), "#"), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid PR number")
	}
	if method == "" {
		method = "squash"
	}
	installID, _, err := b.pickInstallation(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	err = b.gh.MergePullRequest(ctx, installID, owner, repo, number, method)
	if err != nil {
		return err
	}
	b.audit(ctx, msg, "merge_pr", fmt.Sprintf("%s#%d method=%s", trimRepo(owner, repo), number, method), true, "")
	return b.notifier.Send(ctx, msg.Chat.ID, fmt.Sprintf("✅ Merged #%d (%s)", number, method))
}

// cmdIssues lists issues.
func (b *Bot) cmdIssues(ctx context.Context, msg *telegram.Message, args string) error {
	state, rest := splitSpace(args)
	if state == "" {
		state = "open"
	}
	owner, repo, err := b.resolveRepo(ctx, msg.From.ID, rest)
	if err != nil {
		return err
	}
	return b.renderIssues(ctx, msg.From.ID, msg.Chat.ID, owner, repo, state, 0)
}

func (b *Bot) renderIssues(ctx context.Context, tgID, chatID int64, owner, repo, state string, page int) error {
	installID, _, err := b.pickInstallation(ctx, tgID, "")
	if err != nil {
		return err
	}
	iss, hasMore, err := b.gh.GetIssues(ctx, installID, owner, repo, state, github.ListOptions{Page: page, PerPage: perPageDefault})
	if err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>🐛 Issues (%s) · %s</b>\n", state, trimRepo(owner, repo)))
	for _, i := range iss {
		prList := ""
		if i.PullRequest != nil {
			prList = " 🔀"
		}
		sb.WriteString(fmt.Sprintf("#%d %s%s\n", i.Number, telegram.EscapeH(brief(i.Title, 60)), prList))
	}
	if len(iss) == 0 {
		sb.WriteString("(none)")
	}
	kb := telegram.Markup()
	var row []telegram.InlineKeyboardButton
	for _, i := range iss {
		row = append(row, telegram.CbButton(fmt.Sprintf("#%d", i.Number), fmt.Sprintf("issuedetail:%s/%s:%d", owner, repo, i.Number)))
	}
	kb.InlineKeyboard = append(kb.InlineKeyboard, row)
	if page > 0 || hasMore {
		paginated(kb, fmt.Sprintf("issues:%s/%s:%s", owner, repo, state), page, hasMore)
	}
	return b.notifier.Send(ctx, chatID, sb.String(), withMarkup(kb))
}

// cbIssues paginates issues.
func (b *Bot) cbIssues(ctx context.Context, cq *telegram.CallbackQuery, chatID int64, mid int, data string) {
	page := parsePageArg(data, 0)
	core := stripPage(data)
	parts := strings.SplitN(core, ":", 2)
	if len(parts) != 2 {
		return
	}
	ownerrepo := strings.SplitN(parts[0], "/", 2)
	if len(ownerrepo) != 2 {
		return
	}
	state := parts[1]
	if state == "" {
		state = "open"
	}
	installID, _, err := b.pickInstallation(ctx, cq.From.ID, "")
	if err != nil {
		return
	}
	iss, hasMore, err := b.gh.GetIssues(ctx, installID, ownerrepo[0], ownerrepo[1], state, github.ListOptions{Page: page, PerPage: perPageDefault})
	if err != nil {
		return
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>🐛 Issues (%s) · %s</b>\n", telegram.EscapeH(state), trimRepo(ownerrepo[0], ownerrepo[1])))
	for _, i := range iss {
		sb.WriteString(fmt.Sprintf("#%d %s\n", i.Number, telegram.EscapeH(brief(i.Title, 60))))
	}
	kb := telegram.Markup()
	var row []telegram.InlineKeyboardButton
	for _, i := range iss {
		row = append(row, telegram.CbButton(fmt.Sprintf("#%d", i.Number), fmt.Sprintf("issuedetail:%s/%s:%d", ownerrepo[0], ownerrepo[1], i.Number)))
	}
	kb.InlineKeyboard = append(kb.InlineKeyboard, row)
	paginated(kb, fmt.Sprintf("issues:%s/%s:%s", ownerrepo[0], ownerrepo[1], state), page, hasMore)
	_ = b.notifier.Edit(ctx, chatID, mid, sb.String(), kb)
}

// cmdIssue shows issue detail.
func (b *Bot) cmdIssue(ctx context.Context, msg *telegram.Message, args string) error {
	owner, repo, err := b.resolveRepo(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	num, _ := splitSpace(args)
	number, err := strconv.ParseInt(strings.TrimPrefix(num, "#"), 10, 64)
	if err != nil {
		return fmt.Errorf("usage: /issue <number>")
	}
	installID, _, err := b.pickInstallation(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	iss, err := b.gh.GetIssue(ctx, installID, owner, repo, number)
	if err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>#%d %s</b> [%s]\n", iss.Number, telegram.EscapeH(iss.Title), telegram.EscapeH(iss.State)))
	if iss.User.Login != "" {
		sb.WriteString(fmt.Sprintf("Author: @%s\n", telegram.EscapeH(iss.User.Login)))
	}
	sb.WriteString(fmt.Sprintf("Comments: %d\n", iss.Comments))
	if iss.Body != "" {
		sb.WriteString(fmt.Sprintf("\n%s\n", telegram.EscapeH(brief(iss.Body, 300))))
	}
	kb := telegram.Markup(
		telegram.Row(
			telegram.UrlButton("Open on GitHub", iss.HTMLURL),
		),
		telegram.Row(
			telegram.CbButton("💬 Comment", fmt.Sprintf("acts:comment:%s/%s:%d", owner, repo, number)),
		),
	)
	return b.notifier.Send(ctx, msg.Chat.ID, sb.String(), withMarkup(kb))
}

// cmdOpenIssue opens a new issue.
func (b *Bot) cmdOpenIssue(ctx context.Context, msg *telegram.Message, args string) error {
	owner, repo, err := b.resolveRepo(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	if args == "" {
		return fmt.Errorf("usage: /issue open <title> [---body]")
	}
	title, body, _ := strings.Cut(args, " ---")
	installID, _, err := b.pickInstallation(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	iss, err := b.gh.CreateIssue(ctx, installID, owner, repo, strings.TrimSpace(title), strings.TrimSpace(body))
	if err != nil {
		return err
	}
	b.audit(ctx, msg, "create_issue", fmt.Sprintf("%s#%d", trimRepo(owner, repo), iss.Number), true, "")
	return b.notifier.Send(ctx, msg.Chat.ID, fmt.Sprintf("✅ Opened issue #%d", iss.Number))
}

func stripPage(data string) string {
	// data like "owner/repo:state:p5"
	if i := strings.LastIndex(data, ":"); i >= 0 {
		tail := data[i+1:]
		if strings.HasPrefix(tail, "p") {
			if _, err := strconv.Atoi(strings.TrimPrefix(tail, "p")); err == nil {
				return data[:i]
			}
		}
	}
	return data
}

// cmdComment adds a PR/issue comment.
func (b *Bot) cmdComment(ctx context.Context, msg *telegram.Message, args string) error {
	owner, repo, err := b.resolveRepo(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	num, rest, ok := strings.Cut(strings.TrimSpace(args), " ")
	if !ok {
		return fmt.Errorf("usage: /comment <number> <text>")
	}
	number, err := strconv.ParseInt(strings.TrimPrefix(num, "#"), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid number")
	}
	installID, _, err := b.pickInstallation(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	err = b.gh.CreateIssueComment(ctx, installID, owner, repo, number, strings.TrimSpace(rest))
	if err != nil {
		return err
	}
	b.audit(ctx, msg, "comment", fmt.Sprintf("%s#%d", trimRepo(owner, repo), number), true, "")
	return b.notifier.Send(ctx, msg.Chat.ID, "💬 Comment added.")
}
