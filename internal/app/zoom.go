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
			if hint := m.items[i].kind.Hint; len(hint) > 0 {
				pairs = append(pairs, hint...)
			} else {
				pairs = append(pairs, "j k g G", "scroll")
			}
		}
		return hints(m.width, append(pairs, "e", "edit", "?", "keys")...)
	}
}

// zoom gives one note the whole screen. A log starts at its end.
func (m *Model) zoom(it item) {
	m.zoomName, m.mode, m.zoomScroll = it.note.Name, modeZoom, 0
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
		m.reveal = revealNote
	case key == "e" || key == "?":
		if f, ok := boardKeys[key]; ok {
			return f(m)
		}
	case m.items[i].kind.Handles(key):
		m.toWidget(i, key)
	default:
		if offset, ok := widget.ScrollKey(m.zoomScroll, key); ok {
			m.zoomScroll = offset
		}
	}
	return nil
}

// zoomBody draws the zoomed note in a frame that fills the screen. A note
// with a cursor scrolls to keep it in view; other notes scroll by key.
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
	out, cursor := it.w.Draw(max(m.width-4, 1), true)
	lines := strings.Split(out, "\n")
	if cursor >= 0 {
		if cursor >= m.zoomScroll+rows {
			m.zoomScroll = cursor - rows + 1
		}
		if cursor < m.zoomScroll {
			m.zoomScroll = cursor
		}
	}
	m.zoomScroll = widget.ClampOffset(m.zoomScroll, len(lines), rows)
	visible := append([]string(nil), widget.Window(lines, m.zoomScroll, rows)...)
	for len(visible) < rows {
		visible = append(visible, "")
	}
	return strings.Split(frame(box{
		title: title, icon: it.kind.Icon, summary: it.w.Summary(),
		body: strings.Join(visible, "\n"), width: m.width, color: noteColor(it), focused: true,
	}), "\n")
}
