package bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/control-center/github-telegram-control-center/internal/config"
	"github.com/control-center/github-telegram-control-center/internal/database"
	"github.com/control-center/github-telegram-control-center/internal/github"
	"github.com/control-center/github-telegram-control-center/internal/telegram"
)

// cmdStart shows the main menu.
func (b *Bot) cmdStart(ctx context.Context, msg *telegram.Message, role string) error {
	kb := telegram.Markup(
		telegram.Row(
			telegram.CbButton("👤 Account", "acts:account"),
			telegram.CbButton("🏢 Orgs", ":orgs"),
		),
		telegram.Row(
			telegram.CbButton("📦 Repos", "repos:"),
			telegram.CbButton("🌿 Branches", "branches:"),
		),
		telegram.Row(
			telegram.CbButton("🔁 CI/CD", "acts:ci"),
			telegram.CbButton("📋 PRs", "prs:open"),
			telegram.CbButton("🐛 Issues", "issues:open"),
		),
		telegram.Row(
			telegram.CbButton("🔔 Notifications", "notif:view"),
		),
	)
	return b.notifier.Send(ctx, msg.Chat.ID, about(b.cfg, role), withMarkup(kb))
}

func (b *Bot) editStart(ctx context.Context, chatID int64, mid int) {
	role, _ := b.cfg.RoleFor(0)
	role = ""
	text := about(b.cfg, role)
	kb := telegram.Markup(
		telegram.Row(telegram.CbButton("👤 Account", "acts:account"), telegram.CbButton("🏢 Orgs", "orgs:")),
		telegram.Row(telegram.CbButton("📦 Repos", "repos:"), telegram.CbButton("🔁 CI/CD", "acts:ci"), telegram.CbButton("📋 PRs", "prs:open")),
		telegram.Row(telegram.CbButton("🔔 Notifications", "notif:view")),
	)
	_ = b.notifier.Edit(ctx, chatID, mid, text, kb)
}

func about(cfg *config.Config, role string) string {
	return fmt.Sprintf(`🤖 <b>GitHub Control Center</b>

Remote control for your GitHub account via Telegram.

<b>Commands</b>
/start — this menu
/account — account details
/orgs — list organizations
/repos [org] — list repositories
/files [path] — browse repo contents
/branches · /tags · /commits [ref]
/compare &lt;base&gt;...&lt;head&gt;
/search code &lt;query&gt;
/prs [state] · /pr &lt;n&gt; · /merge &lt;n&gt;
/issues [state] · /issue &lt;n&gt; · /open &lt;title&gt;
/workflows · /runs [branch] · /run &lt;id&gt; · /log &lt;id&gt;
/artifacts · /artifact &lt;id&gt; · /deploy
/branch create|delete &lt;name&gt; · /tag create|delete
/release create|delete &lt;tag&gt;
/notify — notification preferences
/sub &lt;repo&gt; [event] · /unsub &lt;repo&gt; [event] · /list
/config — user profile
/audit [n] — audit log
/whoami — who you are
/help — this message

Use inline buttons to navigate. Callback buttons paginate with ◀ ▶.`)
}

// cmdStatus shows bot + rate limit status.
func (b *Bot) cmdStatus(ctx context.Context, msg *telegram.Message) error {
	_, remaining, resetAt, err := b.store.GetRateLimit(ctx, "core")
	if err != nil {
		remaining, resetAt = -1, time.Time{}
	}
	rlS := "n/a"
	if err == nil {
		rlS = fmt.Sprintf("%d (resets %s)", remaining, resetAt.Format("15:04"))
	}
	return b.notifier.Send(ctx, msg.Chat.ID, fmt.Sprintf(`<b>Status</b>
• Bot: online
• Rate limit: %s
• Environment: %s
`, telegram.EscapeH(rlS), telegram.EscapeH(b.cfg.Environment)))
}

// cmdInstall shows install instructions.
func (b *Bot) cmdInstall(ctx context.Context, msg *telegram.Message) error {
	url := b.cfg.WebhookBaseURL
	text := "📦 <b>Install the GitHub App</b>\n\nThe admin must install the GitHub App for this account. Once installed, run /orgs to see available accounts.\n"
	if url != "" {
		text += fmt.Sprintf("Webhook URL: <code>%s/webhooks/github</code>\n", telegram.EscapeH(url))
	}
	return b.notifier.Send(ctx, msg.Chat.ID, text)
}

