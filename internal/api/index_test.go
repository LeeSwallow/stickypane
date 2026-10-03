package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The index is every note of every tab in a line: its shape, title, what
// its widget counts, and a gist: its summary, else what to look at first
// (the next open item, the latest log line, the first line of text).
func TestIndexGivesEveryNoteInALine(t *testing.T) {
	a, dir := newAPI(t, map[string]string{
		"plan.md":  "---\ntype: checklist\ntitle: Plan\n---\n- [x] design\n- [ ] write tests\n- [ ] ship\n",
		"why.md":   "---\nsummary: why the cache is in a cookie\n---\n# Decision\n\nLong text.\n",
		"build.log": "compiling\ntests passed\n",
		"intro.md": "# Hello\n\nFirst **real** line.\n",
	})
	if err := os.MkdirAll(filepath.Join(dir, "deploy"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "deploy", "run.md"), []byte("ship it\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	idx, err := a.Index()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]Entry{}
	for _, e := range idx {
		got[e.Name] = e
	}
	for name, want := range map[string]string{
		"plan.md":       "write tests",
		"why.md":        "why the cache is in a cookie",
		"build.log":     "tests passed",
		"intro.md":      "Hello",
		"deploy/run.md": "ship it",
	} {
		if got[name].Gist != want {
			t.Errorf("%s gist = %q, want %q", name, got[name].Gist, want)
		}
	}
	if e := got["plan.md"]; e.Title != "Plan" || e.Summary != "1/3" || e.Kind != "checklist" || e.Tab != "" {
		t.Errorf("plan = %+v", e)
	}
	if got["deploy/run.md"].Tab != "deploy" {
		t.Errorf("a tab's note says its tab: %+v", got["deploy/run.md"])
	}
	md := IndexMarkdown(idx, "proj")
	for _, want := range []string{"# proj", "## deploy", "plan.md", "Plan", "1/3", "write tests"} {
		if !strings.Contains(md, want) {
			t.Errorf("the markdown index should have %q:\n%s", want, md)
		}
	}
}
