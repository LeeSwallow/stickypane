package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/widget"
)

func init() {
	handlers[modeZoom] = zoomUpdate
	bodies[modeZoom] = zoomBody
	footers[modeZoom] = func(m *Model) string {
		pairs := []string{"esc", "back"}
		if i := m.index(m.zoomName); i >= 0 {
			pairs = append(pairs, m.items[i].kind.Hint...)
		}
		return hintsThen(m.width, "?", "keys", append(pairs, "g G", "top, end", "e", "edit")...)
	}
}

// zoom gives one note the whole screen. It starts where the selection is; a
// log starts at its end.
func (m *Model) zoom(it item) {
	m.zoomName, m.mode, m.zoomScroll = it.note.Name, modeZoom, 0
	m.reveal = revealCursor
	if it.kind.Tail {
		m.zoomScroll = 1 << 30
	}
}

func zoomUpdate(m *Model, msg tea.Msg) tea.Cmd {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}
	i := m.index(m.zoomName)
	if i < 0 {
		m.mode = modeBoard
		return nil
	}
	switch key := k.String(); {
	case key == "esc" || key == "q":
		m.mode = modeBoard
		m.reveal = revealCursor
	case key == "e" || key == "E" || key == "?" || key == "T" || strings.ContainsAny(key, ",.<>") && len(key) == 1:
		if f, ok := boardKeys[key]; ok {
			return f(m)
		}
	case m.items[i].kind.Handles(key):
		m.toWidget(i, key)
	default:
		// Keys the note does not use scroll freely, so every line of a
		// zoomed note can be reached even when it has a cursor.
		if offset, ok := widget.ScrollKey(m.zoomScroll, key, m.zoomRows()); ok {
			m.zoomScroll = offset
		}
	}
	return nil
}

// zoomRows is how many lines of the note fit inside the zoomed frame.
func (m *Model) zoomRows() int { return max(m.height-1-2, 1) }

// layoutZoom draws the zoomed note. It scrolls to the selection only when
// the last update asked for it, which a key handled by the note does; any
// other scrolling is left alone.
func (m *Model) layoutZoom() {
	i := m.index(m.zoomName)
	if i < 0 {
		return
	}
	out, at := m.items[i].w.Draw(max(m.width-4, 1), true)
	m.zoomLines = strings.Split(out, "\n")
	rows := m.zoomRows()
	if m.reveal == revealCursor && at.Ok() {
		m.zoomScroll = show(at, m.zoomScroll, rows)
	}
	m.zoomScroll = widget.ClampOffset(m.zoomScroll, len(m.zoomLines), rows)
	if b, ok := m.items[i].w.(*book); ok {
		b.settle(m.zoomScroll, rows)
		m.items[i].page, m.items[i].kind = b.at, m.items[i].pages[b.at].kind
	}
}

// zoomBody shows the zoomed note in a frame that fills the screen.
func zoomBody(m *Model, h int) []string {
	i := m.index(m.zoomName)
	if i < 0 {
		return nil
	}
	it := m.items[i]
	title := m.heading(it)
	if title == "" {
		title = it.note.Name
	}
	rows := max(h-2, 1)
	visible := append([]string(nil), widget.Window(m.zoomLines, m.zoomScroll, rows)...)
	for len(visible) < rows {
		visible = append(visible, "")
	}
	return strings.Split(frame(box{
		title: title, icon: it.kind.Icon, summary: m.summary(it),
		body: strings.Join(visible, "\n"), width: m.width, color: m.color(it), focused: true,
		offset: m.zoomScroll, total: len(m.zoomLines),
	}), "\n")
}
