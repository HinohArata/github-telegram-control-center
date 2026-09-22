package telegram

import (
	"strings"
	"testing"
)

func TestSplitLongText(t *testing.T) {
	short := "one line"
	if parts := SplitLongText(short, 100); len(parts) != 1 || parts[0] != short {
		t.Fatalf("SplitLongText(short) = %v", parts)
	}

	paragraphs := strings.Repeat("hello world\n", 400) // ~4800 bytes
	parts := SplitLongText(paragraphs, 1000)
	if len(parts) < 2 {
		t.Fatalf("expected split into multiple parts, got %d", len(parts))
	}
	total := strings.Join(parts, "")
	if total != paragraphs {
		t.Fatalf("SplitLongText lost data: input %d bytes, output %d bytes", len(paragraphs), len(total))
	}
	for i, p := range parts {
		if len(p) > 1000 {
			t.Fatalf("part %d exceeds maxLen: %d bytes", i, len(p))
		}
	}
}

func TestSplitLongTextFallsBackToDefault(t *testing.T) {
	in := strings.Repeat("a", MaxMessageLen+50)
	parts := SplitLongText(in, 0)
	if len(parts) != 2 {
		t.Fatalf("SplitLongText(in, 0) = %d parts; want 2", len(parts))
	}
}

func TestSplitLongTextPrefersWordBoundary(t *testing.T) {
	in := strings.Repeat("word ", 500)
	parts := SplitLongText(in, 100)
	if len(parts) < 2 {
		t.Fatalf("expected split, got %d", len(parts))
	}
	// A word-boundary cut keeps whole words: every part but the last should
	// end on whitespace and never split a word in half.
	for _, p := range parts[:len(parts)-1] {
		if len(p) == 0 || p[len(p)-1] != ' ' {
			t.Fatalf("part %q does not end on a word boundary", p)
		}
	}
}

func TestTruncateText(t *testing.T) {
	if got := TruncateText("short", 10); got != "short" {
		t.Fatalf("TruncateText(short) = %q", got)
	}
	got := TruncateText("0123456789abcdef", 8)
	if !strings.HasPrefix(got, "01234567") || !strings.Contains(got, "truncated") {
		t.Fatalf("TruncateText = %q", got)
	}
}

func TestFormatLinesPerPage(t *testing.T) {
	lines := []string{"a", "b", "c", "d", "e"}
	text := strings.Join(lines, "\n")
	pages := FormatLinesPerPage(text, 2)
	if len(pages) != 3 {
		t.Fatalf("FormatLinesPerPage = %d pages; want 3", len(pages))
	}
	want := []string{"a\nb", "c\nd", "e"}
	for i := range want {
		if pages[i] != want[i] {
			t.Fatalf("page %d = %q; want %q", i, pages[i], want[i])
		}
	}
	if joined := strings.Join(pages, "\n"); joined != text {
		t.Fatalf("FormatLinesPerPage lost data: %q", joined)
	}
}

func TestChunkRow(t *testing.T) {
	chunks := ChunkRow("abcdefghij", 4)
	want := []string{"abcd", "efgh", "ij"}
	if len(chunks) != len(want) {
		t.Fatalf("ChunkRow = %v; want %v", chunks, want)
	}
	for i := range want {
		if chunks[i] != want[i] {
			t.Fatalf("chunk %d = %q; want %q", i, chunks[i], want[i])
		}
	}
}

func TestParsePage(t *testing.T) {
	cases := []struct {
		data    string
		current int
		want    int
	}{
		{"repos:5", 1, 5},
		{"repos:5", 1, 5},
		{"repos:p5", 1, 1},
		{"repos", 3, 3},
		{"repos:0", 2, 2},
		{"repos:abc", 7, 7},
		{"", 9, 9},
	}
	for _, c := range cases {
		if got := ParsePage(c.data, c.current); got != c.want {
			t.Errorf("ParsePage(%q, %d) = %d; want %d", c.data, c.current, got, c.want)
		}
	}
}

func TestShortID(t *testing.T) {
	if got := ShortID("run", 42); got != "run:42" {
		t.Errorf("ShortID = %q", got)
	}
}

func TestPageFromData(t *testing.T) {
	cases := []struct {
		data string
		want int
	}{
		{"repos:p3", 3},
		{"prs:p12:yes", 12},
		{"runs", 1},
		{"x:p0", 1},
		{"x:p-1", 1},
	}
	for _, c := range cases {
		if got := PageFromData(c.data); got != c.want {
			t.Errorf("PageFromData(%q) = %d; want %d", c.data, got, c.want)
		}
	}
}
