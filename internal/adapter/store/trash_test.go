package store

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func exists(s *Store, name string) bool {
	_, err := os.Lstat(filepath.Join(s.Dir, filepath.FromSlash(name)))
	return err == nil
}

func TestTrashAndRestore(t *testing.T) {
	s := newStore(t)
	write(t, s, "a.md", "first\n")
	to, err := s.Trash("a.md")
	if err != nil || to != ".trash/a.md" || exists(s, "a.md") || read(t, s, ".trash/a.md") != "first\n" {
		t.Fatalf("Trash = %q, %v", to, err)
	}
	if notes, _ := s.Scan(); len(notes) != 0 {
		t.Errorf("the trash is not shown: %+v", notes)
	}
	// A second note of the same name does not push the first out of the trash.
	write(t, s, "a.md", "second\n")
	to2, err := s.Trash("a.md")
	if err != nil || to2 != ".trash/a-2.md" || read(t, s, ".trash/a.md") != "first\n" {
		t.Fatalf("Trash again = %q, %v", to2, err)
	}
	if err := s.Restore(to2, "a.md"); err != nil || read(t, s, "a.md") != "second\n" || exists(s, to2) {
		t.Fatalf("Restore: %v", err)
	}
	if err := s.Restore(to, "a.md"); err == nil || read(t, s, "a.md") != "second\n" {
		t.Errorf("Restore must not write over a note that is there: %v", err)
	}
	if err := s.Restore(".trash/missing.md", "x.md"); err == nil {
		t.Error("restoring what is not there is an error")
	}
	if err := s.Restore("../outside.md", "x.md"); err == nil {
		t.Error("only the trash and the archive can be restored from")
	}
}

func TestTrashKeepsThePlaceOfAPageAndTidiesAnEmptyBook(t *testing.T) {
	s := newStore(t)
	write(t, s, "docs/a.md", "a\n")
	write(t, s, "docs/b.md", "b\n")
	to, err := s.Trash("docs/b.md")
	if err != nil || to != ".trash/docs/b.md" || !exists(s, "docs/a.md") {
		t.Fatalf("Trash = %q, %v", to, err)
	}
	if _, err := s.Trash("docs/a.md"); err != nil || exists(s, "docs") {
		t.Errorf("a folder left without files is removed: %v", err)
	}
	if err := s.Restore(to, "docs/b.md"); err != nil || read(t, s, "docs/b.md") != "b\n" {
		t.Errorf("restoring a page makes its folder again: %v", err)
	}
	// A whole book can go to the trash too.
	if to, err := s.Trash("docs"); err != nil || to != ".trash/docs-2" || exists(s, "docs") || !exists(s, ".trash/docs-2/b.md") {
		t.Errorf("Trash of a folder = %q, %v", to, err)
	}
}

func TestLatestInTrash(t *testing.T) {
	s := newStore(t)
	if _, ok := s.Trashed("a.md"); ok {
		t.Error("nothing is in the trash yet")
	}
	for _, body := range []string{"one\n", "two\n", "three\n"} {
		write(t, s, "a.md", body)
		if _, err := s.Trash("a.md"); err != nil {
			t.Fatal(err)
		}
	}
	if got, ok := s.Trashed("a.md"); !ok || got != ".trash/a-3.md" {
		t.Errorf("Trashed = %q, %v, want the one deleted last", got, ok)
	}
}

func TestMove(t *testing.T) {
	s := newStore(t)
	write(t, s, "a.md", "a\n")
	write(t, s, "b.md", "b\n")
	write(t, s, "docs/p.md", "p\n")
	if err := s.Move("a.md", "docs/a.md"); err != nil || read(t, s, "docs/a.md") != "a\n" || exists(s, "a.md") {
		t.Fatalf("into a book: %v", err)
	}
	if err := s.Move("docs/a.md", "renamed.md"); err != nil || read(t, s, "renamed.md") != "a\n" {
		t.Fatalf("out of a book, under a new name: %v", err)
	}
	if err := s.Move("b.md", "new/b.md"); err != nil || read(t, s, "new/b.md") != "b\n" {
		t.Fatalf("into a folder that is not there yet: %v", err)
	}
	if err := s.Move("renamed.md", "docs/p.md"); !errors.Is(err, os.ErrExist) || read(t, s, "docs/p.md") != "p\n" {
		t.Errorf("Move must not write over another note: %v", err)
	}
	if err := s.Move("missing.md", "x.md"); err == nil {
		t.Error("moving what is not there is an error")
	}
	if err := s.Move("docs/p.md", "elsewhere/p.md"); err != nil || exists(s, "docs") {
		t.Errorf("a folder left without files is removed: %v", err)
	}
}
