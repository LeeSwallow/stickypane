package app

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/theme"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

func init() {
	handlers[modeBoard] = mainUpdate
	bodies[modeBoard] = mainBody
	footers[modeBoard] = mainFooter

	boardKeys["tab"] = func(m *Model) tea.Cmd { m.stepFocus(1); return nil }
	boardKeys["shift+tab"] = func(m *Model) tea.Cmd { m.stepFocus(-1); return nil }
	boardKeys["r"] = func(m *Model) tea.Cmd { m.reload(); return nil }
	boardKeys["q"] = func(*Model) tea.Cmd { return tea.Quit }
	boardKeys["T"] = func(m *Model) tea.Cmd { m.chooseTheme(theme.Next(m.theme.Get().Name)); return nil }
	boardKeys["]"] = func(m *Model) tea.Cmd { m.flip(1); return nil }
	boardKeys["["] = func(m *Model) tea.Cmd { m.flip(-1); return nil }

	// Tabs: a digit jumps to that tab, ( and ) go to the one before or after.
	for i := 1; i <= 9; i++ {
		n := i
		boardKeys[strconv.Itoa(n)] = func(m *Model) tea.Cmd { m.switchTab(n - 1); return nil }
	}
	boardKeys[")"] = func(m *Model) tea.Cmd { m.switchTab(m.tab + 1); return nil }
	boardKeys["("] = func(m *Model) tea.Cmd { m.switchTab(m.tab - 1); return nil }

	// . and , turn the pages of a book; > and < are the same keys shifted.
	turn := func(delta int) func(*Model) tea.Cmd {
		return func(m *Model) tea.Cmd {
			name := m.focus
			if m.zoomed() {
				name = m.zoomName
			}
			if i := m.index(name); i >= 0 {
				m.turn(i, delta)
			}
			return nil
		}
	}
	boardKeys["."], boardKeys[">"] = turn(1), turn(1)
	boardKeys[","], boardKeys["<"] = turn(-1), turn(-1)

	// enter goes one step deeper: it opens a closed note and zooms an open one.
	boardKeys["enter"] = onFocused(func(m *Model, it item) tea.Cmd {
		if m.isOpen(it) {
			m.zoom(it)
		} else {
			m.setOpen(it, true)
		}
		return nil
	})
	// z zooms too. It is the way in for a note that uses enter itself.
	boardKeys["z"] = onFocused(func(m *Model, it item) tea.Cmd {
		if m.isOpen(it) {
			m.zoom(it)
		}
		return nil
	})
	boardKeys["o"] = onFocused(func(m *Model, it item) tea.Cmd {
		m.setOpen(it, !m.isOpen(it))
		return nil
	})
}

// mainUpdate routes a key. A focused open note gets the keys its kind
// lists; the screen gets the rest; and scroll keys nobody claimed scroll
// the focused note inside its pane.
func mainUpdate(m *Model, msg tea.Msg) tea.Cmd {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}
	key := k.String()
	if i := m.index(m.focus); i >= 0 && m.isOpen(m.items[i]) && m.items[i].kind.Handles(key) {
		m.toWidget(i, key)
		return nil
	}
	if f, ok := boardKeys[key]; ok {
		return f(m)
	}
	// Scroll keys nobody claimed move the focused note inside its pane.
	if p, ok := m.paneOf(m.focus); ok {
		if offset, ok := widget.ScrollKey(p.offset, key, p.rows()); ok {
			m.scrollPane(p, offset)
		}
	}
	return nil
}

func mainBody(m *Model, h int) []string {
	switch {
	case len(m.items) == 0:
		return dialog("", append([]string{tr.NoNotes, ""}, strings.Split(tr.NoNotesHint, "\n")...), 46, m.width, h, m.accent())
	case len(m.canvas) == 0:
		return dialog("", append([]string{tr.NothingOpen, ""}, strings.Split(tr.NothingOpenHint, "\n")...), 46, m.width, h, m.accent())
	}
	return widget.Window(m.canvas, 0, h)
}

