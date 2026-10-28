package api

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Changes says what happened between two readings of the board: notes made
// and removed, and inside a note what its kind says happened.
func TestChangesSaysWhatHappened(t *testing.T) {
	a, dir := newAPI(t, map[string]string{
		"plan.md":  "---\ntype: checklist\n---\n- [ ] tests\n",
		"old.md":   "bye\n",
		"notes.md": "one\n",
	})
	before, err := a.st.Load()
	if err != nil {
		t.Fatal(err)
	}
	writeAndWait(t, dir, "plan.md", "---\ntype: checklist\n---\n- [x] tests\n")
	writeAndWait(t, dir, "notes.md", "two\n")
	writeAndWait(t, dir, "new.md", "hello\n")
	if err := os.Remove(filepath.Join(dir, "old.md")); err != nil {
		t.Fatal(err)
	}
	after, err := a.st.Load()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, e := range a.Changes(before, after) {
		got[e.Note] = e.Type + " " + e.Item
	}
	want := map[string]string{"plan.md": "item.ticked tests", "notes.md": "note.changed ", "new.md": "note.created ", "old.md": "note.removed "}
	for n, w := range want {
		if got[n] != w {
			t.Errorf("%s: %q, want %q (all: %v)", n, got[n], w, got)
		}
	}
}

// Watch hands each change to emit as it happens, filtered by note and type.
func TestWatchFollowsTheBoard(t *testing.T) {
	a, dir := newAPI(t, map[string]string{"work.md": "---\ntype: board\n---\n## To do\n- login\n## Done\n", "other.md": "x\n"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	got := make(chan Event, 4)
	ready := make(chan struct{})
	go func() {
		_ = a.Watch(ctx, Filter{Notes: []string{"work"}, Types: []string{"card"}}, func(e Event) error {
			got <- e
			return nil
		}, ready)
	}()
	<-ready
	writeAndWait(t, dir, "other.md", "y\n")
	writeAndWait(t, dir, "work.md", "---\ntype: board\n---\n## To do\n## Done\n- login\n")
	select {
	case e := <-got:
		if e.Note != "work.md" || e.Type != "card.moved" || e.Item != "login" || e.From != "To do" || e.To != "Done" {
			t.Errorf("event = %+v", e)
		}
	case <-ctx.Done():
		t.Fatal("no event")
	}
}

// writeAndWait writes a file so that its time differs from what was read.
func writeAndWait(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(time.Second)
	if err := os.Chtimes(path, later, later); err != nil {
		t.Fatal(err)
	}
}
