package bot

import (
	"context"
	"fmt"
	"strings"

	"github.com/control-center/github-telegram-control-center/internal/github"
	"github.com/control-center/github-telegram-control-center/internal/telegram"
)

// cbRepo renders repository detail with navigation buttons.
func (b *Bot) cbRepo(ctx context.Context, cq *telegram.CallbackQuery, chatID int64, mid int, data string) {
	owner, repo, ok := strings.Cut(data, "/")
	if !ok {
		return
	}
	installID, _, err := b.pickInstallation(ctx, cq.From.ID, "")
	if err != nil {
		_ = b.notifier.Edit(ctx, chatID, mid, telegram.EscapeH(err.Error()), nil)
		return
	}
	r, err := b.gh.GetRepository(ctx, installID, owner, repo)
	if err != nil {
		_ = b.notifier.Edit(ctx, chatID, mid, telegram.EscapeH(err.Error()), nil)
		return
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s <b>%s</b>\n", repoEmoji(*r), telegram.EscapeH(r.FullName)))
	if r.Description != "" {
		sb.WriteString(fmt.Sprintf("<i>%s</i>\n", telegram.EscapeH(brief(r.Description, 200))))
	}
	flags := []string{}
	if r.Private {
		flags = append(flags, "private")
	}
	if r.Fork {
		flags = append(flags, "fork")
	}
	if r.Archived {
		flags = append(flags, "archived")
	}
	if len(flags) > 0 {
		sb.WriteString(fmt.Sprintf("<b>%s</b>\n", strings.Join(flags, " · ")))
	}
	sb.WriteString(fmt.Sprintf("Default branch: <code>%s</code>\n", telegram.EscapeH(r.DefaultBranch)))
	if r.Language != "" {
		sb.WriteString(fmt.Sprintf("Language: %s\n", telegram.EscapeH(r.Language)))
	}
	sb.WriteString(fmt.Sprintf("⭐ %d · 🍴 %d · 👁 %d\n", r.Stargazers, r.Forks, r.Watchers))
	sb.WriteString(fmt.Sprintf("Open issues: %d · Size: %d KB\n", r.OpenIssues, r.Size))
	if len(r.Topics) > 0 {
		sb.WriteString(fmt.Sprintf("Topics: %s\n", telegram.EscapeH(strings.Join(r.Topics, ", "))))
	}
	kb := telegram.Markup(
		telegram.Row(
			telegram.CbButton("🌿 Branches", "branches:"+trimRepo(owner, repo)),
			telegram.CbButton("🏷 Tags", "tags:"+trimRepo(owner, repo)),
			telegram.CbButton("📜 Commits", "commits:"+trimRepo(owner, repo)),
		),
		telegram.Row(
			telegram.CbButton("🔀 PRs", "prs:"+trimRepo(owner, repo)+":open"),
			telegram.CbButton("🐛 Issues", "issues:"+trimRepo(owner, repo)+":open"),
		),
		telegram.Row(
			telegram.CbButton("⚙ Workflows", "workflows:"+trimRepo(owner, repo)),
			telegram.CbButton("🔁 Runs", "runs:"+trimRepo(owner, repo)),
			telegram.CbButton("📦 Artifacts", "acts:arts:"+trimRepo(owner, repo)),
		),
		telegram.Row(telegram.BackButton("repos:")),
	)
	_ = b.notifier.Edit(ctx, chatID, mid, sb.String(), kb)
}

var _ = github.ListOptions{}