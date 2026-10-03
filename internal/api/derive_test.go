package api

import (
	"strings"
	"testing"
)

// A chart with from: says where its values come from; stickypane computes
// them from that note and writes them into the chart, so the file is true
// without stickypane.
func TestDerivedChartsComputeTheirValues(t *testing.T) {
	a, dir := newAPI(t, map[string]string{
		"plan.md":     "---\ntype: checklist\n---\n- [x] a\n- [x] b\n- [ ] c\n",
		"work.md":     "---\ntype: board\n---\n## To do\n- x\n## Done\n- y\n- z\n",
		"progress.md": "---\ntype: chart\nfrom: plan\n---\nHow far the plan is.\n",
		"cards.md":    "---\ntype: chart\nfrom: work\n---\n",
		"both.md":     "---\ntype: chart\nfrom: plan, work\n---\nold: 1\n",
		"loop.md":     "---\ntype: chart\nfrom: progress\n---\n",
	})
	n, err := a.Derive()
	if err != nil || n != 3 {
		t.Fatalf("Derive = %d, %v; want three charts written", n, err)
	}
	for name, want := range map[string]string{
		"progress.md": "How far the plan is.\ndone: 2\nopen: 1\n",
		"cards.md":    "To do: 1\nDone: 2\n",
		"both.md":     "plan: 67\nwork: 3\n",
	} {
		if got := read(t, dir, name); !strings.HasSuffix(got, "---\n"+want) {
			t.Errorf("%s =\n%s\nwant body\n%s", name, got, want)
		}
	}
	if got := read(t, dir, "loop.md"); strings.Contains(got, ": ") && !strings.Contains(got, "from: progress") {
		t.Errorf("a chart derived from a derived chart is left alone: %q", got)
	}
	if n, _ := a.Derive(); n != 0 {
		t.Errorf("nothing changed, nothing written: %d", n)
	}
}

// A change from the command line recomputes the charts that depend on it.
func TestAChangeRecomputesTheCharts(t *testing.T) {
	a, dir := newAPI(t, map[string]string{
		"plan.md":     "---\ntype: checklist\n---\n- [ ] a\n",
		"progress.md": "---\ntype: chart\nfrom: plan\n---\n",
	})
	if _, err := a.Todo("plan", "check", "a"); err != nil {
		t.Fatal(err)
	}
	if got := read(t, dir, "progress.md"); !strings.Contains(got, "done: 1\nopen: 0") {
		t.Errorf("progress = %q", got)
	}
}

func TestAChartOfSeveralNotesNamesThemByTitle(t *testing.T) {
	a, dir := newAPI(t, map[string]string{
		"20-plan.md":  "---\ntype: checklist\ntitle: Token endpoint\n---\n- [x] a\n- [ ] b\n",
		"10-work.md":  "---\ntype: board\n---\n## To do\n- x\n",
		"progress.md": "---\ntype: chart\nfrom: 20-plan, 10-work\n---\n",
	})
	if _, err := a.Derive(); err != nil {
		t.Fatal(err)
	}
	if got := read(t, dir, "progress.md"); !strings.HasSuffix(got, "---\nToken endpoint: 50\nwork: 1\n") {
		t.Errorf("a source is named by its title, or its file name without the order number: %q", got)
	}
}
