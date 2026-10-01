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

func TestMMovesANoteIntoAndOutOfABook(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": opened("note a\n"), "b.md": opened("note b\n")})
	mkdir(t, dir, "docs")
	writeFile(t, dir, "docs/p.md", "page p\n")
	writeView(t, dir, openAll)
	press(m, "r", "m")
	s := screen(m)
	if m.mode != modeMove || !strings.Contains(s, "Move a") || !strings.Contains(s, "docs") || !strings.Contains(s, "New folder") {
		t.Fatalf("m should ask where to move the note:\n%s", s)
	}
	if strings.Contains(s, "top level") {
		t.Errorf("a note that is not in a folder cannot be moved out of one:\n%s", s)
	}
	press(m, "esc")
	if m.mode != modeBoard || !fileExists(dir, "a.md") {
		t.Fatalf("esc should cancel the move")
	}
	press(m, "m", "enter")
	if fileExists(dir, "a.md") || readFile(t, dir, "docs/a.md") != opened("note a\n") {
		t.Fatalf("enter should move the note into the folder")
	}
	if s := screen(m); m.focus != "docs" || !strings.Contains(s, "note a") || !strings.Contains(s, "1/2") {
		t.Fatalf("the book is focused and shows the page that was moved (focus %q):\n%s", m.focus, s)
	}
	press(m, "m")
	if s := screen(m); !strings.Contains(s, "top level") {
		t.Fatalf("a page can be moved out of its book:\n%s", s)
	}
	press(m, "enter")
	if !fileExists(dir, "a.md") || fileExists(dir, "docs/a.md") || m.focus != "a.md" {
		t.Errorf("the first choice for a page is the top level (focus %q)", m.focus)
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
	mkdir(t, dir, "docs")
	writeFile(t, dir, "docs/a.md", "theirs\n")
	press(m, "r", "m", "enter")
	if readFile(t, dir, "docs/a.md") != "theirs\n" || !fileExists(dir, "a.md") {
		t.Fatal("a move must not write over another note")
	}
	if s := screen(m); !strings.Contains(s, "a.md") {
		t.Errorf("the refusal should be explained:\n%s", s)
	}
}
