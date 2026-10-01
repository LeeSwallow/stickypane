// Package logview is the log note: one entry per line, newest at the end.
package logview

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// previewLines is how many of the latest entries the board shows.
const previewLines = 5

// Kind registers the log.
var Kind = widget.Kind{
	Name:     "log",
	Label:    "Log",
	Template: func(title string) []byte { return widget.NewFile("log", title, "") },
	Parse:    func(d doc.Document) widget.Widget { return parse(d, 0, true) },
}

// Log is the widget for a log note.
type Log struct {
	lines  []string
	offset int
	follow bool // stay at the end as lines arrive
}

func parse(d doc.Document, offset int, follow bool) *Log {
	l := &Log{offset: offset, follow: follow}
	for _, line := range doc.Lines(d.Body) {
		l.lines = append(l.lines, strings.TrimRight(widget.Clean(line), " "))
	}
	for n := len(l.lines); n > 0 && l.lines[n-1] == ""; n = len(l.lines) {
		l.lines = l.lines[:n-1]
	}
	return l
}

// Preview implements widget.Widget.
func (l *Log) Preview(width int) string {
	var out []string
	for i := len(l.lines) - 1; i >= 0 && len(out) < previewLines; i-- {
		if l.lines[i] != "" {
			out = append([]string{widget.Truncate(l.lines[i], width)}, out...)
		}
	}
	if len(out) == 0 {
		return widget.Faint.Render("(no entries yet)")
	}
	return strings.Join(out, "\n")
}

// View implements widget.Widget.
func (l *Log) View(width, height int) string {
	var wrapped []string
	for _, line := range l.lines {
		if width > 0 {
			line = ansi.Wrap(line, width, "")
		}
		wrapped = append(wrapped, strings.Split(line, "\n")...)
	}
	if l.follow {
		l.offset = len(wrapped)
	}
	l.offset = widget.ClampOffset(l.offset, len(wrapped), height)
	return strings.Join(widget.Window(wrapped, l.offset, height), "\n")
}

// Update implements widget.Widget.
func (l *Log) Update(key string) (widget.Widget, widget.Result) {
	if key == "G" {
		l.follow = true
	} else if off, ok := widget.ScrollKey(l.offset, key); ok {
		l.offset, l.follow = off, false
	}
	return l, widget.Result{}
}

// Sync implements widget.Widget.
func (l *Log) Sync(d doc.Document) widget.Widget { return parse(d, l.offset, l.follow) }
