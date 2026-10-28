package app

import (
	"image/color"

	"github.com/LeeSwallow/stickypane/internal/arrange"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// How a note is arranged comes from three places, in this order: what the
// user did on the board, kept in sticky.json; the note's own front matter,
// which is how an agent proposes an arrangement; and the note's kind.

// isOpen reports whether a note is drawn on the main screen. A note that
// neither sticky.json nor its front matter opens is closed, with two
// exceptions that last for this run and are written nowhere: a note that
// appeared while stickypane was running, so that what an agent just wrote
// is seen at once, and a note that was open when a rewrite dropped its key,
// so that an agent rewriting a note does not make it vanish from the screen.
// A note that cannot be read is always open, to say why.
func (m *Model) isOpen(it item) bool {
	if it.note.Err != nil {
		return true
	}
	if open, said := arrange.Open(m.views[it.note.Name], it.note.Doc); said {
		return open
	}
	return m.peek[it.note.Name]
}

// sizeOf returns the note's size.
func (m *Model) sizeOf(it item) string {
	if m.drag.name == it.note.Name && m.drag.size != "" {
		return m.drag.size
	}
	return arrange.Size(m.views[it.note.Name], it.note.Doc, it.kind, it.file().Doc)
}

// rowsOf returns the height in lines the note asks for, or 0 when it asks
// for as much as its content takes.
func (m *Model) rowsOf(it item) int {
	if m.drag.name == it.note.Name && m.drag.rows > 0 {
		return m.drag.rows
	}
	return arrange.Rows(m.views[it.note.Name], it.note.Doc, it.kind)
}

// pinned reports whether the note is kept first.
func (m *Model) pinned(it item) bool {
	return arrange.Pinned(m.views[it.note.Name], it.note.Doc)
}

// color returns the note's color: the one chosen for it, or one derived
// from its name so that it keeps its color between runs.
func (m *Model) color(it item) color.Color {
	return m.noteColor(arrange.ColorIndex(it.note.Name, arrange.Color(m.views[it.note.Name], it.note.Doc)))
}

// setView changes how a note is arranged and shows the result.
func (m *Model) setView(name string, change func(*store.View)) {
	if err := m.store.SetView(name, change); err != nil {
		m.status = say(m.tr.ArrangementNotSaved, map[string]any{"Err": err.Error()})
	}
	m.reload()
}

// widthOf turns a size, a fraction of the screen, into cells.
func (m *Model) widthOf(size string) int {
	return widget.SizeCells(size, m.width)
}

// nameOf names a file: its title, or its file name without the folder and
// the extension.
func nameOf(n store.Note) string {
	return widget.Clean(arrange.Title(store.View{}, n.Doc, n.Name))
}

// label names a note in the title bar and in prompts: the name sticky.json
// gives it, else its title, else its name without the extension. A book
// goes by its folder's name unless it was given one.
func (m *Model) label(it item) string {
	return widget.Clean(arrange.Title(m.views[it.note.Name], it.note.Doc, it.note.Name))
}

// heading is the text in a note's top border. A plain note without a title
// has none, like a sticky note; other shapes fall back to the file name. A
// book shows its name and the page it is on.
func (m *Model) heading(it item) string {
	if len(it.pages) > 0 {
		return m.label(it) + " · " + nameOf(it.file())
	}
	t, _ := it.note.Doc.Get("title")
	if named := m.views[it.note.Name].Title; named != "" || (t == "" && (it.kind.Name != m.reg[0].Name || it.note.Err != nil)) {
		return m.label(it)
	}
	return widget.Clean(t)
}

// summary is the text at the right end of a note's top border: what its
// widget counts and, for a book, which page it is on. A script that is
// running says so instead.
func (m *Model) summary(it item) string {
	if m.mark.name == it.note.Name && m.mark.text != "" {
		return m.mark.text
	}
	if m.running[it.file().Name] {
		return m.tr.Running
	}
	s := it.w.Summary()
	// A log says when it last grew, so one that stopped is seen to.
	if it.kind.Tail && !it.file().ModTime.IsZero() {
		if s != "" {
			s += " · "
		}
		s += m.env.Short(it.file().ModTime)
	}
	return s
}
