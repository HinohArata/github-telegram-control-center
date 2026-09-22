package telegram

import "testing"

func TestEscapeMarkdown(t *testing.T) {
	cases := map[string]string{
		"hello":                      "hello",
		"bold *text*":                `bold \*text\*`,
		"_under_ [x](y)":             `\_under\_ \[x\]\(y\)`,
		"`code` and ~~str~~":         "\\`code\\` and \\~\\~str\\~\\~",
		"a>b c#1 d-e f=g":            `a\>b c\#1 d\-e f\=g`,
		"pipe | brace {1} plus + h!": `pipe \| brace \{1\} plus \+ h\!`,
	}
	for in, want := range cases {
		if got := EscapeMarkdown(in); got != want {
			t.Errorf("EscapeMarkdown(%q) = %q; want %q", in, got, want)
		}
	}
}

func TestEscapeMarkdownPreservesUserContent(t *testing.T) {
	in := "bold *text* and _more_"
	got := EscapeMarkdown(in)
	for _, bad := range []string{"*text*", "_more_"} {
		if contains(got, bad) {
			t.Errorf("EscapeMarkdown(%q) left markdown-active sequence %q: %q", in, bad, got)
		}
	}
}

func TestEscapeHTML(t *testing.T) {
	cases := map[string]string{
		"plain":              "plain",
		"<script>x</script>": "&lt;script&gt;x&lt;/script&gt;",
		"a & b":              "a &amp; b",
		`"q" & 's'`:          "&#34;q&#34; &amp; &#39;s&#39;",
	}
	for in, want := range cases {
		if got := EscapeHTML(in); got != want {
			t.Errorf("EscapeHTML(%q) = %q; want %q", in, got, want)
		}
		if got := EscapeH(in); got != want {
			t.Errorf("EscapeH(%q) = %q; want %q", in, got, want)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
