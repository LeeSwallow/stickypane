package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/api"
	"github.com/LeeSwallow/stickypane/internal/lineedit"
	"github.com/LeeSwallow/stickypane/internal/when"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// The index is every note of every tab in a line, as a wiki's index page
// lists its pages: for a board with more notes than the screen shows.
// Typing filters it, the arrows choose, enter goes to the note.

func init() {
	handlers[modeIndex] = indexUpdate
	bodies[modeIndex] = indexBody
	footers[modeIndex] = func(m *Model) string {
		return hints(m.width, "↑ ↓", tr.L("choose"), "enter", tr.L("go"), "esc", tr.L("close")) + "  " +
			widget.Bold.Render("/ ") + m.indexFilter.View()
	}
	open := func(m *Model) tea.Cmd {
		idx, err := api.New(m.store, m.reg).Index()
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
			lines = append(lines, widget.Bold.Render(widget.Truncate(widget.Clean(e.TabTitle), w)))
		}
		mark := "  "
		if e.Open {
			mark = widget.Accent.Render("● ")
		}
		stamp := when.Short(e.Modified, m.now())
		text := e.Icon + " " + widget.Clean(e.Title)
		if e.Summary != "" {
			text += widget.Faint.Render("  " + e.Summary)
		}
		if e.Gist != "" && e.Gist != e.Title {
			text += widget.Faint.Render("  — " + e.Gist)
		}
		room := max(w-4-widget.Width(stamp)-1, 1)
		line := mark + widget.Pad(text, room) + " " + widget.Faint.Render(stamp)
		if i == m.indexAt {
			at = len(lines)
			line = widget.Selected.Render("› " + widget.Pad(ansi.Strip(text), room) + " " + stamp)
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		lines = []string{widget.Faint.Render(tr.L("nothing matches"))}
	}
	// Keep the chosen line in view.
	offset := 0
	if at >= h {
		offset = at - h + 1
	}
	return widget.Window(lines, offset, h)
}
