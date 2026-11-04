package app

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/core/i18n"
	"github.com/LeeSwallow/stickypane/internal/core/theme"
	"github.com/LeeSwallow/stickypane/internal/core/widget"
)

// An action is one thing a key does on the board, under a name that stays
// when its keys change. The help and the bottom line name keys through it,
// so they say what the keys are, whatever they are.
type action struct {
	name    string   // kebab-case, stable: what a user's key file will name
	keys    []string // the keys that run it, the one hints show first
	english string   // what it does, in English; a %d is filled with n
	text    string   // the name of its words in i18n.Strings.Actions, when not name
	n       int
	zoom    bool // it works on a zoomed note too
	repeat  bool // a count before its key runs it that many times: 3J
	run     func(*Model) tea.Cmd
}

// describe is what the action does, in the screen's language.
func (a action) describe(tr i18n.Strings) string {
	key := a.text
	if key == "" {
		key = a.name
	}
	s := tr.Action(key, a.english)
	if a.n > 0 {
		s = fmt.Sprintf(s, a.n)
	}
	return s
}

// actions is the board's keys: every action, with its default keys. A
// focused open note takes the keys its kind lists first; these get the
// rest.
func actions() []action {
	acts := []action{
		{name: "focus-next", repeat: true, keys: []string{"tab"}, english: "next note", run: func(m *Model) tea.Cmd { m.stepFocus(1); return nil }},
		{name: "focus-prev", repeat: true, keys: []string{"shift+tab"}, english: "previous note", run: func(m *Model) tea.Cmd { m.stepFocus(-1); return nil }},
		{name: "note-open", keys: []string{"enter"}, english: "open, then zoom", run: onFocused(openOrZoom)},
		{name: "note-zoom", keys: []string{"z"}, english: "zoom", run: onFocused(zoomIn)},
		{name: "note-toggle", keys: []string{"o"}, english: "open or close", run: onFocused(toggleOpen)},
		{name: "note-wider", repeat: true, keys: []string{"+", "="}, english: "wider", run: onFocused(func(m *Model, it item) tea.Cmd { m.resize(it, 1); return nil })},
		{name: "note-narrower", repeat: true, keys: []string{"-"}, english: "narrower", run: onFocused(func(m *Model, it item) tea.Cmd { m.resize(it, -1); return nil })},
		{name: "note-taller", repeat: true, keys: []string{"shift+up"}, english: "a line taller", run: stretchBy(1)},
		{name: "note-shorter", repeat: true, keys: []string{"shift+down"}, english: "a line shorter", run: stretchBy(-1)},
		{name: "note-taller-5", repeat: true, keys: []string{"shift+pgup"}, english: "five lines taller", run: stretchBy(5)},
		{name: "note-shorter-5", repeat: true, keys: []string{"shift+pgdown"}, english: "five lines shorter", run: stretchBy(-5)},
		{name: "note-fit", keys: []string{"0"}, english: "content height", run: onFocused(func(m *Model, it item) tea.Cmd { m.setRows(it, 0); return nil })},
		// A note whose shape takes H J K L (a kanban moves its cards with
		// them) keeps them; { } still move it.
		{name: "note-move-earlier", repeat: true, keys: []string{"H", "{"}, english: "move earlier", run: func(m *Model) tea.Cmd { m.move(-1); return nil }},
		{name: "note-move-later", repeat: true, keys: []string{"L", "}"}, english: "move later", run: func(m *Model) tea.Cmd { m.move(1); return nil }},
		{name: "note-move-up", repeat: true, keys: []string{"K"}, english: "move a row up", run: func(m *Model) tea.Cmd { m.moveRow(-1); return nil }},
		{name: "note-move-down", repeat: true, keys: []string{"J"}, english: "move a row down", run: func(m *Model) tea.Cmd { m.moveRow(1); return nil }},
		// < and > are , and . shifted.
		{name: "page-prev", repeat: true, keys: []string{",", "<"}, english: "previous page", zoom: true, run: turnPage(-1)},
		{name: "page-next", repeat: true, keys: []string{".", ">"}, english: "next page", zoom: true, run: turnPage(1)},
		{name: "note-to-folder", keys: []string{"m"}, english: "move to a folder", run: onFocused(chooseFolder)},
		// n jots too unless the focused open note uses n itself (a board
		// adds a card, a checklist an item).
		{name: "jot", keys: []string{"N", "n"}, english: "jot a note", run: jot},
		{name: "add", keys: []string{"a"}, english: "add by shape", run: openCatalog},
		{name: "note-edit", keys: []string{"e"}, english: "edit here (like vi)", zoom: true, run: onFocused(editHere)},
		{name: "note-editor", keys: []string{"E"}, english: "edit in $EDITOR", zoom: true, run: onFocused(editOutside)},
		{name: "note-pin", keys: []string{"p"}, english: "pin", run: onFocused(togglePin)},
		{name: "note-color", keys: []string{"c"}, english: "change color", run: onFocused(nextColor)},
		{name: "note-rename", keys: []string{"R"}, english: "rename", run: onFocused(rename)},
		{name: "note-archive", keys: []string{"x"}, english: "move to archive", run: onFocused(archive)},
		{name: "note-delete", keys: []string{"D"}, english: "delete (to .trash)", run: onFocused(remove)},
		{name: "undo", keys: []string{"u"}, english: "undo archive, delete", run: undoMove},
		{name: "reload", keys: []string{"r"}, english: "reload", run: func(m *Model) tea.Cmd { m.reload(); return nil }},
		{name: "theme-next", keys: []string{"T"}, english: "next theme", zoom: true, run: func(m *Model) tea.Cmd { m.chooseTheme(theme.Next(m.theme.Get().Name)); return nil }},
		{name: "index", keys: []string{"i", "/"}, english: "index of every note", run: openIndex},
		{name: "links", keys: []string{"f"}, english: "linked notes", run: openLinks},
		{name: "settings", keys: []string{"S"}, english: "settings", run: func(m *Model) tea.Cmd { m.back, m.mode = m.mode, modeSettings; return nil }},
		{name: "screen-prev", repeat: true, keys: []string{"["}, english: "previous screen", run: func(m *Model) tea.Cmd { m.flip(-1); return nil }},
		{name: "screen-next", repeat: true, keys: []string{"]"}, english: "next screen", run: func(m *Model) tea.Cmd { m.flip(1); return nil }},
	}
	for n := 1; n <= 9; n++ {
		acts = append(acts, action{name: "tab-" + strconv.Itoa(n), keys: []string{strconv.Itoa(n)}, english: "tab %d", text: "tab-n", n: n,
			run: func(m *Model) tea.Cmd { m.switchTab(n - 1); return nil }})
	}
	return append(acts,
		action{name: "tab-prev", repeat: true, keys: []string{"("}, english: "previous tab", run: func(m *Model) tea.Cmd { m.switchTab(m.tab - 1); return nil }},
		action{name: "tab-next", repeat: true, keys: []string{")"}, english: "next tab", run: func(m *Model) tea.Cmd { m.switchTab(m.tab + 1); return nil }},
		// Also reachable from a zoomed note, which it returns to.
		action{name: "help", keys: []string{"?"}, english: "this help", zoom: true, run: openHelp},
		action{name: "quit", keys: []string{"q"}, english: "quit", run: func(*Model) tea.Cmd { return tea.Quit }},
	)
}

