package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// On three columns, five notes are a b c over d e, the second row shared
// equally. h j k l go to the note on that side as the screen shows them;
// down from c is e, which is under it, and up from e is c, which shares
// most of its columns. A count goes as far as it can.
func TestHJKLGoToTheNoteOnThatSide(t *testing.T) {
	t.Parallel()
	m, _ := newModel(t, fiveNotes())
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 60})
	m.setFocus("a.md")
	for _, step := range []struct{ keys, want string }{
		{"l", "b"}, {"l", "c"}, {"l", "c"}, {"j", "e"}, {"j", "e"}, {"k", "c"},
		{"j h", "d"}, {"h", "d"}, {"k", "a"}, {"2 l", "c"}, {"h h", "a"},
		{"G", "e"}, {"g g", "a"}, {"ㅓ", "d"}, {"ㅏ", "a"},
	} {
		for _, k := range strings.Fields(step.keys) {
			if r := []rune(k); len(r) == 1 && r[0] >= 'ㄱ' {
				m.Update(hangul(r[0]))
				continue
			}
			press(m, k)
		}
		if got := strings.TrimSuffix(m.focus, ".md"); got != step.want {
			t.Fatalf("after %s the focus is on %s, want %s", step.keys, got, step.want)
		}
	}
}

// A closed note has no place on the screen: h and l go along the title
// bar from it.
func TestHAndLGoAlongTheTitleBarFromAClosedNote(t *testing.T) {
	t.Parallel()
	m, _ := newModel(t, map[string]string{"a.md": opened("one\n"), "b.md": "closed\n", "c.md": opened("three\n")})
	m.setFocus("b.md")
	press(m, "l")
	if m.focus != "c.md" {
		t.Fatalf("l from a closed note is the next one: %q", m.focus)
	}
	m.setFocus("b.md")
	press(m, "h")
	if m.focus != "a.md" {
		t.Errorf("h from a closed note is the one before: %q", m.focus)
	}
}

// Down from the last row of a screen of notes is the first row of the
// next screen, which comes into view.
func TestJGoesOnToTheNextScreen(t *testing.T) {
	t.Parallel()
	full := "---\nopen: true\nsize: \"1\"\n---\n" + numbered(20)
	m, _ := newModel(t, map[string]string{"a.md": full, "b.md": full, "c.md": full})
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 14})
	if m.screens < 2 {
		t.Fatalf("three tall notes take more than a screen of 14 lines: %d", m.screens)
	}
	m.setFocus("a.md")
	m.relayout()
	var last string
	for range 3 {
		press(m, "j")
		if p, _ := m.paneOf(m.focus); p.screen == 1 {
			last = m.focus
			break
		}
	}
	if last == "" || m.screen != 1 {
		t.Fatalf("j goes on to the notes of the next screen: %q, screen %d", m.focus, m.screen)
	}
	press(m, "g", "g")
	if m.focus != "a.md" || m.screen != 0 {
		t.Errorf("gg comes back to the first: %q, screen %d", m.focus, m.screen)
	}
}

// yy copies where the note's file is, from the project, and says so.
func TestYYCopiesTheNotesPath(t *testing.T) {
	t.Parallel()
	m, dir := newModel(t, map[string]string{"a.md": opened("one\n")})
	m.setFocus("a.md")
	press(m, "y", "y")
	if s := lastLine(screen(m)); !strings.Contains(s, "Copied .sticky/a.md") {
		t.Errorf("yy says what it copied: %q", s)
	}
	msg := copyPath(m, m.items[0])()
	if got := fmt.Sprintf("%T %v", msg, msg); got != "tea.setClipboardMsg .sticky/a.md" {
		t.Errorf("the terminal is asked to copy the path (OSC 52): %s", got)
	}
	// A linked note is the project's own file.
	project := filepath.Dir(dir)
	if err := os.MkdirAll(filepath.Join(project, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, project, "docs/plan.md", "the plan\n")
	if err := m.store.Link("plan.md", filepath.Join(project, "docs", "plan.md")); err != nil {
		t.Fatal(err)
	}
	m.reload()
	if got := m.projectPath(m.items[m.index("plan.md")].note); got != "docs/plan.md" {
		t.Errorf("a linked note's path is the file it shows: %q", got)
	}
}

// The scroll keys are actions: ctrl+d and ctrl+u scroll a zoomed note half
// a page, and j and k still scroll it, since it has nowhere else to go.
func TestTheScrollKeysWorkOnAZoomedNote(t *testing.T) {
	t.Parallel()
	m, _ := newModel(t, map[string]string{"long.md": opened(numbered(60))})
	press(m, "z", "ctrl+d")
	if m.zoomScroll == 0 {
		t.Fatal("ctrl+d scrolls the zoomed note")
	}
	half := m.zoomScroll
	press(m, "j")
	if m.zoomScroll != half+1 {
		t.Errorf("j scrolls a line: %d after %d", m.zoomScroll, half)
	}
	press(m, "k", "ctrl+u")
	if m.zoomScroll != 0 {
		t.Errorf("ctrl+u scrolls back: %d", m.zoomScroll)
	}
	m.keys = rebound(t, map[string][]string{"scroll-half-down": {"J"}, "note-move-down": {}})
	press(m, "esc", "J")
	if p, _ := m.paneOf("long.md"); p.offset == 0 {
		t.Error("keys.json can give the scroll another key")
	}
}
