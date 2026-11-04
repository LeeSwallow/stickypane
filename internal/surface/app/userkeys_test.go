package app

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/core/i18n"
)

func TestNormalKey(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{
		"x": "x", "X": "X", "?": "?", "+": "+", "-": "-", "가": "가",
		"enter": "enter", "Enter": "enter", "pgdn": "pgdown", "escape": "esc", "f5": "f5",
		"ctrl+x": "ctrl+x", "alt+ctrl+x": "ctrl+alt+x", "Shift+Tab": "shift+tab",
		"shift+x": "X", "ctrl+X": "ctrl+shift+x", "ctrl++": "ctrl++", " u ": "u",
		"g g": "g g", " g   g ": "g g", "d shift+x": "d X", "ctrl+w J": "ctrl+w J", "esc x": "esc x", "0": "0",
		"ㅓ": "j", "ㅃ": "Q", "ctrl+ㅓ": "ctrl+j", "ㅎ ㅎ": "g g",
	} {
		if got, err := NormalKey(in); err != nil || got != want {
			t.Errorf("NormalKey(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for in, why := range map[string]string{
		"":            "empty",
		"1 g":         "starts with a digit",
		"g esc":       "after esc",
		"g foo":       "not a key",
		"foo":         "not a key",
		"cmd+x":       "not a modifier",
		"ctrl+c":      "always quits",
		"shift+1":     "the character shift makes",
		"ctrl+ctrl+x": "twice",
		"\x01":        "not a key",
	} {
		if got, err := NormalKey(in); err == nil || !strings.Contains(err.Error(), why) {
			t.Errorf("NormalKey(%q) = %q, %v; want an error saying %q", in, got, err, why)
		}
	}
}

// keysOf is the keys an action has under want.
func keysUnder(t *testing.T, want map[string][]string, name string) []string {
	t.Helper()
	km, _, _ := bindKeys(want)
	return km.keysOf(name)
}

func TestAKeyFileChangesTheActionsItNames(t *testing.T) {
	t.Parallel()
	km, changed, problems := bindKeys(map[string][]string{"note-archive": {"X", "ctrl+x"}, "jot": {}})
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	if got := km.keysOf("note-archive"); !slices.Equal(got, []string{"X", "ctrl+x"}) {
		t.Errorf("note-archive: %v", got)
	}
	if got := km.keysOf("jot"); len(got) != 0 {
		t.Errorf("jot is unbound: %v", got)
	}
	if _, ok := km.lookup("x"); ok {
		t.Error("x no longer archives")
	}
	if a, _ := km.lookup("X"); a.name != "note-archive" {
		t.Errorf("X runs %q", a.name)
	}
	if !changed["note-archive"] || !changed["jot"] || changed["undo"] {
		t.Errorf("changed = %v", changed)
	}
	if got := km.keysOf("undo"); !slices.Equal(got, []string{"u"}) {
		t.Errorf("undo keeps its key: %v", got)
	}
}

// What cannot be used is a problem, and its action keeps its keys; the
// rest of the file is used.
func TestWhatAKeyFileGetsWrongKeepsTheDefaults(t *testing.T) {
	t.Parallel()
	want := map[string][]string{
		"no-such-thing": {"Q"},
		"note-pin":      {"0 p"},
		"note-color":    {"u"}, // undo's
		"note-rename":   {"W"},
	}
	km, changed, problems := bindKeys(want)
	var got []string
	for _, p := range problems {
		got = append(got, p.String())
	}
	for _, w := range []string{"no-such-thing: there is no such action", `note-pin: "0 p" starts with a digit`, "note-color: u is undo's key too"} {
		if !slices.ContainsFunc(got, func(s string) bool { return strings.HasPrefix(s, w) }) {
			t.Errorf("no problem %q in %q", w, got)
		}
	}
	for name, keys := range map[string][]string{"note-pin": {"p"}, "note-color": {"c"}, "undo": {"u"}, "note-rename": {"W"}} {
		if got := km.keysOf(name); !slices.Equal(got, keys) {
			t.Errorf("%s: %v, want %v", name, got, keys)
		}
	}
	if changed["note-color"] || changed["note-pin"] || !changed["note-rename"] {
		t.Errorf("changed = %v", changed)
	}
}

// Keys traded between two actions are no clash, and a clash that a
// change undoes in turn is found too.
func TestKeysCanBeSwapped(t *testing.T) {
	t.Parallel()
	if got := keysUnder(t, map[string][]string{"note-archive": {"u"}, "undo": {"x"}}, "undo"); !slices.Equal(got, []string{"x"}) {
		t.Errorf("undo: %v", got)
	}
	// note-archive wants u, undo's; it keeps x, which jot wanted: both go back.
	km, _, problems := bindKeys(map[string][]string{"note-archive": {"u"}, "jot": {"x"}})
	if len(problems) != 2 || !slices.Equal(km.keysOf("jot"), []string{"N", "n"}) || !slices.Equal(km.keysOf("note-archive"), []string{"x"}) {
		t.Errorf("problems %v, jot %v, archive %v", problems, km.keysOf("jot"), km.keysOf("note-archive"))
	}
}

// The board with {"note-archive": ["X"]}: X archives, x does nothing, and
// the help and the message after it say X.
func TestTheBoardUsesTheKeyFile(t *testing.T) {
	t.Parallel()
	m, dir := newModel(t, map[string]string{"a.md": "one\n", "b.md": "two\n", "c.md": "three\n"})
	if said := m.UseKeys("keys.json", map[string][]string{"note-archive": {"X"}, "undo": {"U"}}, nil); said != "" {
		t.Fatalf("all of it is used, but: %q", said)
	}
	press(m, "x")
	if _, err := os.Stat(filepath.Join(dir, "a.md")); err != nil {
		t.Fatalf("x archived: %v", err)
	}
	press(m, "X")
	if got := readFile(t, dir, "archive/a.md"); got != "one\n" {
		t.Fatalf("X did not archive: %q", got)
	}
	if s := screen(m); !strings.Contains(s, "U brings it back") {
		t.Errorf("the message names undo's key:\n%s", s)
	}
	help := strings.Join(m.keys.helpLines(m.tr), "\n")
	if !strings.Contains(help, "  X                move to archive") || !strings.Contains(help, "  U                undo") {
		t.Errorf("the help lists X and U:\n%s", help)
	}
}

func TestTheBoardSaysWhatItLeftOut(t *testing.T) {
	t.Parallel()
	m, _ := newModel(t, map[string]string{"a.md": "one\n"})
	said := m.UseKeys("~/.stickypane/keys.json", map[string][]string{"nope": {"Q"}, "note-pin": {"2 p"}, "note-color": {"C"}}, nil)
	if said != "~/.stickypane/keys.json: nope, note-pin kept the keys they come with. stickypane keys says why." {
		t.Errorf("said %q", said)
	}
	if a, _ := m.keys.lookup("C"); a.name != "note-color" {
		t.Error("the rest of the file is used")
	}
	said = m.UseKeys("keys.json", nil, errors.New("not an object"))
	if !strings.HasPrefix(said, "keys.json could not be read") || m.keys != defaultKeys {
		t.Errorf("an unreadable file: %q", said)
	}
	if said := m.UseKeys("keys.json", nil, nil); said != "" || m.keys.key("note-archive") != "x" {
		t.Errorf("no file: %q", said)
	}
}

// Messages that name a key name the key the action has, and say nothing
// of one it does not have.
func TestMessagesNameTheKeysInEffect(t *testing.T) {
	t.Parallel()
	m, _ := newModel(t, map[string]string{})
	m.UseKeys("keys.json", map[string][]string{"jot": {"Y"}, "reload": {}}, nil)
	if s := screen(m); !strings.Contains(s, "Press Y to jot one down") || !strings.Contains(s, "Y jot") {
		t.Errorf("the empty board names Y:\n%s", s)
	}
	if got := m.Message("WatchUnavailable"); got != "File watching is unavailable." {
		t.Errorf("WatchUnavailable = %q", got)
	}
	m.UseKeys("keys.json", map[string][]string{"jot": {}}, nil)
	if s := screen(m); strings.Contains(s, "Press") || !strings.Contains(s, "Ask your agent") {
		t.Errorf("no key to jot with:\n%s", s)
	}
	ko, _ := i18n.Load("ko")
	m.tr = ko
	m.UseKeys("keys.json", map[string][]string{"undo": {"U"}}, nil)
	if got := m.keyed(ko.Archived, map[string]any{"Name": "a"}); !strings.Contains(got, "U 키로 되돌릴 수 있어요") {
		t.Errorf("Korean: %q", got)
	}
}

func TestBindingsListEveryAction(t *testing.T) {
	t.Parallel()
	bs, problems := Bindings(i18n.English(), map[string][]string{"note-archive": {"X"}})
	if len(problems) > 0 || len(bs) != len(actions()) {
		t.Fatalf("%d bindings, problems %v", len(bs), problems)
	}
	for _, b := range bs {
		if b.Name == "note-archive" && (!b.Changed || !slices.Equal(b.Keys, []string{"X"}) || b.Words != "move to archive") {
			t.Errorf("%+v", b)
		}
		if b.Name == "tab-3" && b.Words != "tab 3" {
			t.Errorf("%+v", b)
		}
	}
}