// mainFooter guides the keys that matter for what has the focus: the note's
// own keys first when it is open, then the screen's. The way to the full
// key list is always there, however narrow the pane.
func mainFooter(m *Model) string {
	i := m.index(m.focus)
	switch {
	case i < 0:
		return hintsThen(m.width, "?", "help", "N", "jot", "a", "add", "q", "quit")
	case !m.isOpen(m.items[i]):
		return hintsThen(m.width, "?", "help", "tab", "next", "enter", "open", "N", "jot", "a", "add", "q", "quit")
	case len(m.items[i].kind.Hint) > 0:
		zoom := "enter"
		if m.items[i].kind.Handles("enter") {
			zoom = "z"
		}
		return hintsThen(m.width, "?", "help", append(append(append([]string(nil), m.items[i].kind.Hint...), zoom, "zoom"), m.moving("o", "close", "tab", "next")...)...)
	}
	return hintsThen(m.width, "?", "help", append([]string{"tab", "next", "enter", "zoom"}, m.moving("j k", "scroll", "o", "close", "+ -", "size", "{ }", "move", "m", "to folder", "D", "delete", "N", "jot")...)...)
}

// titleBar is one line of tabs, like the tab strips of yazi and zellij:
// an open note is a tab with a tinted background in the note's color, the
// focused one a ribbon in the accent color, and a closed note plain muted
// text. A note that changed since it was last focused carries a dot. When
// the tabs do not fit, the ones around the focus are shown and the ends say
// how many more there are on each side. The right end says which screen of
// notes is shown when there is more than one.
func (m *Model) titleBar() []string {
	m.tabs = m.tabs[:0]
	var lines []string
	if strip := m.tabStrip(); strip != "" {
		lines = append(lines, strip)
	}
	if len(m.items) == 0 {
		return lines
	}
	th := m.theme.Get()
	type tab struct {
		text string
		w    int
		st   lipgloss.Style
	}
	tabs := make([]tab, len(m.items))
	focus, open := 0, 0
	for i, it := range m.items {
		text := strings.TrimSpace(it.kind.Icon + " " + m.label(it))
		if m.changed(it) {
			text += " •"
		}
		// A tab is never wider than a third of the screen, but on a
		// narrow screen a name still gets a readable length.
		text = " " + widget.Truncate(text, max(min(max(m.width/3, 24), m.width-4), 4)) + " "
		t := tab{text: text, w: widget.Width(text)}
		switch {
		case it.note.Name == m.focus:
			focus = i
			t.st = lipgloss.NewStyle().Background(lipgloss.Color(th.Accent)).Foreground(lipgloss.Color(th.Base)).Bold(true)
		case m.isOpen(it):
			open++
			t.st = lipgloss.NewStyle().Background(lipgloss.Color(th.Select)).Foreground(m.color(it))
		default:
			t.st = widget.Faint
		}
		tabs[i] = t
	}

	right := ""
	if m.screens > 1 {
		right = fmt.Sprintf(" %d/%d", m.screen+1, m.screens)
	}
	avail := m.width - widget.Width(right)

	// The window of tabs: the focused one, then neighbors on either side
	// while they fit, keeping room for the counts of what is left out.
	lo, hi := focus, focus+1
	used := tabs[focus].w
	marker := func(n int, left bool) string {
		if n == 0 {
			return ""
		}
		if left {
			return fmt.Sprintf("‹%d ", n)
		}
		return fmt.Sprintf(" %d›", n)
	}
	room := func() int {
		return avail - widget.Width(marker(lo, true)) - widget.Width(marker(len(tabs)-hi, false))
	}
	for {
		grew := false
		if hi < len(tabs) && used+1+tabs[hi].w <= room()-widget.Width(marker(len(tabs)-hi-1, false))+widget.Width(marker(len(tabs)-hi, false)) {
			used += 1 + tabs[hi].w
			hi++
			grew = true
		}
		if lo > 0 && used+1+tabs[lo-1].w <= room()-widget.Width(marker(lo-1, true))+widget.Width(marker(lo, true)) {
			used += 1 + tabs[lo-1].w
			lo--
			grew = true
		}
		if !grew {
			break
		}
	}

	var line strings.Builder
	x := 0
	if left := marker(lo, true); left != "" {
		line.WriteString(widget.Faint.Render(left))
		x += widget.Width(left)
	}
	for i := lo; i < hi; i++ {
		if i > lo {
			line.WriteString(" ")
			x++
		}
		m.tabs = append(m.tabs, hit{name: m.items[i].note.Name, y: len(lines), x0: x, x1: x + tabs[i].w})
		line.WriteString(tabs[i].st.Render(tabs[i].text))
		x += tabs[i].w
	}
	if rightMarker := marker(len(tabs)-hi, false); rightMarker != "" {
		line.WriteString(widget.Faint.Render(rightMarker))
		x += widget.Width(rightMarker)
	}
	if gap := m.width - x - widget.Width(right); gap > 0 {
		line.WriteString(strings.Repeat(" ", gap))
	}
	if right != "" {
		line.WriteString(widget.Faint.Render(right))
	}
	return append(lines, line.String())
}