// cmdOrg lists / switches organization.
func (b *Bot) cmdOrg(ctx context.Context, msg *telegram.Message, args string) error {
	if args != "" {
		return b.setDefault(ctx, msg, func(u *database.User) { u.DefaultOrg = &args }, "org")
	}
	return b.cmdOrgs(ctx, msg)
}

// cmdOrgs lists installations.
func (b *Bot) cmdOrgs(ctx context.Context, msg *telegram.Message) error {
	insts, err := b.store.ListInstallations(ctx)
	if err != nil {
		return err
	}
	if len(insts) == 0 {
		return fmt.Errorf("no installations — use /install")
	}
	var sb strings.Builder
	sb.WriteString("<b>Installed accounts</b>\n")
	for _, i := range insts {
		login := "?"
		if i.AccountLogin != nil {
			login = *i.AccountLogin
		}
		typ := ""
		if i.AccountType != nil {
			typ = *i.AccountType
		}
		sb.WriteString(fmt.Sprintf("• <code>%s</code> (%s) — <a href=\"%s\">switch</a>\n", telegram.EscapeH(login), telegram.EscapeH(typ), fmt.Sprintf("t.me/%s", "")))
	}
	return b.notifier.Send(ctx, msg.Chat.ID, sb.String(), withMarkup(
		telegram.Markup(func() []telegram.InlineKeyboardButton {
			var row []telegram.InlineKeyboardButton
			for _, i := range insts {
				login := "?"
				if i.AccountLogin != nil {
					login = *i.AccountLogin
				}
				row = append(row, telegram.CbButton("▶ "+login, "org:"+login))
			}
			return row
		}()),
	))
}

// cbOrg switches the default org.
func (b *Bot) cbOrg(ctx context.Context, cq *telegram.CallbackQuery, chatID int64, mid int, login string) {
	user, err := b.resolveUser(ctx, cq.From.ID)
	if err != nil {
		return
	}
	b.setDefault(ctx, &telegram.Message{Chat: &telegram.Chat{ID: chatID}, From: cq.From}, func(u *database.User) { u.DefaultOrg = &login }, "org")
	_ = user
}

// cmdRepo sets default repo (or "owner/repo").
func (b *Bot) cmdRepo(ctx context.Context, msg *telegram.Message, args string) error {
	if args == "" {
		return b.cmdRepos(ctx, msg, "")
	}
	return b.setDefault(ctx, msg, func(u *database.User) { u.DefaultRepo = &args }, "repo")
}

func (b *Bot) setDefault(ctx context.Context, msg *telegram.Message, apply func(*database.User), what string) error {
	u, err := b.resolveUser(ctx, msg.From.ID)
	if err != nil {
		return err
	}
	var org, repo *string
	org, repo = u.DefaultOrg, u.DefaultRepo
	// apply to a copy of strings
	co := u.DefaultOrg
	if co == nil {
		s := ""
		co = &s
	}
	_ = co
	u2 := *u
	apply(&u2)
	_ = org
	_ = repo
	if err := b.store.UpdateUserPrefs(ctx, msg.From.ID, nil, u2.DefaultOrg, u2.DefaultRepo, nil, nil); err != nil {
		return err
	}
	return b.notifier.Send(ctx, msg.Chat.ID, fmt.Sprintf("✅ default %s set", what))
}

// cmdWhoami shows user info.
func (b *Bot) cmdWhoami(ctx context.Context, msg *telegram.Message) error {
	u, err := b.resolveUser(ctx, msg.From.ID)
	if err != nil {
		return err
	}
	role, _ := b.cfg.RoleFor(msg.From.ID)
	text := fmt.Sprintf("<b>You</b>\n• Telegram: %s\n• Role: %s\n", telegram.EscapeH(tgName(msg.From)), telegram.EscapeH(role))
	if u.GitHubUsername != nil {
		text += fmt.Sprintf("• GitHub: %s\n", telegram.EscapeH(*u.GitHubUsername))
	}
	return b.notifier.Send(ctx, msg.Chat.ID, text)
}

func tgName(u *telegram.User) string {
	if u.Username != "" {
		return "@" + u.Username
	}
	return fmt.Sprintf("%s %s", u.FirstName, u.LastName)
}

