package app

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/api"
	"github.com/LeeSwallow/stickypane/internal/prefs"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/theme"
)

// settingsWidth is the dialog that lists the settings.
const settingsWidth = 64

func init() {
	handlers[modeSettings] = settingsUpdate
	bodies[modeSettings] = settingsBody
	footers[modeSettings] = func(m *Model) string {
		return m.hints(m.width, "j k", m.tr.L("choose"), "h l", m.tr.L("change"), "esc", m.tr.L("close"))
	}
	boardKeys["S"] = func(m *Model) tea.Cmd {
		m.back, m.mode = m.mode, modeSettings
		return nil
	}
}

// applyPrefs makes the screen follow the settings: called on every reload,
// so a change from the panel, from stickypane config or from an edit of
// sticky.json applies at once.
func (m *Model) applyPrefs(set store.Settings) {
	if v := prefs.Get(set, "theme"); v != m.themeChoice {
		m.themeChoice = v
		m.useTheme(theme.Pick(v, m.dark))
	}
	if times, stale := prefs.Times(set), prefs.StaleAfter(set); times != m.env.Times || stale != m.env.StaleAfter {
		// Widgets keep the Env they were parsed with: they are made again.
		m.env.Times, m.env.StaleAfter = times, stale
		m.items = nil
	}
	m.editor = prefs.Get(set, "editor")
	m.drawn = nil // what notes drew may say times or flags differently now
}

func settingsUpdate(m *Model, msg tea.Msg) tea.Cmd {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}
	n := len(prefs.All)
	switch k.String() {
	case "j", "down", "tab":
		m.settingsAt = (m.settingsAt + 1) % n
	case "k", "up", "shift+tab":
		m.settingsAt = (m.settingsAt - 1 + n) % n
	case "l", "right", "enter", "space":
		m.changePref(1)
	case "h", "left":
		m.changePref(-1)
	case "esc", "q", "S":
		m.mode = m.back
	}
	return nil
}

// changePref moves the chosen setting to its next or previous value,
// writes it and applies it.
func (m *Model) changePref(step int) {
	p := prefs.All[m.settingsAt]
	next := p.Next(prefs.Get(m.store.Settings(), p.Key), step)
	if err := m.calls().ChangeSettings(api.Setting{Key: p.Key, Value: next}); err != nil {
		m.status = err.Error()
		return
	}
	m.reload()
}

// settingsBody lists the settings with their values in a dialog; a value at
// its default says so, and the chosen setting shows what it changes.
func settingsBody(m *Model, h int) []string {
	set := m.store.Settings()
	var lines []string
	for i, p := range prefs.All {
		v := prefs.Get(set, p.Key)
		value := v
		if v == p.Default {
			value += m.env.Styles.Faint.Render(" (" + m.tr.L("default") + ")")
		}
		label := fmt.Sprintf("%-20s", m.tr.L(p.Label))
		line := "  " + label + "‹ " + value + " ›"
		if i == m.settingsAt {
			line = m.env.Styles.Selected.Render("› " + label + "‹ " + v + " ›")
			if v == p.Default {
				line += m.env.Styles.Faint.Render(" (" + m.tr.L("default") + ")")
			}
		}
		lines = append(lines, line)
	}
	lines = append(lines, "", m.env.Styles.Faint.Render(m.tr.L(prefs.All[m.settingsAt].Help)))
	if m.width < settingsWidth || h < len(lines)+2 {
		return lines
	}
	return m.dialog(m.tr.L("Settings"), lines, settingsWidth, m.width, h, m.accent())
}
