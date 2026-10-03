package app

import (
	"strings"
	"testing"
)

// i (or /) opens the index: every note of every tab in a line. Typing
// filters it; enter goes to the note, switching tab and opening it.
func TestTheIndexFindsAndOpensANote(t *testing.T) {
	m, dir := newModel(t, map[string]string{
		"plan.md":  "---\ntype: checklist\ntitle: Plan\n---\n- [ ] write tests\n",
		"notes.md": "just text\n",
	})
	mkdir(t, dir, "deploy")
	writeFile(t, dir, "deploy/run.md", "---\ntitle: Release steps\n---\nship it\n")
	m.Update(changedMsg{})
	press(m, "/")
	s := screen(m)
	if m.mode != modeIndex {
		t.Fatalf("/ opens the index:\n%s", s)
	}
	for _, want := range []string{"Plan", "write tests", "Release steps", "deploy"} {
		if !strings.Contains(s, want) {
			t.Errorf("the index shows %q:\n%s", want, s)
		}
	}
	typeText(m, "release")
	if s := screen(m); strings.Contains(s, "write tests") || !strings.Contains(s, "Release steps") {
		t.Errorf("typing filters the index:\n%s", s)
	}
	press(m, "enter")
	if m.mode != modeBoard || m.tabName != "deploy" || m.focus != "deploy/run.md" {
		t.Errorf("enter goes to the note: mode %v, tab %q, focus %q", m.mode, m.tabName, m.focus)
	}
	if i := m.index("deploy/run.md"); i < 0 || !m.isOpen(m.items[i]) {
		t.Error("and opens it")
	}
}

// f on a note opens the index with only the notes it links to and the
// notes that link to it; enter goes there.
func TestFFollowsALink(t *testing.T) {
	m, _ := newModel(t, map[string]string{
		"plan.md":  opened("see [[why]]\n"),
		"why.md":   "---\ntitle: Why\n---\nbecause\n",
		"other.md": "---\ntitle: Other\n---\nunrelated\n",
	})
	m.setFocus("plan.md")
	press(m, "f")
	s := screen(m)
	if m.mode != modeIndex || !strings.Contains(s, "Why") || strings.Contains(s, "Other") {
		t.Fatalf("f lists the links of the focused note:\n%s", s)
	}
	press(m, "enter")
	if m.focus != "why.md" {
		t.Errorf("enter follows the link: focus %q", m.focus)
	}
}
