// Package note is the plain note: any Markdown, from one line to a report.
package note

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// A note's default size follows its length, counted in non-blank lines.
const (
	cardLines = 3  // up to this many lines, a note is a small card
	halfLines = 12 // up to this many, it takes half the width; longer notes get the page
)

// Renderer turns Markdown into terminal text no wider than width.
type Renderer func(markdown string, width int) string

// Plain wraps text without styling it.
func Plain(markdown string, width int) string {
	if width <= 0 {
		return markdown
	}
	return ansi.Wrap(markdown, width, "")
}

// NewKind returns the plain note kind drawing with the given renderer.
func NewKind(render Renderer) widget.Kind {
	return widget.Kind{
		Name:     "note",
		Label:    "Note",
		Icon:     "✎",
		Blurb:    "Anything in Markdown, from one line to a full page.",
		Example:  "Decided: tokens live in a session cookie.\n\n- revisit when we add SSO\n",
		Size:     size,
		Template: func(title string) []byte { return widget.NewFile("note", title, "") },
		Parse:    func(d doc.Document) widget.Widget { return newNote(d.Body, render) },
	}
}

func size(d doc.Document) string {
	n := 0
	for _, line := range doc.Lines(d.Body) {
		if strings.TrimSpace(line) != "" {
			n++
		}
	}
	switch {
	case n <= cardLines:
		return widget.SizeCard
	case n <= halfLines:
		return widget.SizeHalf
	}
	return widget.SizePage
}

// Note is the widget for a plain note.
type Note struct {
	body   string
	render Renderer
	cache  map[int][]string // rendered lines by width
}

func newNote(body string, render Renderer) *Note {
	return &Note{body: body, render: render, cache: map[int][]string{}}
}

// lines renders the body at width and trims blank lines from both ends.
func (n *Note) lines(width int) []string {
	if l, ok := n.cache[width]; ok {
		return l
	}
	l := strings.Split(n.render(widget.Clean(n.body), width), "\n")
	blank := func(s string) bool { return strings.TrimSpace(ansi.Strip(s)) == "" }
	for len(l) > 0 && blank(l[0]) {
		l = l[1:]
	}
	for len(l) > 0 && blank(l[len(l)-1]) {
		l = l[:len(l)-1]
	}
	n.cache[width] = l
	return l
}

// Draw implements widget.Widget. A plain note has no cursor.
func (n *Note) Draw(width int, _ bool) (string, widget.Span) {
	l := n.lines(width)
	if len(l) == 0 {
		return widget.Faint.Render("(empty)"), widget.NoSpan
	}
	return strings.Join(l, "\n"), widget.NoSpan
}

// Summary implements widget.Widget. A plain note has nothing to count.
func (n *Note) Summary() string { return "" }

// Update implements widget.Widget. A plain note takes no keys.
func (n *Note) Update(string) (widget.Widget, widget.Result) { return n, widget.Result{} }

// Sync implements widget.Widget.
func (n *Note) Sync(d doc.Document) widget.Widget { return newNote(d.Body, n.render) }
