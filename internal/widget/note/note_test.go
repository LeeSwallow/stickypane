package note

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

func parse(src string) widget.Widget {
	return NewKind(Plain).Parse(doc.Parse([]byte(src)))
}

func numbered(n int) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		b.WriteString("l")
		b.WriteString(strings.Repeat("i", i))
		b.WriteString("\n")
	}
	return b.String()
}

func TestPreviewShowsShortNoteAsIs(t *testing.T) {
	if got := parse("check env before deploy\n").Preview(30); got != "check env before deploy" {
		t.Errorf("Preview = %q", got)
	}
}

func TestPreviewHidesFrontMatter(t *testing.T) {
	if got := parse("---\ntitle: T\n---\nhello\n").Preview(30); got != "hello" {
		t.Errorf("Preview = %q", got)
	}
}

func TestPreviewCapsAtSixLines(t *testing.T) {
	lines := strings.Split(ansi.Strip(parse(numbered(10)).Preview(30)), "\n")
	if len(lines) != 6 {
		t.Fatalf("got %d lines, want 6", len(lines))
	}
	if lines[4] != "liiiii" || lines[5] != "…" {
		t.Errorf("last lines = %q, %q", lines[4], lines[5])
	}
}

func TestPreviewOfEmptyNote(t *testing.T) {
	if got := ansi.Strip(parse("").Preview(30)); got != "(empty)" {
		t.Errorf("Preview = %q", got)
	}
}

func TestPreviewDropsControlCharacters(t *testing.T) {
	if got := parse("a\tb\x1b[31m red\x1b[0m\x07\n").Preview(30); got != "a    b red" {
		t.Errorf("Preview = %q", got)
	}
}

func TestPreviewWrapsToWidth(t *testing.T) {
	for _, line := range strings.Split(parse("토큰은 세션 쿠키로 보관한다 and some more words here\n").Preview(12), "\n") {
		if w := widget.Width(line); w > 12 {
			t.Errorf("line %q is %d cells wide, want at most 12", line, w)
		}
	}
}

func TestViewScrolls(t *testing.T) {
	w := parse(numbered(6))
	if got := w.View(20, 2); got != "li\nlii" {
		t.Fatalf("View = %q", got)
	}
	w, _ = w.Update("j")
	if got := w.View(20, 2); got != "lii\nliii" {
		t.Errorf("after j: %q", got)
	}
	w, _ = w.Update("G")
	if got := w.View(20, 2); got != "liiiii\nliiiiii" {
		t.Errorf("after G: %q", got)
	}
	w, _ = w.Update("k")
	if got := w.View(20, 2); got != "liiii\nliiiii" {
		t.Errorf("after G then k: %q", got)
	}
}

func TestSyncKeepsScrollPosition(t *testing.T) {
	w := parse(numbered(6))
	w, _ = w.Update("j")
	w.View(20, 2)
	w = w.Sync(doc.Parse([]byte("a\nb\nc\nd\n")))
	if got := w.View(20, 2); got != "b\nc" {
		t.Errorf("View after Sync = %q", got)
	}
}

func TestKind(t *testing.T) {
	k := NewKind(Plain)
	if k.Name != "note" || k.FullRow {
		t.Errorf("kind = %+v", k)
	}
	if got := string(k.Template("Plan")); got != "---\ntitle: Plan\n---\n" {
		t.Errorf("Template = %q", got)
	}
}
