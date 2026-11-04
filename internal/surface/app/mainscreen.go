package app

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/LeeSwallow/stickypane/internal/adapter/store"
	"github.com/LeeSwallow/stickypane/internal/core/widget"
)

func init() {
	handlers[modeBoard] = mainUpdate
	bodies[modeBoard] = mainBody
	footers[modeBoard] = mainFooter
}

// openOrZoom goes one step deeper: it opens a closed note and zooms an
// open one.
func openOrZoom(m *Model, it item) tea.Cmd {
	if m.isOpen(it) {
		m.zoom(it)
	} else {
		m.setOpen(it, true)
	}
	return nil
}

// zoomIn zooms an open note. It is the way in for a note that uses enter
// itself.
func zoomIn(m *Model, it item) tea.Cmd {
	if m.isOpen(it) {
		m.zoom(it)
	}
	return nil
}

func toggleOpen(m *Model, it item) tea.Cmd {
	m.setOpen(it, !m.isOpen(it))
	return nil
}

// turnPage turns the pages of a book, the focused one or the zoomed one.
func turnPage(delta int) func(*Model) tea.Cmd {
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

// mainUpdate routes a key. A focused open note gets the keys its kind
// lists, except the ones the board takes first (h j k l and the others
// that move between notes or move them: see action.first); the screen gets
// the rest. While a sequence waits for its next key ("g" of "g g"), that
// key goes to it first; a count alone does not take a key from the note,
// and is dropped when the note takes it. A key typed with the Korean
// layout on is read as the Latin key it is on.
func mainUpdate(m *Model, msg tea.Msg) tea.Cmd {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}
	key := latinKey(k.String())
	if i := m.index(m.focus); len(m.seq.keys) == 0 && i >= 0 && m.isOpen(m.items[i]) && m.items[i].kind.Handles(key) && !m.keys.first[key] {
		m.seq.reset()
		m.toWidget(i, key)
		return nil
	}
	_, cmd := m.feedKey(key)
	return cmd
}

func mainBody(m *Model, h int) []string {
	switch {
	case len(m.items) == 0:
		return m.dialog("", append([]string{m.tr.NoNotes, ""}, strings.Split(m.keyed(m.tr.NoNotesHint, nil), "\n")...), 46, m.width, h, m.accent())
	case len(m.canvas) == 0:
		return m.dialog("", append([]string{m.tr.NothingOpen, ""}, strings.Split(m.keyed(m.tr.NothingOpenHint, nil), "\n")...), 46, m.width, h, m.accent())
	}
	return widget.Window(m.canvas, 0, h)
}

// mainFooter guides the keys that matter for what has the focus: the note's
// own keys first when it is open, then the screen's. The way to the full
// key list is always there, however narrow the pane.
func mainFooter(m *Model) string {
	k := m.keys
	jotAddQuit := joinHints(k.hint("jot", "jot"), k.hint("add", "add"), k.hint("quit", "quit"))
	i := m.index(m.focus)
	switch {
	case i < 0:
		return m.withHelp("help", jotAddQuit...)
	case !m.isOpen(m.items[i]):
		return m.withHelp("help", joinHints(k.hint("go", "focus-left", "focus-down", "focus-up", "focus-right"), k.hint("open", "note-open"), jotAddQuit)...)
	case len(m.items[i].kind.Hint) > 0:
		// A note that uses enter itself is zoomed with z. Zoomed, it
		// takes the keys the board takes first.
		zoom := k.hint("zoom", "note-open")
		if len(zoom) == 0 || m.items[i].kind.Handles(zoom[0]) {
			zoom = k.hint("zoom", "note-zoom")
		}
		own := append(k.ownHints(m.items[i].kind), zoom...)
		return m.withHelp("help", append(own, m.moving(joinHints(k.hint("go", "focus-left", "focus-down", "focus-up", "focus-right"), k.hint("close", "note-toggle"))...)...)...)
	}
	return m.withHelp("help", joinHints(k.hint("go", "focus-left", "focus-down", "focus-up", "focus-right"), k.hint("zoom", "note-open"),
		m.moving(joinHints(k.hint("scroll", "scroll-half-down", "scroll-half-up"), k.hint("close", "note-toggle"), k.hint("size", "note-wider", "note-narrower"),
			k.hint("move", "note-move-earlier", "note-move-down", "note-move-up", "note-move-later"),
			k.hint("to folder", "note-to-folder"), k.hint("copy", "note-copy"), k.hint("delete", "note-delete"), k.hint("jot", "jot"))...))...)
}

