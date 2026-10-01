package api

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/kinds"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/widget/note"
)

func newAPI(t *testing.T, files map[string]string) (*API, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), store.DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return New(store.Open(dir), kinds.Default(note.Plain)), dir
}

func read(t *testing.T, dir, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestListDescribesEveryNote(t *testing.T) {
	a, _ := newAPI(t, map[string]string{
		"b.md": "---\ntype: board\ntitle: Auth\nopen: true\npin: true\n---\n## To do\n",
		"a.md": "one line\n",
		"c.md": "---\ntype: checklist\nsize: page\n---\n- [ ] x\n",
	})
	got, err := a.List()
	if err != nil {
		t.Fatal(err)
	}
	want := []Info{
		{Name: "a.md", Title: "a", Type: "note", Open: false, Size: "card", Pinned: false},
		{Name: "b.md", Title: "Auth", Type: "board", Open: true, Size: "page", Pinned: true},
		{Name: "c.md", Title: "c", Type: "checklist", Open: false, Size: "page", Pinned: false},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d notes, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("note %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestShowReturnsTheFile(t *testing.T) {
	a, _ := newAPI(t, map[string]string{"a.md": "---\ntitle: T\n---\nbody\n"})
	for _, name := range []string{"a.md", "a"} {
		got, err := a.Show(name)
		if err != nil || string(got) != "---\ntitle: T\n---\nbody\n" {
			t.Errorf("Show(%q) = %q, %v", name, got, err)
		}
	}
	if _, err := a.Show("missing"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Show(missing): err = %v, want not-exist", err)
	}
}

func TestWriteCreatesAndReplaces(t *testing.T) {
	a, dir := newAPI(t, nil)
	name, err := a.Write("plan", Options{Type: "checklist", Title: "Release plan", Open: true, Size: "half"}, []byte("- [ ] build\n"))
	if err != nil || name != "plan.md" {
		t.Fatalf("Write = %q, %v", name, err)
	}
	want := "---\ntype: checklist\ntitle: Release plan\nopen: true\nsize: half\n---\n- [ ] build\n"
	if got := read(t, dir, "plan.md"); got != want {
		t.Errorf("file = %q\nwant  %q", got, want)
	}
	if _, err := a.Write("plan.md", Options{}, []byte("just text\n")); err != nil {
		t.Fatal(err)
	}
	if got := read(t, dir, "plan.md"); got != "just text\n" {
		t.Errorf("a write without options replaces the whole file, got %q", got)
	}
}

func TestWriteKeepsFrontMatterGivenInTheBody(t *testing.T) {
	a, dir := newAPI(t, nil)
	body := "---\ntype: board\ncolor: blue\n---\n## To do\n"
	if _, err := a.Write("b", Options{Open: true}, []byte(body)); err != nil {
		t.Fatal(err)
	}
	if got := read(t, dir, "b.md"); got != "---\ntype: board\ncolor: blue\nopen: true\n---\n## To do\n" {
		t.Errorf("file = %q", got)
	}
}

func TestBadNamesAreRejected(t *testing.T) {
	a, dir := newAPI(t, nil)
	for _, name := range []string{"", "../escape", "sub/note", `sub\note`, ".hidden", "archive/x.md", "note.txt"} {
		if _, err := a.Write(name, Options{}, []byte("x\n")); !errors.Is(err, ErrBadName) {
			t.Errorf("Write(%q): err = %v, want ErrBadName", name, err)
		}
		if _, err := a.Show(name); !errors.Is(err, ErrBadName) {
			t.Errorf("Show(%q): err = %v, want ErrBadName", name, err)
		}
	}
	if _, err := a.Write("x", Options{Size: "huge"}, []byte("x\n")); err == nil {
		t.Error("an unknown size should be rejected")
	}
	if _, err := a.Write("x", Options{Type: "spreadsheet"}, []byte("x\n")); err == nil {
		t.Error("an unknown type should be rejected")
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("nothing should have been written: %v", entries)
	}
	if entries, _ := os.ReadDir(filepath.Dir(dir)); len(entries) != 1 {
		t.Errorf("nothing may be written outside the notes folder: %v", entries)
	}
}
