package api

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// A board without a language of its own follows the environment's; the
	// tests pin it, so they do not depend on the machine's.
	os.Setenv("STICKYPANE_LANG", "en")
	os.Exit(m.Run())
}

// korean is an API over a board whose language is Korean.
func korean(t *testing.T) (*API, string) {
	t.Helper()
	a, dir := newAPI(t, nil)
	if err := a.Store().SetLanguage("ko"); err != nil {
		t.Fatal(err)
	}
	return a, dir
}

// A board made on a Korean board starts with the columns in Korean, and a
// card without a column goes to the first of them.
func TestANewBoardStartsInTheBoardsLanguage(t *testing.T) {
	a, dir := korean(t)
	if _, err := a.Card("work", "add", "x", ""); err != nil {
		t.Fatal(err)
	}
	want := "---\ntype: board\ntitle: work\nopen: true\n---\n## 할 일\n- x\n\n## 하는 중\n\n## 끝\n"
	if got := read(t, dir, "work.md"); got != want {
		t.Errorf("file = %q, want %q", got, want)
	}
	body, _, err := a.Make("board", "Sprint", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := plain(string(body)), "---\ntype: board\ntitle: Sprint\n---\n## 할 일\n\n## 하는 중\n\n## 끝\n"; got != want {
		t.Errorf("Make = %q, want %q", got, want)
	}
}

// A card put into one of the usual columns of a board not made yet goes
// there; one put into a column of its own starts the board with that
// column alone, with no empty columns beside it.
func TestACardStartsTheBoardItsColumnNames(t *testing.T) {
	a, dir := korean(t)
	if _, err := a.Card("work", "add", "x", "하는 중"); err != nil {
		t.Fatal(err)
	}
	if got, want := read(t, dir, "work.md"), "---\ntype: board\ntitle: work\nopen: true\n---\n## 할 일\n\n## 하는 중\n- x\n\n## 끝\n"; got != want {
		t.Errorf("a usual column: file = %q, want %q", got, want)
	}
	if _, err := a.Card("진행", "add", "x", "진행 중"); err != nil {
		t.Fatal(err)
	}
	if got, want := read(t, dir, "진행.md"), "---\ntype: board\ntitle: 진행\nopen: true\n---\n## 진행 중\n- x\n"; got != want {
		t.Errorf("a column of its own: file = %q, want %q", got, want)
	}

	// An agent that names the columns in English gets them in English, so
	// its next move to "Done" finds its column.
	if _, err := a.Card("agent", "add", "x", "Doing"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Card("agent", "move", "x", "Done"); err != nil {
		t.Fatal(err)
	}
	if got, want := read(t, dir, "agent.md"), "---\ntype: board\ntitle: agent\nopen: true\n---\n## To do\n\n## Doing\n\n## Done\n- x\n"; got != want {
		t.Errorf("English columns on a Korean board: file = %q, want %q", got, want)
	}

	en, dir := newAPI(t, nil)
	if _, err := en.Card("work", "add", "x", "Backlog"); err != nil {
		t.Fatal(err)
	}
	if got, want := read(t, dir, "work.md"), "---\ntype: board\ntitle: work\nopen: true\n---\n## Backlog\n- x\n"; got != want {
		t.Errorf("English, a column of its own: file = %q, want %q", got, want)
	}
	if _, err := en.Card("work", "add", "y", "Doing"); err != nil {
		t.Fatal(err)
	}
	if got, want := read(t, dir, "work.md"), "---\ntype: board\ntitle: work\nopen: true\n---\n## Backlog\n- x\n## Doing\n- y\n"; got != want {
		t.Errorf("a board that is there gets the column at the end: file = %q, want %q", got, want)
	}
}
