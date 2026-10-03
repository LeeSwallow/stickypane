package store

import (
	"strings"
	"testing"
)

func tabNames(tabs []Tab) string {
	var out []string
	for _, t := range tabs {
		out = append(out, t.Name+"["+names(t.Notes)+"]")
	}
	return strings.Join(out, " ")
}

func TestTabsAreTheFoldersAndBooksTheFoldersInThem(t *testing.T) {
	s := newStore(t)
	write(t, s, "plan.md", "root note\n")
	write(t, s, "deploy/build.log", "x\n")
	write(t, s, "deploy/run.sh", "echo\n")
	write(t, s, "deploy/docs/01.md", "page\n")
	write(t, s, "deploy/docs/02.md", "page\n")
	write(t, s, "deploy/docs/deeper/x.md", "too deep\n")
	write(t, s, "notes/a.md", "a\n")
	write(t, s, "empty/.keep", "")
	write(t, s, "archive/old.md", "old\n")
	write(t, s, ".trash/gone.md", "gone\n")
	tabs, err := s.Tabs()
	if err != nil {
		t.Fatal(err)
	}
	want := "[plan.md] deploy[deploy/build.log,deploy/docs[deploy/docs/01.md deploy/docs/02.md],deploy/run.sh] notes[notes/a.md]"
	if got := tabNames(tabs); got != want {
		t.Errorf("Tabs = %s\nwant %s", got, want)
	}
	if tabs[0].Title == "" || tabs[1].Title != "deploy" {
		t.Errorf("a tab goes by its folder's name; the root tab by the project's: %q, %q", tabs[0].Title, tabs[1].Title)
	}
}

func TestATabHasItsOwnSettingsFile(t *testing.T) {
	s := newStore(t)
	write(t, s, "plan.md", "root\n")
	write(t, s, "deploy/build.log", "x\n")
	write(t, s, "deploy/docs/01.md", "page\n")
	if err := s.SetView("deploy/build.log", func(v *View) { v.Open = yes(); v.Rows = 12 }); err != nil {
		t.Fatal(err)
	}
	if err := s.SetView("deploy/docs", func(v *View) { v.Title = "Docs" }); err != nil {
		t.Fatal(err)
	}
	if err := s.SetView("plan.md", func(v *View) { v.Size = "half" }); err != nil {
		t.Fatal(err)
	}
	if err := s.SetTabTitle("deploy", "배포"); err != nil {
		t.Fatal(err)
	}
	tabFile := read(t, s, "deploy/"+ViewFile)
	if !strings.Contains(tabFile, `"build.log"`) || !strings.Contains(tabFile, `"docs"`) || !strings.Contains(tabFile, `"title": "배포"`) || strings.Contains(tabFile, "plan.md") {
		t.Errorf("the tab's file holds the tab's notes by their name inside the tab, and its title:\n%s", tabFile)
	}
	if root := read(t, s, ViewFile); !strings.Contains(root, `"plan.md"`) || strings.Contains(root, "build.log") {
		t.Errorf("the root file holds the root tab's notes:\n%s", root)
	}
	views, err := s.Views()
	if err != nil || views["deploy/build.log"].Rows != 12 || views["deploy/docs"].Title != "Docs" || views["plan.md"].Size != "half" {
		t.Errorf("Views reads every tab, keyed by the full name: %+v, %v", views, err)
	}
	tabs, _ := s.Tabs()
	if tabs[1].Title != "배포" {
		t.Errorf("a tab's title comes from its file: %+v", tabs[1])
	}
	if err := s.SetOrder("deploy", []string{"deploy/docs", "deploy/build.log"}); err != nil {
		t.Fatal(err)
	}
	if got := s.Order("deploy"); strings.Join(got, ",") != "deploy/docs,deploy/build.log" {
		t.Errorf("Order(deploy) = %v", got)
	}
	if err := s.SetTab("deploy"); err != nil || s.Tab() != "deploy" {
		t.Errorf("the active tab is kept in the root file: %v, %q", err, s.Tab())
	}
	if got := read(t, s, ViewFile); !strings.Contains(got, `"tab": "deploy"`) {
		t.Errorf("root file:\n%s", got)
	}
}

func TestWatchSeesChangesInsideABookInsideATab(t *testing.T) {
	s := newStore(t)
	write(t, s, "deploy/docs/01.md", "page\n")
	if len(s.watched()) < 2 {
		t.Errorf("a tab and the book in it are both watched: %v", s.watched())
	}
}

func TestATabIsTitledWithoutTheNumberThatOrdersIt(t *testing.T) {
	s := newStore(t)
	write(t, s, "20-release/build.log", "x\n")
	write(t, s, "2026-10/notes.md", "x\n")
	tabs, err := s.Tabs()
	if err != nil {
		t.Fatal(err)
	}
	var titles []string
	for _, tab := range tabs[1:] {
		titles = append(titles, tab.Name+"="+tab.Title)
	}
	if got := strings.Join(titles, " "); got != "20-release=release 2026-10=2026-10" {
		t.Errorf("tabs = %s", got)
	}
}

func TestBareTakesOffTheOrderNumberOnly(t *testing.T) {
	for in, want := range map[string]string{"10-plan": "plan", "05_notes": "notes", "3 docs": "docs", "plan": "plan", "2026-10-03": "2026-10-03", "42": "42"} {
		if got := Bare(in); got != want {
			t.Errorf("Bare(%q) = %q, want %q", in, got, want)
		}
	}
}
