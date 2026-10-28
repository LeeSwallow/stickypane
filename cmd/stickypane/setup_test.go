package main

import (
	"github.com/LeeSwallow/stickypane/internal/initcmd"
	"path/filepath"

	"errors"
	"github.com/LeeSwallow/stickypane/internal/store"
	"os"
	"slices"
	"strings"
	"testing"
)

// setup prints its plan and runs nothing until told to: --yes runs it, and
// --dry-run only prints it. The scope, MCP and agents are flags.
func TestSetupAsksBeforeItRuns(t *testing.T) {
	project(t)
	var ran []string
	oldRun, oldCheck, oldLook, oldHas, oldMCP := setupRun, setupCheck, setupLook, setupHas, setupMCP
	t.Cleanup(func() {
		setupRun, setupCheck, setupLook, setupHas, setupMCP = oldRun, oldCheck, oldLook, oldHas, oldMCP
	})
	setupMCP = func(string) (initcmd.MCPServer, bool) { return initcmd.MCPServer{}, false }
	setupRun = func(cmd []string) error { ran = append(ran, strings.Join(cmd, " ")); return nil }
	setupCheck = func([]string) error { return errors.New("not registered") }
	setupLook = func(name string) (string, error) {
		if name == "claude" {
			return "/bin/claude", nil
		}
		return "", errors.New("missing")
	}
	setupHas = func(string) func() (bool, bool) { return func() (bool, bool) { return false, false } }

	code, out, _ := exec(t, "setup", "--scope", "project", "--mcp")
	if code != 0 || len(ran) != 0 || !strings.Contains(out, "claude plugin install board@stickypane --scope project") || !strings.Contains(out, "--yes") {
		t.Fatalf("without --yes nothing runs: code %d, ran %q\n%s", code, ran, out)
	}
	if code, _, _ := exec(t, "setup", "--dry-run", "--yes"); code != 0 || len(ran) != 0 {
		t.Errorf("--dry-run runs nothing: %q", ran)
	}
	code, out, errOut := exec(t, "setup", "--scope", "project", "--mcp", "--yes")
	if code != 0 || len(ran) != 3 || ran[2] != "claude mcp add --scope project stickypane -- stickypane mcp" {
		t.Fatalf("--yes runs the plan: code %d, ran %q, out %q, err %q", code, ran, out, errOut)
	}
	if code, _, errOut := exec(t, "setup", "--scope", "everywhere"); code != exitUsage || !strings.Contains(errOut, "user, project or local") {
		t.Errorf("a bad scope is a usage error: %d %q", code, errOut)
	}
	if code, _, _ := exec(t, "setup", "--agents", "gemini"); code != exitUsage {
		t.Errorf("an unknown agent is a usage error: %d", code)
	}
}

// stubSetup replaces the agents' commands: found are the agents on the
// PATH, and the commands run are collected in the returned slice.
func stubSetup(t *testing.T, found ...string) *[]string {
	t.Helper()
	var ran []string
	oldRun, oldCheck, oldLook, oldHas, oldMCP := setupRun, setupCheck, setupLook, setupHas, setupMCP
	t.Cleanup(func() {
		setupRun, setupCheck, setupLook, setupHas, setupMCP = oldRun, oldCheck, oldLook, oldHas, oldMCP
	})
	setupMCP = func(string) (initcmd.MCPServer, bool) { return initcmd.MCPServer{}, false }
	setupRun = func(cmd []string) error { ran = append(ran, strings.Join(cmd, " ")); return nil }
	setupCheck = func([]string) error { return errors.New("not registered") }
	setupLook = func(name string) (string, error) {
		if slices.Contains(found, name) {
			return "/bin/" + name, nil
		}
		return "", errors.New("missing")
	}
	setupHas = func(string) func() (bool, bool) { return func() (bool, bool) { return false, false } }
	return &ran
}

// setup installs the agents' integration and nothing else: run from a
// folder that is not a project (a desktop app runs it from / or $HOME), it
// makes no board and writes no AGENTS.md there, and says how to make one.
func TestSetupMakesNoBoard(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	ran := stubSetup(t, "claude")
	code, out, errOut := exec(t, "setup", "--yes")
	if code != 0 || len(*ran) == 0 {
		t.Fatalf("setup --yes: code %d, ran %q, err %q", code, *ran, errOut)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("setup wrote into the current folder: %v", entries)
	}
	if !strings.Contains(out, "stickypane init") {
		t.Errorf("setup should say how to make a board:\n%s", out)
	}
}

