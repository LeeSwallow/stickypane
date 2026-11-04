package app

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/core/i18n"
	"github.com/LeeSwallow/stickypane/internal/core/widget"
)

// Every action is named once, in kebab-case, says what it does in every
// language the screen speaks, and has a key.
func TestEveryActionIsNamedDescribedAndBound(t *testing.T) {
	t.Parallel()
	kebab := regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	if _, err := newKeymap(actions()); err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, a := range actions() {
		if !kebab.MatchString(a.name) {
			t.Errorf("action %q: a name is kebab-case", a.name)
		}
		if names[a.name] {
			t.Errorf("action %q is named twice", a.name)
		}
		names[a.name] = true
		if a.english == "" || a.run == nil || len(a.keys) == 0 {
			t.Errorf("action %q: it needs words, a handler and a default key", a.name)
		}
	}
	for _, code := range i18n.Languages()[1:] {
		tr, err := i18n.Load(code)
		if err != nil {
			t.Fatal(err)
		}
		has := func(key string) {
			if tr.Actions[key] == "" {
				t.Errorf("%s: no words for %q", code, key)
			}
		}
		has("help.notes")
		for _, a := range actions() {
			if a.text != "" {
				has(a.text)
			} else {
				has(a.name)
			}
		}
		for _, l := range boardHelp {
			if l.text != "" {
				has(l.text)
			}
		}
	}
}

func TestAKeyRunsOneActionAndANameIsOne(t *testing.T) {
	t.Parallel()
	run := func(*Model) tea.Cmd { return nil }
	if _, err := newKeymap([]action{{name: "a", keys: []string{"x"}, run: run}, {name: "b", keys: []string{"y", "x"}, run: run}}); err == nil {
		t.Error("x bound to two actions should be an error")
	}
	if _, err := newKeymap([]action{{name: "a", keys: []string{"x"}, run: run}, {name: "a", keys: []string{"y"}, run: run}}); err == nil {
		t.Error("a name used twice should be an error")
	}
}

// The help's first section names every action, and only actions there are.
func TestTheHelpNamesEveryAction(t *testing.T) {
	t.Parallel()
	listed := map[string]bool{}
	for _, l := range boardHelp {
		if len(l.actions) == 0 && (l.keys == "" || l.text == "") {
			t.Errorf("a line without actions needs its keys and words: %+v", l)
		}
		if len(l.actions) > 1 && l.text == "" {
			t.Errorf("a line of several actions needs words of its own: %v", l.actions)
		}
		for _, n := range l.actions {
			if _, ok := defaultKeys.byName[n]; !ok {
				t.Errorf("the help names %q, which is no action", n)
			}
			listed[n] = true
		}
	}
	for _, a := range actions() {
		if !listed[a.name] {
			t.Errorf("the help does not list %q", a.name)
		}
	}
}

// The English help of the board's keys, as the table makes it.
const englishBoardHelp = `Notes
  h j k l          the note on that side
  gg G             first, last note
  tab              next note
  shift+tab        previous note
  enter            open, then zoom
  z                zoom
  o                open or close
  > <  + -         wider, narrower
  shift+up down    taller, shorter
  shift+pgup pgdn  five lines at a time
  0                content height
  H L              move earlier, later
  K J              move a row up, down
  , .              turn a folder's pages
  m                move to a folder
  N n              jot a note
  a                add by shape
  e                edit here (like vi)
  E                edit in $EDITOR
  yy               copy its path
  p                pin
  c                change color
  R                rename
  x                move to archive
  dd               delete (to .trash)
  u                undo archive, delete
  r                reload
  T                next theme
  i /              index of every note
  f                linked notes
  S                settings
  ctrl+e y  ↓ ↑    a line down, up
  ctrl+d u         half a page down, up
  pgdn pgup        a page down, up
  [ ]              previous, next screen
  1-9 gt gT        switch tab
  ?                this help
  :                command line
  q                quit`

