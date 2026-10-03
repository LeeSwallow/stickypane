package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMain points the agents' home folders at empty places, so that the
// plugins installed on the machine running the tests do not change what
// init does.
func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "stickypane-home")
	if err != nil {
		panic(err)
	}
	os.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(tmp, "claude"))
	os.Setenv("CODEX_HOME", filepath.Join(tmp, "codex"))
	os.Unsetenv("TMUX")
	code := m.Run()
	os.RemoveAll(tmp)
	os.Exit(code)
}

// repo makes a git repository with no board and moves into a folder inside
// it. It returns the repository's top.
func repo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if real, err := filepath.EvalSymlinks(root); err == nil {
		root = real
	}
	sub := filepath.Join(root, "internal", "pkg")
	for _, dir := range []string{filepath.Join(root, ".git"), sub} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(sub)
	return root
}

// fakeBoard stands in for the terminal UI and records what it was given.
func fakeBoard(t *testing.T) *[]string {
	t.Helper()
	var calls []string
	old := runBoard
	runBoard = func(_ context.Context, dir, status string) error {
		calls = append(calls, dir, status)
		return nil
	}
	t.Cleanup(func() { runBoard = old })
	return &calls
}

func TestBoardInARepositoryWithoutOneSetsItUp(t *testing.T) {
	root := repo(t)
	calls := fakeBoard(t)
	code, out, errOut := exec(t)
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, errOut)
	}
	dir := filepath.Join(root, ".sticky")
	if len(*calls) != 2 || (*calls)[0] != dir {
		t.Fatalf("the board should open on %s: %q", dir, *calls)
	}
	if status := (*calls)[1]; !strings.Contains(status, ".sticky") {
		t.Errorf("the board should say it was just made: %q", status)
	}
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); err != nil {
		t.Error("the agent should be told about the board, as init does")
	}
	if !strings.Contains(out, "Made .sticky/") {
		t.Errorf("after the board closes, say what was set up: %q", out)
	}
}

func TestBoardThatExistsOpensWithoutSettingUp(t *testing.T) {
	root := repo(t)
	if err := os.Mkdir(filepath.Join(root, ".sticky"), 0o755); err != nil {
		t.Fatal(err)
	}
	calls := fakeBoard(t)
	code, out, _ := exec(t)
	if code != 0 || len(*calls) != 2 || (*calls)[1] != "" || out != "" {
		t.Errorf("code = %d, calls = %q, out = %q", code, *calls, out)
	}
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); err == nil {
		t.Error("an existing board is opened, not set up again")
	}
}

func TestAWriteInARepositoryWithoutABoardMakesOne(t *testing.T) {
	root := repo(t)
	code, out, errOut := exec(t, "todo", "plan", "add", "write tests")
	if code != 0 || out != "plan.md: 0/1\n" {
		t.Fatalf("code = %d, out = %q, stderr = %q", code, out, errOut)
	}
	if _, err := os.Stat(filepath.Join(root, ".sticky", "plan.md")); err != nil {
		t.Errorf("the board belongs at the top of the repository: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); err == nil {
		t.Error("an agent's command makes the board only; the agent already knows it")
	}
	if !strings.Contains(errOut, ".sticky") {
		t.Errorf("say where the board was made: %q", errOut)
	}
}

func TestAReadWithoutABoardDoesNotMakeOne(t *testing.T) {
	root := repo(t)
	if code, _, errOut := exec(t, "list"); code != 1 || !strings.Contains(errOut, "stickypane init") {
		t.Errorf("code = %d, stderr = %q", code, errOut)
	}
	if _, err := os.Stat(filepath.Join(root, ".sticky")); err == nil {
		t.Error("reading must not make a board")
	}
}

func TestAWriteOutsideARepositoryAsksForInit(t *testing.T) {
	t.Chdir(t.TempDir())
	code, _, errOut := exec(t, "show", "plan")
	if code != 1 || !strings.Contains(errOut, "stickypane init") {
		t.Errorf("code = %d, stderr = %q", code, errOut)
	}
}