// withHelp draws pairs on the bottom line and ends with the key to the
// help under label, which a narrow pane never drops.
func (m *Model) withHelp(label string, pairs ...string) string {
	key := m.keys.key("help")
	if key == "" {
		return m.hints(m.width, pairs...)
	}
	return m.hintsThen(m.width, key, label, pairs...)
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
			t.st = m.env.Styles.Faint
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
		line.WriteString(m.env.Styles.Faint.Render(left))
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
		line.WriteString(m.env.Styles.Faint.Render(rightMarker))
		x += widget.Width(rightMarker)
	}
	if gap := m.width - x - widget.Width(right); gap > 0 {
		line.WriteString(strings.Repeat(" ", gap))
	}
	if right != "" {
		line.WriteString(m.env.Styles.Faint.Render(right))
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
		st := m.env.Styles.Faint
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

// focusEnd moves to the first note in title bar order, or the last.
func (m *Model) focusEnd(last bool) {
	if len(m.items) == 0 {
		return
	}
	i := 0
	if last {
		i = len(m.items) - 1
	}
	m.setFocus(m.items[i].note.Name)
}

// focusToward moves to the open note on one side of the focused one, as
// the screen shows them: dx -1 is left and 1 right, dy -1 up and 1 down.
// Left and right go to a note beside it, one that shares some of its
// lines. Up and down go to the nearest row that way, to the note there
// that shares most of its columns, or else the nearest; the screens of
// notes stand one above the other, so down from the last row goes on to
// the next screen. A closed note has no side: from it the focus moves in
// the title bar's order.
func (m *Model) focusToward(dx, dy int) {
	from, ok := m.paneOf(m.focus)
	if !ok {
		m.stepFocus(dx + dy)
		return
	}
	// Where a pane is, top to bottom across the screens.
	top := func(p pane) int { return p.screen*(m.bodyHeight()+1) + p.rect.Y }
	// overlap is how much of a and b's spans [a0,a1) and [b0,b1) they share,
	// less than 0 by how far apart they are.
	overlap := func(a0, a1, b0, b1 int) int { return min(a1, b1) - max(a0, b0) }
	best, bestKey := "", [3]int{}
	for _, p := range m.panes {
		if p.name == from.name || p.rect.H < 2 {
			continue
		}
		var gap, shared int
		switch {
		case dx > 0:
			gap = p.rect.X - (from.rect.X + from.rect.W)
		case dx < 0:
			gap = from.rect.X - (p.rect.X + p.rect.W)
		case dy > 0:
			// The notes of a row start on the same line: the next row
			// is the nearest top below this one.
			gap = top(p) - top(from) - 1
		default:
			gap = top(from) - top(p) - 1
		}
		if dx != 0 {
			shared = overlap(from.rect.Y, from.rect.Y+from.rect.H, p.rect.Y, p.rect.Y+p.rect.H)
			if p.screen != from.screen || shared <= 0 {
				continue
			}
		} else {
			shared = overlap(from.rect.X, from.rect.X+from.rect.W, p.rect.X, p.rect.X+p.rect.W)
		}
		if gap < 0 {
			continue
		}
		// Nearest first, then the most shared, then the one further left
		// or up.
		key := [3]int{gap, -shared, p.rect.X + top(p)}
		if best == "" || key[0] < bestKey[0] || key[0] == bestKey[0] && (key[1] < bestKey[1] || key[1] == bestKey[1] && key[2] < bestKey[2]) {
			best, bestKey = p.name, key
		}
	}
	if best != "" {
		m.setFocus(best)
	}
}

// editable reports whether the note's file can take a change, and says so on
// the bottom line when it cannot. A note that could not be read is shown to
// explain why; rewriting its file would do no good.
func (m *Model) editable(it item) bool {
	if err := it.file().Err; err != nil {
		m.status = say(m.tr.CannotChange, map[string]any{"Err": err.Error()})
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
		pairs = append(m.keys.hint("screen", "screen-prev", "screen-next"), pairs...)
	}
	if i := m.index(m.focus); i >= 0 && len(m.items[i].pages) > 1 {
		pairs = append(m.keys.hint("page", "page-prev", "page-next"), pairs...)
	}
	return pairs
}
