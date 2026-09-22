package bot

import (
	"context"
	"fmt"
	"strings"

	"github.com/control-center/github-telegram-control-center/internal/github"
	"github.com/control-center/github-telegram-control-center/internal/telegram"
)

// cmdFiles reads a repo file or shows a path hint.
func (b *Bot) cmdFiles(ctx context.Context, msg *telegram.Message, args string) error {
	owner, repo, err := b.resolveRepo(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	path := strings.TrimSpace(args)
	if path == "" {
		return b.notifier.Send(ctx, msg.Chat.ID,
			fmt.Sprintf("<code>%s</code> — usage: /files &lt;path&gt; to read a file.", trimRepo(owner, repo)))
	}
	return b.renderFile(ctx, msg.From.ID, msg.Chat.ID, owner, repo, path)
}

// cmdBranches lists branches.
func (b *Bot) cmdBranches(ctx context.Context, msg *telegram.Message, args string) error {
	owner, repo, err := b.resolveRepo(ctx, msg.From.ID, args)
	if err != nil {
		return err
	}
	return b.renderBranches(ctx, msg.From.ID, msg.Chat.ID, owner, repo, 0)
}

func (b *Bot) renderBranches(ctx context.Context, tgID, chatID int64, owner, repo string, page int) error {
	installID, _, err := b.pickInstallation(ctx, tgID, "")
	if err != nil {
		return err
	}
	branches, hasMore, err := b.gh.GetBranches(ctx, installID, owner, repo, github.ListOptions{Page: page, PerPage: perPageDefault})
	if err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>🌿 Branches · %s</b>\n", trimRepo(owner, repo)))
	for _, br := range branches {
		shield := ""
		if br.Protected {
			shield = " 🛡"
		}
		short := br.Commit.SHA
		if len(short) > 7 {
			short = short[:7]
		}
		sb.WriteString(fmt.Sprintf("%s%s — %s\n", telegram.EscapeH(br.Name), shield, telegram.EscapeH(trimRepo("", short))))
	}
	kb := telegram.Markup()
	if page > 0 || hasMore {
		kb.InlineKeyboard = append(kb.InlineKeyboard, telegram.PaginationButtons(fmt.Sprintf("branches:%s/%s", owner, repo), page, page+1))
	}
	return b.notifier.Send(ctx, chatID, sb.String(), withMarkup(kb))
}

// cbBranches paginates branches/tags.
func (b *Bot) cbBranches(ctx context.Context, cq *telegram.CallbackQuery, chatID int64, mid int, data, kind string) {
	page := parsePageArg(data, 0)
	ref := strings.ReplaceAll(strings.TrimSuffix(strings.TrimPrefix(data, ":"), fmt.Sprintf(":p%d", page)), ":", "/")
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) != 2 {
		return
	}
	if kind == "tags:" {
		installID, _, err := b.pickInstallation(ctx, cq.From.ID, "")
		if err != nil {
			return
		}
		tags, hasMore, err := b.gh.GetTags(ctx, installID, parts[0], parts[1], github.ListOptions{Page: page, PerPage: perPageDefault})
		if err != nil {
			return
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("<b>🏷 Tags · %s</b>\n", trimRepo(parts[0], parts[1])))
		for _, t := range tags {
			sb.WriteString(fmt.Sprintf("%s", telegram.EscapeH(t.Name)))
			if t.Commit.SHA != "" {
				short := t.Commit.SHA
				if len(short) > 7 {
					short = short[:7]
				}
				sb.WriteString(fmt.Sprintf(" — %s", short))
			}
			sb.WriteString("\n")
		}
		kb := telegram.Markup()
		if page > 0 || hasMore {
			kb.InlineKeyboard = append(kb.InlineKeyboard, telegram.PaginationButtons("tags:"+trimRepo(parts[0], parts[1]), page, page+1))
		}
		_ = b.notifier.Edit(ctx, chatID, mid, sb.String(), kb)
		return
	}
	// branches
	installID, _, err := b.pickInstallation(ctx, cq.From.ID, "")
	if err != nil {
		return
	}
	branches, hasMore, err := b.gh.GetBranches(ctx, installID, parts[0], parts[1], github.ListOptions{Page: page, PerPage: perPageDefault})
	if err != nil {
		return
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>🌿 Branches · %s</b>\n", trimRepo(parts[0], parts[1])))
	for _, br := range branches {
		shield := ""
		if br.Protected {
			shield = " 🛡"
		}
		short := br.Commit.SHA
		if len(short) > 7 {
			short = short[:7]
		}
		sb.WriteString(fmt.Sprintf("%s%s — %s\n", telegram.EscapeH(br.Name), shield, short))
	}
	kb := telegram.Markup()
	if page > 0 || hasMore {
		kb.InlineKeyboard = append(kb.InlineKeyboard, telegram.PaginationButtons("branches:"+trimRepo(parts[0], parts[1]), page, page+1))
	}
	_ = b.notifier.Edit(ctx, chatID, mid, sb.String(), kb)
}

// cmdTags lists tags.
func (b *Bot) cmdTags(ctx context.Context, msg *telegram.Message, args string) error {
	owner, repo, err := b.resolveRepo(ctx, msg.From.ID, args)
	if err != nil {
		return err
	}
	installID, _, err := b.pickInstallation(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	tags, hasMore, err := b.gh.GetTags(ctx, installID, owner, repo, github.ListOptions{Page: 0, PerPage: perPageDefault})
	if err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>🏷 Tags · %s</b>\n", trimRepo(owner, repo)))
	for _, t := range tags {
		sb.WriteString(fmt.Sprintf("%s", telegram.EscapeH(t.Name)))
		if t.Commit.SHA != "" {
			short := t.Commit.SHA
			if len(short) > 7 {
				short = short[:7]
			}
			sb.WriteString(fmt.Sprintf(" — %s", short))
		}
		sb.WriteString("\n")
	}
	kb := telegram.Markup()
	if hasMore {
		kb.InlineKeyboard = append(kb.InlineKeyboard, telegram.PaginationButtons("tags:"+trimRepo(owner, repo), 0, 1))
	}
	return b.notifier.Send(ctx, msg.Chat.ID, sb.String(), withMarkup(kb))
}

// cmdCommits lists commits.
func (b *Bot) cmdCommits(ctx context.Context, msg *telegram.Message, args string) error {
	owner, repo, err := b.resolveRepo(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	branch, rest := "", args
	if rest != "" {
		branch, _ = splitSpace(rest)
	}
	return b.renderCommits(ctx, msg.From.ID, msg.Chat.ID, owner, repo, branch, "", 0)
}

func (b *Bot) renderCommits(ctx context.Context, tgID, chatID int64, owner, repo, branch, path string, page int) error {
	installID, _, err := b.pickInstallation(ctx, tgID, "")
	if err != nil {
		return err
	}
	commits, hasMore, err := b.gh.GetCommits(ctx, installID, owner, repo, branch, path, github.ListOptions{Page: page, PerPage: 10})
	if err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>📜 Commits · %s</b>\n", trimRepo(owner, repo)))
	if branch != "" {
		sb.WriteString(fmt.Sprintf("Branch: %s\n", telegram.EscapeH(branch)))
	}
	for _, c := range commits {
		msg := strings.ReplaceAll(c.Commit.Message, "\n", " ")
		msg = brief(msg, 70)
		author := ""
		if c.Author != nil && c.Author.Login != "" {
			author = c.Author.Login
		} else if c.Commit.Author != nil {
			author = c.Commit.Author.Name
		}
		short := c.SHA
		if len(short) > 7 {
			short = short[:7]
		}
		sb.WriteString(fmt.Sprintf("%s %s — <i>%s</i>\n", short, telegram.EscapeH(msg), telegram.EscapeH(author)))
	}
	kb := telegram.Markup()
	if page > 0 || hasMore {
		kb.InlineKeyboard = append(kb.InlineKeyboard, telegram.PaginationButtons(fmt.Sprintf("commits:%s/%s", owner, repo), page, page+1))
	}
	return b.notifier.Send(ctx, chatID, sb.String(), withMarkup(kb))
}

// cbCommits paginates commits.
func (b *Bot) cbCommits(ctx context.Context, cq *telegram.CallbackQuery, chatID int64, mid int, data string) {
	page := parsePageArg(data, 0)
	ref := strings.ReplaceAll(strings.TrimPrefix(data, ":"), "/", "/")
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) != 2 {
		return
	}
	installID, _, err := b.pickInstallation(ctx, cq.From.ID, "")
	if err != nil {
		return
	}
	commits, hasMore, err := b.gh.GetCommits(ctx, installID, parts[0], parts[1], "", "", github.ListOptions{Page: page, PerPage: 10})
	if err != nil {
		return
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>📜 Commits · %s</b>\n", trimRepo(parts[0], parts[1])))
	for _, c := range commits {
		msg := strings.ReplaceAll(c.Commit.Message, "\n", " ")
		msg = brief(msg, 70)
		short := c.SHA
		if len(short) > 7 {
			short = short[:7]
		}
		sb.WriteString(fmt.Sprintf("%s %s\n", short, telegram.EscapeH(msg)))
	}
	kb := telegram.Markup()
	if page > 0 || hasMore {
		kb.InlineKeyboard = append(kb.InlineKeyboard, telegram.PaginationButtons("commits:"+trimRepo(parts[0], parts[1]), page, page+1))
	}
	_ = b.notifier.Edit(ctx, chatID, mid, sb.String(), kb)
}

// cmdCommit shows commit detail.
func (b *Bot) cmdCommit(ctx context.Context, msg *telegram.Message, args string) error {
	owner, repo, err := b.resolveRepo(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	sha, _ := splitSpace(args)
	if sha == "" {
		return fmt.Errorf("usage: /commit <sha>")
	}
	installID, _, err := b.pickInstallation(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	c, err := b.gh.GetCommit(ctx, installID, owner, repo, sha)
	if err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>Commit</b> %s\n", telegram.EscapeH(shortSHA(c.SHA))))
	sb.WriteString(fmt.Sprintf("<i>%s</i>\n\n", telegram.EscapeH(brief(c.Message, 200))))
	if c.Author != nil {
		sb.WriteString(fmt.Sprintf("Author: %s\n", telegram.EscapeH(c.Author.Name)))
	}
	sb.WriteString(fmt.Sprintf("Files: %d (+%d/-%d)\n", c.Stats.Total, c.Stats.Additions, c.Stats.Deletions))
	_ = shortSHA
	return b.notifier.Send(ctx, msg.Chat.ID, sb.String())
}

func shortSHA(s string) string {
	if len(s) > 7 {
		return s[:7]
	}
	return s
}

// cmdCompare shows diff between refs.
func (b *Bot) cmdCompare(ctx context.Context, msg *telegram.Message, args string) error {
	owner, repo, err := b.resolveRepo(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	base, head, ok := strings.Cut(args, "...")
	if !ok {
		return fmt.Errorf("usage: /compare <base>...<head>")
	}
	base = strings.TrimSpace(base)
	head = strings.TrimSpace(head)
	installID, _, err := b.pickInstallation(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	res, err := b.gh.CompareRefs(ctx, installID, owner, repo, base, head)
	if err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>Compare</b> %s...%s\n", telegram.EscapeH(base), telegram.EscapeH(head)))
	sb.WriteString(fmt.Sprintf("%d commits · %d files · +%d/-%d\n\n", res.TotalCommits, len(res.Files), res.AheadBy, res.BehindBy))
	for _, f := range res.Files {
		sb.WriteString(fmt.Sprintf("%s %s\n", telegram.EscapeH(f.Status), telegram.EscapeH(f.Filename)))
	}
	return b.notifier.Send(ctx, msg.Chat.ID, sb.String())
}

// cmdSearch searches code/repos/issues.
func (b *Bot) cmdSearch(ctx context.Context, msg *telegram.Message, args string) error {
	kind, rest, ok := strings.Cut(strings.TrimSpace(args), " ")
	if !ok {
		return fmt.Errorf("usage: /search <code|repo|issue> <query>")
	}
	rest = strings.TrimSpace(rest)
	installID, _, err := b.pickInstallation(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	switch kind {
	case "repo":
		rs, err := b.gh.SearchRepositories(ctx, installID, rest, github.ListOptions{Page: 0, PerPage: 10})
		if err != nil {
			return err
		}
		var sb strings.Builder
		sb.WriteString("<b>🔎 Repos</b>\n")
		for _, r := range rs {
			sb.WriteString(fmt.Sprintf("%s %s\n", repoEmoji(r), telegram.EscapeH(r.FullName)))
		}
		return b.notifier.Send(ctx, msg.Chat.ID, sb.String())
	case "issue":
		is, err := b.gh.SearchIssues(ctx, installID, rest, github.ListOptions{Page: 0, PerPage: 10})
		if err != nil {
			return err
		}
		var sb strings.Builder
		sb.WriteString("<b>🔎 Issues/PRs</b>\n")
		for _, i := range is {
			sb.WriteString(fmt.Sprintf("#%d %s\n", i.Number, telegram.EscapeH(brief(i.Title, 60))))
		}
		return b.notifier.Send(ctx, msg.Chat.ID, sb.String())
	case "code":
		owner, repo, rerr := b.resolveRepo(ctx, msg.From.ID, "")
		if rerr != nil {
			return rerr
		}
		res, err := b.gh.SearchCode(ctx, installID, owner, repo, rest, github.ListOptions{Page: 0, PerPage: 10})
		if err != nil {
			return err
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("<b>🔎 Code in %s</b>\n", trimRepo(owner, repo)))
		for _, c := range res.Items {
			sb.WriteString(fmt.Sprintf("<code>%s</code>\n", telegram.EscapeH(c.Path)))
			for _, tm := range c.TextMatches {
				sb.WriteString(fmt.Sprintf("<i>%s</i>\n", telegram.EscapeH(brief(tm.Fragment, 100))))
			}
		}
		return b.notifier.Send(ctx, msg.Chat.ID, sb.String())
	}
	return fmt.Errorf("supported: code, repo, issue")
}

func brief(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) > n {
		return strings.TrimSpace(s[:n-1]) + "…"
	}
	return s
}

func splitSpace(s string) (string, string) {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, " \t\n"); i >= 0 {
		return s[:i], strings.TrimSpace(s[i+1:])
	}
	return s, ""
}

// cmdDeploys lists deployments.
func (b *Bot) cmdDeploys(ctx context.Context, msg *telegram.Message, args string) error {
	owner, repo, err := b.resolveRepo(ctx, msg.From.ID, args)
	if err != nil {
		return err
	}
	installID, _, err := b.pickInstallation(ctx, msg.From.ID, "")
	if err != nil {
		return err
	}
	ds, hasMore, err := b.gh.GetDeployments(ctx, installID, owner, repo, github.ListOptions{Page: 0, PerPage: perPageDefault})
	if err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>🚀 Deployments · %s</b>\n", trimRepo(owner, repo)))
	for _, d := range ds {
		sb.WriteString(fmt.Sprintf("%s %s — %s\n", telegram.EscapeH(d.Environment), telegram.EscapeH(d.State), shortSHA(d.SHA)))
	}
	if !hasMore && len(ds) == 0 {
		sb.WriteString("(none)")
	}
	return b.notifier.Send(ctx, msg.Chat.ID, sb.String())
}

func paginated(kb *telegram.InlineKeyboardMarkup, prefix string, page int, hasMore bool) {
	if page > 0 || hasMore {
		kb.InlineKeyboard = append(kb.InlineKeyboard, telegram.PaginationButtons(prefix, page, page+1))
	}
}