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
