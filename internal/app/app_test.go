package app

import (
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/kinds"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/widget"
	"github.com/LeeSwallow/stickypane/internal/widget/note"
)

const boardFile = "---\ntype: board\ntitle: Auth\n---\n## To do\n- payments\n## Doing\n- login API\n## Done\n- schema\n"

// newModel builds a board over a temporary notes folder, sized 80x24.
func newModel(t *testing.T, files map[string]string) (*Model, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), store.DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		writeFile(t, dir, name, content)
	}
	m := New(store.Open(dir), kinds.Default(note.Plain), nil)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	return m, dir
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, dir, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// key builds the message Bubble Tea sends for a key name such as "enter",
// "shift+tab", "n" or "D".
func key(k string) tea.KeyPressMsg {
	switch k {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "shift+tab":
		return tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	case "space":
		return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	case "ctrl+c":
		return tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	}
	r := []rune(k)[0]
	return tea.KeyPressMsg{Code: unicode.ToLower(r), Text: k}
}

func press(m *Model, keys ...string) {
	for _, k := range keys {
		m.Update(key(k))
	}
}

func typeText(m *Model, s string) {
	for _, r := range s {
		m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
}

func screen(m *Model) string { return ansi.Strip(m.render()) }

func TestKeyHelperMatchesBubbleTeaNames(t *testing.T) {
	for _, k := range []string{"enter", "esc", "tab", "shift+tab", "space", "ctrl+c", "n", "D", "?"} {
		if got := key(k).String(); got != k {
			t.Errorf("key(%q).String() = %q", k, got)
		}
	}
}

func TestBoardShowsEveryNote(t *testing.T) {
	m, _ := newModel(t, map[string]string{"a.md": "check env\n", "b.md": boardFile})
	s := screen(m)
	for _, want := range []string{"check env", "Auth", "To do (1)", "payments", "n jot"} {
		if !strings.Contains(s, want) {
			t.Errorf("screen should contain %q:\n%s", want, s)
		}
	}
	if n := strings.Count(s, "\n") + 1; n != 24 {
		t.Errorf("screen is %d lines tall, want 24", n)
	}
}

func TestEmptyBoardInvitesToJot(t *testing.T) {
	m, _ := newModel(t, nil)
	if s := screen(m); !strings.Contains(s, "No notes yet") {
		t.Errorf("screen = %q", s)
	}
}

func TestFocusStartsOnFirstNoteAndCycles(t *testing.T) {
	m, _ := newModel(t, map[string]string{"a.md": "one\n", "b.md": "two\n"})
	if m.focus != "a.md" {
		t.Fatalf("focus = %q, want a.md", m.focus)
	}
	if n := strings.Count(screen(m), "╔"); n != 1 {
		t.Errorf("exactly one note should have the focus border, got %d", n)
	}
	press(m, "tab")
	if m.focus != "b.md" {
		t.Errorf("after tab: focus = %q", m.focus)
	}
	press(m, "tab")
	if m.focus != "a.md" {
		t.Errorf("tab should wrap around, focus = %q", m.focus)
	}
	press(m, "shift+tab")
	if m.focus != "b.md" {
		t.Errorf("after shift+tab: focus = %q", m.focus)
	}
}

func TestArrowKeysFollowTheLayout(t *testing.T) {
	// 80 cells wide gives two columns: a and c on the left, b on the right.
	m, _ := newModel(t, map[string]string{"a.md": "one\n", "b.md": "two\n", "c.md": "three\n"})
	for _, step := range []struct{ key, want string }{
		{"l", "b.md"}, {"h", "a.md"}, {"j", "c.md"}, {"k", "a.md"}, {"k", "a.md"},
	} {
		press(m, step.key)
		if m.focus != step.want {
			t.Fatalf("after %s: focus = %q, want %q", step.key, m.focus, step.want)
		}
	}
}

func TestPinnedNotesComeFirst(t *testing.T) {
	m, _ := newModel(t, map[string]string{"a.md": "one\n", "z.md": "---\npin: true\n---\npinned\n"})
	if m.items[0].note.Name != "z.md" {
		t.Errorf("first note = %q, want the pinned one", m.items[0].note.Name)
	}
	if !strings.Contains(screen(m), "📌") {
		t.Error("a pinned note should show the pin")
	}
}

func TestChangedNotesAreMarkedUntilFocused(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": "one\n", "b.md": "two\n"})
	if strings.Contains(screen(m), "●") {
		t.Fatal("nothing changed yet")
	}
	writeFile(t, dir, "b.md", "two, edited by the agent\n")
	future := time.Now().Add(time.Minute)
	if err := os.Chtimes(filepath.Join(dir, "b.md"), future, future); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "c.md", "new from the agent\n")
	m.Update(changedMsg{})
	if n := strings.Count(screen(m), "●"); n != 2 {
		t.Fatalf("the edited note and the new note should be marked, got %d marks:\n%s", n, screen(m))
	}
	press(m, "tab", "tab")
	if strings.Contains(screen(m), "●") {
		t.Errorf("marks should clear once the notes were focused:\n%s", screen(m))
	}
}

