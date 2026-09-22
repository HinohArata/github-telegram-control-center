package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/control-center/github-telegram-control-center/internal/metrics"
	"github.com/control-center/github-telegram-control-center/internal/telegram"
)

const confirmTTL = 10 * time.Minute

// handleConfirm processes confirm:yes/no callbacks.
func (b *Bot) handleConfirm(ctx context.Context, cq *telegram.CallbackQuery) {
	parts := strings.Split(cq.Data, ":")
	if len(parts) != 3 {
		b.notifier.AnswerCallback(ctx, cq.ID, "Invalid confirm request", true)
		return
	}
	sid, choice := parts[1], parts[2]
	tgID, kind, payload, err := b.store.GetSession(ctx, sid)
	if err != nil {
		b.notifier.AnswerCallback(ctx, cq.ID, "Session expired or invalid", true)
		return
	}
	if tgID != cq.From.ID {
		b.notifier.AnswerCallback(ctx, cq.ID, "Not allowed", true)
		return
	}
	defer b.store.DeleteSession(ctx, sid)
	mid := int(cq.Message.MessageID)
	if choice == "no" {
		_ = b.notifier.Edit(ctx, cq.Message.Chat.ID, mid, "❌ Cancelled.", nil)
		return
	}
	start := time.Now()
	target, err := b.executeConfirm(ctx, cq, kind, payload)
	dur := time.Since(start)

	if err != nil {
		metrics.TelegramErrorsTotal.Inc()
		b.log.Error("confirmed action failed", "kind", kind, "user", cq.From.ID, "err", err)
		b.auditUser(ctx, cq.From.ID, functionOf(kind), target, false, err.Error())
		_ = b.notifier.Edit(ctx, cq.Message.Chat.ID, mid,
			fmt.Sprintf("❌ <b>%s</b> failed: %s", telegram.EscapeH(kind), telegram.EscapeH(err.Error())), nil)
		return
	}
	metrics.CommandDuration.Observe(dur)
	b.auditUser(ctx, cq.From.ID, functionOf(kind), target, true, "")
	_ = b.notifier.Edit(ctx, cq.Message.Chat.ID, mid,
		fmt.Sprintf("✅ <b>%s</b> done.", telegram.EscapeH(functionOf(kind))), nil)
}

// executeConfirm runs the destructive operation recorded in the session payload.
func (b *Bot) executeConfirm(ctx context.Context, cq *telegram.CallbackQuery, kind string, payload map[string]any) (target string, err error) {
	s := func(k string) string {
		v, _ := payload[k].(string)
		return v
	}
	i := func(k string) int64 {
		switch v := payload[k].(type) {
		case int64:
			return v
		case float64:
			return int64(v)
		case string:
			n, _ := strconv.ParseInt(v, 10, 64)
			return n
		}
		return 0
	}
	owner, repo, toks := s("owner"), s("repo"), strings.Split(s("args"), " ")
	switch kind {
	case "merge_pr":
		num := i("num")
		installID, _, e := b.pickInstallation(ctx, cq.From.ID, "")
		if e != nil {
			return "", e
		}
		return fmt.Sprintf("%s#%d", trimRepo(owner, repo), num),
			b.gh.MergePullRequest(ctx, installID, owner, repo, num, "squash")
	case "close_pr":
		num := i("num")
		installID, _, e := b.pickInstallation(ctx, cq.From.ID, "")
		if e != nil {
			return "", e
		}
		st := "closed"
		return fmt.Sprintf("%s#%d", trimRepo(owner, repo), num),
			b.gh.UpdatePullRequest(ctx, installID, owner, repo, num, &st)
	case "close_issue":
		num := i("num")
		installID, _, e := b.pickInstallation(ctx, cq.From.ID, "")
		if e != nil {
			return "", e
		}
		st := "closed"
		return fmt.Sprintf("%s#%d", trimRepo(owner, repo), num),
			b.gh.UpdateIssue(ctx, installID, owner, repo, num, &st)
	case "cancel_run":
		num := i("num")
		installID, _, e := b.pickInstallation(ctx, cq.From.ID, "")
		if e != nil {
			return "", e
		}
		return fmt.Sprintf("%s run=%d", trimRepo(owner, repo), num),
			b.gh.CancelWorkflow(ctx, installID, owner, repo, num)
	case "rerun_run":
		num := i("num")
		installID, _, e := b.pickInstallation(ctx, cq.From.ID, "")
		if e != nil {
			return "", e
		}
		return fmt.Sprintf("%s run=%d", trimRepo(owner, repo), num),
			b.gh.RerunWorkflow(ctx, installID, owner, repo, num, true)
	case "delete_branch":
		name := s("name")
		installID, _, e := b.pickInstallation(ctx, cq.From.ID, "")
		if e != nil {
			return "", e
		}
		return fmt.Sprintf("%s branch=%s", trimRepo(owner, repo), name),
			b.gh.DeleteBranch(ctx, installID, owner, repo, name)
	case "delete_tag":
		name := s("name")
		installID, _, e := b.pickInstallation(ctx, cq.From.ID, "")
		if e != nil {
			return "", e
		}
		return fmt.Sprintf("%s tag=%s", trimRepo(owner, repo), name),
			b.gh.DeleteTag(ctx, installID, owner, repo, name)
	case "delete_file":
		path := s("path")
		installID, _, e := b.pickInstallation(ctx, cq.From.ID, "")
		if e != nil {
			return "", e
		}
		content, e := b.gh.GetContents(ctx, installID, owner, repo, path, "")
		if e != nil {
			return "", e
		}
		return fmt.Sprintf("%s %s", trimRepo(owner, repo), path),
			b.gh.DeleteFile(ctx, installID, owner, repo, path,
				fmt.Sprintf("Delete %s via Telegram", path), content.SHA, "HEAD")
	case "delete_release":
		num := i("num")
		installID, _, e := b.pickInstallation(ctx, cq.From.ID, "")
		if e != nil {
			return "", e
		}
		return fmt.Sprintf("%s release=%d", trimRepo(owner, repo), num),
			b.gh.DeleteRelease(ctx, installID, owner, repo, num)
	}
	_ = toks
	return "", fmt.Errorf("unknown confirmation kind %q", kind)
}

func functionOf(kind string) string {
	m := map[string]string{
		"merge_pr": "merge pull request", "close_pr": "close pull request",
		"close_issue": "close issue", "cancel_run": "cancel workflow run",
		"rerun_run": "rerun workflow run", "delete_branch": "delete branch",
		"delete_tag": "delete tag", "delete_file": "delete file",
		"delete_release": "delete release",
	}
	if v, ok := m[kind]; ok {
		return v
	}
	return kind
}

// startConfirm creates a confirm session and shows the confirm prompt.
func (b *Bot) startConfirm(ctx context.Context, cq *telegram.CallbackQuery, kind string, question string, payload map[string]any) {
	sid := randomID()
	if err := b.store.CreateSession(ctx, sid, cq.From.ID, "confirm:"+kind, payload, confirmTTL); err != nil {
		b.log.Error("create session", "err", err)
		_ = b.notifier.Send(ctx, cq.Message.Chat.ID, "Could not start confirmation.")
		return
	}
	kb := telegram.Markup(
		telegram.Row(
			telegram.CbButton("✅ Confirm", "confirm:"+sid+":yes"),
			telegram.CbButton("❌ Cancel", "confirm:"+sid+":no"),
		),
	)
	_ = b.notifier.Send(ctx, cq.Message.Chat.ID, question, withMarkup(kb))
}
