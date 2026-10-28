package app

import (
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/core/widget"
	"github.com/LeeSwallow/stickypane/internal/surface/lineedit"
	"github.com/LeeSwallow/stickypane/internal/usecase/api"
)

// The index is every note of every tab in a line, as a wiki's index page
// lists its pages: for a board with more notes than the screen shows.
// Typing filters it, the arrows choose, enter goes to the note.

func init() {
	handlers[modeIndex] = indexUpdate
	bodies[modeIndex] = indexBody
	footers[modeIndex] = func(m *Model) string {
		return m.hints(m.width, "↑ ↓", m.tr.L("choose"), "enter", m.tr.L("go"), "esc", m.tr.L("close")) + "  " +
			m.env.Styles.Bold.Render("/ ") + m.indexFilter.View()
	}
	open := func(m *Model) tea.Cmd {
		idx, err := api.NewWithEnv(m.store, m.reg, m.env).Index()
		if err != nil {
			m.status = err.Error()
			return nil
		}
		m.indexAll, m.indexAt = idx, 0
		m.indexFilter = lineedit.New("", max(m.width/3, 10))
		m.back, m.mode = m.mode, modeIndex
		return nil
	}
	boardKeys["i"] = open
	boardKeys["/"] = open
	// f: the notes the focused one links to and the notes that link to it.
	boardKeys["f"] = func(m *Model) tea.Cmd {
		if open(m); m.mode != modeIndex {
			return nil
		}
		var self api.Entry
		for _, e := range m.indexAll {
			if e.Name == m.focus {
				self = e
			}
		}
		var linked []api.Entry
		for _, e := range m.indexAll {
			if slices.Contains(self.Links, e.Name) || slices.Contains(self.Backlinks, e.Name) {
				linked = append(linked, e)
			}
		}
		if len(linked) == 0 {
			m.mode = m.back
			m.status = m.tr.L("no links to or from this note")
			return nil
		}
		m.indexAll = linked
		return nil
	}
}

// indexShown is the index as the filter leaves it: every word of the filter
// in the note's title, file name, gist or tab, in any case.
func (m *Model) indexShown() []api.Entry {
	words := strings.Fields(strings.ToLower(m.indexFilter.Value()))
	if len(words) == 0 {
		return m.indexAll
	}
	var out []api.Entry
	for _, e := range m.indexAll {
		hay := strings.ToLower(e.Title + " " + e.Name + " " + e.Gist + " " + e.TabTitle + " " + e.Kind)
		match := true
		for _, w := range words {
			if !strings.Contains(hay, w) {
				match = false
				break
			}
		}
		if match {
			out = append(out, e)
		}
	}
	return out
}

func indexUpdate(m *Model, msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		n := len(m.indexShown())
		switch msg.String() {
		case "esc":
			m.mode = m.back
		case "down", "ctrl+n", "tab":
			m.indexAt = min(m.indexAt+1, max(n-1, 0))
		case "up", "ctrl+p", "shift+tab":
			m.indexAt = max(m.indexAt-1, 0)
		case "enter":
			if shown := m.indexShown(); m.indexAt < len(shown) {
				m.goTo(shown[m.indexAt])
			}
		default:
			if !m.indexFilter.Key(msg.String()) && msg.Text != "" {
				m.indexFilter.Type(msg.Text)
			}
			m.indexAt = 0
		}
	case tea.PasteMsg:
		m.indexFilter.Type(msg.Content)
		m.indexAt = 0
	}
	return nil
}

// goTo shows a note from the index: its tab, open, with the focus.
func (m *Model) goTo(e api.Entry) {
	m.mode = modeBoard
	if e.Tab != m.tabName {
		for i, t := range m.tabList {
			if t.Name == e.Tab {
				m.switchTab(i)
			}
		}
	}
	if i := m.index(e.Name); i >= 0 {
		if !m.isOpen(m.items[i]) {
			m.setOpen(m.items[i], true)
		}
		m.setFocus(e.Name)
		m.reveal = revealNote
	}
}

// indexBody draws the index: a heading per tab, a line per note with its
// shape, title, count, gist and when it changed; open notes are marked.
func indexBody(m *Model, h int) []string {
	shown := m.indexShown()
	w := m.width
	var lines []string
	at := -1
	tab := "\x00"
	for i, e := range shown {
		if e.Tab != tab {
			tab = e.Tab
			if len(lines) > 0 {
				lines = append(lines, "")
			}
			lines = append(lines, m.env.Styles.Bold.Render(widget.Truncate(widget.Clean(e.TabTitle), w)))
		}
		mark := "  "
		if e.Open {
			mark = m.env.Styles.Accent.Render("● ")
		}
		stamp := m.env.Short(e.Modified)
		text := e.Icon + " " + widget.Clean(e.Title)
		if e.Summary != "" {
			text += m.env.Styles.Faint.Render("  " + e.Summary)
		}
		if n := len(e.Backlinks); n > 0 {
			text += m.env.Styles.Faint.Render(fmt.Sprintf("  ← %d", n))
		}
		if e.Gist != "" && e.Gist != e.Title {
			text += m.env.Styles.Faint.Render("  — " + e.Gist)
		}
		room := max(w-4-widget.Width(stamp)-1, 1)
		line := mark + widget.Pad(text, room) + " " + m.env.Styles.Faint.Render(stamp)
		if i == m.indexAt {
			at = len(lines)
			line = m.env.Styles.Selected.Render("› " + widget.Pad(ansi.Strip(text), room) + " " + stamp)
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		lines = []string{m.env.Styles.Faint.Render(m.tr.L("nothing matches"))}
	}
	// Keep the chosen line in view.
	offset := 0
	if at >= h {
		offset = at - h + 1
	}
	return widget.Window(lines, offset, h)
}
