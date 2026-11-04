package initcmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/LeeSwallow/stickypane"
)

// The board plugin's marketplace, as `plugin marketplace add` takes it,
// and the name the agents know it by.
const (
	marketplace     = "LeeSwallow/stickypane"
	marketplaceName = "stickypane"
)

// wantVersion is the plugin version setup installs, or updates an older
// one to: the one this program carries. Tests change it.
var wantVersion = stickypane.PluginVersion

// SetupOptions are what the user chooses for stickypane setup.
type SetupOptions struct {
	// Scope is where Claude Code installs the plugin and registers the MCP
	// server: "user" (every project, the default), "project" (this
	// repository, shared through .claude/settings.json and .mcp.json) or
	// "local" (this repository, for you only). Codex has the user's only.
	Scope string
	// MCP also registers `stickypane mcp` as an MCP server, for agents that
	// would rather call tools than run the command line.
	MCP bool
	// NoPlugin leaves the board plugin as it is: with MCP, setup registers
	// (or with Undo removes) the MCP server only.
	NoPlugin bool
	// Agents limits setup to these ("claude", "codex"); empty means every
	// agent found on the PATH.
	Agents []string
	// Undo removes what setup adds.
	Undo bool
	// Command is the stickypane the MCP server runs: "stickypane" (found
	// on the PATH) when empty. The desktop app gives its own copy's
	// absolute path, so an older stickypane earlier on the PATH is not the
	// one an agent starts; a server registered in the scope with another
	// command is then replaced.
	Command string
}

// MCPServer is how an agent runs the stickypane MCP server.
type MCPServer struct {
	Command string
}

// Step is one command setup runs, through the agent's own command line,
// so that stickypane never edits an agent's settings itself.
type Step struct {
	Agent  string
	Cmd    []string
	Unless []string // a command that, when it succeeds, makes the step unneeded
}

// PlanSetup says what setup would run, from where the agents stand (st,
// from ReadStatus): it installs the plugin where it is missing, updates
// it where it is older than this stickypane's or fails to load, and where
// the agent's marketplace is a folder that is gone, points the marketplace
// at GitHub again and installs the plugin anew.
func PlanSetup(o SetupOptions, st Status) []Step {
	var steps []Step
	for _, a := range st.Agents {
		if !a.Found || len(o.Agents) > 0 && !slices.Contains(o.Agents, a.Agent) {
			continue
		}
		steps = append(steps, planAgent(o, a)...)
	}
	return steps
}

func planAgent(o SetupOptions, a AgentStatus) []Step {
	s := o.Scope
	if s == "" || !slices.Contains(a.Scopes, s) {
		s = "user" // Codex has no other
	}
	command := o.Command
	if command == "" {
		command = "stickypane"
	}
	claude := a.Agent == "claude"
	// scoped is a Claude Code command with the scope; Codex's take none.
	scoped := func(cmd ...string) []string {
		if claude {
			return append(cmd, "--scope", s)
		}
		return cmd
	}
	var steps []Step
	add := func(cmd ...string) { steps = append(steps, Step{Agent: a.Agent, Cmd: cmd}) }
	pl, has := a.Plugin[s]
	m, registered := a.MCP[s]
	mcpRemove := []string{a.Agent, "mcp", "remove", "stickypane"}
	mcpAdd := []string{a.Agent, "mcp", "add", "stickypane", "--", command, "mcp"}
	if claude {
		mcpRemove = []string{"claude", "mcp", "remove", "--scope", s, "stickypane"}
		mcpAdd = []string{"claude", "mcp", "add", "--scope", s, "stickypane", "--", command, "mcp"}
	}
	install := scoped(a.Agent, "plugin", "add", pluginName)
	update := []string{a.Agent, "plugin", "marketplace", "upgrade", marketplaceName}
	if claude {
		install = scoped("claude", "plugin", "install", pluginName)
		update = []string{"claude", "plugin", "marketplace", "update", marketplaceName}
	}

	if o.Undo {
		if !o.NoPlugin && has {
			if claude {
				add(scoped("claude", "plugin", "uninstall", pluginName)...)
			} else {
				add("codex", "plugin", "remove", pluginName)
			}
		}
		if o.MCP && registered {
			add(mcpRemove...)
		}
		return steps
	}
	if !o.NoPlugin {
		switch {
		case a.Marketplace != nil && a.Marketplace.Gone:
			add(a.Agent, "plugin", "marketplace", "remove", marketplaceName)
			add(a.Agent, "plugin", "marketplace", "add", marketplace)
			if !claude {
				add(install...)
				break
			}
			// Removing Claude Code's marketplace uninstalls the plugin from
			// every scope: install it again where it was.
			for _, sc := range a.Scopes {
				if _, ok := a.Plugin[sc]; ok || sc == s {
					add("claude", "plugin", "install", pluginName, "--scope", sc)
				}
			}
		case !has:
			if a.Marketplace == nil {
				add(a.Agent, "plugin", "marketplace", "add", marketplace)
			}
			add(install...)
		case pl.State == StateOutdated || pl.State == StateBroken:
			add(update...)
			if claude {
				add(scoped("claude", "plugin", "update", pluginName)...)
			} else {
				add(install...) // Codex adds an installed plugin again from the new snapshot
			}
		}
	}
	if o.MCP && (!registered || o.Command != "" && m.Command != o.Command) {
		if registered {
			add(mcpRemove...)
		}
		add(mcpAdd...)
	}
	return steps
}

// ParseCodexMCP reads the command of the [mcp_servers.stickypane] table of
// Codex's config.toml.
func ParseCodexMCP(toml string) (MCPServer, bool) {
	in := false
	for _, line := range strings.Split(toml, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") {
			in = line == "[mcp_servers.stickypane]"
			continue
		}
		if k, v, ok := strings.Cut(line, "="); in && ok && strings.TrimSpace(k) == "command" {
			v = strings.Trim(strings.TrimSpace(v), `"'`)
			return MCPServer{Command: v}, v != ""
		}
	}
	return MCPServer{}, false
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
