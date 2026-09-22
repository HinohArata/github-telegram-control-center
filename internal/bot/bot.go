package bot

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/control-center/github-telegram-control-center/internal/config"
	"github.com/control-center/github-telegram-control-center/internal/database"
	"github.com/control-center/github-telegram-control-center/internal/events"
	"github.com/control-center/github-telegram-control-center/internal/github"
	"github.com/control-center/github-telegram-control-center/internal/metrics"
	"github.com/control-center/github-telegram-control-center/internal/notifications"
	"github.com/control-center/github-telegram-control-center/internal/telegram"
)

const (
	maxCols        = 30
	perPageDefault = 15
)

// Bot is the Telegram control center bot.
type Bot struct {
	cfg      *config.Config
	store    *database.Store
	tg       *telegram.Client
	notifier *notifications.Notifier
	gh       *github.Service
	events   *events.Dispatcher
	log      *slog.Logger
}

// New creates a bot.
func New(cfg *config.Config, store *database.Store, tg *telegram.Client, notifier *notifications.Notifier, svc *github.Service, disp *events.Dispatcher, log *slog.Logger) *Bot {
	return &Bot{cfg: cfg, store: store, tg: tg, notifier: notifier, gh: svc, events: disp, log: log}
}

// HandleUpdate processes one Telegram update.
func (b *Bot) HandleUpdate(ctx context.Context, upd *telegram.Update) {
	metrics.TelegramUpdatesTotal.Inc()
	if upd.Message != nil {
		b.handleMessage(ctx, upd.Message)
		return
	}
	if upd.CallbackQuery != nil {
		b.handleCallback(ctx, upd.CallbackQuery)
		return
	}
}

func (b *Bot) handleMessage(ctx context.Context, msg *telegram.Message) {
	if msg.From == nil {
		return
	}
	if msg.From.IsBot {
		return
	}
	role, ok := b.cfg.RoleFor(msg.From.ID)
	if !ok {
		b.notifier.Send(ctx, msg.Chat.ID, "⛔ Unauthorized. You are not on the authorized user list.")
		return
	}
	if b.isAdmin(msg.From.ID) {
		role = "admin"
	}
	b.log.Debug("telegram command", "user", msg.From.ID, "text", msg.Text)

	start := time.Now()
	defer func() { metrics.CommandDuration.Observe(time.Since(start)) }()

	cmd, args := parseCommand(msg.Text)
	if cmd == "" {
		return
	}
	if err := b.dispatchCommand(ctx, msg, role, cmd, args); err != nil {
		metrics.TelegramErrorsTotal.Inc()
		b.log.Error("command failed", "cmd", cmd, "user", msg.From.ID, "err", err)
		b.notifier.Send(ctx, msg.Chat.ID, fmt.Sprintf("❌ <b>Error:</b> %s", telegram.EscapeH(err.Error())))
	}
}

// dispatchCommand routes a command. Returns error handled by caller (sent to user).
func (b *Bot) dispatchCommand(ctx context.Context, msg *telegram.Message, role, cmd string, args string) error {
	requireAdmin := map[string]bool{
		"users": true,
	}
	if requireAdmin[cmd] && !b.isAdmin(msg.From.ID) {
		return fmt.Errorf("admin only")
	}

	switch cmd {
	case "start", "help":
		return b.cmdStart(ctx, msg, role)
	case "status":
		return b.cmdStatus(ctx, msg)
	case "install":
		return b.cmdInstall(ctx, msg)
	case "org":
		return b.cmdOrg(ctx, msg, args)
	case "repo", "switch":
		return b.cmdRepo(ctx, msg, args)
	case "whoami":
		return b.cmdWhoami(ctx, msg)
	case "account":
		return b.cmdAccount(ctx, msg)
	case "orgs":
		return b.cmdOrgs(ctx, msg)
	case "repos":
		return b.cmdRepos(ctx, msg, args)
	case "files", "tree":
		return b.cmdFiles(ctx, msg, args)
	case "branches":
		return b.cmdBranches(ctx, msg, args)
	case "tags":
		return b.cmdTags(ctx, msg, args)
	case "commits":
		return b.cmdCommits(ctx, msg, args)
	case "commit":
		return b.cmdCommit(ctx, msg, args)
	case "compare":
		return b.cmdCompare(ctx, msg, args)
	case "search":
		return b.cmdSearch(ctx, msg, args)
	case "prs":
		return b.cmdPullRequests(ctx, msg, args)
	case "pr":
		return b.cmdPullRequest(ctx, msg, args)
	case "issues":
		return b.cmdIssues(ctx, msg, args)
	case "issue":
		return b.cmdIssue(ctx, msg, args)
	case "open":
		return b.cmdOpenIssue(ctx, msg, args)
	case "workflows":
		return b.cmdWorkflows(ctx, msg, args)
	case "runs":
		return b.cmdRuns(ctx, msg, args)
	case "run":
		return b.cmdRun(ctx, msg, args)
	case "jobs":
		return b.cmdJobs(ctx, msg, args)
	case "log":
		return b.cmdLog(ctx, msg, args)
	case "artifacts":
		return b.cmdArtifacts(ctx, msg, args)
	case "artifact":
		return b.cmdArtifact(ctx, msg, args)
	case "deploy":
		return b.cmdDeploys(ctx, msg, args)
	case "notify":
		return b.cmdNotify(ctx, msg)
	case "sub":
		return b.cmdSub(ctx, msg, args)
	case "unsub":
		return b.cmdUnsub(ctx, msg, args)
	case "list":
		return b.cmdList(ctx, msg)
	case "audit":
		return b.cmdAudit(ctx, msg, args)
	case "set":
		return b.cmdSet(ctx, msg, args)
	case "branch":
		return b.cmdBranch(ctx, msg, args)
	case "tag":
		return b.cmdTag(ctx, msg, args)
	case "release":
		return b.cmdRelease(ctx, msg, args)
	case "merge":
		return b.cmdMerge(ctx, msg, args)
	case "comment":
		return b.cmdComment(ctx, msg, args)
	case "file":
		return b.cmdFile(ctx, msg, args)
	case "users":
		return b.cmdUsers(ctx, msg)
	}
	return fmt.Errorf("unknown command %q — try /help", cmd)
}

