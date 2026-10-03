package app

import (
	"strings"
	"testing"
)

// twoTabs makes a root tab with one note and a "deploy" tab with two.
func twoTabs(t *testing.T) (*Model, string) {
	t.Helper()
	m, dir := newModel(t, map[string]string{"plan.md": opened("the plan\n")})
	mkdir(t, dir, "deploy")
	writeFile(t, dir, "deploy/build.log", "compiled\n")
	writeFile(t, dir, "deploy/notes.md", "deploy notes\n")
	mkdir(t, dir, "deploy/docs")
	writeFile(t, dir, "deploy/docs/01.md", "page one\n")
	writeFile(t, dir, "deploy/sticky.json", `{"notes":{"build.log":{"open":true},"docs":{"open":true}}}`)
	press(m, "r")
	return m, dir
}

func TestFoldersAreTabsAndOneIsActive(t *testing.T) {
	m, _ := twoTabs(t)
	s := screen(m)
	top := strings.Split(s, "\n")[0]
	if !strings.Contains(top, "deploy") || !strings.Contains(top, "1") || !strings.Contains(top, "2") {
		t.Fatalf("the first line lists the tabs with their numbers:\n%s", s)
	}
	if !strings.Contains(s, "the plan") || strings.Contains(s, "compiled") {
		t.Fatalf("the root tab is active first: its notes show, the other tab's do not:\n%s", s)
	}
	press(m, "2")
	s = screen(m)
	if strings.Contains(s, "the plan") || !strings.Contains(s, "compiled") || !strings.Contains(s, "page one") {
		t.Fatalf("2 switches to the second tab, whose files are notes and whose folder is a book:\n%s", s)
	}
	if !strings.Contains(s, "≣ build") || !strings.Contains(s, "✎ notes") || !strings.Contains(s, "docs") {
		t.Errorf("the second line lists the active tab's notes:\n%s", s)
	}
	press(m, ")")
	if s := screen(m); !strings.Contains(s, "compiled") {
		t.Errorf(") stops at the last tab:\n%s", s)
	}
	press(m, "(")
	if s := screen(m); !strings.Contains(s, "the plan") {
		t.Errorf("( goes back to the first tab:\n%s", s)
	}
}

func TestTheActiveTabIsRememberedAndFollowsStickyJSON(t *testing.T) {
	m, dir := twoTabs(t)
	press(m, "2")
	if got := readFile(t, dir, "sticky.json"); !strings.Contains(got, `"tab": "deploy"`) {
		t.Fatalf("the active tab goes to the root sticky.json:\n%s", got)
	}
	// An agent's `stickypane show deploy/notes` writes the tab; the board follows.
	press(m, "1")
	writeView(t, dir, `{"notes":{"plan.md":{"open":true}},"tab":"deploy"}`)
	m.Update(changedMsg{})
	if s := screen(m); !strings.Contains(s, "compiled") {
		t.Errorf("a tab chosen from outside is shown:\n%s", s)
	}
}

func TestEachTabKeepsItsOwnFocusAndArrangement(t *testing.T) {
	m, dir := twoTabs(t)
	press(m, "2", "tab", "o") // close the second note of tab 2
	if m.focus != "deploy/docs" && m.focus != "deploy/notes.md" {
		t.Fatalf("focus = %q", m.focus)
	}
	closed := m.focus
	if got := readFile(t, dir, "deploy/sticky.json"); !strings.Contains(got, `"open": false`) || strings.Contains(got, "plan.md") {
		t.Errorf("the tab's arrangement is written to the tab's own file:\n%s", got)
	}
	press(m, "1")
	if m.focus != "plan.md" {
		t.Errorf("each tab has its own focus, got %q", m.focus)
	}
	press(m, "2")
	if m.focus != closed {
		t.Errorf("coming back finds the focus where it was, got %q", m.focus)
	}
}

func TestATabCanBeNamed(t *testing.T) {
	m, dir := twoTabs(t)
	writeFile(t, dir, "deploy/sticky.json", `{"title":"배포","notes":{"build.log":{"open":true}}}`)
	press(m, "r")
	if top := strings.Split(screen(m), "\n")[0]; !strings.Contains(top, "배포") || strings.Contains(top, "deploy") {
		t.Errorf("a tab goes by the title its file gives it: %q", top)
	}
}

func TestClickingATabSwitchesToIt(t *testing.T) {
	m, _ := twoTabs(t)
	clickOn(t, m, "deploy")
	if s := screen(m); !strings.Contains(s, "compiled") {
		t.Errorf("a click on a tab switches to it:\n%s", s)
	}
}

func TestMoveOffersTabsAndBooks(t *testing.T) {
	m, dir := twoTabs(t)
	press(m, "m")
	if s := screen(m); !strings.Contains(s, "deploy") || strings.Contains(s, "top level") {
		t.Fatalf("from the root, the other tabs are offered:\n%s", s)
	}
	press(m, "enter")
	if !fileExists(dir, "deploy/plan.md") {
		t.Fatalf("the note should have moved into the tab")
	}
	if s := screen(m); !strings.Contains(s, "the plan") || !strings.Contains(s, "compiled") {
		t.Fatalf("the board follows the note into its tab:\n%s", s)
	}
	press(m, "m")
	if s := screen(m); !strings.Contains(s, "docs") || !strings.Contains(s, "top level") {
		t.Errorf("in a tab, its books and the top level are offered:\n%s", s)
	}
	press(m, "esc")
}
