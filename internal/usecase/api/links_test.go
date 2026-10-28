package api

import "testing"

// Notes link to each other with [[name]]; the index says what each note
// links to and what links to it, by file.
func TestTheIndexKnowsLinksAndBacklinks(t *testing.T) {
	a, _ := newAPI(t, map[string]string{
		"plan.md":   "---\ntype: checklist\n---\n- [ ] write tests, see [[why]]\n- [ ] ship ([[deploy.sh]])\n",
		"why.md":    "Because [[plan#write tests]] and [[nowhere]].\n",
		"deploy.sh": "echo\n",
	})
	idx, err := a.Index()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]Entry{}
	for _, e := range idx {
		got[e.Name] = e
	}
	if l := got["plan.md"].Links; len(l) != 2 || l[0] != "why.md" || l[1] != "deploy.sh" {
		t.Errorf("plan links = %q", l)
	}
	if b := got["plan.md"].Backlinks; len(b) != 1 || b[0] != "why.md" {
		t.Errorf("plan backlinks = %q", b)
	}
	if l := got["why.md"].Links; len(l) != 1 || l[0] != "plan.md" {
		t.Errorf("a link to nothing is left out: %q", l)
	}
}