func parseCommand(text string) (cmd, args string) {
	text = strings.TrimSpace(text)
	if len(text) <= 1 || text[0] != '/' || !strings.Contains(text, "@") && text[0:] == "/" {
		if text == "" || text[0] != '/' {
			return "", ""
		}
	}
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "/") {
		return "", ""
	}
	fields := strings.Fields(text)
	cmd = strings.TrimPrefix(fields[0], "/")
	if i := strings.IndexByte(cmd, '@'); i >= 0 {
		cmd = cmd[:i]
	}
	if len(fields) > 1 {
		args = strings.Join(fields[1:], " ")
	}
	return cmd, args
}

// handleCallback routes callback queries.
func (b *Bot) handleCallback(ctx context.Context, cq *telegram.CallbackQuery) {
	defer b.answerCallback(ctx, cq)
	if cq.From == nil {
		return
	}
	if _, ok := b.cfg.RoleFor(cq.From.ID); !ok {
		b.notifier.AnswerCallback(ctx, cq.ID, "Unauthorized", true)
		return
	}
	b.log.Debug("callback", "user", cq.From.ID, "data", cq.Data)

	if strings.HasPrefix(cq.Data, "confirm:") {
		b.handleConfirm(ctx, cq)
		return
	}

	chatID := cq.Message.Chat.ID
	mid := int(cq.Message.MessageID)

	switch {
	case cq.Data == "noop":
		return
	case cq.Data == "menu" || cq.Data == "back:menu":
		b.editStart(ctx, chatID, mid)
		return
	case strings.HasPrefix(cq.Data, "org:"):
		b.cbOrg(ctx, cq, chatID, mid, strings.TrimPrefix(cq.Data, "org:"))
		return
	case strings.HasPrefix(cq.Data, "repos:"):
		b.cbRepos(ctx, cq, chatID, mid, strings.TrimPrefix(cq.Data, "repos:"))
		return
	case strings.HasPrefix(cq.Data, "repo:"):
		b.cbRepo(ctx, cq, chatID, mid, strings.TrimPrefix(cq.Data, "repo:"))
		return
	case strings.HasPrefix(cq.Data, "branches:"):
		b.cbBranches(ctx, cq, chatID, mid, strings.TrimPrefix(cq.Data, "branches:"), "branches:")
		return
	case strings.HasPrefix(cq.Data, "tags:"):
		b.cbBranches(ctx, cq, chatID, mid, strings.TrimPrefix(cq.Data, "tags:"), "tags:")
		return
	case strings.HasPrefix(cq.Data, "commits:"):
		b.cbCommits(ctx, cq, chatID, mid, strings.TrimPrefix(cq.Data, "commits:"))
		return
	case strings.HasPrefix(cq.Data, "prs:"):
		b.cbPRs(ctx, cq, chatID, mid, strings.TrimPrefix(cq.Data, "prs:"))
		return
	case strings.HasPrefix(cq.Data, "issues:"):
		b.cbIssues(ctx, cq, chatID, mid, strings.TrimPrefix(cq.Data, "issues:"))
		return
	case strings.HasPrefix(cq.Data, "runs:"):
		b.cbRuns(ctx, cq, chatID, mid, strings.TrimPrefix(cq.Data, "runs:"))
		return
	case strings.HasPrefix(cq.Data, "workflows:"):
		b.cbWorkflows(ctx, cq, chatID, mid, strings.TrimPrefix(cq.Data, "workflows:"))
		return
	case strings.HasPrefix(cq.Data, "notif:"):
		b.cbNotif(ctx, cq, chatID, mid, strings.TrimPrefix(cq.Data, "notif:"))
		return
	case strings.HasPrefix(cq.Data, "acts:"):
		b.handleActions(ctx, cq, chatID, mid, strings.TrimPrefix(cq.Data, "acts:"))
		return
	}
}

