package app

import (
	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/layout"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

func init() {
	handlers[modeBoard] = boardUpdate
	bodies[modeBoard] = boardBody
	footers[modeBoard] = func(*Model) string { return "n jot  enter open  a add  ? help  q quit" }

	for key, dir := range map[string]layout.Dir{
		"up": layout.Up, "k": layout.Up,
		"down": layout.Down, "j": layout.Down,
		"left": layout.Left, "h": layout.Left,
		"right": layout.Right, "l": layout.Right,
	} {
		boardKeys[key] = func(m *Model) tea.Cmd { m.moveFocus(dir); return nil }
	}
	boardKeys["tab"] = func(m *Model) tea.Cmd { m.stepFocus(1); return nil }
	boardKeys["shift+tab"] = func(m *Model) tea.Cmd { m.stepFocus(-1); return nil }
	boardKeys["r"] = func(m *Model) tea.Cmd { m.reload(); return nil }
	boardKeys["q"] = func(*Model) tea.Cmd { return tea.Quit }
}

func boardUpdate(m *Model, msg tea.Msg) tea.Cmd {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}
	if f, ok := boardKeys[k.String()]; ok {
		return f(m)
	}
	return nil
}

func boardBody(m *Model, h int) []string {
	if len(m.items) == 0 {
		return []string{"", "  No notes yet. Press n to jot one down."}
	}
	return widget.Window(m.canvas, m.scroll, h)
}

// moveFocus moves to the nearest note in a direction on the screen.
func (m *Model) moveFocus(d layout.Dir) {
	i := m.index(m.focus)
	if j := layout.Neighbor(m.rects, i, d); j >= 0 && j < len(m.items) {
		m.setFocus(m.items[j].note.Name)
	}
}

// stepFocus moves to the next or previous note in board order, wrapping.
func (m *Model) stepFocus(delta int) {
	if n := len(m.items); n > 0 {
		i := (m.index(m.focus) + delta + n) % n
		m.setFocus(m.items[i].note.Name)
	}
}