func TestTheHelpListsTheBoardKeys(t *testing.T) {
	t.Parallel()
	if got := strings.Join(defaultKeys.helpLines(i18n.English()), "\n"); got != englishBoardHelp {
		t.Errorf("the board's keys in the help:\n%s\nwant:\n%s", got, englishBoardHelp)
	}
	for _, code := range i18n.Languages() {
		tr, err := i18n.Load(code)
		if err != nil {
			t.Fatal(err)
		}
		lines := defaultKeys.helpLines(tr)
		for _, line := range lines {
			if w := widget.Width(line); w > 40 {
				t.Errorf("%s: help line %q is %d cells wide, want at most 40", code, line, w)
			}
		}
		for _, k := range []string{"h j k l", "gg G", "> <", "shift+up down", "shift+pgup pgdn", "0", "yy", "dd", "ctrl+d u", "1-9 gt gT"} {
			if !slices.ContainsFunc(lines, func(l string) bool { return strings.HasPrefix(strings.TrimSpace(l), k+"  ") }) {
				t.Errorf("%s: the help should list %q", code, k)
			}
		}
	}
}

func TestShortKeys(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		keys []string
		want string
	}{
		{[]string{"shift+up", "shift+down"}, "shift+up down"},
		{[]string{"shift+pgup", "shift+pgdown"}, "shift+pgup pgdn"},
		{[]string{"1", "2", "3", "4", "(", ")"}, "1-4 ( )"},
		{[]string{"1", "2", "(", ")"}, "1 2 ( )"},
		{[]string{"i", "/"}, "i /"},
		{[]string{"ctrl+e", "ctrl+y", "down", "up"}, "ctrl+e y ↓ ↑"},
		{[]string{"ctrl+a", "b"}, "ctrl+a b"},
	} {
		if got := shortKeys(c.keys); got != c.want {
			t.Errorf("shortKeys(%q) = %q, want %q", c.keys, got, c.want)
		}
	}
}

// rebound is the default keys with some actions given other keys.
func rebound(t *testing.T, keys map[string][]string) *keymap {
	t.Helper()
	acts := actions()
	for i, a := range acts {
		if k, ok := keys[a.name]; ok {
			acts[i].keys = k
		}
	}
	km, err := newKeymap(acts)
	if err != nil {
		t.Fatal(err)
	}
	return km
}

// The help, the bottom line and the keys themselves all come from the
// table: another key there is another key everywhere, and an action
// without one drops out.
func TestTheHelpAndTheBottomLineFollowTheKeys(t *testing.T) {
	t.Parallel()
	m, dir := newModel(t, map[string]string{"a.md": opened("one\n")})
	m.keys = rebound(t, map[string][]string{"note-archive": {"X"}, "help": {"F"}, "note-delete": {}, "note-toggle": {"O"}})
	m.Update(tea.WindowSizeMsg{Width: 160, Height: 24})

	help := strings.Join(m.keys.helpLines(m.tr), "\n")
	if !strings.Contains(help, "\n  X                move to archive") || strings.Contains(help, "\n  x ") {
		t.Errorf("the help should say X archives:\n%s", help)
	}
	if strings.Contains(help, "delete (to .trash)") {
		t.Errorf("an action without a key is not in the help:\n%s", help)
	}
	foot := lastLine(screen(m))
	if !strings.Contains(foot, "F help") || !strings.Contains(foot, "O close") || strings.Contains(foot, "delete") {
		t.Errorf("the bottom line should follow the keys: %q", foot)
	}

	press(m, "x")
	if got := readFile(t, dir, "a.md"); got != opened("one\n") {
		t.Fatal("x no longer archives")
	}
	press(m, "X")
	if len(m.items) != 0 {
		t.Errorf("X should archive the note, notes = %d", len(m.items))
	}
}

func lastLine(s string) string {
	lines := strings.Split(s, "\n")
	return lines[len(lines)-1]
}
