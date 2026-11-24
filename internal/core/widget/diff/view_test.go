package diff

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/core/widget"
)

func page(t *testing.T, fixture string, width int) string {
	t.Helper()
	b, err := os.ReadFile("testdata/" + fixture + ".diff")
	if err != nil {
		t.Fatal(err)
	}
	out, _ := NewPage(string(b), widget.DefaultEnv()).Draw(width, true)
	return ansi.Strip(out)
}

func TestPageSaysHowEachFileChanged(t *testing.T) {
	t.Parallel()
	for fixture, want := range map[string][]string{
		"rename":      {"R moved.txt → moved-to.txt  +0 −0", "similarity 100%"},
		"rename-edit": {"R renamed-edit.txt → renamed-done.txt  +1 −1", "- gamma", "+ GAMMA", "  alpha"},
		"binary":      {"M image.bin  binary"},
		"mode":        {"M run.sh  +0 −0", "mode 100644 → 100755"},
		"no-newline":  {"+ c", `\ No newline at end of file`},
		"added":       {"A added.txt  +2 −0", "+ fresh"},
		"deleted":     {"D gone.txt  +0 −2", "- bye"},
		"empty":       {"no changes"},
	} {
		out := page(t, fixture, 80)
		for _, w := range want {
			if !strings.Contains(out, w) {
				t.Errorf("%s lacks %q:\n%s", fixture, w, out)
			}
		}
	}
}

func TestPageColorsAddedAndRemovedLines(t *testing.T) {
	t.Parallel()
	e := widget.DefaultEnv()
	out, _ := NewPage("diff --git a/x b/x\n--- a/x\n+++ b/x\n@@ -1 +1 @@\n-old\n+new\n", e).Draw(40, false)
	if !strings.Contains(out, e.Styles.Good.Render("+ new")) || !strings.Contains(out, e.Styles.Bad.Render("- old")) {
		t.Errorf("lines not colored: %q", out)
	}
}

// Code wraps at the pane's width and keeps its indent: nothing is cut.
func TestPageWrapsCodeKeepingItsIndent(t *testing.T) {
	t.Parallel()
	out, _ := NewPage("diff --git a/x b/x\n--- a/x\n+++ b/x\n@@ -0,0 +1 @@\n+\tif ok { return verylongname }\n", widget.DefaultEnv()).Draw(12, false)
	lines := strings.Split(ansi.Strip(out), "\n")
	joined := strings.Join(lines[2:], "")
	if !strings.Contains(joined, "+     if ok { return verylongname }") {
		t.Errorf("wrapped code lost text: %q", lines)
	}
	for _, l := range lines {
		if ansi.StringWidth(l) > 12 {
			t.Errorf("line %q is wider than 12", l)
		}
	}
}

func TestListShowsWorktreesOverlapsAndThePullRequest(t *testing.T) {
	t.Parallel()
	o := Overview{
		Worktrees: []Worktree{
			{Name: "main", Base: "main", Current: true},
			{Name: "feat/chart-ui", Base: "main", Files: 7, Add: 128, Del: 41},
			{Name: "refactor/widget", Base: "main", Files: 3, Add: 22, Del: 19},
			{Name: "gone", Closed: true},
			{Name: "lost", Base: ""},
		},
		Overlaps: []Overlap{{Path: "internal/core/widget/chart/view.go", By: []string{"feat/chart-ui", "refactor/widget"}}},
		Chosen:   "feat/chart-ui",
		PR:       &PR{Title: "feat: chart", State: "OPEN", URL: "https://x"},
	}
	l := NewList(o, "", widget.DefaultEnv())
	out, at := l.Draw(80, true)
	out = ansi.Strip(out)
	for _, want := range []string{
		"▸ feat/chart-ui  main…  7 files  +128 −41",
		"  refactor/widget  main…  3 files  +22 −19",
		"  gone  closed",
		"  lost  no base: b chooses one",
		"⚠ internal/core/widget/chart/view.go",
		"changed in feat/chart-ui · refactor/widget",
		"PR open · feat: chart",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("list lacks %q:\n%s", want, out)
		}
	}
	if at.Start != 1 || l.Cursor() != "feat/chart-ui" || l.Summary() != "⚠ 1" {
		t.Errorf("span %+v, cursor %q, summary %q", at, l.Cursor(), l.Summary())
	}
	l.Update("j")
	if l.Cursor() != "refactor/widget" {
		t.Errorf("j moved to %q", l.Cursor())
	}
	for range 10 {
		l.Update("j")
	}
	l.Update("k")
	if l.Cursor() != "gone" {
		t.Errorf("k from the end is on %q", l.Cursor())
	}
	if c := NewList(o, "refactor/widget", widget.DefaultEnv()).Cursor(); c != "refactor/widget" {
		t.Errorf("a list made again keeps its cursor: %q", c)
	}
	for _, w := range []int{1, 7, 20} {
		out, _ := l.Draw(w, true)
		for _, line := range strings.Split(out, "\n") {
			if widget.Width(line) > w {
				t.Errorf("width %d: %q", w, line)
			}
		}
	}
}

func TestAnEmptyListSaysSo(t *testing.T) {
	t.Parallel()
	l := NewList(Overview{}, "", widget.DefaultEnv())
	out, _ := l.Draw(40, false)
	if !strings.Contains(out, "no git worktrees here") || l.Cursor() != "" {
		t.Errorf("empty list = %q", out)
	}
	l.Update("j")
	l.Update("k")
}
