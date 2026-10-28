// Package script is the note that runs: a shell script shown with a button.
// The widget only asks for the run; the app starts the script and collects
// its output.
package script

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/LeeSwallow/stickypane/internal/core/doc"
	"github.com/LeeSwallow/stickypane/internal/core/widget"
)

const label = "▶ Run"

// Kind registers the script. It is the kind of every ".sh" and ".ps1" file.
var Kind = widget.Kind{
	Name:     "script",
	Exts:     []string{".sh", ".ps1"},
	New:      ".sh",
	Label:    "Script",
	Icon:     "▶",
	Hint:     []string{"enter", "run"},
	Blurb:    "A shell script with a Run button. It runs in the project folder and its output is a log next to it.",
	Command:  "stickypane show deploy.sh",
	Usage:    "A .sh file, or .ps1 for PowerShell. enter, a click on Run or a double click asks the user first, then runs it in the project folder with sh or PowerShell, whichever the file needs; the output shows under the script, from a .log of the same name.",
	Example:  "#!/bin/sh\ngo test ./...\n",
	Keys:     "enter space",
	Size:     func(doc.Document) string { return widget.SizeHalf },
	Template: func(string) []byte { return []byte("#!/bin/sh\n") },
	Options: []widget.Option{
		{Name: "commands", Type: widget.OptLines, Label: "Commands"},
	},
	Compose: func(_ string, opts map[string]string) ([]byte, error) {
		// The commands as written: indentation means something in a script.
		file := "#!/bin/sh\n"
		if commands := strings.Trim(opts["commands"], "\n"); strings.TrimSpace(commands) != "" {
			file += commands + "\n"
		}
		return []byte(file), nil
	},
	Parse: func(d doc.Document, e widget.Env) widget.Widget { return parse(d, e) },
	Data:  data,
}

// Script is the widget for a script.
type Script struct {
	lines  []string
	button widget.Span // the lines of the button in the last Draw
	env    widget.Env
}

func parse(d doc.Document, e widget.Env) *Script {
	s := &Script{env: e}
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
		return widget.Fit(s.env.Styles.Faint.Render(s.env.T("(empty script)")), width), widget.NoSpan
	}
	var out []string
	inner := widget.Width(label)
	if width >= inner+4 {
		edge, text := s.env.Styles.Faint, lipgloss.NewStyle()
		tl, tr, bl, br, h, v := "╭", "╮", "╰", "╯", "─", "│"
		if active {
			edge, text = s.env.Styles.Bold, s.env.Styles.Selected
			tl, tr, bl, br, h, v = "┏", "┓", "┗", "┛", "━", "┃"
		}
		rule := strings.Repeat(h, inner+2)
		out = append(out,
			edge.Render(tl+rule+tr),
			edge.Render(v)+text.Render(" "+label+" ")+edge.Render(v),
			edge.Render(bl+rule+br),
		)
	} else {
		out = append(out, s.env.Styles.Bold.Render(widget.Truncate(label, width)))
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
			out = append(out, s.env.Styles.Faint.Render(number)+part)
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
		return s.env.T("1 line")
	default:
		return fmt.Sprintf(s.env.T("%d lines"), n)
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
func (s *Script) Sync(d doc.Document) widget.Widget { return parse(d, s.env) }

// data is the script as written and the language to run it in. A document
// does not know its file's name, so the language comes from the first line:
// "ps1" when it is a shebang for PowerShell, "sh" otherwise.
func data(d doc.Document, _ widget.Env) any {
	first, _, _ := strings.Cut(d.Body, "\n")
	lang := "sh"
	if strings.HasPrefix(first, "#!") && (strings.Contains(first, "pwsh") || strings.Contains(first, "powershell")) {
		lang = "ps1"
	}
	return struct {
		Source string `json:"source"`
		Lang   string `json:"lang"`
	}{d.Body, lang}
}
