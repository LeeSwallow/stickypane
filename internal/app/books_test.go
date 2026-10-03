package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/store"
)

// viewsOf reads sticky.json.
func viewsOf(t *testing.T, dir string) store.Views {
	t.Helper()
	v, err := store.Open(dir).Views()
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func writeView(t *testing.T, dir, json string) {
	t.Helper()
	writeFile(t, dir, store.ViewFile, json)
}

func mkdir(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, name), 0o755); err != nil {
		t.Fatal(err)
	}
}

const openAll = `{"notes":{"build.log":{"open":true},"docs":{"open":true}}}`

func TestALogFileIsShownAsItIs(t *testing.T) {
	m, dir := newModel(t, map[string]string{"build.log": "---\nnot front matter\n---\ncompiled **ok**\n"})
	if s := screen(m); !strings.Contains(s, "≣ build") || m.isOpen(m.items[0]) {
		t.Fatalf("a .log file is a note, a log, and closed until it is opened:\n%s", s)
	}
	press(m, "o")
	s := screen(m)
	for _, want := range []string{"≣ build", "not front matter", "compiled **ok**", "4 lines"} {
		if !strings.Contains(s, want) {
			t.Errorf("a log shows its file as it is, missing %q:\n%s", want, s)
		}
	}
	if got := readFile(t, dir, "build.log"); got != "---\nnot front matter\n---\ncompiled **ok**\n" {
		t.Errorf("opening a log must not write to it: %q", got)
	}
	if v := viewsOf(t, dir)["build.log"]; v.Open == nil || !*v.Open {
		t.Errorf("the log's place on the screen is kept in sticky.json: %+v", v)
	}
}

func TestAFolderIsOneNoteWithPages(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": opened("another note\n")})
	mkdir(t, dir, "docs")
	writeFile(t, dir, "docs/01-intro.md", "---\ntitle: Introduction\n---\nthe first page\n")
	writeFile(t, dir, "docs/02-usage.md", "the second page\n")
	writeFile(t, dir, "docs/03-run.log", "a log page\n")
	writeView(t, dir, openAll)
	press(m, "r", "tab")
	s := screen(m)
	if strings.Count(s, "docs") < 2 || !strings.Contains(s, "docs · Introduction") || !strings.Contains(s, "1/3") {
		t.Fatalf("a folder is one note in the bar and one pane, on its first page:\n%s", s)
	}
	if !strings.Contains(s, "─ Introduction ─") || !strings.Contains(s, "the first page") || !strings.Contains(s, "another note") {
		t.Fatalf("the pages follow one another under a rule with the page's name:\n%s", s)
	}
	press(m, ".")
	if s := screen(m); !strings.Contains(s, "the second page") || !strings.Contains(s, "docs · 02-usage") || !strings.Contains(s, "2/3") {
		t.Fatalf(". should scroll to the next page:\n%s", s)
	}
	press(m, ".")
	if s := screen(m); !strings.Contains(s, "a log page") || !strings.Contains(s, "3/3") {
		t.Fatalf("a page takes its shape from its own file:\n%s", s)
	}
	press(m, ".", ".")
	if s := screen(m); !strings.Contains(s, "3/3") {
		t.Errorf(". stops at the last page:\n%s", s)
	}
	press(m, ",", ",", ",", ",")
	if s := screen(m); !strings.Contains(s, "the first page") || !strings.Contains(s, "1/3") {
		t.Errorf(", should go back to the first page:\n%s", s)
	}
	if s := screen(m); !strings.Contains(s, ", . page") {
		t.Errorf("the bottom line should offer the page keys:\n%s", s)
	}
}

