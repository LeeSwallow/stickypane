package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

const failing = "#!/bin/sh\npwd > where.txt\necho hello from the script\necho a warning >&2\nexit 3\n"

// finish runs the command an update returned, as Bubble Tea would, and
// hands its message back to the model.
func finish(t *testing.T, m *Model, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		t.Fatal("the update should have started something")
	}
	m.Update(cmd())
}

func TestAScriptRunsOnlyAfterTheUserSaysYes(t *testing.T) {
	m, dir := newModel(t, map[string]string{"deploy.sh": failing})
	press(m, "o", "enter")
	if s := screen(m); m.mode != modeConfirm || !strings.Contains(s, "Run deploy.sh") || !strings.Contains(s, "(y/n)") {
		t.Fatalf("enter on a script should ask first:\n%s", s)
	}
	if _, cmd := m.Update(key("n")); cmd != nil || fileExists(dir, "deploy.log") {
		t.Fatal("answering n must not run anything")
	}
	press(m, "enter")
	_, cmd := m.Update(key("y"))
	if s := screen(m); !strings.Contains(s, "running") {
		t.Errorf("the script should be shown as running:\n%s", s)
	}
	press(m, "enter")
	if s := screen(m); m.mode == modeConfirm || !strings.Contains(s, "already running") {
		t.Errorf("a script that is running is not started again:\n%s", s)
	}
	finish(t, m, cmd)

	log := readFile(t, dir, "deploy.log")
	for _, want := range []string{"$ sh deploy.sh", "hello from the script", "a warning", "[exit 3"} {
		if !strings.Contains(log, want) {
			t.Errorf("the log should contain %q:\n%s", want, log)
		}
	}
	root, _ := filepath.EvalSymlinks(filepath.Dir(dir))
	where, err := os.ReadFile(filepath.Join(filepath.Dir(dir), "where.txt"))
	if got, _ := filepath.EvalSymlinks(strings.TrimSpace(string(where))); err != nil || got != root {
		t.Errorf("the script should run in the project folder %q: %q, %v", root, where, err)
	}
	s := screen(m)
	if !strings.Contains(s, "exit 3") || strings.Contains(s, "running") {
		t.Errorf("the bottom line should say how the run ended:\n%s", s)
	}
	if !strings.Contains(s, "hello from the script") || !strings.Contains(s, "≣ deploy") {
		t.Errorf("the output is a log, open next to the script:\n%s", s)
	}
}

func TestASecondRunStartsTheLogAgain(t *testing.T) {
	m, dir := newModel(t, map[string]string{"ok.sh": "echo run\n", "ok.log": "from an earlier run\n"})
	press(m, "tab", "o", "enter")
	_, cmd := m.Update(key("y"))
	finish(t, m, cmd)
	log := readFile(t, dir, "ok.log")
	if strings.Contains(log, "earlier") || !strings.Contains(log, "run\n") || !strings.Contains(log, "[exit 0") {
		t.Errorf("a run replaces the log of the run before:\n%s", log)
	}
}

func TestAScriptInABookLogsIntoThatBook(t *testing.T) {
	m, dir := newModel(t, nil)
	mkdir(t, dir, "ops")
	writeFile(t, dir, "ops/build.sh", "echo built\n")
	writeView(t, dir, `{"notes":{"ops":{"open":true}}}`)
	press(m, "r", "enter")
	_, cmd := m.Update(key("y"))
	finish(t, m, cmd)
	if got := readFile(t, dir, "ops/build.log"); !strings.Contains(got, "built") {
		t.Errorf("ops/build.log = %q", got)
	}
	if s := screen(m); !strings.Contains(s, "/2") {
		t.Errorf("the log is a new page of the same book:\n%s", s)
	}
}

func TestAddingAScriptMakesAShellFile(t *testing.T) {
	m, dir := newModel(t, nil)
	press(m, "a")
	for i := 0; i < len(m.reg) && m.reg[m.catalogIdx].Name != "script"; i++ {
		press(m, "j")
	}
	press(m, "enter")
	typeText(m, "Run tests")
	press(m, "enter")
	if got := readFile(t, dir, "run-tests.sh"); got != "#!/bin/sh\n" {
		t.Fatalf("a new script is a .sh file without front matter: %q", got)
	}
	if s := screen(m); !strings.Contains(s, "▶ run-tests") || !m.isOpen(m.items[0]) {
		t.Errorf("the new script should be open:\n%s", s)
	}
}
