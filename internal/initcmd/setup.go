package initcmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"
)

// The board plugin's marketplace, as `plugin marketplace add` takes it.
const marketplace = "LeeSwallow/stickypane"

// SetupOptions are what the user chooses for stickypane setup.
type SetupOptions struct {
	// Scope is where Claude Code installs the plugin and registers the MCP
	// server: "user" (every project, the default), "project" (this
	// repository, shared through .claude/settings.json and .mcp.json) or
	// "local" (this repository, for you only). Codex has no scopes.
	Scope string
	// MCP also registers `stickypane mcp` as an MCP server, for agents that
	// would rather call tools than run the command line.
	MCP bool
	// Agents limits setup to these ("claude", "codex"); empty means every
	// agent found on the PATH.
	Agents []string
	// Undo removes what setup adds.
	Undo bool
}

// Step is one command setup runs, through the agent's own command line,
// so that stickypane never edits an agent's settings itself.
type Step struct {
	Agent  string
	Cmd    []string
	Unless []string // a command that, when it succeeds, makes the step unneeded
}

// PlanSetup says what setup would run. look finds an agent's command
// (exec.LookPath); has says whether Claude Code and Codex have the plugin.
func PlanSetup(o SetupOptions, look func(string) (string, error), has func() (claude, codex bool)) []Step {
	scope := o.Scope
	if scope == "" {
		scope = "user"
	}
	hasClaude, hasCodex := has()
	want := func(agent string) bool {
		if len(o.Agents) > 0 && !slices.Contains(o.Agents, agent) {
			return false
		}
		_, err := look(agent)
		return err == nil
	}
	var steps []Step
	add := func(agent string, unless []string, cmd ...string) {
		steps = append(steps, Step{Agent: agent, Cmd: cmd, Unless: unless})
	}
	if want("claude") {
		switch {
		case o.Undo:
			add("claude", nil, "claude", "plugin", "uninstall", pluginName, "--scope", scope)
			if o.MCP {
				add("claude", nil, "claude", "mcp", "remove", "stickypane")
			}
		default:
			if !hasClaude {
				add("claude", nil, "claude", "plugin", "marketplace", "add", marketplace)
				add("claude", nil, "claude", "plugin", "install", pluginName, "--scope", scope)
			}
			if o.MCP {
				add("claude", []string{"claude", "mcp", "get", "stickypane"}, "claude", "mcp", "add", "--scope", scope, "stickypane", "--", "stickypane", "mcp")
			}
		}
	}
	if want("codex") {
		switch {
		case o.Undo:
			add("codex", nil, "codex", "plugin", "remove", pluginName)
			if o.MCP {
				add("codex", nil, "codex", "mcp", "remove", "stickypane")
			}
		default:
			if !hasCodex {
				add("codex", nil, "codex", "plugin", "marketplace", "add", marketplace)
				add("codex", nil, "codex", "plugin", "add", pluginName)
			}
			if o.MCP {
				add("codex", []string{"codex", "mcp", "get", "stickypane"}, "codex", "mcp", "add", "stickypane", "--", "stickypane", "mcp")
			}
		}
	}
	return steps
}

// InstalledPlugins says whether Claude Code and Codex have the board plugin
// for the project at root, from the agents' own records.
func InstalledPlugins(root string) func() (claude, codex bool) {
	return func() (bool, bool) { return plugins(root) }
}

// RunSetup runs the steps in order, saying each one, and stops at the first
// that fails. run runs a step (RunCommand) and check runs a step's Unless
// without showing it (QuietCommand).
func RunSetup(steps []Step, run, check func(cmd []string) error, out io.Writer) error {
	for _, s := range steps {
		line := strings.Join(s.Cmd, " ")
		if len(s.Unless) > 0 && check(s.Unless) == nil {
			fmt.Fprintf(out, "  %s  (already done)\n", line)
			continue
		}
		fmt.Fprintf(out, "  %s\n", line)
		if err := run(s.Cmd); err != nil {
			return fmt.Errorf("%s: %w", line, err)
		}
	}
	return nil
}

// RunCommand runs a command with the user's terminal, so an agent's own
// prompts and messages reach them; a check (Unless) runs silently.
func RunCommand(cmd []string) error {
	c := exec.Command(cmd[0], cmd[1:]...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	return c.Run()
}

// QuietCommand runs a command without showing its output.
func QuietCommand(cmd []string) error {
	return exec.Command(cmd[0], cmd[1:]...).Run()
}
