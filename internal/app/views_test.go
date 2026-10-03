package app

import (
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestArrangingTheScreenLeavesTheNotesAlone(t *testing.T) {
	const note = "---\ntitle: Plan\n---\nthe plan\n"
	m, dir := newModel(t, map[string]string{"a.md": note})
	press(m, "o", "+", "p", "c")
	if got := readFile(t, dir, "a.md"); got != note {
		t.Fatalf("opening, sizing, pinning and coloring must not rewrite the note: %q", got)
	}
	v := viewsOf(t, dir)["a.md"]
	if v.Open == nil || !*v.Open || v.Pin == nil || !*v.Pin || v.Size == "" || v.Color == "" {
		t.Errorf("the arrangement goes to sticky.json: %+v", v)
	}
	if s := screen(m); !strings.Contains(s, "✎ Plan") || !strings.Contains(s, "the plan") || !m.pinned(m.items[0]) {
		t.Errorf("the screen should show the arrangement:\n%s", s)
	}
	press(m, "o", "p")
	if v := viewsOf(t, dir)["a.md"]; v.Open == nil || *v.Open || v.Pin == nil || *v.Pin {
		t.Errorf("closing and unpinning are remembered too: %+v", v)
	}
}

func TestStickyJSONWinsOverFrontMatter(t *testing.T) {
	m, dir := newModel(t, map[string]string{
		"a.md": "---\nopen: true\nsize: page\npin: true\ncolor: blue\n---\nnote a\n",
		"b.md": "---\nopen: true\n---\nnote b\n",
	})
	if s := screen(m); !strings.Contains(s, "note a") || !m.pinned(m.items[0]) {
		t.Fatalf("front matter arranges a note until the user says otherwise:\n%s", s)
	}
	writeView(t, dir, `{"notes":{"a.md":{"open":false,"pin":false}}}`)
	press(m, "r")
	if s := screen(m); strings.Contains(s, "note a") || m.pinned(m.items[0]) || !strings.Contains(s, "note b") {
		t.Errorf("what sticky.json says wins:\n%s", s)
	}
}

func TestStickyJSONIsReadAgainWhenItChanges(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": "note a\n"})
	if s := screen(m); strings.Contains(s, "note a") {
		t.Fatalf("the note starts closed:\n%s", s)
	}
	writeView(t, dir, `{"notes":{"a.md":{"open":true}}}`)
	m.Update(changedMsg{})
	if s := screen(m); !strings.Contains(s, "note a") {
		t.Errorf("an agent opening a note through sticky.json should show at once:\n%s", s)
	}
}

func TestABrokenStickyJSONIsSaidAndNotOverwritten(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": opened("note a\n")})
	writeView(t, dir, "{ broken")
	press(m, "r")
	if s := screen(m); !strings.Contains(s, "sticky.json") || !strings.Contains(s, "note a") {
		t.Fatalf("the board should say the file is broken and still show the notes:\n%s", s)
	}
	press(m, "o")
	if got := readFile(t, dir, "sticky.json"); got != "{ broken" {
		t.Errorf("a broken file must not be overwritten: %q", got)
	}
	if s := screen(m); !strings.Contains(s, "sticky.json") {
		t.Errorf("the failed change should be explained:\n%s", s)
	}
}

func TestBracesMoveANoteAmongTheOthers(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": opened("note a\n"), "b.md": opened("note b\n"), "c.md": opened("note c\n")})
	order := func() string {
		var names []string
		for _, it := range m.items {
			names = append(names, it.note.Name)
		}
		return strings.Join(names, " ")
	}
	press(m, "}")
	if got := order(); got != "b.md a.md c.md" || m.focus != "a.md" {
		t.Fatalf("} should move the focused note one place later: %s (focus %s)", got, m.focus)
	}
	press(m, "}", "}")
	if got := order(); got != "b.md c.md a.md" {
		t.Errorf("} stops at the end: %s", got)
	}
	press(m, "{")
	if got := order(); got != "b.md a.md c.md" {
		t.Errorf("{ should move it back: %s", got)
	}
	if got := readFile(t, dir, "sticky.json"); !strings.Contains(got, `"order"`) {
		t.Errorf("the order is kept in sticky.json:\n%s", got)
	}
	// A new note takes its place by name after the ones that were moved.
	writeFile(t, dir, "0-new.md", "new\n")
	press(m, "r")
	if got := order(); got != "b.md a.md c.md 0-new.md" {
		t.Errorf("order with a new note: %s", got)
	}
	// A pinned note stays first and moves only among pinned notes.
	press(m, "p", "{")
	if got := order(); got != "a.md b.md c.md 0-new.md" {
		t.Errorf("a pinned note is first and does not trade places with an unpinned one: %s", got)
	}
}

func TestANoteWithoutFrontMatterIsNamedInStickyJSON(t *testing.T) {
	m, dir := newModel(t, map[string]string{"build.log": "compiled\n"})
	mkdir(t, dir, "docs")
	writeFile(t, dir, "docs/a.md", "page a\n")
	press(m, "r", "R")
	if m.mode != modeInput {
		t.Fatalf("R on a log should ask for a name, mode = %v", m.mode)
	}
	typeText(m, "CI build")
	press(m, "enter")
	if got := readFile(t, dir, "build.log"); got != "compiled\n" {
		t.Fatalf("a log has no front matter to put a title in: %q", got)
	}
	if v := viewsOf(t, dir)["build.log"]; v.Title != "CI build" {
		t.Errorf("the name goes to sticky.json: %+v", v)
	}
	if s := screen(m); !strings.Contains(s, "≣ CI build") {
		t.Errorf("the title bar should show the name:\n%s", s)
	}
	// A book is named the same way; the folder keeps its name.
	writeView(t, dir, `{"notes":{"docs":{"open":true,"title":"Handbook"}}}`)
	press(m, "r")
	if s := screen(m); !strings.Contains(s, "Handbook · a") || !fileExists(dir, "docs/a.md") {
		t.Errorf("a book goes by the name sticky.json gives it:\n%s", s)
	}
}

func TestRenamingALinkedNoteLeavesItsFileAlone(t *testing.T) {
	m, dir := newModel(t, nil)
	outside := filepath.Join(filepath.Dir(dir), "README.md")
	if err := os.WriteFile(outside, []byte("# Readme\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "README.md")); err != nil {
		t.Skip("symlinks are not available:", err)
	}
	press(m, "r", "R")
	typeText(m, "Read me")
	press(m, "enter")
	if b, _ := os.ReadFile(outside); string(b) != "# Readme\n" {
		t.Fatalf("the project's file must not get front matter: %q", b)
	}
	if v := viewsOf(t, dir)["README.md"]; v.Title != "Read me" {
		t.Errorf("the name goes to sticky.json: %+v", v)
	}
}

func TestTCyclesThemesAndKeepsTheChoice(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": opened("note a\n")})
	first := m.theme.Get().Name
	before := m.render()
	press(m, "T")
	next := m.theme.Get().Name
	if next == first || !strings.Contains(screen(m), "Theme: "+next) {
		t.Fatalf("T should switch to the next theme and say so: %q -> %q\n%s", first, next, screen(m))
	}
	if m.render() == before {
		t.Error("the screen should be drawn in the new colors")
	}
	if got := readFile(t, dir, "sticky.json"); !strings.Contains(got, `"theme": "`+next+`"`) {
		t.Errorf("the choice goes to sticky.json:\n%s", got)
	}
	// A chosen theme stays whatever the terminal says about its background.
	m.Update(tea.BackgroundColorMsg{Color: color.White})
	if got := m.theme.Get().Name; got != next {
		t.Errorf("theme after a background report = %q, want %q", got, next)
	}
}

func TestTheScreenSpeaksTheChosenLanguage(t *testing.T) {
	t.Setenv("STICKYPANE_LANG", "en")
	m, dir := newModel(t, map[string]string{"a.md": "note a\n"})
	if s := screen(m); !strings.Contains(s, "Nothing is open") {
		t.Fatalf("English by default:\n%s", s)
	}
	writeView(t, dir, `{"language":"ko"}`)
	UseLanguage(m.store.Language())
	m.reload()
	s := screen(m)
	if !strings.Contains(s, "열린 노트가 없습니다") || !strings.Contains(s, "도움말") {
		t.Errorf("the screen and the hints should be Korean:\n%s", s)
	}
	press(m, "o", "D")
	if s := screen(m); !strings.Contains(s, "삭제할까요") {
		t.Errorf("messages with names are filled in Korean:\n%s", s)
	}
	press(m, "n")
	UseLanguage("en")
	if s := screen(m); strings.Contains(s, "도움말") {
		t.Errorf("back to English:\n%s", s)
	}
}
