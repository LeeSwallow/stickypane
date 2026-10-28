package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LeeSwallow/stickypane/internal/kinds"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/widget/note"
)

// at is the moment a fixed clock says.
var at = time.Date(2026, 10, 3, 14, 2, 5, 0, time.Local)

// Every time the API writes comes from its clock: when a note was made,
// when an item was ticked, when a message was said, when a note went.
func TestTheAPIWritesTheTimeOfItsClock(t *testing.T) {
	dir := filepath.Join(t.TempDir(), store.DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	a := NewWithClock(store.Open(dir), kinds.Default(note.Plain), func() time.Time { return at })
	if !a.Now().Equal(at) {
		t.Errorf("Now = %v", a.Now())
	}

	if _, err := a.Create("fresh.md", Options{}, []byte("hi\n")); err != nil {
		t.Fatal(err)
	}
	if got := readRaw(t, dir, "fresh.md"); !strings.Contains(got, "created: 2026-10-03 14:02\n") {
		t.Errorf("created:\n%s", got)
	}

	if _, err := a.Todo("plan", "add", "tests"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Todo("plan", "check", "tests"); err != nil {
		t.Fatal(err)
	}
	if got := readRaw(t, dir, "plan.md"); !strings.Contains(got, "- [x] tests ✅ 2026-10-03 14:02\n") || !strings.Contains(got, "created: 2026-10-03 14:02\n") {
		t.Errorf("ticked:\n%s", got)
	}

	if _, err := a.Write("done.md", Options{Type: "checklist"}, []byte("- [x] shipped\n")); err != nil {
		t.Fatal(err)
	}
	if got := readRaw(t, dir, "done.md"); !strings.Contains(got, "- [x] shipped ✅ 2026-10-03 14:02\n") {
		t.Errorf("an item ticked without a time is given the clock's:\n%s", got)
	}

	if _, err := a.Say("talk", "agent", "hello"); err != nil {
		t.Fatal(err)
	}
	if got := readRaw(t, dir, "talk.md"); !strings.Contains(got, "@agent 14:02 hello\n") || !strings.Contains(got, "2026-10-03") {
		t.Errorf("said:\n%s", got)
	}

	was, err := a.Store().Load()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "fresh.md")); err != nil {
		t.Fatal(err)
	}
	now, err := a.Store().Load()
	if err != nil {
		t.Fatal(err)
	}
	var removed bool
	for _, ev := range a.Changes(was, now) {
		if ev.Type == "note.removed" && ev.Note == "fresh.md" {
			removed = true
			if !ev.Time.Equal(at) {
				t.Errorf("note.removed at %v, want the clock's %v", ev.Time, at)
			}
		}
	}
	if !removed {
		t.Error("no note.removed")
	}
}
