package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fileExists(dir, name string) bool {
	_, err := os.Lstat(filepath.Join(dir, filepath.FromSlash(name)))
	return err == nil
}

func TestDeleteGoesToTheTrashAndUBringsItBack(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": opened("note a\n"), "b.md": opened("note b\n")})
	press(m, "D", "y")
	if fileExists(dir, "a.md") || readFile(t, dir, ".trash/a.md") != opened("note a\n") {
		t.Fatalf("a deleted note is moved to the trash, not removed")
	}
	if s := screen(m); !strings.Contains(s, "u") || !strings.Contains(s, "Deleted a") || strings.Contains(s, "note a") {
		t.Fatalf("the bottom line should say how to take it back:\n%s", s)
	}
	press(m, "u")
	if !fileExists(dir, "a.md") || fileExists(dir, ".trash/a.md") {
		t.Fatalf("u should bring the note back")
	}
	if s := screen(m); !strings.Contains(s, "note a") || m.focus != "a.md" || !strings.Contains(s, "Restored a") {
		t.Errorf("the restored note is shown and has the focus (%q):\n%s", m.focus, s)
	}
	press(m, "u")
	if s := screen(m); !strings.Contains(s, "Nothing to undo") {
		t.Errorf("a second u has nothing to do:\n%s", s)
	}
}

func TestArchiveCanBeUndone(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": opened("note a\n")})
	press(m, "x")
	if !fileExists(dir, "archive/a.md") {
		t.Fatal("x should move the note to the archive")
	}
	press(m, "u")
	if !fileExists(dir, "a.md") || fileExists(dir, "archive/a.md") {
		t.Errorf("u should bring the note back from the archive")
	}
}

func TestUndoDoesNotWriteOverANewNote(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": opened("old\n")})
	press(m, "D", "y")
	writeFile(t, dir, "a.md", "written since\n")
	press(m, "r", "u")
	if got := readFile(t, dir, "a.md"); got != "written since\n" {
		t.Fatalf("undo must not write over a note made in the meantime: %q", got)
	}
	if s := screen(m); !strings.Contains(s, ".trash") {
		t.Errorf("the user should be told where the deleted note still is:\n%s", s)
	}
}

func TestMMovesANoteIntoATabThenABookAndBack(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": opened("note a\n"), "b.md": opened("note b\n")})
	mkdir(t, dir, "t/docs")
	writeFile(t, dir, "t/docs/p.md", "page p\n")
	press(m, "r", "m")
	s := screen(m)
	if m.mode != modeMove || !strings.Contains(s, "Move a") || !strings.Contains(s, "▣ t") || !strings.Contains(s, "New folder") {
		t.Fatalf("m should offer the other tabs:\n%s", s)
	}
	if strings.Contains(s, "top level") || strings.Contains(s, "docs") {
		t.Errorf("from the root, neither the top level nor another tab's books are offered:\n%s", s)
	}
	press(m, "esc")
	if m.mode != modeBoard || !fileExists(dir, "a.md") {
		t.Fatalf("esc should cancel the move")
	}
	press(m, "m", "enter")
	if fileExists(dir, "a.md") || readFile(t, dir, "t/a.md") != opened("note a\n") {
		t.Fatalf("enter should move the note into the tab")
	}
	if s := screen(m); m.tabName != "t" || m.focus != "t/a.md" || !strings.Contains(s, "note a") {
		t.Fatalf("the board follows the note into its tab (tab %q, focus %q):\n%s", m.tabName, m.focus, s)
	}
	press(m, "m")
	if s := screen(m); !strings.Contains(s, "top level") || !strings.Contains(s, "▤ t/docs") {
		t.Fatalf("in a tab, the top level and the tab's books are offered:\n%s", s)
	}
	press(m, "j", "enter")
	if fileExists(dir, "t/a.md") || readFile(t, dir, "t/docs/a.md") != opened("note a\n") || m.focus != "t/docs" {
		t.Fatalf("the note becomes a page of the book (focus %q)", m.focus)
	}
	press(m, "m", "enter")
	if !fileExists(dir, "a.md") || m.tabName != "" || m.focus != "a.md" {
		t.Errorf("the first choice for a page is the top level (tab %q, focus %q)", m.tabName, m.focus)
	}
}

func TestMCanMakeANewFolder(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": opened("note a\n")})
	press(m, "m", "enter") // the only choice is a new folder
	if m.mode != modeInput {
		t.Fatalf("a new folder needs a name, mode = %v", m.mode)
	}
	typeText(m, "Release notes")
	press(m, "enter")
	if !fileExists(dir, "release-notes/a.md") {
		entries, _ := os.ReadDir(dir)
		t.Fatalf("the note should be in the new folder: %v", entries)
	}
}

func TestMovingOntoATakenNameIsRefused(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": opened("mine\n")})
	mkdir(t, dir, "t")
	writeFile(t, dir, "t/a.md", "theirs\n")
	press(m, "r", "m", "enter")
	if readFile(t, dir, "t/a.md") != "theirs\n" || !fileExists(dir, "a.md") {
		t.Fatal("a move must not write over another note")
	}
	if s := screen(m); !strings.Contains(s, "a.md") {
		t.Errorf("the refusal should be explained:\n%s", s)
	}
}
