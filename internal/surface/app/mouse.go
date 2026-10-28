package app

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/adapter/store"
	"github.com/LeeSwallow/stickypane/internal/core/widget"
)

const (
	wheelStep   = 3                      // lines one notch of the wheel scrolls
	doubleClick = 400 * time.Millisecond // two clicks this close are one double click
)

// hit is where a note's title is in the title bar.
type hit struct {
	name      string
	y, x0, x1 int
}

// lastClick remembers a click on a note that hit nothing, to tell a double
// click from two clicks.
type lastClick struct {
	name string
	at   time.Time
}

// dragging is a note's border held with the mouse: the right border sets
// the width, snapped to the nearest fraction, and the bottom border the
// height in lines. The screen follows while the button is held; sticky.json
// is written once, when it is let go.
type dragging struct {
	name  string
	right bool   // the right border; else the bottom one
	from  string // the size when the border was taken
	x     int    // where the note starts, and what its row's sizes add
	row   int    // up to, when the border was taken
	size  string // the width so far, "" before the pointer moved
	rows  int    // the height so far, 0 before the pointer moved
}

// grab starts dragging a border when the press is on one: the right
// border of a pane (its corners included), or its bottom border. Two
// quick presses on the bottom border give the note back its own height
// instead. It reports whether the press was on a border.
func (m *Model) grab(x, y int) bool {
	p, ok := m.paneAt(x, y)
	if !ok {
		return false
	}
	r, row := p.rect, y-min(len(m.bar), m.height-1)
	right, bottom := x == r.X+r.W-1, row == r.Y+r.H-1
	i := m.index(p.name)
	if !right && !bottom || i < 0 {
		return false
	}
	if m.focus != p.name {
		m.setFocus(p.name)
	}
	now := m.now()
	edge := "edge:" + p.name // not a click on the note itself
	if !right && m.lastClick.name == edge && now.Sub(m.lastClick.at) <= doubleClick {
		m.lastClick = lastClick{}
		m.setRows(m.items[i], 0)
		return true
	}
	m.lastClick = lastClick{name: edge, at: now}
	m.drag = dragging{name: p.name, right: right, from: m.sizeOf(m.items[i]), x: r.X, row: p.row}
	return true
}

// pull follows the pointer while a border is held.
func (m *Model) pull(e tea.Mouse) {
	p, ok := m.paneOf(m.drag.name)
	if !ok {
		return
	}
	if m.drag.right {
		// The pane is its row's share of the screen, stretched: the
		// pointer's width is turned back into the width the size asks for.
		asks := (e.X - m.drag.x + 1) * max(m.drag.row, 1) / max(m.width, 1)
		m.drag.size = widget.SnapSize(m.drag.from, asks, m.width)
		m.mark = mark{name: p.name, text: m.tr.L("width") + " " + m.drag.size}
		return
	}
	h := e.Y - min(len(m.bar), m.height-1) - p.rect.Y + 1
	m.drag.rows = max(min(h-2, m.bodyHeight()-2), 1)
	m.mark = mark{name: p.name, text: fmt.Sprintf(m.tr.L("%d lines"), m.drag.rows)}
}

// drop writes what a dragged border was let go at.
func (m *Model) drop() {
	d := m.drag
	m.drag = dragging{}
	i := m.index(d.name)
	switch {
	case i < 0:
	case d.size != "" && d.size != d.from:
		m.setView(d.name, func(v *store.View) { v.Size = d.size })
	case d.rows > 0:
		m.setRows(m.items[i], d.rows)
	}
}

// wheel scrolls what is on the screen.
func (m *Model) wheel(e tea.Mouse) {
	delta := wheelStep
	switch e.Button {
	case tea.MouseWheelUp:
		delta = -wheelStep
	case tea.MouseWheelDown:
	default:
		return
	}
	switch m.mode {
	case modeBoard:
		// The wheel scrolls the note under the pointer, focused or not.
		if p, ok := m.paneAt(e.X, e.Y); ok {
			m.scrollPane(p, p.offset+delta)
		}
	case modeZoom:
		m.zoomScroll += delta
	case modeHelp:
		m.helpScroll += delta
	case modeEdit:
		m.edit.Scroll(delta)
	}
}