func TestFocusSurvivesWhenFocusedNoteDisappears(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": "one\n", "b.md": "two\n", "c.md": "three\n"})
	press(m, "tab")
	os.Remove(filepath.Join(dir, "b.md"))
	m.Update(changedMsg{})
	if m.focus != "c.md" {
		t.Errorf("focus = %q, want the note that took its place", m.focus)
	}
}

func TestUnreadableNoteShowsWhy(t *testing.T) {
	m, _ := newModel(t, map[string]string{"big.md": strings.Repeat("x", store.MaxSize+1)})
	s := screen(m)
	if !strings.Contains(s, "Cannot show this note") || !strings.Contains(s, "big") {
		t.Errorf("screen should explain the problem and name the file:\n%s", s)
	}
}

func TestScrollKeepsFocusedNoteVisible(t *testing.T) {
	files := map[string]string{}
	for i := 0; i < 30; i++ {
		files[string(rune('a'+i%26))+strings.Repeat("x", i/26)+".md"] = "note\n"
	}
	m, _ := newModel(t, files)
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
	for i := 0; i < 29; i++ {
		press(m, "tab")
		if !strings.Contains(screen(m), "╔") {
			t.Fatalf("focused note scrolled out of view after %d tabs", i+1)
		}
	}
}

func TestTinyTerminalDoesNotPanic(t *testing.T) {
	m, _ := newModel(t, map[string]string{"a.md": "한글 메모입니다\n", "b.md": boardFile})
	for _, size := range [][2]int{{0, 0}, {1, 1}, {5, 1}, {5, 3}, {10, 3}, {35, 2}, {200, 2}} {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		press(m, "tab", "j", "l")
		out := m.render()
		if out == "" {
			continue
		}
		lines := strings.Split(out, "\n")
		if len(lines) > size[1] {
			t.Errorf("%v: %d lines, want at most %d", size, len(lines), size[1])
		}
		for _, line := range lines {
			if w := widget.Width(line); w > size[0] {
				t.Errorf("%v: line is %d cells wide", size, w)
			}
		}
	}
}

func TestStatusShowsUntilNextKey(t *testing.T) {
	m, _ := newModel(t, map[string]string{"a.md": "one\n"})
	m.SetStatus("File watching is unavailable.")
	if !strings.Contains(screen(m), "File watching is unavailable.") {
		t.Fatal("status should replace the hint line")
	}
	press(m, "tab")
	if strings.Contains(screen(m), "File watching is unavailable.") {
		t.Error("status should clear on the next key")
	}
}

func TestBackgroundReportRedrawsNotes(t *testing.T) {
	m, _ := newModel(t, map[string]string{"a.md": "one\n", "b.md": "two\n"})
	press(m, "tab")
	var reports []bool
	m.OnBackground = func(dark bool) { reports = append(reports, dark) }
	before := m.items[0].w
	m.Update(tea.BackgroundColorMsg{Color: color.White})
	if len(reports) != 1 || reports[0] {
		t.Errorf("OnBackground calls = %v, want one call with dark=false", reports)
	}
	if m.items[0].w == before {
		t.Error("widgets should be rebuilt so they redraw with the new colors")
	}
	if m.focus != "b.md" || strings.Contains(screen(m), "●") {
		t.Errorf("focus and change marks must survive the redraw, focus = %q", m.focus)
	}
}

func TestQuitKeys(t *testing.T) {
	m, _ := newModel(t, nil)
	for _, k := range []string{"q", "ctrl+c"} {
		if _, cmd := m.Update(key(k)); cmd == nil {
			t.Errorf("%s should return the quit command", k)
		}
	}
}
