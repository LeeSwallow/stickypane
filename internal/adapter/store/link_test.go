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
