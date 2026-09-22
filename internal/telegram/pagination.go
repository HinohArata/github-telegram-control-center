package telegram

import (
	"fmt"
	"strconv"
	"strings"
)

const MaxMessageLen = 4000

func SplitLongText(text string, maxLen int) []string {
	if maxLen <= 0 {
		maxLen = MaxMessageLen
	}
	if len(text) <= maxLen {
		return []string{text}
	}
	var parts []string
	for len(text) > maxLen {
		cut := maxLen
		if idx := strings.LastIndexAny(text[:maxLen], "\n"); idx > maxLen/2 {
			cut = idx + 1
		} else if idx := strings.LastIndex(text[:maxLen], " "); idx > maxLen/2 {
			cut = idx + 1
		}
		parts = append(parts, text[:cut])
		text = text[cut:]
	}
	if len(text) > 0 {
		parts = append(parts, text)
	}
	return parts
}

func TruncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "\n… (truncated)"
}

func FormatLinesPerPage(text string, perPage int) []string {
	lines := strings.Split(text, "\n")
	var parts []string
	var buf []string
	for _, l := range lines {
		if len(buf) >= perPage {
			parts = append(parts, strings.Join(buf, "\n"))
			buf = nil
		}
		buf = append(buf, l)
	}
	if len(buf) > 0 {
		parts = append(parts, strings.Join(buf, "\n"))
	}
	return parts
}

type Cursor struct {
	Token string
	Page  int
}

func ChunkRow(s string, n int) []string {
	runes := []rune(s)
	if len(runes) <= n {
		return []string{s}
	}
	var out []string
	for i := 0; i < len(runes); i += n {
		end := i + n
		if end > len(runes) {
			end = len(runes)
		}
		out = append(out, string(runes[i:end]))
	}
	return out
}

func ShortID(prefix string, id int64) string {
	return fmt.Sprintf("%s:%d", prefix, id)
}

func ParsePage(data string, current int) int {
	parts := strings.Split(data, ":")
	if len(parts) > 0 {
		if n, err := strconv.Atoi(parts[len(parts)-1]); err == nil && n > 0 {
			return n
		}
	}
	return current
}
