package initcmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/core/i18n"
)

// agentsAt is a status: the agents found, each with the plugin at the
// scopes given ("user", "project", ...) at version 1.0.0, and nothing else.
func agentsAt(found ...string) Status {
	st := Status{Plugin: "1.0.0"}
	for _, a := range found {
		st.Agents = append(st.Agents, AgentStatus{Agent: a, Found: true, Scopes: scopesOf[a], Plugin: map[string]PluginStatus{}, MCP: map[string]MCPStatus{}})
	}
	return st
}

// with gives agent the plugin at scope, in state, and returns st.
func (st Status) with(agent, scope, state string) Status {
	for _, a := range st.Agents {
		if a.Agent == agent {
			a.Plugin[scope] = PluginStatus{Version: "1.0.0", State: state}
		}
	}
	return st
}

func (st Status) mcp(agent, scope, command string) Status {
	for _, a := range st.Agents {
		if a.Agent == agent {
			a.MCP[scope] = MCPStatus{Command: command}
		}
	}
	return st
}

func (st Status) market(agent string, m Marketplace) Status {
	for i := range st.Agents {
		if st.Agents[i].Agent == agent {
			st.Agents[i].Marketplace = &m
		}
	}
	return st
}

func cmds(steps []Step) string {
	var out []string
	for _, s := range steps {
		out = append(out, strings.Join(s.Cmd, " "))
	}
	return strings.Join(out, "\n")
}

func plan(t *testing.T, o SetupOptions, st Status, want ...string) {
	t.Helper()
	if got := cmds(PlanSetup(o, st)); got != strings.Join(want, "\n") {
		t.Errorf("plan =\n%s\nwant\n%s", got, strings.Join(want, "\n"))
	}
}

// setup installs the plugin for every agent it finds, through the agents'
// own commands, in the scope the user chose; the MCP server only when asked.
func TestSetupPlansTheAgentsOwnCommands(t *testing.T) {
	plan(t, SetupOptions{Scope: "project", MCP: true}, agentsAt("claude", "codex"),
		"claude plugin marketplace add LeeSwallow/stickypane",
		"claude plugin install board@stickypane --scope project",
		"claude mcp add --scope project stickypane -- stickypane mcp",
		"codex plugin marketplace add LeeSwallow/stickypane",
		"codex plugin add board@stickypane",
		"codex mcp add stickypane -- stickypane mcp",
	)
}

func TestSetupSkipsWhatIsThereAndWhatIsMissing(t *testing.T) {
	// Codex is not installed; Claude Code has the plugin already.
	st := agentsAt("claude").with("claude", "user", StateOK)
	st.Agents = append(st.Agents, AgentStatus{Agent: "codex", State: StateMissing})
	plan(t, SetupOptions{}, st)
	// Only the agents asked for.
	plan(t, SetupOptions{Agents: []string{"codex"}}, agentsAt("claude", "codex"),
		"codex plugin marketplace add LeeSwallow/stickypane",
		"codex plugin add board@stickypane")
	// A marketplace already there is not added again.
	plan(t, SetupOptions{}, agentsAt("claude").market("claude", Marketplace{Source: "github", Where: "LeeSwallow/stickypane"}),
		"claude plugin install board@stickypane --scope user")
	// The MCP server registered in the scope with the same command: nothing.
	plan(t, SetupOptions{MCP: true}, agentsAt("claude").with("claude", "user", StateOK).mcp("claude", "user", "stickypane"))
	// The plugin in the user's scope is not the project's.
	plan(t, SetupOptions{Scope: "project"}, agentsAt("claude").with("claude", "user", StateOK).market("claude", Marketplace{Source: "github"}),
		"claude plugin install board@stickypane --scope project")
}

// Codex has the user's scope only: a project scope asked for is the user's.
func TestSetupCodexHasTheUsersScopeOnly(t *testing.T) {
	plan(t, SetupOptions{Scope: "local", MCP: true}, agentsAt("codex").with("codex", "user", StateOK),
		"codex mcp add stickypane -- stickypane mcp")
}

// --plugin=false leaves the plugin: the MCP server only, added or removed.
func TestSetupMCPOnly(t *testing.T) {
	plan(t, SetupOptions{MCP: true, NoPlugin: true}, agentsAt("claude", "codex"),
		"claude mcp add --scope user stickypane -- stickypane mcp",
		"codex mcp add stickypane -- stickypane mcp")
	st := agentsAt("claude", "codex").with("claude", "user", StateOK).with("codex", "user", StateOK).mcp("claude", "user", "x").mcp("codex", "user", "x")
	plan(t, SetupOptions{MCP: true, NoPlugin: true, Undo: true}, st,
		"claude mcp remove --scope user stickypane",
		"codex mcp remove stickypane")
}

