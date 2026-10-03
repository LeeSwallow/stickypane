package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/kinds"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/theme"
)

// A line ![[plan]] in a note draws the plan there, as its own shape and
// read only; when the plan changes, the note that shows it does too.
func TestANoteEmbedsAnother(t *testing.T) {
	dir := filepath.Join(t.TempDir(), store.DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "summary.md", "---\nopen: true\n---\nWhere we are:\n\n![[plan]]\n\nThat is all.\n")
	writeFile(t, dir, "plan.md", "---\ntype: checklist\n---\n- [x] design\n- [ ] write tests\n")
	th := theme.NewHolder(theme.Pick("auto", true))
	m := New(store.Open(dir), kinds.Default(kinds.Markdown(th)), nil, th)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	s := screen(m)
	if !strings.Contains(s, "Where we are") || !strings.Contains(s, "1/2") || !strings.Contains(s, "write tests") || strings.Contains(s, "![[") {
		t.Fatalf("the plan is drawn inside the note:\n%s", s)
	}
	writeFile(t, dir, "plan.md", "---\ntype: checklist\n---\n- [x] design\n- [x] write tests\n")
	m.Update(changedMsg{})
	if s := screen(m); !strings.Contains(s, "2/2") {
		t.Errorf("the embed follows the plan:\n%s", s)
	}
}

// A chart computed from a checklist follows it on the board.
func TestADerivedChartFollowsItsSourceOnTheBoard(t *testing.T) {
	m, dir := newModel(t, map[string]string{
		"plan.md":     "---\ntype: checklist\n---\n- [x] a\n- [ ] b\n",
		"progress.md": "---\ntype: chart\nopen: true\nfrom: plan\n---\n",
	})
	if got := readFile(t, dir, "progress.md"); !strings.Contains(got, "done: 1\nopen: 1") {
		t.Fatalf("the board computes the chart when it reads it:\n%s", got)
	}
	writeFile(t, dir, "plan.md", "---\ntype: checklist\n---\n- [x] a\n- [x] b\n")
	m.Update(changedMsg{})
	if got := readFile(t, dir, "progress.md"); !strings.Contains(got, "done: 2\nopen: 0") {
		t.Errorf("and again when the source changes:\n%s", got)
	}
}