// With no agent on the PATH nothing is set up, and setup says so rather
// than that every agent found is set up.
func TestSetupSaysWhenNoAgentIsFound(t *testing.T) {
	t.Chdir(t.TempDir())
	stubSetup(t)
	for _, args := range [][]string{{"setup", "--yes"}, {"setup", "--undo", "--yes"}, {"setup", "--agents", "codex", "--yes"}} {
		code, out, _ := exec(t, args...)
		if code != 0 || strings.Contains(out, "every agent found") || !strings.Contains(out, "No agent found") {
			t.Errorf("%q: code %d\n%s", args, code, out)
		}
	}
	stubSetup(t, "claude")
	setupHas = func(string) func() (bool, bool) { return func() (bool, bool) { return true, true } }
	if code, out, _ := exec(t, "setup", "--yes"); code != 0 || !strings.Contains(out, "Nothing to do") {
		t.Errorf("all set up: code %d\n%s", code, out)
	}
}

// --command puts a full path in the MCP server's registration; a relative
// one is a usage error.
func TestSetupCommandNamesTheMCPServersProgram(t *testing.T) {
	t.Chdir(t.TempDir())
	ran := stubSetup(t, "claude")
	setupHas = func(string) func() (bool, bool) { return func() (bool, bool) { return true, true } }
	code, _, errOut := exec(t, "setup", "--mcp", "--command", "/opt/app/stickypane", "--yes")
	if code != 0 || len(*ran) != 1 || (*ran)[0] != "claude mcp add --scope user stickypane -- /opt/app/stickypane mcp" {
		t.Fatalf("code %d, ran %q, err %q", code, *ran, errOut)
	}
	setupMCP = func(string) (initcmd.MCPServer, bool) {
		return initcmd.MCPServer{Command: "stickypane", Scope: "user"}, true
	}
	*ran = nil
	code, out, _ := exec(t, "setup", "--mcp", "--command", "/opt/app/stickypane", "--dry-run")
	if code != 0 || len(*ran) != 0 || !strings.Contains(out, "claude mcp remove --scope user stickypane") {
		t.Errorf("an old registration is replaced, in the plan first: %d %q\n%s", code, *ran, out)
	}
	if code, _, errOut := exec(t, "setup", "--mcp", "--command", "stickypane"); code != exitUsage || !strings.Contains(errOut, "full path") {
		t.Errorf("a relative --command: %d %q", code, errOut)
	}
}

// init refuses the home folder, as the desktop app does: a board there
// would be found from every project below it.
func TestInitRefusesTheHomeFolder(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Chdir(home)
	code, _, errOut := exec(t, "init")
	if code == 0 || !strings.Contains(errOut, "home folder") {
		t.Errorf("init in home: code %d, %q", code, errOut)
	}
	if entries, _ := os.ReadDir(home); len(entries) != 0 {
		t.Errorf("init wrote into home: %v", entries)
	}
}

// init --view and --style write how the board shows to sticky.json; "app"
// is the desktop app.
func TestInitSetsTheViewAndStyle(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if code, _, errOut := exec(t, "init", "--no-agent-docs", "--view", "app", "--style", "neat"); code != 0 {
		t.Fatalf("init: %s", errOut)
	}
	st := store.Open(filepath.Join(dir, ".sticky"))
	if st.ViewMode() != store.ViewOverlay || st.Style() != store.StyleNeat {
		t.Errorf("view %q style %q", st.ViewMode(), st.Style())
	}
	if code, _, _ := exec(t, "init", "--view", "both"); code != 0 || store.Open(filepath.Join(dir, ".sticky")).ViewMode() != store.ViewBoth {
		t.Errorf("init again with --view both: %d", code)
	}
	for _, args := range [][]string{{"init", "--view", "window"}, {"init", "--style", "fancy"}} {
		if code, _, errOut := exec(t, args...); code != exitUsage {
			t.Errorf("%q: %d %q", args, code, errOut)
		}
	}
}

// init --language is the board's language: it writes it to sticky.json,
// and the welcome note is written in it.
func TestInitSetsTheLanguageAndWritesTheWelcomeInIt(t *testing.T) {
	t.Setenv("STICKYPANE_LANG", "en")
	dir := t.TempDir()
	t.Chdir(dir)
	if code, _, errOut := exec(t, "init", "--no-agent-docs", "--language", "ko"); code != 0 {
		t.Fatalf("init: %s", errOut)
	}
	if got := store.Open(filepath.Join(dir, ".sticky")).Settings().Language(); got != "ko" {
		t.Errorf("the board's language = %q, want ko", got)
	}
	b, err := os.ReadFile(filepath.Join(dir, ".sticky", "welcome.md"))
	if err != nil || !strings.Contains(string(b), "환영") {
		t.Errorf("the welcome note should be Korean: %q %v", b, err)
	}
	if code, _, errOut := exec(t, "init", "--language", "xx"); code != exitUsage || !strings.Contains(errOut, "ko") {
		t.Errorf("--language xx: %d %q", code, errOut)
	}
}
