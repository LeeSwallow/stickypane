package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

func init() {
	handlers[modeModal] = modalUpdate
	bodies[modeModal] = modalBody
	footers[modeModal] = func(*Model) string { return "esc close  e edit  ? keys" }

	boardKeys["enter"] = func(m *Model) tea.Cmd {
		if m.index(m.focus) >= 0 {
			m.modalName, m.mode = m.focus, modeModal
		}
		return nil
	}
}

func modalUpdate(m *Model, msg tea.Msg) tea.Cmd {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}
	i := m.index(m.modalName)
	if i < 0 {
		m.mode = modeBoard
		return nil
	}
	switch key := k.String(); key {
	case "esc", "q":
		m.mode = modeBoard
	case "e", "?":
		if f, ok := boardKeys[key]; ok {
			return f(m)
		}
	default:
		name := m.modalName
		w, res := m.items[i].w.Update(key)
		m.items[i].w = w
		if res.Op != nil {
			m.apply(name, res.Op)
		}
		if p := res.Prompt; p != nil {
			m.ask(p.Label, "", func(text string) { m.apply(name, p.Submit(text)) })
		}
	}
	return nil
}

// modalBody draws the open note in a focused frame that fills the screen.
func modalBody(m *Model, h int) []string {
	i := m.index(m.modalName)
	if i < 0 {
		return nil
	}
	it := m.items[i]
	title := m.heading(it)
	if title == "" {
		title = it.note.Name
	}
	rows := max(h-2, 1)
	lines := strings.Split(it.w.View(max(m.width-4, 1), rows), "\n")
	for len(lines) < rows {
		lines = append(lines, "")
	}
	return strings.Split(frame(title, strings.Join(lines[:rows], "\n"), m.width, noteColor(it), true), "\n")
}
