package telegram

import (
	"fmt"
	"strconv"
)

func Row(buttons ...InlineKeyboardButton) []InlineKeyboardButton {
	return buttons
}

func CbButton(text, data string) InlineKeyboardButton {
	return InlineKeyboardButton{Text: text, CallbackData: data}
}

func UrlButton(text, url string) InlineKeyboardButton {
	return InlineKeyboardButton{Text: text, URL: url}
}

func Markup(rows ...[]InlineKeyboardButton) *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{InlineKeyboard: rows}
}

func PaginationButtons(prefix string, page, totalPages int) []InlineKeyboardButton {
	var row []InlineKeyboardButton
	if page > 1 {
		row = append(row, CbButton("⬅️", fmt.Sprintf("%s:p%d", prefix, page-1)))
	}
	row = append(row, CbButton(fmt.Sprintf("%d/%d", page, totalPages), "noop"))
	if page < totalPages {
		row = append(row, CbButton("➡️", fmt.Sprintf("%s:p%d", prefix, page+1)))
	}
	return row
}

func BackButton(data string) InlineKeyboardButton {
	return CbButton("🔙 Back", data)
}

func PageFromData(data string) int {
	parts := splitN(data, ":", 3)
	for _, p := range parts {
		if len(p) >= 2 && p[0] == 'p' {
			if n, err := strconv.Atoi(p[1:]); err == nil && n > 0 {
				return n
			}
		}
	}
	return 1
}

func splitN(s, sep string, n int) []string {
	var out []string
	start := 0
	for i := 0; i < n-1; i++ {
		idx := index(s, sep, start)
		if idx < 0 {
			break
		}
		out = append(out, s[start:idx])
		start = idx + len(sep)
	}
	out = append(out, s[start:])
	return out
}

func index(s, sub string, from int) int {
	for i := from; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}