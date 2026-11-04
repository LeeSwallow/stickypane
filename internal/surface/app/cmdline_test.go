package app

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/core/i18n"
	"github.com/LeeSwallow/stickypane/internal/core/widget"
	"github.com/LeeSwallow/stickypane/internal/usecase/prefs"
)

// runLine types : and a line, and enter.
func runLine(m *Model, line string) {
	press(m, ":")
	typeText(m, line)
	press(m, "enter")
}

// A command is its name, an alias of it, or a start of its name no other
// command shares; a start that several share names them.
func TestACommandIsNamedByAnyStartOfIt(t *testing.T) {
	t.Parallel()
	m, _ := newModel(t, map[string]string{"a.md": opened("one\n")})
	for word, want := range map[string]string{
		"q": "quit", "quit": "quit", "wq": "quit", "w": "write", "col": "columns", "colu": "columns", "c": "columns",
		"lang": "language", "l": "language", "th": "theme", "ta": "tab", "e": "edit", "ed": "edit", "u": "restore", "undo": "restore",
		"SET": "set", "h": "help",
	} {
		c, bad := m.findCommand(word)
		if bad != "" || c.name != want {
			t.Errorf("%s = %q (%s), want %s", word, c.name, bad, want)
		}
	}
	if _, bad := m.findCommand("t"); bad != "t could be tab, theme." {
		t.Errorf("t is short for two: %q", bad)
	}
	if _, bad := m.findCommand("foo"); bad != "Not a command: foo" {
		t.Errorf("foo: %q", bad)
	}
}

// : opens a line at the bottom; enter runs it, esc drops it, and the board
// goes on after a line that is no command.
func TestTheCommandLineOpensRunsAndCancels(t *testing.T) {
	t.Parallel()
	m, _ := newModel(t, map[string]string{"a.md": opened("one\n")})
	press(m, ":")
	typeText(m, "col 2")
	if m.mode != modeCommand || !strings.HasSuffix(strings.TrimRight(screen(m), " "), ":col 2") {
		t.Fatalf(": opens the line at the bottom:\n%s", screen(m))
	}
	press(m, "esc")
	if m.mode != modeBoard || m.columns != 3 {
		t.Fatalf("esc drops the line: mode %v, columns %d", m.mode, m.columns)
	}
	runLine(m, "foo")
	if m.mode != modeBoard || !strings.Contains(screen(m), "Not a command: foo") {
		t.Errorf("an unknown command is said, and the board goes on:\n%s", screen(m))
	}
	press(m, ":", "backspace")
	if m.mode != modeBoard {
		t.Errorf("backspace on an empty line closes it, mode %v", m.mode)
	}
}

// :set shows and changes every setting, as the settings panel does; the
// shorthands do the same for theirs.
func TestSetChangesASetting(t *testing.T) {
	t.Parallel()
	m, _ := newModel(t, map[string]string{"a.md": opened("one\n")})
	runLine(m, "col 2")
	if got := prefs.Get(m.store.Settings(), "columns"); got != "2" || m.columns != 2 || m.status != "columns: 2." {
		t.Fatalf(":col 2: columns %q (%d on the screen), status %q", got, m.columns, m.status)
	}
	runLine(m, "set theme nord")
	if m.store.Theme() != "nord" || m.theme.Get().Name != "nord" {
		t.Errorf(":set theme nord: stored %q, shown %q", m.store.Theme(), m.theme.Get().Name)
	}
	runLine(m, "set stale=1h")
	if got := prefs.Get(m.store.Settings(), "stale"); got != "1h" {
		t.Errorf(":set stale=1h: %q", got)
	}
	runLine(m, "set th")
	if m.status != "theme is nord." {
		t.Errorf(":set th says the theme: %q", m.status)
	}
	runLine(m, "theme")
	if m.status != "theme is nord." {
		t.Errorf(":theme says the theme: %q", m.status)
	}
	runLine(m, "set")
	for _, want := range []string{"theme=nord", "columns=2", "stale=1h", "language=auto"} {
		if !strings.Contains(m.status, want) {
			t.Errorf(":set lists %q: %q", want, m.status)
		}
	}
	runLine(m, "set colour red")
	if !strings.Contains(m.status, "Not a setting: colour") || !strings.Contains(m.status, "theme") {
		t.Errorf("an unknown setting: %q", m.status)
	}
	runLine(m, "col 7")
	if m.columns != 2 || !strings.Contains(m.status, "3, 4, 1, 2") {
		t.Errorf("a value refused keeps the setting and says the values: %d %q", m.columns, m.status)
	}
	runLine(m, "lang ko")
	if m.store.Language() != "ko" || m.tr.Keys != mustLoad(t, "ko").Keys {
		t.Errorf(":lang ko: stored %q", m.store.Language())
	}
}

