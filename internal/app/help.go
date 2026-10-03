package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/widget"
)

// helpWidth is the dialog that holds helpText: its widest line plus the frame.
const helpWidth = 44

func init() {
	handlers[modeHelp] = helpUpdate
	bodies[modeHelp] = helpBody
	footers[modeHelp] = func(m *Model) string {
		return hints(m.width, "j k", "scroll") + "  " + widget.Faint.Render(tr.L("any other key closes"))
	}

	// Also reachable from a zoomed note, which it returns to.
	boardKeys["?"] = func(m *Model) tea.Cmd {
		m.back, m.mode, m.helpScroll = m.mode, modeHelp, 0
		return nil
	}
}

func helpUpdate(m *Model, msg tea.Msg) tea.Cmd {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}
	if offset, scrolled := widget.ScrollKey(m.helpScroll, k.String(), max(m.height-4, 1)); scrolled {
		m.helpScroll = offset
		return nil
	}
	m.mode = m.back
	return nil
}

// helpBody draws the key reference: section names stand out and the keys
// line up. On a screen with room it sits in a centered dialog.
func helpBody(m *Model, h int) []string {
	lines := strings.Split(tr.Help, "\n")
	for i, l := range lines {
		if l != "" && !strings.HasPrefix(l, " ") {
			lines[i] = widget.Bold.Render(l)
		}
	}
	rows := h
	framed := m.width >= helpWidth && h >= 6
	if framed {
		rows = min(h-2, len(lines))
	}
	m.helpScroll = widget.ClampOffset(m.helpScroll, len(lines), rows)
	visible := append([]string(nil), widget.Window(lines, m.helpScroll, rows)...)
	if !framed {
		return visible
	}
	return dialog(tr.Keys, visible, helpWidth, m.width, h, m.accent())
}
