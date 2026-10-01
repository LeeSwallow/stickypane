// Package script is the note that runs: a shell script shown with a button.
// The widget only asks for the run; the app starts the script and collects
// its output.
package script

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

const label = "▶ Run"

// Kind registers the script. It is the kind of every ".sh" file.
var Kind = widget.Kind{
	Name:     "script",
	Exts:     []string{".sh"},
	New:      ".sh",
	Label:    "Script",
	Icon:     "▶",
	Hint:     []string{"enter", "run"},
	Blurb:    "A shell script with a Run button. It runs in the project folder and its output is a log next to it.",
	Example:  "#!/bin/sh\ngo test ./...\n",
	Keys:     "enter space",
	Size:     func(doc.Document) string { return widget.SizeHalf },
	Template: func(string) []byte { return []byte("#!/bin/sh\n") },
	Parse:    func(d doc.Document) widget.Widget { return parse(d) },
}

// Script is the widget for a script.
type Script struct {
	lines  []string
	button widget.Span // the lines of the button in the last Draw
}

func parse(d doc.Document) *Script {
	s := &Script{}
	for _, l := range doc.Lines(d.Body) {
		s.lines = append(s.lines, strings.TrimRight(widget.Clean(l), " "))
	}
	for n := len(s.lines); n > 0 && s.lines[n-1] == ""; n = len(s.lines) {
		s.lines = s.lines[:n-1]
	}
	return s
}

// Draw implements widget.Widget: the Run button, then the script with line
// numbers so that what will run can be read before it is run.
func (s *Script) Draw(width int, active bool) (string, widget.Span) {
	if len(s.lines) == 0 {
		s.button = widget.NoSpan
		return widget.Fit(widget.Faint.Render("(empty script)"), width), widget.NoSpan
	}
	var out []string
	inner := widget.Width(label)
	if width >= inner+4 {
		edge, text := widget.Faint, lipgloss.NewStyle()
		tl, tr, bl, br, h, v := "╭", "╮", "╰", "╯", "─", "│"
		if active {
			edge, text = widget.Bold, widget.Selected
			tl, tr, bl, br, h, v = "┏", "┓", "┗", "┛", "━", "┃"
		}
		rule := strings.Repeat(h, inner+2)
		out = append(out,
			edge.Render(tl+rule+tr),
			edge.Render(v)+text.Render(" "+label+" ")+edge.Render(v),
			edge.Render(bl+rule+br),
		)
	} else {
		out = append(out, widget.Bold.Render(widget.Truncate(label, width)))
	}
	s.button = widget.Span{Start: 0, End: len(out)}
	out = append(out, "")
	digits := len(fmt.Sprint(len(s.lines)))
	for i, l := range s.lines {
		number := fmt.Sprintf("%*d ", digits, i+1)
		for j, part := range widget.Wrap(l, max(width-digits-1, 1)) {
			if j > 0 {
				number = strings.Repeat(" ", digits+1)
			}
			out = append(out, widget.Faint.Render(number)+part)
		}
	}
	return widget.Fit(strings.Join(out, "\n"), width), s.button
}

// Summary implements widget.Widget: how long the script is.
func (s *Script) Summary() string {
	switch n := len(s.lines); n {
	case 0:
		return ""
	case 1:
		return "1 line"
	default:
		return fmt.Sprintf("%d lines", n)
	}
}

// Update implements widget.Widget: enter and space ask to run the script.
func (s *Script) Update(key string) (widget.Widget, widget.Result) {
	if (key == "enter" || key == "space") && len(s.lines) > 0 {
		return s, widget.Result{Run: true}
	}
	return s, widget.Result{}
}

// Click implements widget.Clicker: a press on the button asks to run.
func (s *Script) Click(line, _ int) (widget.Widget, widget.Result, bool) {
	if s.button.Ok() && line >= s.button.Start && line < s.button.End && len(s.lines) > 0 {
		return s, widget.Result{Run: true}, true
	}
	return s, widget.Result{}, false
}

// Sync implements widget.Widget.
func (s *Script) Sync(d doc.Document) widget.Widget { return parse(d) }