func mustLoad(t *testing.T, code string) i18n.Strings {
	t.Helper()
	s, err := i18n.Load(code)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// tab completes a command's name, then what it takes: one way to go is
// filled in, several are filled as far as they agree and shown.
func TestTabCompletes(t *testing.T) {
	t.Parallel()
	m, _ := newModel(t, map[string]string{"a.md": opened("one\n")})
	press(m, ":")
	typeText(m, "th")
	press(m, "tab")
	if got := m.cmdLine.Value(); got != "theme " {
		t.Fatalf("th<tab> = %q", got)
	}
	press(m, "tab")
	if !strings.Contains(m.cmdHint, "nord") || !strings.Contains(screen(m), "dracula") {
		t.Errorf("tab after the name shows the themes: %q", m.cmdHint)
	}
	typeText(m, "no")
	press(m, "tab")
	if got := m.cmdLine.Value(); got != "theme nord " {
		t.Errorf("theme no<tab> = %q", got)
	}
	press(m, "esc", ":")
	typeText(m, "set co")
	press(m, "tab")
	typeText(m, "4")
	if got := m.cmdLine.Value(); got != "set columns 4" {
		t.Errorf("set co<tab>4 = %q", got)
	}
	press(m, "esc", ":")
	typeText(m, "t")
	press(m, "tab")
	if m.cmdLine.Value() != "t" || m.cmdHint != "theme tab" {
		t.Errorf("t<tab> shows both: %q %q", m.cmdLine.Value(), m.cmdHint)
	}
}

// ↑ and ↓ go through the lines run this session; past the newest is the
// line being typed.
func TestTheArrowsGoThroughTheHistory(t *testing.T) {
	t.Parallel()
	m, _ := newModel(t, map[string]string{"a.md": opened("one\n")})
	runLine(m, "col 2")
	runLine(m, "col 4")
	runLine(m, "col 4")
	press(m, ":")
	typeText(m, "se")
	for _, step := range []struct{ key, want string }{
		{"up", "col 4"}, {"up", "col 2"}, {"up", "col 2"}, {"down", "col 4"}, {"down", "se"}, {"down", "se"},
	} {
		press(m, step.key)
		if got := m.cmdLine.Value(); got != step.want {
			t.Fatalf("%s: %q, want %q", step.key, got, step.want)
		}
	}
}

// :e zooms a note by its name or title, in any tab; :3 goes to the third
// note, as :3 goes to a line in vi.
func TestEditAndANumberGoToANote(t *testing.T) {
	t.Parallel()
	m, dir := newModel(t, map[string]string{"a.md": opened("one\n"), "b.md": "two\n", "plan.md": "---\ntitle: Launch plan\n---\nship\n"})
	if err := os.MkdirAll(filepath.Join(dir, "work"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "work"), "deep.md", "far away\n")
	runLine(m, "e plan")
	if m.mode != modeZoom || m.zoomName != "plan.md" || !m.isOpen(m.items[m.index("plan.md")]) {
		t.Fatalf(":e plan zooms it: mode %v, %q", m.mode, m.zoomName)
	}
	press(m, "esc")
	runLine(m, "e launch")
	if m.zoomName != "plan.md" {
		t.Errorf(":e by the title's words: %q", m.zoomName)
	}
	press(m, "esc")
	runLine(m, "e deep")
	if m.tabName != "work" || m.zoomName != "work/deep.md" {
		t.Errorf(":e goes to the note's tab: %q %q", m.tabName, m.zoomName)
	}
	runLine(m, "e nothing")
	if m.status != "No note matches nothing." {
		t.Errorf("no match: %q", m.status)
	}
	runLine(m, "tab 1")
	if m.tabName != "" || m.mode != modeBoard {
		t.Errorf(":tab 1 is the root: %q", m.tabName)
	}
	runLine(m, "tab wo")
	if m.tabName != "work" {
		t.Errorf(":tab wo is work: %q", m.tabName)
	}
	runLine(m, "tab 1")
	runLine(m, "2")
	if m.focus != "b.md" {
		t.Errorf(":2 focuses the second note: %q", m.focus)
	}
	runLine(m, "9")
	if m.status != "There is no note 9." {
		t.Errorf(":9: %q", m.status)
	}
}

// :new and :jot make a note as a and N do; :archive, :delete and
// :restore are x, dd and u.
func TestCommandsMakeAndMoveNotes(t *testing.T) {
	t.Parallel()
	m, dir := newModel(t, map[string]string{"a.md": opened("one\n")})
	runLine(m, "new idea")
	if !strings.Contains(readFile(t, dir, "idea.md"), "idea") || m.focus != "idea.md" {
		t.Fatalf(":new idea jots it: focus %q", m.focus)
	}
	runLine(m, "new checklist Release")
	if s := readFile(t, dir, "release.md"); !strings.Contains(s, "type: checklist") {
		t.Errorf(":new checklist Release makes a checklist:\n%s", s)
	}
	runLine(m, "new")
	if m.mode != modeCatalog {
		t.Errorf(":new alone opens the shapes: %v", m.mode)
	}
	press(m, "esc")
	runLine(m, "jot")
	if m.mode != modeInput {
		t.Errorf(":jot alone asks for the line: %v", m.mode)
	}
	press(m, "esc")
	m.setFocus("idea.md")
	runLine(m, "archive")
	if _, err := os.Stat(filepath.Join(dir, "archive", "idea.md")); err != nil {
		t.Fatalf(":archive moves it to archive/: %v", err)
	}
	runLine(m, "restore")
	if _, err := os.Stat(filepath.Join(dir, "idea.md")); err != nil {
		t.Fatalf(":restore brings it back: %v", err)
	}
	m.setFocus("idea.md")
	runLine(m, "delete")
	if m.mode != modeConfirm {
		t.Fatalf(":delete asks first: %v", m.mode)
	}
	press(m, "y")
	if _, err := os.Stat(filepath.Join(dir, "idea.md")); err == nil {
		t.Error(":delete then y deletes it")
	}
	runLine(m, "w")
	if m.status != m.tr.SavesAsYouGo {
		t.Errorf(":w: %q", m.status)
	}
	runLine(m, "keys")
	if m.mode != modeHelp {
		t.Errorf(":keys shows the keys: %v", m.mode)
	}
}

// :q quits, with or without !; :help names what a command does.
func TestQuitAndHelp(t *testing.T) {
	t.Parallel()
	m, _ := newModel(t, map[string]string{"a.md": opened("one\n")})
	for _, line := range []string{"q", "q!", "quit", "wq"} {
		cmd := m.runCommand(line)
		if cmd == nil {
			t.Fatalf(":%s returns nothing", line)
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Errorf(":%s does not quit", line)
		}
	}
	runLine(m, "help set")
	if !strings.HasPrefix(m.status, ":set [setting [value]]") {
		t.Errorf(":help set: %q", m.status)
	}
	runLine(m, "help")
	if m.mode != modeHelp {
		t.Fatalf(":help opens the help: %v", m.mode)
	}
	for _, code := range i18n.Languages() {
		m.tr = mustLoad(t, code)
		lines := m.commandHelp()
		for _, l := range lines {
			if w := widget.Width(l); w > 40 {
				t.Errorf("%s: %q is %d cells wide", code, l, w)
			}
		}
		if !slices.ContainsFunc(lines, func(l string) bool { return strings.HasPrefix(l, "  :set ") }) {
			t.Errorf("%s: the help lists :set", code)
		}
	}
}

// In Korean the answers are Korean and the names stay English; what is
// typed on the line is taken as typed, a jamo too.
func TestTheCommandLineInKorean(t *testing.T) {
	t.Parallel()
	m, _ := newModel(t, map[string]string{"sticky.json": `{"language":"ko"}`, "a.md": opened("one\n")})
	runLine(m, "foo")
	if m.status != "없는 명령이에요: foo" {
		t.Errorf("Korean: %q", m.status)
	}
	press(m, ":")
	m.Update(hangul('ㅓ'))
	if got := m.cmdLine.Value(); got != "ㅓ" {
		t.Errorf("a jamo stays a jamo on the line: %q", got)
	}
}

// keys.json can give the command line another key.
func TestAKeyFileMovesTheCommandLine(t *testing.T) {
	t.Parallel()
	m, _ := newModel(t, map[string]string{"a.md": opened("one\n")})
	if msg := m.UseKeys("keys.json", map[string][]string{"command-line": {";"}}, nil); msg != "" {
		t.Fatal(msg)
	}
	press(m, ":")
	if m.mode == modeCommand {
		t.Error(": is no longer the command line")
	}
	press(m, ";")
	if m.mode != modeCommand {
		t.Errorf("; opens it: %v", m.mode)
	}
}
