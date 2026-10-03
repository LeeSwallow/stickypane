package app

import (
	"strings"

	"github.com/LeeSwallow/stickypane/internal/layout"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// show returns the scroll offset that brings a span into a window of the
// given height, moving as little as possible from offset. A span taller than
// the window is shown from its start.
func show(at widget.Span, offset, height int) int {
	if at.End > offset+height {
		offset = at.End - height
	}
	if at.Start < offset {
		offset = at.Start
	}
	return offset
}

// bodyHeight is how many lines the open notes get on the main screen.
func (m *Model) bodyHeight() int {
	return max(m.height-1-min(len(m.bar), max(m.height-1, 0)), 0)
}

// pane is an open note in its place: a fixed rectangle on one of the
// screens, showing a window of the note's lines.
type pane struct {
	name   string // the note
	file   string // the file shown: the note's, or a page's. Scrolling is kept by file
	screen int
	rect   layout.Rect
	total  int // lines the note has
	offset int // the first line shown
}

// rows is how many lines of the note fit in the pane.
func (p pane) rows() int { return max(p.rect.H-2, 0) }

// relayout redraws the title bar and the open notes. The notes tile the
// screen: each has a fixed place and shows as much of itself as fits there,
// and the rest is reached by scrolling inside the note. Notes that do not
// fit on one screen go to the next, and the screen with the focused note is
// the one shown when the update asked to see it.
func (m *Model) relayout() {
	m.bar, m.panes, m.canvas, m.zoomLines = nil, nil, nil, nil
	if m.width <= 0 {
		return
	}
	if len(m.items) > 0 || len(m.tabList) > 1 {
		m.bar = []string{""} // one line, drawn once the screens are known
		if len(m.tabList) > 1 {
			m.bar = append(m.bar, "") // and one for the tabs above it
		}
	}
	h := m.bodyHeight()

	var open []item
	var widths []int
	for _, it := range m.items {
		if m.isOpen(it) {
			open = append(open, it)
			widths = append(widths, m.widthOf(m.sizeOf(it)))
		}
	}
	cells := layout.Rows(m.width, widths)

	// Draw every note at the width its place gives it. A row asks for the
	// height of its tallest note, or for the height a note fixes.
	lines := make([][]string, len(open))
	spans := make([]widget.Span, len(open))
	var need []int
	for i, it := range open {
		lines[i], spans[i] = m.draw(it, max(cells[i].W-4, 1), it.note.Name == m.focus)
		want := len(lines[i]) + 2
		if rows := m.rowsOf(it); rows > 0 {
			want = rows + 2
		}
		if cells[i].Row == len(need) {
			need = append(need, 0)
		}
		need[cells[i].Row] = max(need[cells[i].Row], want)
	}
	slots := layout.Stack(h, need, minPane)

	m.screens = 1
	for _, s := range slots {
		m.screens = max(m.screens, s.Screen+1)
	}
	for i, it := range open {
		if it.note.Name == m.focus && m.reveal != revealNothing {
			m.screen = slots[cells[i].Row].Screen
		}
	}
	m.screen = max(min(m.screen, m.screens-1), 0)

	var rects []layout.Rect
	var boxes []string
	for i, it := range open {
		slot := slots[cells[i].Row]
		p := pane{
			name: it.note.Name, file: it.note.Name, screen: slot.Screen, total: len(lines[i]),
			rect: layout.Rect{X: cells[i].X, Y: slot.Y, W: cells[i].W, H: slot.H},
		}
		// Where the user left it; a log that was never scrolled shows its end.
		offset, scrolled := m.offsets[p.file]
		if !scrolled && it.kind.Tail && len(it.pages) == 0 {
			offset = p.total
		}
		if b, ok := it.w.(*book); ok {
			// A book keeps its place by page: when the page the view is
			// on moved, because pages before it came or went, the view
			// moves with it.
			if a, ok := m.anchors[p.name]; ok && b.at < len(b.starts) && a.page == it.pages[b.at].note.Name && a.start != b.starts[b.at] {
				offset, scrolled = b.starts[b.at]+a.within, true
				m.offsets[p.file] = offset
			}
		}
		focused := p.name == m.focus
		if focused && m.reveal == revealCursor && spans[i].Ok() {
			offset = show(spans[i], offset, p.rows())
			m.offsets[p.file] = offset
		}
		p.offset = widget.ClampOffset(offset, p.total, p.rows())
		if scrolled && it.kind.Tail && len(it.pages) == 0 && p.offset >= p.total-p.rows() {
			delete(m.offsets, p.file) // back at the end: follow the log again
		}
		m.settle(i, p.offset, p.rows())
		m.panes = append(m.panes, p)
		if p.screen != m.screen || p.rect.H < 2 {
			continue
		}
		body := append([]string(nil), widget.Window(lines[i], p.offset, p.rows())...)
		for len(body) < p.rows() {
			body = append(body, "")
		}
		rects = append(rects, p.rect)
		boxes = append(boxes, frame(box{
			title: m.heading(it), icon: it.kind.Icon, summary: m.summary(it),
			body: strings.Join(body, "\n"), width: p.rect.W, color: m.color(it), focused: focused,
			offset: p.offset, total: p.total,
		}))
	}
	m.canvas = layout.Compose(rects, boxes)
	m.bar = m.titleBar()

	if m.zoomed() {
		m.layoutZoom()
	}
	m.reveal = revealNothing
}

// settle notes which page of a book the view is on after a scroll, so the
// next keys go to that page and the place survives a reload. i is the
// index into the open notes of the current layout.
func (m *Model) settle(i, offset, rows int) {
	for k := range m.items {
		it := &m.items[k]
		b, ok := it.w.(*book)
		if !ok || it.note.Name != m.openName(i) {
			continue
		}
		b.settle(offset, rows)
		it.page, it.kind = b.at, it.pages[b.at].kind
		if b.at < len(b.starts) {
			m.anchors[it.note.Name] = anchor{page: it.pages[b.at].note.Name, within: offset - b.starts[b.at], start: b.starts[b.at]}
		}
	}
}

// openName returns the name of the i-th open note.
func (m *Model) openName(i int) string {
	n := 0
	for _, it := range m.items {
		if m.isOpen(it) {
			if n == i {
				return it.note.Name
			}
			n++
		}
	}
	return ""
}

// paneOf returns the pane of an open note.
func (m *Model) paneOf(name string) (pane, bool) {
	for _, p := range m.panes {
		if p.name == name {
			return p, true
		}
	}
	return pane{}, false
}

// scrollPane moves the window of an open note to offset. Out of range
// values are fixed when the screen is drawn.
func (m *Model) scrollPane(p pane, offset int) {
	m.offsets[p.file] = max(offset, 0)
}

// flip shows the next or previous screen of notes and moves the focus to
// its first note.
func (m *Model) flip(delta int) {
	to := max(min(m.screen+delta, m.screens-1), 0)
	if to == m.screen {
		return
	}
	m.screen = to
	for _, p := range m.panes {
		if p.screen == to {
			m.focus = p.name
			m.markSeen(p.name)
			break
		}
	}
}

// drawn is a note as it was last drawn on the main screen.
type drawn struct {
	width  int
	active bool
	lines  []string
	at     widget.Span
}

// draw draws a note at width, or hands back the lines it drew last time
// when nothing about it changed: a key to another note redraws the screen
// but not every note on it. forget says when a note changed. A book is
// always drawn, since scrolling it changes the page it is on.
func (m *Model) draw(it item, width int, active bool) ([]string, widget.Span) {
	_, isBook := it.w.(*book)
	if d, ok := m.drawn[it.note.Name]; ok && !isBook && d.width == width && d.active == active {
		return d.lines, d.at
	}
	out, at := it.w.Draw(width, active)
	lines := strings.Split(out, "\n")
	if m.drawn == nil {
		m.drawn = map[string]drawn{}
	}
	m.drawn[it.note.Name] = drawn{width, active, lines, at}
	return lines, at
}

// forget drops what a note last drew, after its widget changed.
func (m *Model) forget(name string) { delete(m.drawn, name) }
