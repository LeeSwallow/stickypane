package app

import (
	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/widget"
)

func init() {
	handlers[modeCatalog] = catalogUpdate
	bodies[modeCatalog] = catalogBody
	footers[modeCatalog] = func(*Model) string { return "enter choose  esc cancel" }

	// Jot: one line becomes a plain note. No shape, no title.
	boardKeys["n"] = func(m *Model) tea.Cmd {
		m.ask("Jot", "", func(text string) { m.create(text, []byte(text+"\n")) })
		return nil
	}
	boardKeys["a"] = func(m *Model) tea.Cmd {
		m.catalogIdx, m.mode = 0, modeCatalog
		return nil
	}
}

// create writes a new note named after text and focuses it.
func (m *Model) create(text string, content []byte) {
	name, err := m.store.Create(text, content, m.now())
	if err != nil {
		m.status = "Cannot create the note: " + err.Error()
		return
	}
	m.reload()
	m.setFocus(name)
}

func catalogUpdate(m *Model, msg tea.Msg) tea.Cmd {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}
	switch k.String() {
	case "esc", "q":
		m.mode = modeBoard
	case "j", "down":
		m.catalogIdx = min(m.catalogIdx+1, len(m.reg)-1)
	case "k", "up":
		m.catalogIdx = max(m.catalogIdx-1, 0)
	case "enter":
		kind := m.reg[m.catalogIdx]
		m.mode = modeBoard
		m.ask("Title", "", func(title string) { m.create(title, kind.Template(title)) })
	}
	return nil
}

func catalogBody(m *Model, _ int) []string {
	lines := []string{"", "  Add a note", ""}
	for i, k := range m.reg {
		if i == m.catalogIdx {
			lines = append(lines, "  "+widget.Selected.Render("› "+k.Label))
		} else {
			lines = append(lines, "    "+k.Label)
		}
	}
	return lines
}
