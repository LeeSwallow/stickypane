package diff

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/core/doc"
	"github.com/LeeSwallow/stickypane/internal/core/widget"
)

// Page draws the files of a diff: each one's name and counts, how it
// changed, then its hunks, added lines in the good color and removed ones
// in the bad.
type Page struct {
	files []File
	env   widget.Env
}

// NewPage reads text, a unified diff, into a page.
func NewPage(text string, e widget.Env) *Page { return &Page{files: Parse(text), env: e} }

// Head is the first line of a changed file: its status, its path and the
// lines it added and removed, "M internal/x.go +12 −3".
func Head(status, path string, add, del int, binary bool) string {
	if binary {
		return status + " " + path + "  binary"
	}
	return fmt.Sprintf("%s %s  +%d −%d", status, path, add, del)
}

// Draw implements widget.Widget.
func (p *Page) Draw(width int, _ bool) (string, widget.Span) {
	s := p.env.Styles
	var out []string
	put := func(st lipgloss.Style, line string) {
		for _, l := range wrap(line, width) {
			out = append(out, st.Render(l))
		}
	}
	if len(p.files) == 0 {
		put(s.Faint, p.env.T("no changes"))
	}
	for i, f := range p.files {
		if i > 0 {
			out = append(out, "")
		}
		add, del := f.Counts()
		path := f.Path()
		if f.Status == "R" || f.Status == "C" {
			path = f.Old + " → " + f.New
		}
		put(s.Bold, Head(f.Status, path, add, del, f.Binary))
		if f.Similarity != "" {
			put(s.Faint, p.env.T("similarity")+" "+f.Similarity)
		}
		if f.OldMode != "" && f.NewMode != "" {
			put(s.Faint, "mode "+f.OldMode+" → "+f.NewMode)
		}
		for _, h := range f.Hunks {
			put(s.Info, h.Header)
			for _, l := range h.Lines {
				switch l.Op {
				case '+':
					put(s.Good, "+ "+l.Text)
				case '-':
					put(s.Bad, "- "+l.Text)
				case '\\':
					put(s.Faint, `\`+l.Text)
				default:
					put(lipgloss.NewStyle(), "  "+l.Text)
				}
			}
		}
	}
	return widget.Fit(strings.Join(out, "\n"), width), widget.NoSpan
}

// wrap breaks a line of code at the width, spaces and all, so that its
// indent stays where it was: a diff is never cut, only wrapped.
func wrap(line string, width int) []string {
	return strings.Split(ansi.Hardwrap(widget.Clean(line), max(width, 1), true), "\n")
}

// Summary implements widget.Widget: the lines added and removed.
func (p *Page) Summary() string {
	add, del := 0, 0
	for _, f := range p.files {
		a, d := f.Counts()
		add, del = add+a, del+d
	}
	if add+del == 0 {
		return ""
	}
	return fmt.Sprintf("+%d −%d", add, del)
}

// Update implements widget.Widget. A diff takes no keys; the pane scrolls.
func (p *Page) Update(string) (widget.Widget, widget.Result) { return p, widget.Result{} }

// Sync implements widget.Widget.
func (p *Page) Sync(d doc.Document) widget.Widget { return NewPage(d.Body, p.env) }

// List is the first page: the worktrees, one a line, with the files two of
// them changed and the pull request of the chosen one under them. j and k
// move the cursor; what enter and b do the board decides, since they
// change what it keeps in sticky.json.
type List struct {
	o   Overview
	at  int
	env widget.Env
}

// NewList makes the list page of o with the cursor on the worktree called
// cursor, or on the chosen one when there is none by that name.
func NewList(o Overview, cursor string, e widget.Env) *List {
	l := &List{o: o, env: e}
	for i, w := range o.Worktrees {
		if w.Name == o.Chosen {
			l.at = i
		}
	}
	for i, w := range o.Worktrees {
		if w.Name == cursor {
			l.at = i
		}
	}
	return l
}

// Cursor is the worktree the cursor is on, or "" when there is none.
func (l *List) Cursor() string {
	if l.at < len(l.o.Worktrees) {
		return l.o.Worktrees[l.at].Name
	}
	return ""
}

// Draw implements widget.Widget.
func (l *List) Draw(width int, active bool) (string, widget.Span) {
	s := l.env.Styles
	var out []string
	put := func(st lipgloss.Style, line string) {
		for _, w := range wrap(line, width) {
			out = append(out, st.Render(w))
		}
	}
	at := widget.NoSpan
	if len(l.o.Worktrees) == 0 {
		put(s.Faint, l.env.T("no git worktrees here"))
	}
	for i, w := range l.o.Worktrees {
		mark, st := "  ", lipgloss.NewStyle()
		if i == l.at {
			mark = "▸ "
			if active {
				st = s.Accent
			}
		}
		if w.Name == l.o.Chosen {
			st = st.Bold(true)
		}
		start := len(out)
		put(st, mark+w.Name+"  "+l.row(w))
		if i == l.at {
			at = widget.Span{Start: start, End: len(out)}
		}
	}
	for _, o := range l.o.Overlaps {
		put(s.Warn, "⚠ "+o.Path)
		put(s.Faint, "  "+fmt.Sprintf(l.env.T("changed in %s"), strings.Join(o.By, " · ")))
	}
	if pr := l.o.PR; pr != nil {
		put(s.Faint, l.env.T("PR "+strings.ToLower(pr.State))+" · "+pr.Title)
	}
	return widget.Fit(strings.Join(out, "\n"), width), at
}

// row is what a worktree's line says after its name: its base, then its
// changed files and lines, or why they are not known.
func (l *List) row(w Worktree) string {
	switch {
	case w.Closed:
		return l.env.T("closed")
	case w.Err != "":
		return w.Err
	case w.Base == "":
		return l.env.T("no base: b chooses one")
	}
	files := fmt.Sprintf(l.env.T("%d files"), w.Files)
	if w.Files == 1 {
		files = l.env.T("1 file")
	}
	return fmt.Sprintf("%s…  %s  +%d −%d", w.Base, files, w.Add, w.Del)
}

// Summary implements widget.Widget: how many files two worktrees changed.
func (l *List) Summary() string {
	if n := len(l.o.Overlaps); n > 0 {
		return fmt.Sprintf("⚠ %d", n)
	}
	return ""
}

// Update implements widget.Widget: j and k move the cursor.
func (l *List) Update(key string) (widget.Widget, widget.Result) {
	switch key {
	case "j", "down":
		l.at = min(l.at+1, max(len(l.o.Worktrees)-1, 0))
	case "k", "up":
		l.at = max(l.at-1, 0)
	}
	return l, widget.Result{}
}

// Sync implements widget.Widget: the board makes the list again from git,
// not from a file.
func (l *List) Sync(doc.Document) widget.Widget { return l }