// A helpLine is a line of the help's first section: the actions it names,
// keys side by side, and what they do. A line with one action says what
// that action does. A line of keys that are not the board's actions (the
// scroll keys every note shares) gives them as they are.
type helpLine struct {
	actions []string
	text    string // the name of the words in i18n.Strings.Actions
	english string
	keys    string
}

// boardHelp is the order of the help's first section. Every action is on
// a line of it.
var boardHelp = []helpLine{
	{actions: []string{"focus-next"}},
	{actions: []string{"focus-prev"}},
	{actions: []string{"note-open"}},
	{actions: []string{"note-zoom"}},
	{actions: []string{"note-toggle"}},
	{actions: []string{"note-wider", "note-narrower"}, text: "help.size", english: "wider, narrower"},
	{actions: []string{"note-taller", "note-shorter"}, text: "help.height", english: "taller, shorter"},
	{actions: []string{"note-taller-5", "note-shorter-5"}, text: "help.height5", english: "five lines at a time"},
	{actions: []string{"note-fit"}},
	{actions: []string{"note-move-earlier", "note-move-later"}, text: "help.order", english: "move earlier, later"},
	{actions: []string{"note-move-up", "note-move-down"}, text: "help.rows", english: "move a row up, down"},
	{actions: []string{"page-prev", "page-next"}, text: "help.pages", english: "turn a folder's pages"},
	{actions: []string{"note-to-folder"}},
	{actions: []string{"jot"}},
	{actions: []string{"add"}},
	{actions: []string{"note-edit"}},
	{actions: []string{"note-editor"}},
	{actions: []string{"note-pin"}},
	{actions: []string{"note-color"}},
	{actions: []string{"note-rename"}},
	{actions: []string{"note-archive"}},
	{actions: []string{"note-delete"}},
	{actions: []string{"undo"}},
	{actions: []string{"reload"}},
	{actions: []string{"theme-next"}},
	{actions: []string{"index"}},
	{actions: []string{"links"}},
	{actions: []string{"settings"}},
	{keys: "j k g G", text: "help.scroll", english: "scroll in the note"},
	{keys: "pgdn pgup", text: "help.paging", english: "a page down, up"},
	{actions: []string{"screen-prev", "screen-next"}, text: "help.screens", english: "previous, next screen"},
	{actions: []string{"tab-1", "tab-2", "tab-3", "tab-4", "tab-5", "tab-6", "tab-7", "tab-8", "tab-9", "tab-prev", "tab-next"}, text: "help.tabs", english: "switch tab"},
	{actions: []string{"help"}},
	{actions: []string{"quit"}},
}

