package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

func init() {
	handlers[modeBoard] = mainUpdate
	bodies[modeBoard] = mainBody
	footers[modeBoard] = func(*Model) string {
		return "tab move  enter open/zoom  o close  + - size  N jot  a add  ? help"
	}

	boardKeys["tab"] = func(m *Model) tea.Cmd { m.stepFocus(1); return nil }
	boardKeys["shift+tab"] = func(m *Model) tea.Cmd { m.stepFocus(-1); return nil }
	boardKeys["r"] = func(m *Model) tea.Cmd { m.reload(); return nil }
	boardKeys["q"] = func(*Model) tea.Cmd { return tea.Quit }

	// enter goes one step deeper: it opens a closed note and zooms an open one.
	boardKeys["enter"] = onFocused(func(m *Model, it item) tea.Cmd {
		if m.isOpen(it) {
			m.zoom(it)
		} else {
			m.setOpen(it, true)
		}
		return nil
	})
	boardKeys["o"] = onFocused(func(m *Model, it item) tea.Cmd {
		m.setOpen(it, !m.isOpen(it))
		return nil
	})
}

// mainUpdate routes a key. A focused open note gets the keys its kind
// lists; the screen gets the rest; and scroll keys nobody claimed move the
// screen.
func mainUpdate(m *Model, msg tea.Msg) tea.Cmd {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}
	key := k.String()
	if i := m.index(m.focus); i >= 0 && m.isOpen(m.items[i]) && m.items[i].kind.Handles(key) {
		m.toWidget(i, key)
		return nil
	}
	if f, ok := boardKeys[key]; ok {
		return f(m)
	}
	if offset, ok := widget.ScrollKey(m.scroll, key); ok {
		m.scroll = offset
	}
	return nil
}

func mainBody(m *Model, h int) []string {
	switch {
	case len(m.items) == 0:
		return []string{"", "  No notes yet. Press N to jot one down."}
	case len(m.canvas) == 0:
		return []string{"", "  Nothing is open. Pick a note with tab and press enter."}
	}
	return widget.Window(m.canvas, m.scroll, h)
}

// titleBar lists every note, wrapping onto more lines when they do not fit:
// "▾" marks an open note and "▸" a closed one, "📌" a pinned note and "●"
// one that changed since it was last focused.
func (m *Model) titleBar() []string {
	var lines []string
	var line strings.Builder
	used := 0
	for _, it := range m.items {
		text := "▸ " + label(it)
		if m.isOpen(it) {
			text = "▾ " + label(it)
		}
		if it.note.Doc.Pinned() {
			text += " 📌"
		}
		if m.changed(it) {
			text += " ●"
		}
		text = widget.Truncate(text, max(m.width-2, 1))
		w := widget.Width(text) + 2
		if used > 0 && used+w > m.width {
			lines = append(lines, line.String())
			line.Reset()
			used = 0
		}
		switch {
		case it.note.Name == m.focus:
			text = widget.Selected.Render(text)
		case !m.isOpen(it):
			text = widget.Faint.Render(text)
		}
		line.WriteString(" " + text + " ")
		used += w
	}
	if used > 0 {
		lines = append(lines, line.String())
	}
	return lines
}

// stepFocus moves to the next or previous note in title bar order, wrapping.
func (m *Model) stepFocus(delta int) {
	if n := len(m.items); n > 0 {
		i := (m.index(m.focus) + delta + n) % n
		m.setFocus(m.items[i].note.Name)
	}
}

// setOpen opens or closes a note by writing its "open" key.
func (m *Model) setOpen(it item, open bool) {
	value := "false"
	if open {
		value = "true"
	}
	m.apply(it.note.Name, doc.SetKey{Key: "open", Value: value})
	m.reveal = revealNote
}

// onFocused runs f for the focused note and does nothing when there is none.
func onFocused(f func(*Model, item) tea.Cmd) func(*Model) tea.Cmd {
	return func(m *Model) tea.Cmd {
		i := m.index(m.focus)
		if i < 0 {
			return nil
		}
		return f(m, m.items[i])
	}
}
