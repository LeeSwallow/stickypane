package initcmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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
	// Command is the stickypane the MCP server runs: "stickypane" (found
	// on the PATH) when empty. The desktop app gives its own copy's
	// absolute path, so an older stickypane earlier on the PATH is not the
	// one an agent starts.
	Command string
	// Registered says how an agent's "stickypane" MCP server runs now, if
	// it is registered: its command and, for Claude Code, its scope. When
	// Command is given and an older registration runs another command (a
	// bare stickypane from the PATH, say), setup replaces it. nil: not
	// known, and a registration is left as it is.
	Registered func(agent string) (MCPServer, bool)
}

// MCPServer is how an agent runs the stickypane MCP server.
type MCPServer struct {
	Command string
	Scope   string // Claude Code's: user, project or local
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
	command := o.Command
	if command == "" {
		command = "stickypane"
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
				if old, ok := o.stale("claude"); ok {
					remove := []string{"claude", "mcp", "remove", "stickypane"}
					if old.Scope != "" {
						remove = []string{"claude", "mcp", "remove", "--scope", old.Scope, "stickypane"}
					}
					add("claude", nil, remove...)
					add("claude", nil, "claude", "mcp", "add", "--scope", scope, "stickypane", "--", command, "mcp")
				} else {
					add("claude", []string{"claude", "mcp", "get", "stickypane"}, "claude", "mcp", "add", "--scope", scope, "stickypane", "--", command, "mcp")
				}
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
				if _, ok := o.stale("codex"); ok {
					add("codex", nil, "codex", "mcp", "remove", "stickypane")
					add("codex", nil, "codex", "mcp", "add", "stickypane", "--", command, "mcp")
				} else {
					add("codex", []string{"codex", "mcp", "get", "stickypane"}, "codex", "mcp", "add", "stickypane", "--", command, "mcp")
				}
			}
		}
	}
	return steps
}

// stale says whether the agent's stickypane MCP server is registered with
// another command than the full path asked for, and so is to be replaced.
func (o SetupOptions) stale(agent string) (MCPServer, bool) {
	if o.Command == "" || o.Registered == nil {
		return MCPServer{}, false
	}
	m, ok := o.Registered(agent)
	if !ok || m.Command == o.Command {
		return MCPServer{}, false
	}
	return m, true
}

// RegisteredMCP says how an agent's stickypane MCP server is registered:
// Claude Code through `claude mcp get stickypane`, Codex from its
// config.toml.
func RegisteredMCP(agent string) (MCPServer, bool) {
	switch agent {
	case "claude":
		out, err := exec.Command("claude", "mcp", "get", "stickypane").Output()
		if err != nil {
			return MCPServer{}, false
		}
		return ParseClaudeMCP(string(out))
	case "codex":
		home := agentHome("CODEX_HOME", ".codex")
		if home == "" {
			return MCPServer{}, false
		}
		b, err := os.ReadFile(filepath.Join(home, "config.toml"))
		if err != nil {
			return MCPServer{}, false
		}
		return ParseCodexMCP(string(b))
	}
	return MCPServer{}, false
}

// ParseClaudeMCP reads what `claude mcp get stickypane` prints: its
// Command and Scope lines.
func ParseClaudeMCP(out string) (MCPServer, bool) {
	var m MCPServer
	for _, line := range strings.Split(out, "\n") {
		k, v, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok {
			continue
		}
		v = strings.TrimSpace(v)
		switch k {
		case "Command":
			m.Command = v
		case "Scope":
			switch f := strings.ToLower(v); {
			case strings.HasPrefix(f, "user"):
				m.Scope = "user"
			case strings.HasPrefix(f, "project"):
				m.Scope = "project"
			case strings.HasPrefix(f, "local"):
				m.Scope = "local"
			}
		}
	}
	return m, m.Command != ""
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
