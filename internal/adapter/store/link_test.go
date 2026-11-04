package store

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestLinkPutsAProjectFileOnTheBoard(t *testing.T) {
	s := newStore(t)
	readme := filepath.Join(filepath.Dir(s.Dir), "README.md")
	if err := os.WriteFile(readme, []byte("# hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.Link("README.md", filepath.Join("..", "README.md")); err != nil {
		t.Skip("no symbolic links here:", err)
	}
	if !s.Linked("README.md") {
		t.Fatal("the note is not a link")
	}
	if got := read(t, s, "README.md"); got != "# hi\n" {
		t.Fatalf("read %q through the link", got)
	}
	if err := s.Link("README.md", "elsewhere.md"); !errors.Is(err, fs.ErrExist) {
		t.Fatalf("a second link over the first: err = %v, want fs.ErrExist", err)
	}
}

func TestExistsSeesANoteABookAndALinkToNothing(t *testing.T) {
	s := newStore(t)
	if err := s.Write("deploy/plan.md", []byte("x\n")); err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(s.Dir, "deploy", "plan.md"); s.Path("deploy/plan.md") != want {
		t.Errorf("Path = %q, want %q", s.Path("deploy/plan.md"), want)
	}
	if !s.Exists("deploy/plan.md") || !s.Exists("deploy") || s.Exists("missing.md") {
		t.Error("Exists should see the page and its book, and nothing that is not there")
	}
	if err := s.Link("gone.md", "nowhere.md"); err != nil {
		t.Skip("no symbolic links here:", err)
	}
	if !s.Exists("gone.md") {
		t.Error("a link to nothing is still there")
	}
}
