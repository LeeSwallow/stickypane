package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/registry"
)

// appHome makes the folder the app would make and points the registry at it.
func appHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("STICKYPANE_HOME", home)
	return home
}

// recorded reads every event written to home.
func recorded(t *testing.T, home string) []registry.Event {
	t.Helper()
	files, _ := filepath.Glob(filepath.Join(home, "events", "*.jsonl"))
	var out []registry.Event
	for _, f := range files {
		fh, err := os.Open(f)
		if err != nil {
			t.Fatal(err)
		}
		sc := bufio.NewScanner(fh)
		for sc.Scan() {
			var e registry.Event
			if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
				t.Fatalf("bad line %q: %v", sc.Text(), err)
			}
			out = append(out, e)
		}
		fh.Close()
	}
	return out
}

// same says whether two paths are the one place, whatever symbolic links
// lie between.
func same(a, b string) bool {
	ra, _ := filepath.EvalSymlinks(a)
	rb, _ := filepath.EvalSymlinks(b)
	return ra != "" && ra == rb
}

func TestInitRecordsTheProject(t *testing.T) {
	home := appHome(t)
	root := t.TempDir()
	t.Chdir(root)
	if code, _, errOut := exec(t, "init", "--no-agent-docs"); code != 0 {
		t.Fatalf("init failed: %s", errOut)
	}
	got := recorded(t, home)
	if len(got) != 1 || got[0].Type != "project.init" || !same(got[0].Project, root) {
		t.Errorf("want one project.init for %s: %+v", root, got)
	}
}

func TestEditTouchesTheProject(t *testing.T) {
	root := project(t)
	home := appHome(t)
	if code, _, errOut := exec(t, "todo", "plan", "add", "write tests"); code != 0 {
		t.Fatalf("todo failed: %s", errOut)
	}
	got := recorded(t, home)
	if len(got) != 1 || got[0].Type != "project.touch" || got[0].Cmd != "todo" || !same(got[0].Project, root) {
		t.Errorf("want one project.touch by todo for %s: %+v", root, got)
	}
	// A read changes nothing, so it records nothing.
	exec(t, "list")
	if n := len(recorded(t, home)); n != 1 {
		t.Errorf("list should record nothing, events = %d", n)
	}
}

func TestMCPIsASession(t *testing.T) {
	root := project(t)
	home := appHome(t)
	if code, _, errOut := exec(t, "mcp"); code != 0 {
		t.Fatalf("mcp failed: %s", errOut)
	}
	got := recorded(t, home)
	if len(got) != 2 || got[0].Type != "session.start" || got[0].Kind != "mcp" ||
		got[1].Type != "session.end" || !same(got[0].Project, root) {
		t.Errorf("want session.start and session.end of mcp: %+v", got)
	}
	if left, _ := filepath.Glob(filepath.Join(home, "run", "*.json")); len(left) != 0 {
		t.Errorf("the run file should be gone: %v", left)
	}
}

func TestBoardIsASession(t *testing.T) {
	root := project(t)
	home := appHome(t)
	fakeBoard(t)
	if code, _, errOut := exec(t); code != 0 {
		t.Fatalf("board failed: %s", errOut)
	}
	got := recorded(t, home)
	if len(got) != 2 || got[0].Kind != "tui" || got[1].Type != "session.end" || !same(got[0].Project, root) {
		t.Errorf("want session.start (tui) and session.end: %+v", got)
	}
}

func TestCommandsSucceedWhenTheHomeIsReadOnly(t *testing.T) {
	home := appHome(t)
	if err := os.Chmod(home, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(home, 0o755) })
	root := t.TempDir()
	t.Chdir(root)
	for _, args := range [][]string{
		{"init", "--no-agent-docs"},
		{"todo", "plan", "add", "write tests"},
		{"card", "work", "add", "one"},
	} {
		if code, _, errOut := exec(t, args...); code != 0 {
			t.Errorf("%v: code = %d, stderr = %q", args, code, errOut)
		}
	}
}

// TestEveryCommandIsSortedIntoChanges makes the commands that only read
// say so; any other command is taken to change a board and must be in
// changes, so a new command cannot be forgotten.
func TestEveryCommandIsSortedIntoChanges(t *testing.T) {
	reads := map[string]bool{
		"init": true, "setup": true, "theme": true, "language": true,
		"config": true, "kinds": true, "env": true, "guide": true,
		"version": true, "help": true, "list": true, "index": true,
		"cat": true, "answers": true, "wait": true, "api": true,
		"watch": true, "mcp": true, "bridge": true,
	}
	for _, c := range commands {
		name := c.names[0]
		if reads[name] == changes[name] {
			t.Errorf("%s: it must be in exactly one of the read-only list and changes", name)
		}
	}
	for name := range changes {
		if _, ok := lookup(name); !ok {
			t.Errorf("changes names %s, which is no command", name)
		}
	}
}
