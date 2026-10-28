package initcmd

import (
	"errors"
	"strings"
	"testing"
)

func lookFor(have ...string) func(string) (string, error) {
	return func(name string) (string, error) {
		for _, h := range have {
			if h == name {
				return "/bin/" + name, nil
			}
		}
		return "", errors.New("not found")
	}
}

func cmds(steps []Step) []string {
	var out []string
	for _, s := range steps {
		out = append(out, strings.Join(s.Cmd, " "))
	}
	return out
}

// setup installs the plugin for every agent it finds, through the agents'
// own commands, in the scope the user chose; the MCP server only when asked.
func TestSetupPlansTheAgentsOwnCommands(t *testing.T) {
	got := cmds(PlanSetup(SetupOptions{Scope: "project", MCP: true}, lookFor("claude", "codex"), func() (bool, bool) { return false, false }))
	want := []string{
		"claude plugin marketplace add LeeSwallow/stickypane",
		"claude plugin install board@stickypane --scope project",
		"claude mcp add --scope project stickypane -- stickypane mcp",
		"codex plugin marketplace add LeeSwallow/stickypane",
		"codex plugin add board@stickypane",
		"codex mcp add stickypane -- stickypane mcp",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("plan =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestSetupSkipsWhatIsThereAndWhatIsMissing(t *testing.T) {
	// Codex is not installed; Claude Code has the plugin already.
	got := cmds(PlanSetup(SetupOptions{}, lookFor("claude"), func() (bool, bool) { return true, false }))
	if len(got) != 0 {
		t.Errorf("nothing to do, got %q", got)
	}
	// Only the agents asked for.
	got = cmds(PlanSetup(SetupOptions{Agents: []string{"codex"}}, lookFor("claude", "codex"), func() (bool, bool) { return false, false }))
	if len(got) != 2 || !strings.HasPrefix(got[0], "codex ") {
		t.Errorf("only codex: %q", got)
	}
	// The MCP step skips itself when the server is registered.
	steps := PlanSetup(SetupOptions{MCP: true}, lookFor("claude"), func() (bool, bool) { return true, false })
	if len(steps) != 1 || strings.Join(steps[0].Unless, " ") != "claude mcp get stickypane" {
		t.Errorf("mcp step = %+v", steps)
	}
}

// Command names the stickypane the MCP server runs, by its full path.
func TestSetupMCPRunsTheCommandGiven(t *testing.T) {
	const app = "/Applications/Stickypane.app/Contents/Helpers/stickypane"
	got := cmds(PlanSetup(SetupOptions{MCP: true, Command: app}, lookFor("claude", "codex"), func() (bool, bool) { return true, true }))
	want := []string{
		"claude mcp add --scope user stickypane -- " + app + " mcp",
		"codex mcp add stickypane -- " + app + " mcp",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("mcp with a command =\n%s", strings.Join(got, "\n"))
	}
}

func TestSetupUndoRemovesWhatItAdded(t *testing.T) {
	got := cmds(PlanSetup(SetupOptions{Undo: true, Scope: "user", MCP: true}, lookFor("claude", "codex"), func() (bool, bool) { return true, true }))
	want := []string{
		"claude plugin uninstall board@stickypane --scope user",
		"claude mcp remove stickypane",
		"codex plugin remove board@stickypane",
		"codex mcp remove stickypane",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("undo =\n%s", strings.Join(got, "\n"))
	}
	// Without --mcp, undo takes out the plugin only, as the README says.
	got = cmds(PlanSetup(SetupOptions{Undo: true}, lookFor("claude", "codex"), func() (bool, bool) { return true, true }))
	if strings.Join(got, "\n") != "claude plugin uninstall board@stickypane --scope user\ncodex plugin remove board@stickypane" {
		t.Errorf("undo without --mcp =\n%s", strings.Join(got, "\n"))
	}
}

func TestRunSetupRunsStepsAndStopsAtAFailure(t *testing.T) {
	var ran []string
	run := func(cmd []string) error {
		ran = append(ran, strings.Join(cmd, " "))
		if cmd[0] == "fail" {
			return errors.New("boom")
		}
		return nil
	}
	steps := []Step{{Cmd: []string{"ok", "1"}}, {Cmd: []string{"skip"}, Unless: []string{"ok", "check"}}, {Cmd: []string{"fail"}}, {Cmd: []string{"never"}}}
	var out strings.Builder
	err := RunSetup(steps, run, run, &out)
	if err == nil || strings.Join(ran, ",") != "ok 1,ok check,fail" {
		t.Errorf("ran %q, err %v", ran, err)
	}
	if !strings.Contains(out.String(), "skip") || !strings.Contains(out.String(), "already") {
		t.Errorf("out = %q", out.String())
	}
}

// An MCP server registered with another command (a bare stickypane from
// the PATH) is removed and added again with the full path asked for; one
// that has it already is left.
func TestSetupReplacesAnMCPServerThatRunsAnotherCommand(t *testing.T) {
	const app = "/Applications/Stickypane.app/Contents/Helpers/stickypane"
	reg := func(cmd string) func(string) (MCPServer, bool) {
		return func(agent string) (MCPServer, bool) {
			if agent == "claude" {
				return MCPServer{Command: cmd, Scope: "local"}, true
			}
			return MCPServer{Command: cmd}, true
		}
	}
	got := cmds(PlanSetup(SetupOptions{MCP: true, Command: app, Registered: reg("stickypane")}, lookFor("claude", "codex"), func() (bool, bool) { return true, true }))
	want := []string{
		"claude mcp remove --scope local stickypane",
		"claude mcp add --scope user stickypane -- " + app + " mcp",
		"codex mcp remove stickypane",
		"codex mcp add stickypane -- " + app + " mcp",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("replace =\n%s", strings.Join(got, "\n"))
	}
	steps := PlanSetup(SetupOptions{MCP: true, Command: app, Registered: reg(app)}, lookFor("claude"), func() (bool, bool) { return true, true })
	if len(steps) != 1 || len(steps[0].Unless) == 0 {
		t.Errorf("already the full path: %+v", steps)
	}
	// without --command nothing is replaced
	if got := cmds(PlanSetup(SetupOptions{MCP: true, Registered: reg("/old/stickypane")}, lookFor("claude"), func() (bool, bool) { return true, true })); len(got) != 1 {
		t.Errorf("no command: %q", got)
	}
}

func TestParseTheAgentsMCPRegistrations(t *testing.T) {
	out := "stickypane:\n  Scope: User config (available in all your projects)\n  Status: ✓ Connected\n  Type: stdio\n  Command: stickypane\n  Args: mcp\n"
	if m, ok := ParseClaudeMCP(out); !ok || m.Command != "stickypane" || m.Scope != "user" {
		t.Errorf("claude: %+v %v", m, ok)
	}
	if _, ok := ParseClaudeMCP("No MCP server found"); ok {
		t.Error("claude: none")
	}
	toml := "[mcp_servers.other]\ncommand = \"x\"\n\n[mcp_servers.stickypane]\ncommand = \"/usr/local/bin/stickypane\"\nargs = [\"mcp\"]\n"
	if m, ok := ParseCodexMCP(toml); !ok || m.Command != "/usr/local/bin/stickypane" {
		t.Errorf("codex: %+v %v", m, ok)
	}
	if _, ok := ParseCodexMCP("[mcp_servers.other]\ncommand = \"x\"\n"); ok {
		t.Error("codex: none")
	}
}
