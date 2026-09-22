package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/control-center/github-telegram-control-center/internal/logging"
	"github.com/control-center/github-telegram-control-center/internal/metrics"
)

const maxSendRetries = 3

type clientOptions struct {
	updateMode string
	baseURL    string
}

type Option func(*clientOptions)

func WithUpdateMode(mode string) Option {
	return func(o *clientOptions) { o.updateMode = mode }
}

type Client struct {
	token    string
	http     *http.Client
	baseURL  string
	mode     string
	log      interface{ Debug(string, ...any) }
}

func NewClient(token string, opts ...Option) *Client {
	copts := &clientOptions{updateMode: "webhook"}
	for _, o := range opts {
		o(copts)
	}
	return &Client{
		token:   token,
		baseURL: "https://api.telegram.org/bot" + token,
		mode:    copts.updateMode,
		http:    &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *Client) call(ctx context.Context, method string, w http.Handler, params url.Values, out any) error {
	var body io.Reader
	if params != nil {
		body = strings.NewReader(params.Encode())
	}
	var lastErr error
	for attempt := 1; attempt <= maxSendRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/"+method, body)
		if err != nil {
			return err
		}
		if params != nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("telegram %s: status %d: %s", method, resp.StatusCode, truncate(raw, 200))
			metrics.TelegramErrorsTotal.Inc()
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
			continue
		}
		var apiResp struct {
			OK          bool            `json:"ok"`
			Description string          `json:"description"`
			Result      json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(raw, &apiResp); err != nil {
			return fmt.Errorf("telegram %s: decode: %w", method, err)
		}
		if !apiResp.OK {
			return fmt.Errorf("telegram %s: %s", method, apiResp.Description)
		}
		if out != nil && len(apiResp.Result) > 0 {
			return json.Unmarshal(apiResp.Result, out)
		}
		return nil
	}
	return lastErr
}

func truncate(b []byte, n int) string {
	s := string(b)
	if len(s) > n {
		return s[:n]
	}
	return s
}

type SendMessageParams struct {
	ChatID                int64
	Text                  string
	ParseMode             string
	ReplyMarkup           *InlineKeyboardMarkup
	DisableWebPagePreview bool
	ReplyToMessageID      int
}

func (c *Client) SendMessage(ctx context.Context, p SendMessageParams) (*Message, error) {
	params := url.Values{}
	params.Set("chat_id", fmt.Sprint(p.ChatID))
	params.Set("text", p.Text)
	if p.ParseMode != "" {
		params.Set("parse_mode", p.ParseMode)
	}
	if p.ReplyMarkup != nil {
		if err := appendKeyboard(params, p.ReplyMarkup); err != nil {
			return nil, err
		}
	}
	if p.DisableWebPagePreview {
		params.Set("disable_web_page_preview", "true")
	}
	if p.ReplyToMessageID > 0 {
		params.Set("reply_to_message_id", fmt.Sprint(p.ReplyToMessageID))
	}
	var m Message
	if err := c.call(ctx, "sendMessage", nil, params, &m); err != nil {
		metrics.TelegramErrorsTotal.Inc()
		return nil, err
	}
	return &m, nil
}

func (c *Client) EditMessageText(ctx context.Context, chatID, messageID int64, text, parseMode string, replyMarkup *InlineKeyboardMarkup) error {
	params := url.Values{}
	params.Set("chat_id", fmt.Sprint(chatID))
	params.Set("message_id", fmt.Sprint(messageID))
	params.Set("text", text)
	if parseMode != "" {
		params.Set("parse_mode", parseMode)
	}
	if replyMarkup != nil {
		if err := appendKeyboard(params, replyMarkup); err != nil {
			return err
		}
	}
	return c.call(ctx, "editMessageText", nil, params, nil)
}

func (c *Client) AnswerCallbackQuery(ctx context.Context, callbackID, text string, showAlert bool) error {
	params := url.Values{}
	params.Set("callback_query_id", callbackID)
	if text != "" {
		params.Set("text", text)
	}
	if showAlert {
		params.Set("show_alert", "true")
	}
	return c.call(ctx, "answerCallbackQuery", nil, params, nil)
}

func (c *Client) SendDocument(ctx context.Context, chatID int64, name string, content []byte) error {
	var buf bytes.Buffer
	boundary := "----gtccboundary"
	buf.WriteString("--" + boundary + "\r\n")
	buf.WriteString(fmt.Sprintf("Content-Disposition: form-data; name=%q; filename=%q\r\n", "document", name))
	buf.WriteString("Content-Type: text/plain\r\n\r\n")
	buf.Write(content)
	buf.WriteString("\r\n--" + boundary + "--\r\n")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/sendDocument", &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "multipart/form-data; boundary="+boundary)
	q := req.URL.Query()
	q.Set("chat_id", fmt.Sprint(chatID))
	req.URL.RawQuery = q.Encode()

	resp, err := c.http.Do(req)
	if err != nil {
		metrics.TelegramErrorsTotal.Inc()
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	logging.FromContext(ctx).Debug("sendDocument", "status", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram sendDocument: status %d: %s", resp.StatusCode, truncate(raw, 200))
	}
	return nil
}

func appendKeyboard(params url.Values, mk *InlineKeyboardMarkup) error {
	b, err := json.Marshal(mk)
	if err != nil {
		return err
	}
	params.Set("reply_markup", string(b))
	return nil
}

func (c *Client) SetWebhook(ctx context.Context, webhookURL, secret string) error {
	params := url.Values{}
	params.Set("url", webhookURL)
	if secret != "" {
		params.Set("secret_token", secret)
	}
	params.Set("allowed_updates", `["message","callback_query"]`)
	return c.call(ctx, "setWebhook", nil, params, nil)
}

func (c *Client) DeleteWebhook(ctx context.Context) error {
	return c.call(ctx, "deleteWebhook", nil, nil, nil)
}

func (c *Client) GetUpdates(ctx context.Context, offset int64) ([]Update, error) {
	params := url.Values{}
	params.Set("offset", fmt.Sprint(offset))
	params.Set("timeout", "25")
	var updates []Update
	if err := c.call(ctx, "getUpdates", nil, params, &updates); err != nil {
		return nil, err
	}
	return updates, nil
}