package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMoveArrangedCarriesTheView(t *testing.T) {
	s := newStore(t)
	write(t, s, "a.md", "a\n")
	if err := s.SetView("a.md", func(v *View) { v.Color = "red" }); err != nil {
		t.Fatal(err)
	}
	if err := s.MoveArranged("a.md", "b.md"); err != nil {
		t.Fatal(err)
	}
	views, err := s.Views()
	if err != nil {
		t.Fatal(err)
	}
	if views["b.md"].Color != "red" {
		t.Fatalf("views = %+v, want b.md red", views)
	}
}

// A move whose view cannot be written is taken back: the note is where it
// was, with its view, and nothing is at the new name.
func TestMoveArrangedThatCannotKeepTheViewMovesBack(t *testing.T) {
	s := newStore(t)
	write(t, s, "t/a.md", "a\n")
	if err := s.SetView("t/a.md", func(v *View) { v.Color = "red" }); err != nil {
		t.Fatal(err)
	}
	// The root's file is broken, so the view cannot be written there.
	write(t, s, ViewFile, "{not json")
	if err := s.MoveArranged("t/a.md", "a.md"); err == nil {
		t.Fatal("the move succeeded without its view")
	}
	if got := read(t, s, "t/a.md"); got != "a\n" {
		t.Fatalf("t/a.md holds %q after the failed move", got)
	}
	if _, err := os.Lstat(filepath.Join(s.Dir, "a.md")); !os.IsNotExist(err) {
		t.Fatalf("a.md is there after the failed move: %v", err)
	}
	if got := read(t, s, ViewFile); got != "{not json" {
		t.Fatalf("the broken file was rewritten: %q", got)
	}
}
