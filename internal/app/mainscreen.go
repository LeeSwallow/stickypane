package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// ruleHeight is the shortest screen that spends a line on the rule under
// the title bar.
const ruleHeight = 8

func init() {
	handlers[modeBoard] = mainUpdate
	bodies[modeBoard] = mainBody
	footers[modeBoard] = mainFooter

	boardKeys["tab"] = func(m *Model) tea.Cmd { m.stepFocus(1); return nil }
	boardKeys["shift+tab"] = func(m *Model) tea.Cmd { m.stepFocus(-1); return nil }
	boardKeys["r"] = func(m *Model) tea.Cmd { m.reload(); return nil }
	boardKeys["q"] = func(*Model) tea.Cmd { return tea.Quit }

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
// lists; the screen gets the rest; and scroll keys nobody claimed move the
// screen.
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
	if offset, ok := widget.ScrollKey(m.scroll, key); ok {
		m.scroll = offset
	}
	return nil
}

func mainBody(m *Model, h int) []string {
	switch {
	case len(m.items) == 0:
		return dialog("", []string{"No notes yet.", "", "Press N to jot one down,", "or ask your agent to stick a note here."}, 46, m.width, h)
	case len(m.canvas) == 0:
		return dialog("", []string{"Nothing is open.", "", "Pick a note with tab and press enter."}, 46, m.width, h)
	}
	return widget.Window(m.canvas, m.scroll, h)
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
		return hintsThen(m.width, "?", "help", append(append([]string(nil), m.items[i].kind.Hint...), zoom, "zoom", "o", "close", "tab", "next")...)
	}
	return hintsThen(m.width, "?", "help", "tab", "next", "enter", "zoom", "o", "close", "+ -", "size", "N", "jot")
}

// titleBar lists every note like a row of tabs, wrapping onto more lines
// when they do not fit: "●" marks an open note and "○" a closed one, then the
// shape's icon and the title, "📌" for a pinned note and "*" for one that
// changed since it was last focused. The bar never takes more than a third
// of the screen: with more notes than that it shows the lines around the
// focused tab. On a screen tall enough, a rule sets it apart from the notes.
func (m *Model) titleBar() []string {
	var lines []string
	var line strings.Builder
	used, focusLine := 0, 0
	m.tabs = m.tabs[:0]
	for _, it := range m.items {
		open := m.isOpen(it)
		text := "○ "
		if open {
			text = "● "
		}
		if it.kind.Icon != "" {
			text += it.kind.Icon + " "
		}
		text += label(it)
		if it.note.Doc.Pinned() {
			text += " 📌"
		}
		if m.changed(it) {
			text += " *"
		}
		text = widget.Truncate(text, max(m.width-2, 1))
		w := widget.Width(text) + 2
		if used > 0 && used+w > m.width {
			lines = append(lines, line.String())
			line.Reset()
			used = 0
		}
		st := lipgloss.NewStyle().Foreground(noteColor(it))
		switch {
		case it.note.Name == m.focus:
			focusLine = len(lines)
			text = st.Bold(true).Reverse(true).Render(" " + text + " ")
		case open:
			text = " " + st.Render(text) + " "
		default:
			text = " " + widget.Faint.Render(text) + " "
		}
		line.WriteString(text)
		m.tabs = append(m.tabs, tab{name: it.note.Name, y: len(lines), x0: used, x1: used + w})
		used += w
	}
	if used > 0 {
		lines = append(lines, line.String())
	}
	if limit := max((m.height-1)/3, 1); len(lines) > limit {
		start := widget.ClampOffset(focusLine-limit/2, len(lines), limit)
		lines = lines[start : start+limit]
		for i := range m.tabs {
			m.tabs[i].y -= start // a tab scrolled out of the bar gets a line that is not there
		}
	}
	if len(lines) > 0 && m.height >= ruleHeight {
		lines = append(lines, widget.Faint.Render(strings.Repeat("─", m.width)))
	}
	return lines
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
	if it.note.Err != nil {
		m.status = "This note cannot be changed from here: " + it.note.Err.Error()
		return false
	}
	return true
}

// setOpen opens or closes a note by writing its "open" key.
func (m *Model) setOpen(it item, open bool) {
	if !m.editable(it) {
		return
	}
	value := "false"
	if open {
		value = "true"
	}
	m.apply(it.note.Name, doc.SetKey{Key: "open", Value: value})
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