// cmdAccount shows GitHub account detail.
func (b *Bot) cmdAccount(ctx context.Context, msg *telegram.Message) error {
	installID, _, err := b.pickInstallation(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	user, err := b.gh.GetAuthenticatedUser(ctx, installID)
	if err != nil {
		return err
	}
	text := fmt.Sprintf("<b>GitHub account</b>\n• User: <code>%s</code>\n• Name: %s\n• Blog: %s\n• Followers: %d\n", telegram.EscapeH(user.Login), telegram.EscapeH(user.Name), telegram.EscapeH(user.Blog), user.Followers)
	return b.notifier.Send(ctx, msg.Chat.ID, text)
}

// cmdRepos lists repositories (optionally for org).
func (b *Bot) cmdRepos(ctx context.Context, msg *telegram.Message, args string) error {
	installID, _, err := b.pickInstallation(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	org := ""
	if args != "" {
		org = args
	}
	repos, hasMore, err := b.gh.GetRepositories(ctx, installID, github.ListOptions{Page: 0, PerPage: 10})
	if err != nil {
		return err
	}
	_ = hasMore
	_ = org
	return b.renderRepos(ctx, msg.Chat.ID, repos, 0, "")
}

func (b *Bot) renderRepos(ctx context.Context, chatID int64, repos []github.Repository, page int, prefix string) error {
	if len(repos) == 0 {
		return b.notifier.Send(ctx, chatID, "No repositories.")
	}
	var sb strings.Builder
	sb.WriteString("<b>Repositories</b>\n")
	for _, r := range repos {
		sb.WriteString(fmt.Sprintf("%s %s — <code>%s</code>\n", repoEmoji(r), r.Name, trimRepo(r.Owner.Login, r.Name)))
	}
	kb := telegram.Markup()
	var row []telegram.InlineKeyboardButton
	for _, r := range repos {
		row = append(row, telegram.CbButton(r.Name, "repo:"+trimRepo(r.Owner.Login, r.Name)))
	}
	kb.InlineKeyboard = append(kb.InlineKeyboard, chunkRow(row))
	nav := telegram.PaginationButtons("repos:", page, reposLen(repos))
	if len(nav) > 0 {
		kb.InlineKeyboard = append(kb.InlineKeyboard, nav)
	}
	return b.notifier.Send(ctx, chatID, sb.String(), withMarkup(kb))
}

func reposLen(repos []github.Repository) int { return len(repos) }

// cbRepos paginates.
func (b *Bot) cbRepos(ctx context.Context, cq *telegram.CallbackQuery, chatID int64, mid int, data string) {
	page := parsePageArg(data, 0)
	installID, _, err := b.pickInstallation(ctx, cq.From.ID, "")
	if err != nil {
		return
	}
	var page2 *int
	p := page
	page2 = &p
	_ = page2
	repos, _, err := b.gh.GetRepositories(ctx, installID, github.ListOptions{Page: page, PerPage: 10})
	if err != nil {
		return
	}
	_ = b.renderRepoListEdit(ctx, chatID, mid, repos, page)
}

func (b *Bot) renderRepoListEdit(ctx context.Context, chatID int64, mid int, repos []github.Repository, page int) error {
	var sb strings.Builder
	sb.WriteString("<b>Repositories</b>\n")
	for _, r := range repos {
		sb.WriteString(fmt.Sprintf("%s %s — %s\n", repoEmoji(r), r.Name, trimRepo(r.Owner.Login, r.Name)))
	}
	kb := telegram.Markup()
	var row []telegram.InlineKeyboardButton
	for _, r := range repos {
		row = append(row, telegram.CbButton(r.Name, "repo:"+trimRepo(r.Owner.Login, r.Name)))
	}
	kb.InlineKeyboard = append(kb.InlineKeyboard, chunkRow(row))
	kb.InlineKeyboard = append(kb.InlineKeyboard, telegram.PaginationButtons("repos:", page, page+1))
	return b.notifier.Edit(ctx, chatID, mid, sb.String(), kb)
}

// repoEmoji picks an emoji for a repository.
func repoEmoji(r github.Repository) string {
	switch {
	case r.Fork:
		return "🍴"
	case r.Archived:
		return "🗄️"
	case r.Private:
		return "🔒"
	default:
		return "📦"
	}
}

func chunkRow(row []telegram.InlineKeyboardButton) []telegram.InlineKeyboardButton {
	return row
}
