package notifications

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/control-center/github-telegram-control-center/internal/database"
	"github.com/control-center/github-telegram-control-center/internal/metrics"
	"github.com/control-center/github-telegram-control-center/internal/telegram"
)

type Notifier struct {
	tg    *telegram.Client
	store *database.Store
	log   *slog.Logger
}

func New(tg *telegram.Client, store *database.Store, log *slog.Logger) *Notifier {
	return &Notifier{tg: tg, store: store, log: log}
}

type SendOptions struct {
	ReplyMarkup *telegram.InlineKeyboardMarkup
}

func track(err error) {
	if err != nil {
		metrics.NotificationsFailedTotal.Inc()
	} else {
		metrics.NotificationsSentTotal.Inc()
	}
}

func (n *Notifier) Send(ctx context.Context, chatID int64, text string, opts ...SendOptions) error {
	start := time.Now()
	params := telegram.SendMessageParams{ChatID: chatID, Text: text, ParseMode: "HTML"}
	if len(opts) > 0 {
		params.ReplyMarkup = opts[0].ReplyMarkup
	}
	_, err := n.tg.SendMessage(ctx, params)
	metrics.CommandDuration.Observe(time.Since(start))
	track(err)
	if err != nil {
		n.log.Error("notify send failed", "chat_id", chatID, "err", err)
	}
	return err
}

func (n *Notifier) Reply(ctx context.Context, chatID, replyTo int64, text string) error {
	start := time.Now()
	_, err := n.tg.SendMessage(ctx, telegram.SendMessageParams{
		ChatID: chatID, Text: text, ParseMode: "HTML", ReplyToMessageID: int(replyTo),
	})
	metrics.CommandDuration.Observe(time.Since(start))
	track(err)
	if err != nil {
		n.log.Error("notify reply failed", "chat_id", chatID, "err", err)
	}
	return err
}

func (n *Notifier) Edit(ctx context.Context, chatID int64, messageID int, text string, markup *telegram.InlineKeyboardMarkup) error {
	start := time.Now()
	err := n.tg.EditMessageText(ctx, chatID, int64(messageID), text, "HTML", markup)
	metrics.CommandDuration.Observe(time.Since(start))
	track(err)
	if err != nil {
		n.log.Error("notify edit failed", "chat_id", chatID, "err", err)
	}
	return err
}

func (n *Notifier) AnswerCallback(ctx context.Context, callbackID, text string, alert bool) error {
	return n.tg.AnswerCallbackQuery(ctx, callbackID, text, alert)
}

func (n *Notifier) SendDocument(ctx context.Context, chatID int64, filename string, content []byte, caption string) error {
	start := time.Now()
	err := n.tg.SendDocument(ctx, chatID, filename, content)
	metrics.CommandDuration.Observe(time.Since(start))
	track(err)
	if err != nil {
		return fmt.Errorf("send document: %w", err)
	}
	return nil
}
