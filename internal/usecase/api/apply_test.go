package api

import (
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/core/doc"
	"github.com/LeeSwallow/stickypane/internal/core/widget"
)

func versionOf(t *testing.T, dir, name string) string {
	t.Helper()
	return widget.Version(doc.Parse([]byte(readRaw(t, dir, name))))
}

func TestApplyTicksAndStampsTheTime(t *testing.T) {
	a, dir := newAPI(t, map[string]string{"plan.md": "---\ntype: checklist\n---\n- [ ] one\n- [ ] two\n"})
	v, err := a.Apply("plan", versionOf(t, dir, "plan.md"), widget.Action{Name: "toggle", Args: map[string]string{"i": "1"}})
	if err != nil {
		t.Fatal(err)
	}
	got := readRaw(t, dir, "plan.md")
	if !regexp.MustCompile(`- \[x\] two .*\d{2}:\d{2}`).MatchString(got) || !strings.Contains(got, "- [ ] one") {
		t.Errorf("the item is ticked with the time: %q", got)
	}
	if v != versionOf(t, dir, "plan.md") {
		t.Errorf("Apply returns the new version: %q", v)
	}
}

func TestApplyRefusesAStaleVersion(t *testing.T) {
	a, dir := newAPI(t, map[string]string{"plan.md": "---\ntype: checklist\n---\n- [ ] one\n"})
	old := versionOf(t, dir, "plan.md")
	if _, err := a.Todo("plan", "add", "two"); err != nil {
		t.Fatal(err)
	}
	before := readRaw(t, dir, "plan.md")
	if _, err := a.Apply("plan", old, widget.Action{Name: "toggle", Args: map[string]string{"i": "0"}}); !errors.Is(err, ErrStale) {
		t.Fatalf("err = %v, want ErrStale", err)
	}
	if got := readRaw(t, dir, "plan.md"); got != before {
		t.Errorf("a stale action must not touch the file: %q", got)
	}
}

func TestApplyOnAKindWithoutAct(t *testing.T) {
	a, dir := newAPI(t, map[string]string{"n.md": "plain\n"})
	if _, err := a.Apply("n", versionOf(t, dir, "n.md"), widget.Action{Name: "toggle"}); err == nil || errors.Is(err, ErrStale) {
		t.Errorf("a plain note takes no action: %v", err)
	}
	if got := readRaw(t, dir, "n.md"); got != "plain\n" {
		t.Errorf("file touched: %q", got)
	}
}

// Do is how the terminal writes what a key asked: the op on the file as it
// is now, and what the kind fills in by itself.
func TestDoAppliesAnOpAndTendsTheNote(t *testing.T) {
	a, dir := newAPI(t, map[string]string{"plan.md": "---\ntype: checklist\n---\n- [x] one\n"})
	if err := a.Do("plan.md", doc.SetKey{Key: "title", Value: "Plan"}); err != nil {
		t.Fatal(err)
	}
	got := readRaw(t, dir, "plan.md")
	if !strings.Contains(got, "title: Plan") || !regexp.MustCompile(`- \[x\] one .*\d{2}:\d{2}`).MatchString(got) {
		t.Errorf("Do:\n%s", got)
	}
}

// Save writes the editor's text as it is, and only over the file the
// editor loaded unless told to.
func TestSaveWritesTheTextAsItIsOverWhatWasLoaded(t *testing.T) {
	a, dir := newAPI(t, map[string]string{"n.md": "---\nopen: true\n---\nold\n"})
	if err := a.Save("n.md", []byte("not what is there\n"), []byte("mine\n")); !errors.Is(err, ErrStale) {
		t.Errorf("a file someone else changed: %v", err)
	}
	if err := a.Save("n.md", []byte("---\nopen: true\n---\nold\n"), []byte("mine\n")); err != nil || readRaw(t, dir, "n.md") != "mine\n" {
		t.Errorf("Save = %v, file %q: the text is the user's, nothing kept or added", err, readRaw(t, dir, "n.md"))
	}
	if err := a.Save("n.md", nil, []byte("forced\n")); err != nil || readRaw(t, dir, "n.md") != "forced\n" {
		t.Errorf("a forced save: %v", err)
	}
}

func TestApplyOnAMissingNote(t *testing.T) {
	a, _ := newAPI(t, nil)
	if _, err := a.Apply("nope", "0123456789abcdef", widget.Action{Name: "toggle"}); !errors.Is(err, doc.ErrConflict) {
		t.Errorf("err = %v, want doc.ErrConflict", err)
	}
}
