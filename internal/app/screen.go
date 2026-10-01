package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/widget"
)

// View implements tea.Model.
func (m *Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}

// render draws the screen: on the main screen the title bar, then the
// current body, then one bottom line. No line is wider than the terminal and
// there are never more lines than rows.
func (m *Model) render() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}
	var lines []string
	if m.bodyMode() == modeBoard {
		lines = append(lines, m.bar[:min(len(m.bar), m.height-1)]...)
	}
	h := m.height - 1 - len(lines)
	if body, ok := bodies[m.bodyMode()]; ok && h > 0 {
		lines = append(lines, widget.Window(body(m, h), 0, h)...)
	}
	for len(lines) < m.height-1 {
		lines = append(lines, "")
	}
	lines = append(lines, m.footer())
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
		return widget.Warn.Render(m.status)
	}
	if f, ok := footers[m.mode]; ok {
		return f(m)
	}
	return ""
}
