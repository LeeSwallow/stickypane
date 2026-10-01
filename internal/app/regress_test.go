package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/widget"
)

func TestFailedWriteRevertsTheScreen(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can write everywhere")
	}
	m, dir := newModel(t, map[string]string{"c.md": checklistFile})
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })
	press(m, "enter", "j", "space")
	if got := readFile(t, dir, "c.md"); got != checklistFile {
		t.Fatalf("the write should have failed, file = %q", got)
	}
	s := screen(m)
	if !strings.Contains(s, "Write failed") {
		t.Errorf("the failure should be reported:\n%s", s)
	}
	if strings.Contains(s, "☑ ship") || !strings.Contains(s, "☐ ship") || !strings.Contains(s, "1/2") {
		t.Errorf("the screen must show the file as it is, not the change that failed:\n%s", s)
	}
}

func TestPromptClosesWhenNoteIsRemoved(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": "one\n", "b.md": boardFile})
	press(m, "tab", "enter", "n")
	if err := os.Remove(filepath.Join(dir, "b.md")); err != nil {
		t.Fatal(err)
	}
	m.Update(changedMsg{})
	if m.mode != modeBoard {
		t.Fatalf("mode = %v, want the board", m.mode)
	}
	s := screen(m)
	if !strings.Contains(s, "The note was removed.") || !strings.Contains(s, "one") {
		t.Errorf("the board should be back with a message:\n%s", s)
	}
}

func TestHelpFitsANarrowPane(t *testing.T) {
	m, _ := newModel(t, map[string]string{"a.md": "one\n"})
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 12})
	for _, line := range strings.Split(helpText, "\n") {
		if w := widget.Width(line); w > 40 {
			t.Errorf("help line %q is %d cells wide, want at most 40", line, w)
		}
	}
	press(m, "?")
	seen := screen(m)
	for i := 0; i < 40; i++ {
		press(m, "j")
		seen += "\n" + screen(m)
	}
	if m.mode != modeHelp {
		t.Fatalf("j should scroll the help, mode = %v", m.mode)
	}
	for _, want := range []string{"jot a note", "move to archive", "quit", "reorder a card", "tick an item"} {
		if !strings.Contains(seen, want) {
			t.Errorf("scrolling the help should reveal %q", want)
		}
	}
	press(m, "k", "x")
	if m.mode != modeBoard || len(m.items) != 1 {
		t.Errorf("another key should close help without acting, mode = %v, notes = %d", m.mode, len(m.items))
	}
}

func TestHelpFromAnOpenNoteReturnsToIt(t *testing.T) {
	m, _ := newModel(t, map[string]string{"b.md": boardFile})
	press(m, "enter")
	if s := screen(m); !strings.Contains(s, "? keys") {
		t.Errorf("the open note should point to the key help:\n%s", s)
	}
	press(m, "?")
	if m.mode != modeHelp {
		t.Fatalf("mode = %v, want help", m.mode)
	}
	press(m, "x")
	if m.mode != modeModal {
		t.Errorf("mode = %v, want the open note again", m.mode)
	}
}
