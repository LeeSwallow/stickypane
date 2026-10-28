package bridge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LeeSwallow/stickypane/internal/adapter/store"
	"github.com/LeeSwallow/stickypane/internal/core/kinds"
	"github.com/LeeSwallow/stickypane/internal/core/when"
	"github.com/LeeSwallow/stickypane/internal/core/widget"
	"github.com/LeeSwallow/stickypane/internal/core/widget/note"
	"github.com/LeeSwallow/stickypane/internal/usecase/api"
)

// The bridge reads the time from the API's clock: the name of a note with
// nothing to name it by, when it was made and when an item was ticked.
func TestTheBridgeWritesTheTimeOfItsClock(t *testing.T) {
	at := time.Date(2026, 10, 3, 14, 2, 5, 0, time.Local)
	dir := board(t, map[string]string{"plan.md": plan})
	s := serveAPI(t, api.NewWithClock(store.Open(dir), kinds.Default(note.Plain), func() time.Time { return at }))

	var got struct{ Note string }
	s.ok("note.create", map[string]any{"kind": "note"}, &got)
	if got.Note != "note-20261003-140205.md" {
		t.Errorf("a note with no title or body is named for the clock: %q", got.Note)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, got.Note)); !strings.Contains(string(b), "created: 2026-10-03 14:02\n") {
		t.Errorf("created:\n%s", b)
	}

	var v struct{ Version string }
	s.ok("note.act", map[string]any{
		"note": "plan.md", "version": versionOf(t, dir, "plan.md"),
		"action": widget.Action{Name: "toggle", Args: map[string]string{"i": "0"}},
	}, &v)
	if b, _ := os.ReadFile(filepath.Join(dir, "plan.md")); !strings.Contains(string(b), "- [x] one ✅ 2026-10-03 14:02") {
		t.Errorf("ticked:\n%s", b)
	}
}

// What a widget writes for an action from the desktop app, a message said
// in a chat or the time a form was sent, is on the API's clock too.
func TestActionsFromTheAppAreStampedByTheAPIClock(t *testing.T) {
	at := time.Date(2026, 10, 3, 14, 2, 5, 0, time.Local)
	dir := board(t, map[string]string{
		"talk.md": "---\ntype: chat\n---\n",
		"ask.md":  "---\ntype: form\n---\n- ( ) yes\n\n[ Send ]\n",
	})
	s := serveAPI(t, api.NewWithClock(store.Open(dir), kinds.Default(note.Plain), func() time.Time { return at }))

	var v struct{ Version string }
	s.ok("note.act", map[string]any{
		"note": "talk.md", "version": versionOf(t, dir, "talk.md"),
		"action": widget.Action{Name: "say", Args: map[string]string{"text": "hello", "who": "min"}},
	}, &v)
	if b, _ := os.ReadFile(filepath.Join(dir, "talk.md")); !strings.Contains(string(b), "## 2026-10-03\n@min 14:02 hello\n") {
		t.Errorf("said:\n%s", b)
	}

	s.ok("note.act", map[string]any{
		"note": "ask.md", "version": versionOf(t, dir, "ask.md"),
		"action": widget.Action{Name: "submit", Args: map[string]string{"label": "Send"}},
	}, &v)
	if b, _ := os.ReadFile(filepath.Join(dir, "ask.md")); !strings.Contains(string(b), "submitted_at: "+at.Format(when.Moment)) {
		t.Errorf("sent:\n%s", b)
	}
}
