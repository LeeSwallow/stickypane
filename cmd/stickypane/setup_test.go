package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/adapter/store"
	"github.com/LeeSwallow/stickypane/internal/usecase/initcmd"
)

// fakeAgents is the machine stubSetup makes: the agents found, and what
// each has.
type fakeAgents struct{ st initcmd.Status }

func (f *fakeAgents) agent(name string) *initcmd.AgentStatus {
	for i := range f.st.Agents {
		if f.st.Agents[i].Agent == name {
			return &f.st.Agents[i]
		}
	}
	return nil
}

// stubSetup replaces the agents' commands: found are the agents on the
// PATH, with nothing installed, and the commands run are collected in the
// returned slice.
func stubSetup(t *testing.T, found ...string) (*[]string, *fakeAgents) {
	t.Helper()
	var ran []string
	oldRun, oldCheck, oldStatus := setupRun, setupCheck, setupStatus
	t.Cleanup(func() { setupRun, setupCheck, setupStatus = oldRun, oldCheck, oldStatus })
	f := &fakeAgents{st: initcmd.Status{Plugin: "0.2.1"}}
	for _, a := range []string{"claude", "codex"} {
		scopes := []string{"user"}
		if a == "claude" {
			scopes = []string{"user", "project", "local"}
		}
		s := initcmd.AgentStatus{Agent: a, State: initcmd.StateMissing, Scopes: scopes, Plugin: map[string]initcmd.PluginStatus{}, MCP: map[string]initcmd.MCPStatus{}}
		for _, g := range found {
			if g == a {
				s.Found, s.Path, s.State = true, "/bin/"+a, initcmd.StateOff
			}
		}
		f.st.Agents = append(f.st.Agents, s)
	}
	setupRun = func(cmd []string) error { ran = append(ran, strings.Join(cmd, " ")); return nil }
	setupCheck = func([]string) error { return errors.New("not registered") }
	setupStatus = func(_ string, agents []string) initcmd.Status {
		out := initcmd.Status{Plugin: f.st.Plugin}
		for _, a := range f.st.Agents {
			for _, w := range agents {
				if a.Agent == w {
					out.Agents = append(out.Agents, a)
				}
			}
		}
		return out
	}
	return &ran, f
}

// setup prints its plan and runs nothing until told to: --yes runs it, and
// --dry-run only prints it. The scope, MCP and agents are flags.
func TestSetupAsksBeforeItRuns(t *testing.T) {
	project(t)
	ran, _ := stubSetup(t, "claude")
	code, out, _ := exec(t, "setup", "--scope", "project", "--mcp")
	if code != 0 || len(*ran) != 0 || !strings.Contains(out, "claude plugin install board@stickypane --scope project") || !strings.Contains(out, "--yes") {
		t.Fatalf("without --yes nothing runs: code %d, ran %q\n%s", code, *ran, out)
	}
	if code, _, _ := exec(t, "setup", "--dry-run", "--yes"); code != 0 || len(*ran) != 0 {
		t.Errorf("--dry-run runs nothing: %q", *ran)
	}
	code, out, errOut := exec(t, "setup", "--scope", "project", "--mcp", "--yes")
	if code != 0 || len(*ran) != 3 || (*ran)[2] != "claude mcp add --scope project stickypane -- stickypane mcp" {
		t.Fatalf("--yes runs the plan: code %d, ran %q, out %q, err %q", code, *ran, out, errOut)
	}
	for _, args := range [][]string{
		{"setup", "--scope", "everywhere"},
		{"setup", "--agents", "gemini"},
		{"setup", "--json"},
		{"setup", "--plugin=false"},
	} {
		if code, _, _ := exec(t, args...); code != exitUsage {
			t.Errorf("%q is a usage error: %d", args, code)
		}
	}
}

