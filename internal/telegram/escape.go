package telegram

import (
	"html"
	"regexp"
	"strings"
)

var (
	mdReserved = regexp.MustCompile(`[_*\[\]()~` + "`" + `>#+\-=|{}.!\\]`)
	htmlTagsRe = regexp.MustCompile(`<[^>]*>`)
)

func EscapeMarkdown(s string) string {
	return mdReserved.ReplaceAllStringFunc(s, func(m string) string {
		return "\\" + m
	})
}

func EscapeHTML(s string) string {
	safe := html.EscapeString(s)
	safe = strings.ReplaceAll(safe, "&amp;", "&amp;")
	return safe
}

func EscapeH(s string) string {
	return html.EscapeString(s)
}

type PlainEscape struct{}

func (PlainEscape) Escape(s string) string { return html.EscapeString(s) }