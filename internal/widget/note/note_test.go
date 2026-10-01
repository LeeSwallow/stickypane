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

func TestDrawShowsTheWholeNote(t *testing.T) {
	out, at := parse(numbered(40)).Draw(60, true)
	lines := strings.Split(out, "\n")
	if len(lines) != 40 || lines[39] != "l"+strings.Repeat("i", 40) {
		t.Errorf("got %d lines, last %q; want all 40 lines", len(lines), lines[len(lines)-1])
	}
	if at.Ok() {
		t.Errorf("span = %+v, want none: a plain note has no cursor", at)
	}
}

func TestDrawHidesFrontMatter(t *testing.T) {
	if got, _ := parse("---\ntitle: T\n---\nhello\n").Draw(30, false); got != "hello" {
		t.Errorf("Draw = %q", got)
	}
}

func TestDrawOfEmptyNote(t *testing.T) {
	if got, _ := parse("").Draw(30, false); ansi.Strip(got) != "(empty)" {
		t.Errorf("Draw = %q", got)
	}
}

func TestDrawDropsControlCharacters(t *testing.T) {
	if got, _ := parse("a\tb\x1b[31m red\x1b[0m\x07\n").Draw(30, false); got != "a    b red" {
		t.Errorf("Draw = %q", got)
	}
}

func TestDrawWrapsToWidth(t *testing.T) {
	out, _ := parse("토큰은 세션 쿠키로 보관한다 and some more words here\n").Draw(12, false)
	for _, line := range strings.Split(out, "\n") {
		if w := widget.Width(line); w > 12 {
			t.Errorf("line %q is %d cells wide, want at most 12", line, w)
		}
	}
	if !strings.Contains(strings.ReplaceAll(out, "\n", " "), "words here") {
		t.Errorf("wrapping lost text: %q", out)
	}
}

func TestSyncReplacesContent(t *testing.T) {
	w := parse("old\n").Sync(doc.Parse([]byte("new\n")))
	if got, _ := w.Draw(30, false); got != "new" {
		t.Errorf("Draw after Sync = %q", got)
	}
}

func TestSizeFollowsLength(t *testing.T) {
	k := NewKind(Plain)
	cases := []struct {
		body string
		want string
	}{
		{"one line\n", widget.SizeCard},
		{"a\n\nb\n\nc\n", widget.SizeCard}, // blank lines do not count
		{numbered(4), widget.SizeHalf},
		{numbered(12), widget.SizeHalf},
		{numbered(13), widget.SizePage},
	}
	for _, c := range cases {
		if got := k.Size(doc.Document{Body: c.body}); got != c.want {
			t.Errorf("Size of %d lines = %q, want %q", strings.Count(c.body, "\n"), got, c.want)
		}
	}
}

func TestKind(t *testing.T) {
	k := NewKind(Plain)
	if k.Name != "note" || k.Keys != "" || k.Tail || k.Rows != 0 {
		t.Errorf("kind = %+v", k)
	}
	if got := string(k.Template("Plan")); got != "---\ntitle: Plan\n---\n" {
		t.Errorf("Template = %q", got)
	}
}

func TestSummaryIsEmptyAndKindDescribesItself(t *testing.T) {
	if got := parse("hello\n").Summary(); got != "" {
		t.Errorf("Summary = %q, want none for a plain note", got)
	}
	k := NewKind(Plain)
	if k.Icon == "" || widget.Width(k.Icon) != 1 || k.Blurb == "" || k.Example == "" {
		t.Errorf("Kind needs a one-cell icon, a blurb and an example: %+v", k)
	}
}