// click handles a press of the left button on the main screen or on a
// zoomed note. Prompts, dialogs and the help screen do not take clicks.
func (m *Model) click(e tea.Mouse) {
	if e.Button != tea.MouseLeft || e.Y < 0 {
		return
	}
	// A question is answered with its buttons on the bottom line; a click
	// anywhere else answers no.
	if m.mode == modeConfirm {
		m.mode = m.back
		if e.Y == m.height-1 && e.X >= m.confirmYes[0] && e.X < m.confirmYes[1] {
			m.onConfirm()
		}
		return
	}
	if e.Y >= m.height-1 {
		return
	}
	switch m.mode {
	case modeBoard:
		m.status, m.mark = "", mark{}
		bar := min(len(m.bar), m.height-1)
		if e.Y < bar {
			m.clickTab(e.X, e.Y)
			return
		}
		if m.grab(e.X, e.Y) {
			return
		}
		if p, ok := m.paneAt(e.X, e.Y); ok {
			m.clickNote(p.name, e.Y-bar-p.rect.Y-1+p.offset, e.X-p.rect.X-2)
		}
	case modeZoom:
		m.status = ""
		m.clickNote(m.zoomName, e.Y-1+m.zoomScroll, e.X-2)
	}
}

// paneAt returns the pane shown at a cell of the screen.
func (m *Model) paneAt(x, y int) (pane, bool) {
	y -= min(len(m.bar), m.height-1)
	for _, p := range m.panes {
		r := p.rect
		if p.screen == m.screen && x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H {
			return p, true
		}
	}
	return pane{}, false
}

// clickTab handles a click on the title bar: on the tab strip it switches
// tabs; on a note's title, a closed note opens, an open note takes the
// focus, and the note that already has the focus closes.
func (m *Model) clickTab(x, y int) {
	for i, t := range m.tabHits {
		if t.y == y && x >= t.x0 && x < t.x1 {
			m.switchTab(i)
			return
		}
	}
	for _, t := range m.tabs {
		i := m.index(t.name)
		if t.y != y || x < t.x0 || x >= t.x1 || i < 0 {
			continue
		}
		it := m.items[i]
		switch {
		case !m.isOpen(it):
			m.setFocus(t.name)
			m.setOpen(it, true)
		case m.focus != t.name:
			m.setFocus(t.name)
		default:
			m.setOpen(it, false)
		}
		return
	}
}

// clickNote handles a click inside a note, at a line and cell of what its
// widget drew. The note takes the focus. A click on something the widget
// can work with does that; two quick clicks on anything else zoom in.
func (m *Model) clickNote(name string, line, col int) {
	i := m.index(name)
	if i < 0 {
		return
	}
	if m.focus != name {
		// The note is under the pointer already: do not scroll to it.
		m.focus = name
		m.markSeen(name)
	}
	if c, ok := m.items[i].w.(widget.Clicker); ok && line >= 0 && col >= 0 {
		if w, res, hit := c.Click(line, col); hit {
			m.items[i].w = w
			m.forget(name)
			m.lastClick = lastClick{}
			m.act(name, res)
			return
		}
	}
	now := m.now()
	if m.mode == modeBoard && m.lastClick.name == name && now.Sub(m.lastClick.at) <= doubleClick {
		m.lastClick = lastClick{}
		// A double click on a script asks to run it, so a script runs
		// again with the mouse alone; any other note zooms.
		if m.items[i].kind.Name == "script" {
			m.askRun(m.items[i].file().Name)
			return
		}
		m.zoom(m.items[i])
		return
	}
	m.lastClick = lastClick{name: name, at: now}
}