func TestKeysActOnThePageTheViewIsOn(t *testing.T) {
	m, dir := newModel(t, nil)
	mkdir(t, dir, "docs")
	writeFile(t, dir, "docs/a-notes.md", "just text\n")
	writeFile(t, dir, "docs/b-todo.md", "---\ntype: checklist\n---\n- [ ] one\n- [ ] two\n")
	writeView(t, dir, openAll)
	press(m, "r", ".", "j", "space")
	if got := readFile(t, dir, "docs/b-todo.md"); !strings.HasSuffix(got, "- [ ] one\n- [x] two\n") {
		t.Fatalf("space should tick the item of the page the view is on: %q", got)
	}
	if s := screen(m); !strings.Contains(s, "2/2 · 1/2") {
		t.Errorf("the border should show the page and the page's own summary:\n%s", s)
	}
	// The page the view is on stays when the files change.
	writeFile(t, dir, "docs/a-first.md", "a new first page\n")
	press(m, "r")
	if s := screen(m); !strings.Contains(s, "☑ two") || !strings.Contains(s, "3/3") {
		t.Errorf("a new page must not move the view off its page:\n%s", s)
	}
}

func TestABookScrollsAsOneAndTheKeysJumpBetweenPages(t *testing.T) {
	m, dir := newModel(t, nil)
	mkdir(t, dir, "docs")
	writeFile(t, dir, "docs/a.md", numbered(40))
	writeFile(t, dir, "docs/b.md", strings.ReplaceAll(numbered(40), "line", "row"))
	writeView(t, dir, openAll)
	press(m, "r", "G")
	if s := screen(m); !strings.Contains(s, "row 40") || !strings.Contains(s, "2/2") {
		t.Fatalf("G scrolls through every page to the end:\n%s", s)
	}
	press(m, ",")
	if s := screen(m); !strings.Contains(s, "line 01") || !strings.Contains(s, "1/2") {
		t.Errorf(", should jump to the start of the first page:\n%s", s)
	}
	press(m, ".")
	if s := screen(m); !strings.Contains(s, "─ b ─") || !strings.Contains(s, "row 01") || strings.Contains(s, "line 40") {
		t.Errorf(". should put the next page at the top of the pane:\n%s", s)
	}
	for i := 0; i < 50; i++ {
		press(m, "k")
	}
	if s := screen(m); !strings.Contains(s, "line 01") || !strings.Contains(s, "1/2") {
		t.Errorf("scrolling up through the pages is one scroll:\n%s", s)
	}
}

func TestEditingAndDeletingAPage(t *testing.T) {
	m, dir := newModel(t, nil)
	mkdir(t, dir, "docs")
	writeFile(t, dir, "docs/a.md", "page a\n")
	writeFile(t, dir, "docs/b.md", "page b\n")
	writeView(t, dir, openAll)
	press(m, "r", ".", "e")
	if s := screen(m); m.mode != modeEdit || !strings.Contains(s, "docs/b.md") || !strings.Contains(s, "page b") {
		t.Fatalf("e should edit the file of the page the view is on:\n%s", s)
	}
	typeKeys(m, "A!<esc>:wq<enter>")
	if got := readFile(t, dir, "docs/b.md"); got != "page b!\n" {
		t.Fatalf("docs/b.md = %q", got)
	}
	press(m, "D")
	if s := screen(m); !strings.Contains(s, "b") || m.mode != modeConfirm {
		t.Fatalf("D should ask before deleting the page:\n%s", s)
	}
	press(m, "y")
	if _, err := os.Stat(filepath.Join(dir, "docs", "b.md")); !os.IsNotExist(err) {
		t.Errorf("the page's file should be gone: %v", err)
	}
	if s := screen(m); !strings.Contains(s, "page a") {
		t.Errorf("the book stays, on its remaining page:\n%s", s)
	}
}

func TestABookZooms(t *testing.T) {
	m, dir := newModel(t, nil)
	mkdir(t, dir, "docs")
	writeFile(t, dir, "docs/a.md", "page a\n")
	writeFile(t, dir, "docs/b.md", "page b\n")
	writeView(t, dir, openAll)
	press(m, "r", "z", ".")
	if s := screen(m); m.mode != modeZoom || !strings.Contains(s, "page b") || !strings.Contains(s, "2/2") {
		t.Errorf("a zoomed book turns its pages too:\n%s", s)
	}
}
