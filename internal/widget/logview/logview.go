// Package logview is the log note: one entry per line, newest at the end.
package logview

import (
	"fmt"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// rows is how tall a log is on the board unless the note says otherwise.
const rows = 10

// Kind registers the log. Its fixed-height view shows the end, so the
// newest entries stay in sight as the file grows.
var Kind = widget.Kind{
	Name:     "log",
	Label:    "Log",
	Icon:     "≣",
	Blurb:    "One entry per line. The newest entries stay in sight.",
	Example:  "14:02 tests passed\n14:10 started on review feedback\n14:31 pushed the fix\n",
	Tail:     true,
	Rows:     rows,
	Size:     func(doc.Document) string { return widget.SizeHalf },
	Template: func(title string) []byte { return widget.NewFile("log", title, "") },
	Parse:    func(d doc.Document) widget.Widget { return parse(d) },
}

// Log is the widget for a log note.
type Log struct{ lines []string }

func parse(d doc.Document) *Log {
	l := &Log{}
	for _, line := range doc.Lines(d.Body) {
		l.lines = append(l.lines, strings.TrimRight(widget.Clean(line), " "))
	}
	for n := len(l.lines); n > 0 && l.lines[n-1] == ""; n = len(l.lines) {
		l.lines = l.lines[:n-1]
	}
	return l
}

// Draw implements widget.Widget. A log has no cursor.
func (l *Log) Draw(width int, _ bool) (string, int) {
	if len(l.lines) == 0 {
		return widget.Faint.Render("(no entries yet)"), -1
	}
	var out []string
	// A log shows the file as it is: no styling, only wrapping so that
	// nothing is cut.
	for _, line := range l.lines {
		out = append(out, widget.Wrap(line, width)...)
	}
	return strings.Join(out, "\n"), -1
}

// Summary implements widget.Widget: how many entries the log has.
func (l *Log) Summary() string {
	n := 0
	for _, line := range l.lines {
		if line != "" {
			n++
		}
	}
	switch n {
	case 0:
		return ""
	case 1:
		return "1 line"
	}
	return fmt.Sprintf("%d lines", n)
}

// Update implements widget.Widget. A log takes no keys.
func (l *Log) Update(string) (widget.Widget, widget.Result) { return l, widget.Result{} }

// Sync implements widget.Widget.
func (l *Log) Sync(d doc.Document) widget.Widget { return parse(d) }
