package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LeeSwallow/stickypane/internal/adapter/store"
	"github.com/LeeSwallow/stickypane/internal/core/kinds"
	"github.com/LeeSwallow/stickypane/internal/core/widget/board"
	"github.com/LeeSwallow/stickypane/internal/core/widget/note"
	"github.com/LeeSwallow/stickypane/internal/usecase/api"
)

func TestJotCreatesAPlainNoteAndShowsIt(t *testing.T) {
	t.Parallel()
	m, dir := newModel(t, map[string]string{"a.md": "one\n"})
	press(m, "N")
	if s := screen(m); !strings.Contains(s, "Jot:") {
		t.Fatalf("the jot prompt should show:\n%s", s)
	}
	typeText(m, "Check env before deploy")
	press(m, "enter")
	if got := plain(readFile(t, dir, "check-env-before-deploy.md")); got != "Check env before deploy\n" {
		t.Errorf("file = %q, want just the text", got)
	}
	if got := isOpenIn(t, dir, "check-env-before-deploy.md"); got != "true" {
		t.Errorf("the note should be opened in sticky.json so it is still on screen after a restart: %q", got)
	}
	if m.focus != "check-env-before-deploy.md" {
		t.Errorf("focus = %q, want the new note", m.focus)
	}
	s := screen(m)
	if !strings.Contains(s, "│ Check env before deploy") && !strings.Contains(s, "║ Check env before deploy") {
		t.Errorf("the new note should be open on the screen:\n%s", s)
	}
	if strings.Contains(s, "*") {
		t.Errorf("a note you just wrote is not marked as changed:\n%s", s)
	}
}

func TestLowercaseNJotsUnlessTheNoteTakesIt(t *testing.T) {
	t.Parallel()
	m, dir := newModel(t, map[string]string{"a.md": opened("plain\n")})
	press(m, "n")
	typeText(m, "배포 전에 확인")
	press(m, "enter")
	if got := plain(readFile(t, dir, "배포-전에-확인.md")); got != "배포 전에 확인\n" {
		t.Errorf("n on a plain note should jot, file = %q", got)
	}

	m, dir = newModel(t, map[string]string{"b.md": boardFile})
	press(m, "n")
	if s := screen(m); !strings.Contains(s, "New card in To do:") {
		t.Errorf("n on a focused board belongs to the board:\n%s", s)
	}
	press(m, "esc", "N")
	if s := screen(m); !strings.Contains(s, "Jot:") {
		t.Errorf("N should jot even on a board:\n%s", s)
	}
	_ = dir
}

func TestEmptyJotCreatesNothing(t *testing.T) {
	t.Parallel()
	m, dir := newModel(t, nil)
	press(m, "N", "enter")
	press(m, "N", "esc")
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("no file should exist, got %v", entries)
	}
}

// A note the screen makes and one api makes from the same words on the same
// clock are the same file under the same name: one rule names and stamps
// them.
func TestTheScreenMakesNotesAsTheAPIDoes(t *testing.T) {
	clock := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	for _, c := range []struct {
		keys  []string
		text  string
		ext   string
		body  []byte
		named string
	}{
		{[]string{"N"}, "Ship it", "", []byte("Ship it\n"), "ship-it.md"},
		{[]string{"a", "j", "enter"}, "Auth work", ".md", board.Kind.Template("Auth work"), "auth-work.md"},
	} {
		m, dir := newModel(t, nil)
		m.now = func() time.Time { return clock }
		press(m, c.keys...)
		typeText(m, c.text)
		press(m, "enter")

		other := filepath.Join(t.TempDir(), store.DirName)
		if err := os.MkdirAll(other, 0o755); err != nil {
			t.Fatal(err)
		}
		a := api.NewWithClock(store.Open(other), kinds.Default(note.Plain), func() time.Time { return clock })
		name, err := a.CreateFrom("", c.text, c.ext, api.Options{}, c.body)
		if err != nil || name != c.named {
			t.Fatalf("api made %q, %v; want %q", name, err, c.named)
		}
		if screen, made := readFile(t, dir, c.named), readFile(t, other, c.named); screen != made {
			t.Errorf("%s: the screen wrote\n%q\napi wrote\n%q", c.named, screen, made)
		}
	}
}

func TestCatalogCreatesTheChosenShape(t *testing.T) {
	t.Parallel()
	m, dir := newModel(t, nil)
	press(m, "a")
	s := screen(m)
	for _, want := range []string{"Add a note", "Note", "Board (kanban)", "Checklist", "Log"} {
		if !strings.Contains(s, want) {
			t.Fatalf("catalog should list %q:\n%s", want, s)
		}
	}
	if !strings.Contains(s, "Anything in Markdown") {
		t.Errorf("the catalog should say what the selected shape is for:\n%s", s)
	}
	press(m, "j")
	if s := screen(m); !strings.Contains(s, "Cards in columns") || !strings.Contains(s, "To do (1)") {
		t.Errorf("the catalog should describe the board and show an example of it:\n%s", s)
	}
	press(m, "enter")
	if s := screen(m); !strings.Contains(s, "Title:") {
		t.Fatalf("choosing a shape should ask for a title:\n%s", s)
	}
	typeText(m, "Auth work")
	press(m, "enter")
	want := string(board.Kind.Template("Auth work"))
	if got := plain(readFile(t, dir, "auth-work.md")); got != want || isOpenIn(t, dir, "auth-work.md") != "true" {
		t.Errorf("file = %q\nwant  %q", got, want)
	}
	if m.focus != "auth-work.md" || m.mode != modeBoard {
		t.Errorf("focus = %q, mode = %v", m.focus, m.mode)
	}
	if s := screen(m); !strings.Contains(s, "To do (0)") {
		t.Errorf("the new board should be open:\n%s", s)
	}
}

func TestCatalogCanBeCancelled(t *testing.T) {
	t.Parallel()
	m, dir := newModel(t, nil)
	press(m, "a", "j", "esc")
	if m.mode != modeBoard {
		t.Errorf("mode = %v", m.mode)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("no file should exist, got %v", entries)
	}
}