// setup installs the agents' integration and nothing else: run from a
// folder that is not a project (a desktop app runs it from / or $HOME), it
// makes no board and writes no AGENTS.md there, and says how to make one.
func TestSetupMakesNoBoard(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	ran, _ := stubSetup(t, "claude")
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
	_, f := stubSetup(t, "claude")
	f.agent("claude").Plugin["user"] = initcmd.PluginStatus{Version: "0.2.1", State: initcmd.StateOK}
	if code, out, _ := exec(t, "setup", "--yes"); code != 0 || !strings.Contains(out, "Nothing to do") {
		t.Errorf("all set up: code %d\n%s", code, out)
	}
}

// --command puts a full path in the MCP server's registration; a relative
// one is a usage error. --plugin=false leaves the plugin.
func TestSetupCommandNamesTheMCPServersProgram(t *testing.T) {
	t.Chdir(t.TempDir())
	ran, f := stubSetup(t, "claude")
	code, _, errOut := exec(t, "setup", "--mcp", "--plugin=false", "--command", "/opt/app/stickypane", "--yes")
	if code != 0 || len(*ran) != 1 || (*ran)[0] != "claude mcp add --scope user stickypane -- /opt/app/stickypane mcp" {
		t.Fatalf("code %d, ran %q, err %q", code, *ran, errOut)
	}
	f.agent("claude").MCP["user"] = initcmd.MCPStatus{Command: "stickypane"}
	*ran = nil
	code, out, _ := exec(t, "setup", "--mcp", "--plugin=false", "--command", "/opt/app/stickypane", "--dry-run")
	if code != 0 || len(*ran) != 0 || !strings.Contains(out, "claude mcp remove --scope user stickypane") {
		t.Errorf("an old registration is replaced, in the plan first: %d %q\n%s", code, *ran, out)
	}
	if code, _, errOut := exec(t, "setup", "--mcp", "--command", "stickypane"); code != exitUsage || !strings.Contains(errOut, "full path") {
		t.Errorf("a relative --command: %d %q", code, errOut)
	}
}

// --status says where each agent stands and runs nothing; --json says it
// for the desktop app.
func TestSetupStatus(t *testing.T) {
	t.Chdir(t.TempDir())
	ran, f := stubSetup(t, "claude")
	c := f.agent("claude")
	c.State = initcmd.StateBroken
	c.Plugin["user"] = initcmd.PluginStatus{Version: "0.2.1", State: initcmd.StateBroken, Error: "Marketplace stickypane failed to load: cache-miss"}
	c.Marketplace = &initcmd.Marketplace{Source: "directory", Where: "/Users/min/Projects/stickypane", Gone: true}
	c.MCP["project"] = initcmd.MCPStatus{Command: "/usr/local/bin/stickypane"}
	code, out, _ := exec(t, "setup", "--status")
	for _, want := range []string{"Claude Code: broken", "user 0.2.1 (broken)", "project /usr/local/bin/stickypane", "the folder is gone", "cache-miss", "Codex: not found", "stickypane setup --agents claude"} {
		if !strings.Contains(out, want) {
			t.Errorf("--status should say %q:\n%s", want, out)
		}
	}
	if code != 0 || len(*ran) != 0 {
		t.Errorf("--status runs nothing: %d %q", code, *ran)
	}
	code, out, _ = exec(t, "setup", "--status", "--json")
	var st initcmd.Status
	if code != 0 || json.Unmarshal([]byte(out), &st) != nil || len(st.Agents) != 2 || st.Agents[0].State != "broken" || !st.Agents[0].Marketplace.Gone || st.Agents[1].Found {
		t.Errorf("--status --json: %d\n%s", code, out)
	}
	if !strings.Contains(out, `"scopes"`) || !strings.Contains(out, `"plugin": "0.2.1"`) {
		t.Errorf("--status --json names the scopes and the plugin's version:\n%s", out)
	}
	// and setup repairs it
	*ran = nil
	if code, _, _ := exec(t, "setup", "--yes"); code != 0 || strings.Join(*ran, "\n") != "claude plugin marketplace remove stickypane\nclaude plugin marketplace add LeeSwallow/stickypane\nclaude plugin install board@stickypane --scope user" {
		t.Errorf("repair: %d\n%s", code, strings.Join(*ran, "\n"))
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