// tabStrip is the first line of the screen when the board has more than
// one tab: every tab with its number, the active one in the accent color.
// Its hits are recorded for the mouse.
func (m *Model) tabStrip() string {
	m.tabHits = m.tabHits[:0]
	if len(m.tabList) < 2 {
		return ""
	}
	th := m.theme.Get()
	var line strings.Builder
	x := 0
	for i, t := range m.tabList {
		text := " " + strconv.Itoa(i+1) + " " + widget.Truncate(widget.Clean(t.Title), max(m.width/4, 8)) + " "
		if x+widget.Width(text) > m.width {
			break
		}
		st := widget.Faint
		if i == m.tab {
			st = lipgloss.NewStyle().Background(lipgloss.Color(th.Accent)).Foreground(lipgloss.Color(th.Base)).Bold(true)
		}
		m.tabHits = append(m.tabHits, hit{name: t.Name, y: 0, x0: x, x1: x + widget.Width(text)})
		line.WriteString(st.Render(text))
		x += widget.Width(text)
		if i < len(m.tabList)-1 {
			line.WriteString(" ")
			x++
		}
	}
	return line.String()
}

// stepFocus moves to the next or previous note in title bar order, wrapping.
func (m *Model) stepFocus(delta int) {
	if n := len(m.items); n > 0 {
		i := (m.index(m.focus) + delta + n) % n
		m.setFocus(m.items[i].note.Name)
	}
}

// editable reports whether the note's file can take a change, and says so on
// the bottom line when it cannot. A note that could not be read is shown to
// explain why; rewriting its file would do no good.
func (m *Model) editable(it item) bool {
	if err := it.file().Err; err != nil {
		m.status = say(tr.CannotChange, map[string]any{"Err": err.Error()})
		return false
	}
	return true
}

// setOpen opens or closes a note. Like everything about where a note is on
// the screen, that is written to sticky.json and not to the note.
func (m *Model) setOpen(it item, open bool) {
	m.setView(it.note.Name, func(v *store.View) { v.Open = &open })
	m.reveal = revealNote
}

// onFocused runs f for the focused note and does nothing when there is none.
func onFocused(f func(*Model, item) tea.Cmd) func(*Model) tea.Cmd {
	return func(m *Model) tea.Cmd {
		i := m.index(m.focus)
		if i < 0 {
			return nil
		}
		return f(m, m.items[i])
	}
}

// moving puts the keys that turn pages and screens in front of other hints
// when the focused note is a book and when the open notes take more than
// one screen.
func (m *Model) moving(pairs ...string) []string {
	if m.screens > 1 {
		pairs = append([]string{"[ ]", "screen"}, pairs...)
	}
	if i := m.index(m.focus); i >= 0 && len(m.items[i].pages) > 1 {
		pairs = append([]string{", .", "page"}, pairs...)
	}
	return pairs
}
