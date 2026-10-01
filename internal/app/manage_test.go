package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/widget"
)

func TestPinToggles(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": "one\n", "b.md": "two\n"})
	press(m, "tab", "p")
	if v := viewsOf(t, dir)["b.md"].Pin; v == nil || !*v || readFile(t, dir, "b.md") != "two\n" {
		t.Fatalf("pin = %v, file = %q", v, readFile(t, dir, "b.md"))
	}
	if m.items[0].note.Name != "b.md" || m.focus != "b.md" {
		t.Errorf("the pinned note should move first and keep the focus")
	}
	press(m, "p")
	if v := viewsOf(t, dir)["b.md"].Pin; v == nil || *v {
		t.Errorf("pin = %v", v)
	}
}

func TestColorCycles(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": "one\n"})
	first := palette[(colorIndex("a.md", "")+1)%len(palette)].name
	second := palette[(colorIndex("a.md", "")+2)%len(palette)].name
	press(m, "c")
	if got := viewsOf(t, dir)["a.md"].Color; got != first {
		t.Fatalf("color = %q, want %s", got, first)
	}
	press(m, "c")
	if got := viewsOf(t, dir)["a.md"].Color; got != second || readFile(t, dir, "a.md") != "one\n" {
		t.Errorf("color = %q, want %s", got, second)
	}
}

func TestRenameEditsTheTitle(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": "---\ntitle: Old\n---\nx\n"})
	press(m, "R")
	typeText(m, "er")
	press(m, "enter")
	if got := readFile(t, dir, "a.md"); got != "---\ntitle: Older\n---\nx\n" {
		t.Errorf("file = %q", got)
	}
	if !strings.Contains(screen(m), "Older") {
		t.Error("the new title should show in the title bar")
	}
}

func TestArchiveMovesTheNoteAway(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": "one\n", "b.md": "two\n"})
	press(m, "x")
	if got := readFile(t, dir, "archive/a.md"); got != "one\n" {
		t.Errorf("archived file = %q", got)
	}
	if m.focus != "b.md" || len(m.items) != 1 {
		t.Errorf("focus = %q, notes = %d", m.focus, len(m.items))
	}
	if !strings.Contains(screen(m), "archive") {
		t.Error("the user should be told where the note went")
	}
}

func TestDeleteAsksFirst(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": "---\ntitle: Plan\n---\nx\n"})
	path := filepath.Join(dir, "a.md")
	press(m, "D")
	if s := screen(m); !strings.Contains(s, "Delete Plan? (y/n)") {
		t.Fatalf("a confirmation should show:\n%s", s)
	}
	press(m, "n")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("answering n must keep the file: %v", err)
	}
	press(m, "D", "y")
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("answering y should delete the file, stat err = %v", err)
	}
	if len(m.items) != 0 || m.mode != modeBoard {
		t.Errorf("notes = %d, mode = %v", len(m.items), m.mode)
	}
}

func TestKeysDoNothingOnAnEmptyScreen(t *testing.T) {
	m, _ := newModel(t, nil)
	press(m, "p", "c", "R", "x", "D", "e", "enter", "o", "+", "-", "tab", "j", "G")
	if m.mode != modeBoard {
		t.Errorf("mode = %v", m.mode)
	}
}

func TestEditorCommand(t *testing.T) {
	t.Setenv("EDITOR", "code --wait")
	if got := editorCommand("/tmp/a.md").Args; strings.Join(got, " ") != "code --wait /tmp/a.md" {
		t.Errorf("Args = %q", got)
	}
	t.Setenv("EDITOR", "")
	if got := editorCommand("/tmp/a.md").Args; strings.Join(got, " ") != "vi /tmp/a.md" {
		t.Errorf("Args without EDITOR = %q", got)
	}
}

func TestEditorFailureIsReported(t *testing.T) {
	m, _ := newModel(t, map[string]string{"a.md": "one\n"})
	m.Update(reloadMsg{what: "Editor failed", err: errors.New("exit status 1")})
	if s := screen(m); !strings.Contains(s, "Editor failed: exit status 1") {
		t.Errorf("screen:\n%s", s)
	}
}

func TestHelpOpensScrollsAndCloses(t *testing.T) {
	m, _ := newModel(t, map[string]string{"a.md": "one\n"})
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 12})
	for _, line := range strings.Split(helpText, "\n") {
		if w := widget.Width(line); w > 40 {
			t.Errorf("help line %q is %d cells wide, want at most 40", line, w)
		}
	}
	press(m, "?")
	seen := screen(m)
	if !strings.Contains(seen, "any other key closes") {
		t.Fatalf("help should say how to close it:\n%s", seen)
	}
	for i := 0; i < 60; i++ {
		press(m, "j")
		seen += "\n" + screen(m)
	}
	if m.mode != modeHelp {
		t.Fatalf("j should scroll the help, mode = %v", m.mode)
	}
	for _, want := range []string{"open or close", "jot a note", "move to archive", "quit", "reorder a card", "tick an item", "bigger"} {
		if !strings.Contains(seen, want) {
			t.Errorf("scrolling the help should reveal %q", want)
		}
	}
	press(m, "x") // any other key closes help and must not archive
	if m.mode != modeBoard || len(m.items) != 1 {
		t.Errorf("mode = %v, notes = %d", m.mode, len(m.items))
	}
}

func TestHelpFromAZoomedNoteReturnsToIt(t *testing.T) {
	m, _ := newModel(t, map[string]string{"b.md": boardFile})
	press(m, "enter")
	if s := screen(m); !strings.Contains(s, "? keys") {
		t.Errorf("the zoomed note should point to the key help:\n%s", s)
	}
	press(m, "?")
	if m.mode != modeHelp {
		t.Fatalf("mode = %v, want help", m.mode)
	}
	press(m, "x")
	if m.mode != modeZoom {
		t.Errorf("mode = %v, want the zoomed note again", m.mode)
	}
}
