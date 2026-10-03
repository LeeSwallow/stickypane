package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/store"
)

func TestNamesMayPointIntoABook(t *testing.T) {
	a, dir := newAPI(t, nil)
	if file, err := a.Write("docs/guide", Options{Title: "Guide"}, []byte("how to\n")); err != nil || file != "docs/guide.md" {
		t.Fatalf("Write = %q, %v", file, err)
	}
	if b, err := a.Cat("docs/guide"); err != nil || string(b) != "---\ntitle: Guide\n---\nhow to\n" {
		t.Errorf("Show = %q, %v", b, err)
	}
	if got, err := a.Todo("docs/todo", "add", "first"); err != nil || got != "docs/todo.md: 0/1" {
		t.Errorf("Todo in a book = %q, %v", got, err)
	}
	if got, err := a.Log("build.log", "compiled"); err != nil || got != "build.log: 1 line" || read(t, dir, "build.log") != "compiled\n" {
		t.Errorf("Log to a .log file = %q, %v, %q", got, err, read(t, dir, "build.log"))
	}
	for _, bad := range []string{"../x", "a/b/c/d", "/abs", "docs/.hidden", "docs/", "/x", "a\\b", "x.png", "docs/x.png", ".x/y"} {
		if _, err := a.Cat(bad); err != ErrBadName {
			t.Errorf("Show(%q) = %v, want ErrBadName", bad, err)
		}
	}
}

func TestListShowsPagesAndTheArrangement(t *testing.T) {
	a, dir := newAPI(t, map[string]string{"a.md": "---\nopen: true\nsize: page\n---\nx\n", "build.log": "x\n"})
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "intro.md"), []byte("---\ntype: checklist\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, store.ViewFile), []byte(`{"notes":{"a.md":{"open":false,"size":"card","pin":true}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", store.ViewFile), []byte(`{"notes":{"intro.md":{"open":true}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	infos, err := a.List()
	if err != nil || len(infos) != 3 {
		t.Fatalf("List = %+v, %v", infos, err)
	}
	if n := infos[0]; n.Name != "a.md" || n.Open || n.Size != "card" || !n.Pinned {
		t.Errorf("sticky.json wins over front matter: %+v", n)
	}
	if n := infos[1]; n.Name != "build.log" || n.Type != "log" || n.Title != "build" {
		t.Errorf("a log file: %+v", n)
	}
	if n := infos[2]; n.Name != "docs/intro.md" || n.Type != "checklist" || !n.Open || n.Title != "intro" {
		t.Errorf("a note of a tab is listed by its full name, arranged by the tab's file: %+v", n)
	}
}

func TestSetPutsTheArrangementInStickyJSON(t *testing.T) {
	a, dir := newAPI(t, map[string]string{"a.md": "---\ntitle: Old\n---\nbody\n", "build.log": "x\n"})
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "p.md"), []byte("p\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := a.Set("a", []string{"open=true", "size=half", "rows=8", "pin=true", "color=blue", "title=New"}); err != nil || got != "a.md: set open, size, rows, pin, color, title" {
		t.Fatalf("Set = %q, %v", got, err)
	}
	if got := read(t, dir, "a.md"); got != "---\ntitle: New\n---\nbody\n" {
		t.Errorf("only keys about the content go into the note: %q", got)
	}
	v, _ := store.Open(dir).Views()
	if n := v["a.md"]; n.Open == nil || !*n.Open || n.Size != "half" || n.Rows != 8 || n.Pin == nil || !*n.Pin || n.Color != "blue" {
		t.Errorf("the arrangement goes to sticky.json: %+v", n)
	}
	if _, err := a.Set("a", []string{"open=", "size="}); err != nil {
		t.Fatal(err)
	}
	if n := (func() store.View { v, _ := store.Open(dir).Views(); return v["a.md"] })(); n.Open != nil || n.Size != "" || n.Rows != 8 {
		t.Errorf("an empty value takes the key back out: %+v", n)
	}
	if _, err := a.Set("build.log", []string{"open=true"}); err != nil {
		t.Errorf("a log can be arranged: %v", err)
	}
	if _, err := a.Set("build.log", []string{"type=board"}); err == nil || !strings.Contains(err.Error(), "Markdown") {
		t.Errorf("a log has no front matter to put a type in: %v", err)
	}
	if got, err := a.Set("docs", []string{"open=true"}); err != nil || got != "docs: set open" {
		t.Errorf("a book is arranged by its folder's name: %q, %v", got, err)
	}
	for _, bad := range [][]string{{"size=huge"}, {"open=maybe"}, {"rows=many"}, {"rows=-1"}, {"color=mauve"}} {
		if _, err := a.Set("a", bad); err == nil {
			t.Errorf("Set(%v) should fail", bad)
		}
	}
}

func TestSetNamesANoteThatHasNoFrontMatter(t *testing.T) {
	a, dir := newAPI(t, map[string]string{"build.log": "x\n"})
	if _, err := a.Write("docs/p", Options{}, []byte("p\n")); err != nil {
		t.Fatal(err)
	}
	if got, err := a.Set("build.log", []string{"title=CI build"}); err != nil || got != "build.log: set title" {
		t.Fatalf("Set = %q, %v", got, err)
	}
	if got, err := a.Set("docs", []string{"title=Handbook"}); err != nil || got != "docs: set title" {
		t.Fatalf("Set on a book = %q, %v", got, err)
	}
	v, _ := store.Open(dir).Views()
	if v["build.log"].Title != "CI build" || v["docs"].Title != "Handbook" || read(t, dir, "build.log") != "x\n" {
		t.Errorf("Views = %+v", v)
	}
	infos, _ := a.List()
	if infos[0].Title != "CI build" {
		t.Errorf("List should show the name: %+v", infos[0])
	}
}