// helpKeyColumn is how wide the keys of a help line are, with their gap.
const helpKeyColumn = 17

// keymap is the actions with the keys bound to them, looked up by key. A
// key may be a sequence, its keys written with spaces between ("g g"); the
// board follows it key by key down a tree, one for the board and one for
// the actions that work on a zoomed note.
type keymap struct {
	actions []action
	byKey   map[string]int // the index in actions of the action a key runs
	byName  map[string]int
	board   *keyNode
	zoom    *keyNode
}

// A keyNode is where the keys pressed so far lead: the action they run,
// when they are a whole key, and the keys that go on from there.
type keyNode struct {
	act  int // the index in actions, -1 when these keys run nothing yet
	next map[string]*keyNode
}

func newNode() *keyNode { return &keyNode{act: -1, next: map[string]*keyNode{}} }

// add puts the sequence keys under n, running act.
func (n *keyNode) add(keys []string, act int) {
	for _, k := range keys {
		if n.next[k] == nil {
			n.next[k] = newNode()
		}
		n = n.next[k]
	}
	n.act = act
}

// newKeymap binds the keys of acts. A name used twice, or a key bound to
// two actions, is an error: one of them could never run. A key that
// begins a longer one ("g" and "g g") is not: the board waits a moment
// after it for the rest.
func newKeymap(acts []action) (*keymap, error) {
	km := &keymap{actions: acts, byKey: map[string]int{}, byName: map[string]int{}, board: newNode(), zoom: newNode()}
	for i, a := range acts {
		if _, dup := km.byName[a.name]; dup || a.name == "" {
			return nil, fmt.Errorf("action %q is named twice or not at all", a.name)
		}
		km.byName[a.name] = i
		for _, k := range a.keys {
			if j, dup := km.byKey[k]; dup {
				return nil, fmt.Errorf("key %q is bound to both %s and %s", k, acts[j].name, a.name)
			}
			km.byKey[k] = i
			km.board.add(strings.Fields(k), i)
			if a.zoom {
				km.zoom.add(strings.Fields(k), i)
			}
		}
	}
	return km, nil
}

// defaultKeys is the board's keys as they come.
var defaultKeys = func() *keymap {
	km, err := newKeymap(actions())
	if err != nil {
		panic(err)
	}
	return km
}()

// lookup is the action a key runs.
func (km *keymap) lookup(key string) (action, bool) {
	i, ok := km.byKey[key]
	if !ok {
		return action{}, false
	}
	return km.actions[i], true
}

// keysOf is the keys of the action called name; none when it has none or
// there is no such action.
func (km *keymap) keysOf(name string) []string {
	if i, ok := km.byName[name]; ok {
		return km.actions[i].keys
	}
	return nil
}

