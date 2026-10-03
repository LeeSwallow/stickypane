package app

import (
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/when"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// logRows is how much of a script's log its pane shows: the end of it.
// Zoomed in, the whole log is shown.
const logRows = 10

// exitLine is the last line run.go writes into a script's log.
var exitLine = regexp.MustCompile(`^\[exit (-?\d+) · (.+)\]$`)

// scriptExts are the files that are scripts, whose log of the same name is
// shown in their pane instead of as a note of its own.
var scriptExts = []string{".sh", ".ps1"}

// withLog is the widget of a script that has a log: the script, with its
// Run button, and under it a panel with the log of its last run. Keys and
// clicks go to the script; the panel only shows.
type withLog struct {
	owner   string // the extension of the note the log belongs to
	script  widget.Widget
	log     store.Note
	now     func() time.Time
	full    bool // zoomed in: the whole log, not its end
	scriptH int  // lines the script took in the last Draw
}

// logsOfScripts finds, among a tab's notes, the logs that belong to a
// note next to them, by that note's name: a script's output, or what the
// agent did with a form's answers (deploy.md and deploy.log).
func logsOfScripts(notes []store.Note) map[string]store.Note {
	byName := make(map[string]bool, len(notes))
	for _, n := range notes {
		if !n.Book() && (slices.Contains(scriptExts, path.Ext(n.Name)) || (path.Ext(n.Name) == ".md" && n.Doc.Type() == "form")) {
			byName[n.Name] = true
		}
	}
	logs := map[string]store.Note{}
	for _, n := range notes {
		if n.Book() || path.Ext(n.Name) != ".log" {
			continue
		}
		stem := strings.TrimSuffix(n.Name, ".log")
		for _, ext := range append([]string{".md"}, scriptExts...) {
			if byName[stem+ext] {
				logs[stem+ext] = n
			}
		}
	}
	return logs
}

// run reads what the log says about the last run: its output, and the
// exit code and duration from the last line.
func (w *withLog) run() (output []string, ended string, failed bool) {
	lines := doc.Lines(strings.TrimRight(w.log.Doc.Body, "\n"))
	if len(lines) > 0 && strings.HasPrefix(lines[0], "$ ") {
		lines = lines[1:] // the command and when, said by the divider
	}
	if n := len(lines); n > 0 {
		if m := exitLine.FindStringSubmatch(strings.TrimSpace(lines[n-1])); m != nil {
			lines = lines[:n-1]
			ended, failed = "exit "+m[1]+" · "+m[2], m[1] != "0"
		}
	}
	return lines, ended, failed
}

// Draw implements widget.Widget.
func (w *withLog) Draw(width int, active bool) (string, widget.Span) {
	out, at := w.script.Draw(width, active)
	lines := strings.Split(out, "\n")
	w.scriptH = len(lines)
	output, ended, failed := w.run()

	label := " " + widget.T("log")
	if path.Ext(w.log.Name) == ".log" && !slices.Contains(scriptExts, w.owner) {
		label = " " + widget.T("output") // what the agent did with a form
	}
	if t := when.Short(w.log.ModTime, w.now()); t != "" {
		label += " · " + t
	}
	rule := widget.Faint.Render("─" + label)
	if ended != "" {
		style := widget.Faint
		if failed {
			style = widget.Bad
		}
		rule += widget.Faint.Render(" · ") + style.Render(ended)
	}
	rule += " " + widget.Faint.Render(strings.Repeat("─", max(width-widget.Width(rule)-1, 0)))
	lines = append(lines, "", rule)

	if !w.full && len(output) > logRows {
		lines = append(lines, widget.Faint.Render(fmt.Sprintf(widget.T("… %d earlier lines"), len(output)-logRows)))
		output = output[len(output)-logRows:]
	}
	for _, l := range output {
		lines = append(lines, widget.Wrap(widget.Clean(l), max(width, 1))...)
	}
	return widget.Fit(strings.Join(lines, "\n"), width), at
}

// Summary implements widget.Widget: how the last run ended, else the
// script's own summary.
func (w *withLog) Summary() string {
	if _, ended, _ := w.run(); ended != "" {
		if t := when.Short(w.log.ModTime, w.now()); t != "" {
			return ended + " · " + t
		}
		return ended
	}
	return w.script.Summary()
}

// Update implements widget.Widget: keys belong to the script.
func (w *withLog) Update(key string) (widget.Widget, widget.Result) {
	s, res := w.script.Update(key)
	w.script = s
	return w, res
}

// Click implements widget.Clicker: a press on the script part goes to it.
func (w *withLog) Click(line, col int) (widget.Widget, widget.Result, bool) {
	if c, ok := w.script.(widget.Clicker); ok && line < w.scriptH {
		s, res, hit := c.Click(line, col)
		w.script = s
		return w, res, hit
	}
	return w, widget.Result{}, false
}

// Sync implements widget.Widget: the script changed; the log stays.
func (w *withLog) Sync(d doc.Document) widget.Widget {
	w.script = w.script.Sync(d)
	return w
}
