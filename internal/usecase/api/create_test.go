package api

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// A book is a folder in a tab: CreateFrom makes a page in it as it makes a
// note in a tab, named after the text, with -2 when the name is taken.
func TestCreateFromMakesAPageInABook(t *testing.T) {
	a, dir := newAPI(t, nil)
	for _, f := range []string{"deploy/docs/01-intro.md", "deploy/docs/02-usage.md", "deploy/plan.md"} {
		if err := os.MkdirAll(filepath.Join(dir, filepath.Dir(f)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, f), []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, c := range []struct{ in, text, want string }{
		{"", "Top", "top.md"},
		{"deploy", "In the tab", "deploy/in-the-tab.md"},
		{"deploy/docs", "Notes", "deploy/docs/notes.md"},
		{"/deploy/docs/", "Notes", "deploy/docs/notes-2.md"},
	} {
		got, err := a.CreateFrom(c.in, c.text, "", Options{}, []byte(c.text+"\n"))
		if err != nil || got != c.want {
			t.Errorf("CreateFrom(%q, %q) = %q, %v; want %q", c.in, c.text, got, err, c.want)
		}
	}
	if got := read(t, dir, "deploy/docs/notes.md"); got != "Notes\n" {
		t.Errorf("the page holds %q", got)
	}
	// Only a tab or a book: not a folder that is not there, not a note,
	// and not a folder deeper than a book, which the board does not show.
	if err := os.MkdirAll(filepath.Join(dir, "deploy/docs/deeper"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, in := range []string{"nope", "deploy/nope", "deploy/plan", "deploy/plan.md", "deploy/docs/deeper"} {
		if _, err := a.CreateFrom(in, "x", "", Options{}, nil); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("CreateFrom(%q) = %v, want fs.ErrNotExist", in, err)
		}
	}
}

func TestPagesListsABooksPagesInOrder(t *testing.T) {
	a, dir := newAPI(t, nil)
	for _, f := range []string{"deploy/docs/02-usage.md", "deploy/docs/01-intro.md", "deploy/plan.md"} {
		if err := os.MkdirAll(filepath.Join(dir, filepath.Dir(f)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, f), []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	pages, err := a.Pages("deploy/docs")
	if err != nil || len(pages) != 2 || pages[0].Name != "deploy/docs/01-intro.md" || pages[1].Name != "deploy/docs/02-usage.md" || pages[0].Title != "intro" {
		t.Fatalf("Pages = %+v, %v", pages, err)
	}
	for _, in := range []string{"deploy", "deploy/plan", "nope/docs"} {
		if _, err := a.Pages(in); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("Pages(%q) = %v, want fs.ErrNotExist", in, err)
		}
	}
}