// key is the first key of the action called name, as the screen writes
// it (see showKey), or "".
func (km *keymap) key(name string) string {
	if k := km.keysOf(name); len(k) > 0 {
		return showKey(k[0])
	}
	return ""
}

// showKey writes a key as the screen shows it: a sequence of characters
// side by side ("g g" is gg), a named key in a sequence in angle brackets
// ("ctrl+w j" is <ctrl+w>j). A key of its own is as it is.
func showKey(key string) string {
	parts := strings.Fields(key)
	if len(parts) < 2 {
		return key
	}
	var sb strings.Builder
	for _, p := range parts {
		if utf8.RuneCountInString(p) == 1 {
			sb.WriteString(p)
		} else {
			sb.WriteString("<" + p + ">")
		}
	}
	return sb.String()
}

// hint is a pair for the bottom line: the first key of each action, side
// by side, and label. It is nothing when none of them has a key.
func (km *keymap) hint(label string, names ...string) []string {
	var keys []string
	for _, n := range names {
		if k := km.key(n); k != "" {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return nil
	}
	return []string{strings.Join(keys, " "), label}
}

// joinHints joins pairs from hint, leaving out the empty ones.
func joinHints(pairs ...[]string) []string {
	var out []string
	for _, p := range pairs {
		out = append(out, p...)
	}
	return out
}

// helpLines is the help's first section, the board's keys, in the
// screen's language: a heading, then a line for each line of boardHelp
// that has a key.
func (km *keymap) helpLines(tr i18n.Strings) []string {
	lines := []string{tr.Action("help.notes", "Notes")}
	for _, l := range boardHelp {
		keys, text := l.keys, tr.Action(l.text, l.english)
		if len(l.actions) > 0 {
			keys = km.helpKeys(l.actions)
			if l.text == "" {
				text = km.actions[km.byName[l.actions[0]]].describe(tr)
			}
		}
		if keys == "" {
			continue
		}
		lines = append(lines, "  "+keys+strings.Repeat(" ", max(helpKeyColumn-widget.Width(keys), 1))+text)
	}
	return lines
}

// helpKeys writes the keys of a line of actions. One action's keys are
// written in a row: "i /". Several actions are written side by side, a
// group for each of their keys: "H L  { }" is earlier and later, then
// their other keys. A group stops where an action has no more keys.
func (km *keymap) helpKeys(names []string) string {
	var lists [][]string
	for _, n := range names {
		if k := km.keysOf(n); len(k) > 0 {
			lists = append(lists, k)
		}
	}
	switch len(lists) {
	case 0:
		return ""
	case 1:
		return shortKeys(lists[0])
	}
	groups := len(lists[0])
	for _, l := range lists {
		groups = min(groups, len(l))
	}
	parts := make([]string, groups)
	for g := range parts {
		col := make([]string, len(lists))
		for i, l := range lists {
			col[i] = l[g]
		}
		parts[g] = shortKeys(col)
	}
	return strings.Join(parts, "  ")
}

// shortKeys writes keys as the help shows them: pgdown as pgdn, a
// modifier once ("shift+up down"), and a run of three digits or more as a
// range ("1-9").
func shortKeys(keys []string) string {
	out := make([]string, 0, len(keys))
	prefix := ""
	if i := strings.LastIndex(keys[0], "+"); i > 0 && !strings.Contains(keys[0], " ") {
		prefix = keys[0][:i+1]
	}
	for i := 0; i < len(keys); i++ {
		if strings.Contains(keys[i], " ") {
			out = append(out, showKey(keys[i]))
			continue
		}
		if j := digitRun(keys[i:]); j >= 3 {
			out = append(out, keys[i]+"-"+keys[i+j-1])
			i += j - 1
			continue
		}
		k := strings.ReplaceAll(keys[i], "pgdown", "pgdn")
		if i > 0 && prefix != "" && strings.HasPrefix(k, prefix) {
			k = strings.TrimPrefix(k, prefix)
		}
		out = append(out, k)
	}
	return strings.Join(out, " ")
}

// digitRun is how many keys from the first are digits counting up by one.
func digitRun(keys []string) int {
	n := 0
	for n < len(keys) && len(keys[n]) == 1 && keys[n][0] >= '0' && keys[n][0] <= '9' &&
		(n == 0 || keys[n][0] == keys[n-1][0]+1) {
		n++
	}
	return n
}