// answerCallback acknowledges the callback query.
func (b *Bot) answerCallback(ctx context.Context, cq *telegram.CallbackQuery) {
	if cq.ID == "" {
		return
	}
	_ = b.notifier.AnswerCallback(ctx, cq.ID, "", false)
}

func (b *Bot) isAdmin(tgID int64) bool {
	return b.cfg.AdminUsers[tgID]
}

// resolveUser loads the telegram user record, ensuring it exists.
func (b *Bot) resolveUser(ctx context.Context, tgID int64) (*database.User, error) {
	role, _ := b.cfg.RoleFor(tgID)
	return b.store.EnsureUser(ctx, tgID, role)
}

// pickInstallation resolves an installation id. hint is an org/login to match, else defaultOrg, else unique installation.
func (b *Bot) pickInstallation(ctx context.Context, tgID int64, hint string) (int64, string, error) {
	user, err := b.resolveUser(ctx, tgID)
	if err != nil {
		return 0, "", err
	}
	insts, err := b.store.ListInstallations(ctx)
	if err != nil {
		return 0, "", err
	}
	if len(insts) == 0 {
		return 0, "", fmt.Errorf("no GitHub installations found — run /install first and install the GitHub App")
	}
	match := hint
	if match == "" && user.DefaultOrg != nil {
		match = *user.DefaultOrg
	}
	if match != "" {
		for _, i := range insts {
			if (i.AccountLogin != nil && *i.AccountLogin == match) || i.AccountID != nil && fmt.Sprintf("%d", *i.AccountID) == match {
				return i.InstallationID, loginOf(i), nil
			}
		}
		return 0, "", fmt.Errorf("no installation for %q — try /org to list accounts", match)
	}
	if len(insts) == 1 {
		return insts[0].InstallationID, loginOf(insts[0]), nil
	}
	return 0, "", fmt.Errorf("multiple accounts installed (%s) — pick one with /org <name>", joinLogins(insts))
}

func loginOf(i database.Installation) string {
	if i.AccountLogin != nil {
		return *i.AccountLogin
	}
	return ""
}

func joinLogins(insts []database.Installation) string {
	var names []string
	for _, i := range insts {
		if l := loginOf(i); l != "" {
			names = append(names, l)
		} else if i.AccountID != nil {
			names = append(names, fmt.Sprintf("#%d", *i.AccountID))
		}
	}
	return strings.Join(names, ", ")
}

// resolveRepo returns owner, repo. Handles "owner/repo", "repo", or user default.
func (b *Bot) resolveRepo(ctx context.Context, tgID int64, s string) (string, string, error) {
	user, err := b.resolveUser(ctx, tgID)
	if err != nil {
		return "", "", err
	}
	s = strings.TrimSpace(s)
	if s == "" && user.DefaultRepo != nil {
		s = *user.DefaultRepo
	}
	if s == "" {
		return "", "", fmt.Errorf("specify a repo (owner/name) or pick a default with /repo")
	}
	if i := strings.IndexByte(s, '/'); i >= 0 {
		if i == 0 || i == len(s)-1 {
			return "", "", fmt.Errorf("invalid repo %q", s)
		}
		return s[:i], s[i+1:], nil
	}
	if user.DefaultOrg == nil || *user.DefaultOrg == "" {
		return "", "", fmt.Errorf("ambiguous repo %q — no default org set; use /org <account>", s)
	}
	return *user.DefaultOrg, s, nil
}

func parsePageArg(data string, defaultVal int) int {
	for _, tok := range strings.Split(data, ":") {
		if strings.HasPrefix(tok, "p") {
			if n, err := strconv.Atoi(strings.TrimPrefix(tok, "p")); err == nil && n >= 0 {
				return n
			}
		}
	}
	return defaultVal
}

func trimRepo(owner, repo string) string {
	return owner + "/" + repo
}