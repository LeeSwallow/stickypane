package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPinToggles(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": "one\n", "b.md": "two\n"})
	press(m, "tab", "p")
	if got := readFile(t, dir, "b.md"); got != "---\npin: true\n---\ntwo\n" {
		t.Fatalf("file = %q", got)
	}
	if m.items[0].note.Name != "b.md" || m.focus != "b.md" {
		t.Errorf("the pinned note should move first and keep the focus")
	}
	press(m, "p")
	if got := readFile(t, dir, "b.md"); got != "---\npin: false\n---\ntwo\n" {
		t.Errorf("file = %q", got)
	}
}

func TestColorCycles(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": "one\n"})
	first := palette[(colorIndex("a.md", "")+1)%len(palette)].name
	second := palette[(colorIndex("a.md", "")+2)%len(palette)].name
	press(m, "c")
	if got := readFile(t, dir, "a.md"); got != "---\ncolor: "+first+"\n---\none\n" {
		t.Fatalf("file = %q, want color %s", got, first)
	}
	press(m, "c")
	if got := readFile(t, dir, "a.md"); got != "---\ncolor: "+second+"\n---\none\n" {
		t.Errorf("file = %q, want color %s", got, second)
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
		t.Error("the new title should show")
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

func TestManageKeysDoNothingOnAnEmptyBoard(t *testing.T) {
	m, _ := newModel(t, nil)
	press(m, "p", "c", "R", "x", "D", "e", "enter")
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

func TestHelpOpensAndCloses(t *testing.T) {
	m, _ := newModel(t, map[string]string{"a.md": "one\n"})
	press(m, "?")
	s := screen(m)
	for _, want := range []string{"jot a note", "move to archive", "press any key to close"} {
		if !strings.Contains(s, want) {
			t.Fatalf("help should mention %q:\n%s", want, s)
		}
	}
	press(m, "x") // any key closes help and must not archive
	if m.mode != modeBoard || len(m.items) != 1 {
		t.Errorf("mode = %v, notes = %d", m.mode, len(m.items))
	}
}
