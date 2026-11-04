package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/adapter/store"
	"github.com/LeeSwallow/stickypane/internal/core/theme"
	"github.com/LeeSwallow/stickypane/internal/usecase/api"
	"github.com/LeeSwallow/stickypane/internal/usecase/prefs"
)

// settingsWidth is the dialog that lists the settings.
const settingsWidth = 64

// settingsLabel is the cells a setting's name takes, so the values line up
// in every language.
const settingsLabel = 16

func init() {
	handlers[modeSettings] = settingsUpdate
	bodies[modeSettings] = settingsBody
	footers[modeSettings] = func(m *Model) string {
		return m.hints(m.width, "j k", m.tr.L("choose"), "h l", m.tr.L("change"), "esc", m.tr.L("close"))
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
	m.columns = prefs.Columns(set)
	m.drawn = nil // what notes drew may say times or flags differently now
}

func settingsUpdate(m *Model, msg tea.Msg) tea.Cmd {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}
	n := len(prefs.Panel())
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
	p := prefs.Panel()[m.settingsAt]
	next := p.Next(prefs.Get(m.store.Settings(), p.Key), step)
	if err := m.calls().ChangeSettings(api.Setting{Key: p.Key, Value: next}); err != nil {
		m.status = err.Error()
		return
	}
	m.reload()
}

// settingsBody lists the panel's settings with their values in a dialog; a
// value at its default says so, and the chosen setting shows what it
// changes.
func settingsBody(m *Model, h int) []string {
	set := m.store.Settings()
	panel := prefs.Panel()
	var lines []string
	for i, p := range panel {
		v := prefs.Get(set, p.Key)
		label := m.tr.L(p.Label)
		label += strings.Repeat(" ", max(0, settingsLabel-ansi.StringWidth(label)))
		line := "  " + label + "‹ " + v + " ›"
		if i == m.settingsAt {
			line = m.env.Styles.Selected.Render("› " + label + "‹ " + v + " ›")
		}
		if v == p.Default {
			line += m.env.Styles.Faint.Render(" (" + m.tr.L("default") + ")")
		}
		lines = append(lines, line)
	}
	lines = append(lines, "", m.env.Styles.Faint.Render(m.tr.L(panel[m.settingsAt].Help)),
		m.env.Styles.Faint.Render(m.tr.L("more: stickypane config")))
	if m.width < settingsWidth || h < len(lines)+2 {
		return lines
	}
	return m.dialog(m.tr.L("Settings"), lines, settingsWidth, m.width, h, m.accent())
}
