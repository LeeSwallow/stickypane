package logview

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

func parseBody(s string) widget.Widget { return Kind.Parse(doc.Document{Body: s}) }

func TestDrawShowsEveryEntry(t *testing.T) {
	out, at := parseBody("one\ntwo\n\nthree\n\n\n").Draw(40, true)
	if out != "one\ntwo\n\nthree" {
		t.Errorf("Draw = %q, want every entry and no trailing blank lines", out)
	}
	if at.Ok() {
		t.Errorf("span = %+v, want none", at)
	}
}

func TestDrawOfEmptyLog(t *testing.T) {
	if got, _ := parseBody("\n").Draw(40, false); ansi.Strip(got) != "(no entries yet)" {
		t.Errorf("Draw = %q", got)
	}
}

func TestDrawWrapsAndCleans(t *testing.T) {
	out, _ := parseBody("14:02 \x1b[32mtests passed\x1b[0m and a very long tail that goes on\r\n").Draw(18, false)
	if strings.Contains(out, "\x1b") || strings.Contains(out, "\r") {
		t.Errorf("control characters were kept: %q", out)
	}
	for _, line := range strings.Split(out, "\n") {
		if w := widget.Width(line); w > 18 {
			t.Errorf("line %q is %d cells wide", line, w)
		}
	}
	if !strings.Contains(strings.ReplaceAll(out, "\n", " "), "goes on") {
		t.Errorf("wrapping lost the end of the line: %q", out)
	}
}

func TestSyncReplacesContent(t *testing.T) {
	w := parseBody("a\n").Sync(doc.Document{Body: "a\nb\n"})
	if got, _ := w.Draw(40, false); got != "a\nb" {
		t.Errorf("Draw after Sync = %q", got)
	}
}

func TestKind(t *testing.T) {
	if Kind.Name != "log" || !Kind.Tail || Kind.Rows != 10 || Kind.Keys != "" {
		t.Errorf("Kind = %+v", Kind)
	}
	if got := Kind.Size(doc.Document{}); got != widget.SizeHalf {
		t.Errorf("Size = %q", got)
	}
	if got := string(Kind.Template("Work log")); got != "---\ntype: log\ntitle: Work log\n---\n" {
		t.Errorf("Template = %q", got)
	}
}

func TestSummaryCountsEntries(t *testing.T) {
	if got := parseBody("one\ntwo\n\nthree\n").Summary(); got != "3 lines" {
		t.Errorf("Summary = %q", got)
	}
	if got := parseBody("one\n").Summary(); got != "1 line" {
		t.Errorf("Summary = %q", got)
	}
	if got := parseBody("").Summary(); got != "" {
		t.Errorf("Summary of an empty log = %q", got)
	}
}

func TestKindDescribesItself(t *testing.T) {
	if Kind.Icon == "" || widget.Width(Kind.Icon) != 1 || Kind.Blurb == "" || Kind.Example == "" {
		t.Errorf("Kind needs a one-cell icon, a blurb and an example: %+v", Kind)
	}
}
