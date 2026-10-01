package logview

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

func parseBody(s string) widget.Widget { return Kind.Parse(doc.Document{Body: s}) }

func numbered(n int) string {
	var sb strings.Builder
	for i := 1; i <= n; i++ {
		sb.WriteString("line ")
		sb.WriteString(strings.Repeat("i", i))
		sb.WriteString("\n")
	}
	return sb.String()
}

func TestPreviewShowsLastFiveLines(t *testing.T) {
	got := parseBody(numbered(8) + "\n\n").Preview(40)
	want := "line iiii\nline iiiii\nline iiiiii\nline iiiiiii\nline iiiiiiii"
	if got != want {
		t.Errorf("Preview = %q, want %q", got, want)
	}
}

func TestPreviewOfEmptyLog(t *testing.T) {
	if got := ansi.Strip(parseBody("\n").Preview(40)); got != "(no entries yet)" {
		t.Errorf("Preview = %q", got)
	}
}

func TestPreviewTruncatesAndCleans(t *testing.T) {
	got := parseBody("14:02 \x1b[32mtests passed\x1b[0m and a very long tail\r\n").Preview(18)
	if got != "14:02 tests passe…" {
		t.Errorf("Preview = %q", got)
	}
}

func TestViewFollowsTheEnd(t *testing.T) {
	w := parseBody(numbered(6))
	if got := w.View(40, 2); got != "line iiiii\nline iiiiii" {
		t.Fatalf("View = %q", got)
	}
	w = w.Sync(doc.Document{Body: numbered(7)})
	if got := w.View(40, 2); got != "line iiiiii\nline iiiiiii" {
		t.Errorf("after a new line: %q", got)
	}
}

func TestScrollingStopsFollowingAndGResumes(t *testing.T) {
	w := parseBody(numbered(6))
	w.View(40, 2)
	w, _ = w.Update("k")
	if got := w.View(40, 2); got != "line iiii\nline iiiii" {
		t.Fatalf("after k: %q", got)
	}
	w = w.Sync(doc.Document{Body: numbered(7)})
	if got := w.View(40, 2); got != "line iiii\nline iiiii" {
		t.Errorf("should stay put while not following: %q", got)
	}
	w, _ = w.Update("G")
	if got := w.View(40, 2); got != "line iiiiii\nline iiiiiii" {
		t.Errorf("after G: %q", got)
	}
}

func TestViewWrapsLongLines(t *testing.T) {
	for _, line := range strings.Split(parseBody("a very long log line that does not fit\n").View(10, 20), "\n") {
		if w := widget.Width(line); w > 10 {
			t.Errorf("line %q is %d cells wide", line, w)
		}
	}
}

func TestKindTemplate(t *testing.T) {
	if got := string(Kind.Template("Work log")); got != "---\ntype: log\ntitle: Work log\n---\n" {
		t.Errorf("Template = %q", got)
	}
}
