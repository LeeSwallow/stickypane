// Package note is the plain note: any Markdown, from one line to a report.
package note

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// previewLines is the tallest a plain note gets on the board.
const previewLines = 6

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
		Template: func(title string) []byte { return widget.NewFile("note", title, "") },
		Parse:    func(d doc.Document) widget.Widget { return newNote(d.Body, render, 0) },
	}
}

// Note is the widget for a plain note.
type Note struct {
	body   string
	render Renderer
	cache  map[int][]string // rendered lines by width
	offset int
}

func newNote(body string, render Renderer, offset int) *Note {
	return &Note{body: body, render: render, cache: map[int][]string{}, offset: offset}
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

// Preview implements widget.Widget.
func (n *Note) Preview(width int) string {
	l := n.lines(width)
	if len(l) == 0 {
		return widget.Faint.Render("(empty)")
	}
	if len(l) > previewLines {
		l = append(append([]string(nil), l[:previewLines-1]...), widget.Faint.Render("…"))
	}
	return strings.Join(l, "\n")
}

// View implements widget.Widget.
func (n *Note) View(width, height int) string {
	l := n.lines(width)
	n.offset = widget.ClampOffset(n.offset, len(l), height)
	return strings.Join(widget.Window(l, n.offset, height), "\n")
}

// Update implements widget.Widget. A plain note only scrolls.
func (n *Note) Update(key string) (widget.Widget, widget.Result) {
	n.offset, _ = widget.ScrollKey(n.offset, key)
	return n, widget.Result{}
}

// Sync implements widget.Widget.
func (n *Note) Sync(d doc.Document) widget.Widget {
	return newNote(d.Body, n.render, n.offset)
}