// Command names the stickypane the MCP server runs, by its full path.
func TestSetupMCPRunsTheCommandGiven(t *testing.T) {
	const app = "/Applications/Stickypane.app/Contents/Helpers/stickypane"
	st := agentsAt("claude", "codex").with("claude", "user", StateOK).with("codex", "user", StateOK)
	plan(t, SetupOptions{MCP: true, Command: app}, st,
		"claude mcp add --scope user stickypane -- "+app+" mcp",
		"codex mcp add stickypane -- "+app+" mcp")
}

func TestSetupUndoRemovesWhatItAdded(t *testing.T) {
	st := agentsAt("claude", "codex").with("claude", "user", StateOK).with("codex", "user", StateOK).mcp("claude", "user", "stickypane").mcp("codex", "user", "stickypane")
	plan(t, SetupOptions{Undo: true, Scope: "user", MCP: true}, st,
		"claude plugin uninstall board@stickypane --scope user",
		"claude mcp remove --scope user stickypane",
		"codex plugin remove board@stickypane",
		"codex mcp remove stickypane")
	// Without --mcp, undo takes out the plugin only, as the README says.
	plan(t, SetupOptions{Undo: true}, st,
		"claude plugin uninstall board@stickypane --scope user",
		"codex plugin remove board@stickypane")
	// Nothing in the scope asked for: nothing to take out.
	plan(t, SetupOptions{Undo: true, Scope: "local", MCP: true}, agentsAt("claude").with("claude", "user", StateOK))
}

// A plugin older than this stickypane's, or one that fails to load, is
// updated from its marketplace.
func TestSetupUpdatesAnOlderPlugin(t *testing.T) {
	st := agentsAt("claude", "codex").with("claude", "user", StateOutdated).with("codex", "user", StateBroken)
	plan(t, SetupOptions{}, st,
		"claude plugin marketplace update stickypane",
		"claude plugin update board@stickypane --scope user",
		"codex plugin marketplace upgrade stickypane",
		"codex plugin add board@stickypane")
}

// A marketplace added from a folder that is gone (a checkout that moved)
// is pointed at GitHub again, and the plugin installed anew: in Claude
// Code at every scope it had, since removing the marketplace uninstalls it.
func TestSetupRepairsAMarketplaceWhoseFolderIsGone(t *testing.T) {
	gone := Marketplace{Source: "directory", Where: "/Users/min/Projects/stickypane", Gone: true}
	st := agentsAt("claude").with("claude", "user", StateBroken).with("claude", "local", StateBroken).market("claude", gone)
	plan(t, SetupOptions{}, st,
		"claude plugin marketplace remove stickypane",
		"claude plugin marketplace add LeeSwallow/stickypane",
		"claude plugin install board@stickypane --scope user",
		"claude plugin install board@stickypane --scope local")
	gone.Source = "local"
	plan(t, SetupOptions{}, agentsAt("codex").with("codex", "user", StateBroken).market("codex", gone),
		"codex plugin marketplace remove stickypane",
		"codex plugin marketplace add LeeSwallow/stickypane",
		"codex plugin add board@stickypane")
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
	err := RunSetup(steps, run, run, &out, i18n.English())
	if err == nil || strings.Join(ran, ",") != "ok 1,ok check,fail" {
		t.Errorf("ran %q, err %v", ran, err)
	}
	if !strings.Contains(out.String(), "skip") || !strings.Contains(out.String(), "already") {
		t.Errorf("out = %q", out.String())
	}
}

// An MCP server registered in the scope with another command (a bare
// stickypane from the PATH) is removed and added again with the full path
// asked for; one that has it already is left.
func TestSetupReplacesAnMCPServerThatRunsAnotherCommand(t *testing.T) {
	const app = "/Applications/Stickypane.app/Contents/Helpers/stickypane"
	st := agentsAt("claude", "codex").with("claude", "local", StateOK).with("codex", "user", StateOK).mcp("claude", "local", "stickypane").mcp("codex", "user", "stickypane")
	plan(t, SetupOptions{MCP: true, Command: app, Scope: "local"}, st,
		"claude mcp remove --scope local stickypane",
		"claude mcp add --scope local stickypane -- "+app+" mcp",
		"codex mcp remove stickypane",
		"codex mcp add stickypane -- "+app+" mcp")
	plan(t, SetupOptions{MCP: true, Command: app, Scope: "local"}, agentsAt("claude").with("claude", "local", StateOK).mcp("claude", "local", app))
	// without --command nothing is replaced
	plan(t, SetupOptions{MCP: true, Scope: "local"}, st)
}

func TestParseCodexMCP(t *testing.T) {
	toml := "[mcp_servers.other]\ncommand = \"x\"\n\n[mcp_servers.stickypane]\ncommand = \"/usr/local/bin/stickypane\"\nargs = [\"mcp\"]\n"
	if m, ok := ParseCodexMCP(toml); !ok || m.Command != "/usr/local/bin/stickypane" {
		t.Errorf("codex: %+v %v", m, ok)
	}
	if _, ok := ParseCodexMCP("[mcp_servers.other]\ncommand = \"x\"\n"); ok {
		t.Error("codex: none")
	}
}
