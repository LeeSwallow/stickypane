package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// View implements tea.Model.
func (m *Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}

// render draws the screen: the current body above one bottom line. No line
// is wider than the terminal and there are never more lines than rows.
func (m *Model) render() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}
	h := m.height - 1
	var lines []string
	if body, ok := bodies[m.bodyMode()]; ok {
		lines = append(lines, body(m, h)...)
	}
	for len(lines) < h {
		lines = append(lines, "")
	}
	lines = append(lines[:h], m.footer())
	for i, l := range lines {
		lines[i] = ansi.Truncate(l, m.width, "")
	}
	return strings.Join(lines, "\n")
}

// bodyMode is the mode whose body is drawn. Prompts and confirmations have
// no body of their own and keep showing the screen they were opened from.
func (m *Model) bodyMode() mode {
	if _, ok := bodies[m.mode]; ok {
		return m.mode
	}
	return m.back
}

func (m *Model) footer() string {
	asking := m.mode == modeInput || m.mode == modeConfirm
	if m.status != "" && !asking {
		return m.status
	}
	if f, ok := footers[m.mode]; ok {
		return f(m)
	}
	return ""
}
