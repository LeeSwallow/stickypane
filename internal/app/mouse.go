package app

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/widget"
)

const (
	wheelStep   = 3                      // lines one notch of the wheel scrolls
	doubleClick = 400 * time.Millisecond // two clicks this close are one double click
)

// tab is where a note's title is in the title bar.
type tab struct {
	name      string
	y, x0, x1 int
}

// lastClick remembers a click on a note that hit nothing, to tell a double
// click from two clicks.
type lastClick struct {
	name string
	at   time.Time
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
			m.scrollPane(p.name, p.offset+delta)
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
	if e.Button != tea.MouseLeft || e.Y < 0 || e.Y >= m.height-1 {
		return
	}
	switch m.mode {
	case modeBoard:
		m.status = ""
		bar := min(len(m.bar), m.height-1)
		if e.Y < bar {
			m.clickTab(e.X, e.Y)
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

// clickTab handles a click on the title bar. A closed note opens; an open
// note takes the focus; the note that already has the focus closes.
func (m *Model) clickTab(x, y int) {
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
			m.lastClick = lastClick{}
			m.act(name, res)
			return
		}
	}
	now := m.now()
	if m.mode == modeBoard && m.lastClick.name == name && now.Sub(m.lastClick.at) <= doubleClick {
		m.lastClick = lastClick{}
		m.zoom(m.items[i])
		return
	}
	m.lastClick = lastClick{name: name, at: now}
}
