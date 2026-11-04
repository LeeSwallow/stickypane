package initcmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// The states of an agent, or of the plugin in one scope, as stickypane
// setup --status says them.
const (
	StateMissing  = "missing"  // the agent's command is not on the PATH
	StateOff      = "off"      // the board plugin is not installed
	StateOK       = "ok"       // installed, loads, and as new as this stickypane's
	StateOutdated = "outdated" // installed, older than this stickypane's
	StateBroken   = "broken"   // installed, but the agent cannot load it
)

// Status is where each agent stands with the board plugin and the MCP
// server, as their own commands and records say, for the project setup
// runs in.
type Status struct {
	// Plugin is the version of the plugin this stickypane carries.
	Plugin string        `json:"plugin"`
	Agents []AgentStatus `json:"agents"`
}

// AgentStatus is one agent's. Plugin and MCP are keyed by scope: "user"
// (every project), and for Claude Code "project" (this repository, shared
// through its files) and "local" (this repository, this machine only). A
// scope where nothing is installed is left out.
type AgentStatus struct {
	Agent string `json:"agent"` // claude or codex
	Found bool   `json:"found"` // its command is on the PATH
	Path  string `json:"path,omitempty"`
	// State sums up the plugin's scopes: missing, off when no scope has it,
	// else the worst of them (broken, outdated, ok).
	State string `json:"state"`
	// Scopes are the scopes the agent has: Codex only "user".
	Scopes []string                `json:"scopes"`
	Plugin map[string]PluginStatus `json:"plugin"`
	MCP    map[string]MCPStatus    `json:"mcp"`
	// Marketplace is where the agent gets the plugin from; nil when it has
	// no marketplace named stickypane.
	Marketplace *Marketplace `json:"marketplace,omitempty"`
}

// PluginStatus is the board plugin in one scope of one agent.
type PluginStatus struct {
	Version string `json:"version,omitempty"`
	State   string `json:"state"`           // ok, outdated or broken
	Error   string `json:"error,omitempty"` // why it does not load
}

// MCPStatus is the stickypane MCP server in one scope of one agent.
type MCPStatus struct {
	Command string `json:"command"`
}

// Marketplace is the agent's "stickypane" marketplace.
type Marketplace struct {
	Source string `json:"source"` // github, git, directory (Claude Code), local (Codex), ...
	Where  string `json:"where"`  // the repository or the folder
	// Gone: a folder that is no longer there, as when the checkout it was
	// added from moved. setup points the marketplace at GitHub again.
	Gone bool `json:"gone,omitempty"`
}

// The scopes of each agent.
var scopesOf = map[string][]string{
	"claude": {"user", "project", "local"},
	"codex":  {"user"},
}

// Probe is how status reaches the agents: their commands and their files.
// Tests pass fakes.
type Probe struct {
	Look   func(name string) (string, error)  // exec.LookPath
	Output func(cmd []string) (string, error) // what a command printed
	Read   func(path string) ([]byte, error)  // os.ReadFile
	Exists func(path string) bool             // whether a folder is there
	Env    func(key string) string            // os.Getenv
	Home   func() (string, error)             // os.UserHomeDir
	Root   string                             // the project setup runs in; "" for none
}

// SystemProbe is the probe on this machine, for the project at root.
func SystemProbe(root string) Probe {
	return Probe{
		Look: exec.LookPath,
		Output: func(cmd []string) (string, error) {
			out, err := exec.Command(cmd[0], cmd[1:]...).Output()
			if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
				return string(out) + string(ee.Stderr), err
			}
			return string(out), err
		},
		Read:   os.ReadFile,
		Exists: func(p string) bool { fi, err := os.Stat(p); return err == nil && fi.IsDir() },
		Env:    os.Getenv,
		Home:   os.UserHomeDir,
		Root:   root,
	}
}

// ReadStatus asks each agent in agents ("claude", "codex") where it
// stands. It runs only commands that read (plugin list, marketplace list)
// and reads the agents' own records of their MCP servers.
func ReadStatus(p Probe, agents []string) Status {
	st := Status{Plugin: wantVersion}
	for _, a := range agents {
		st.Agents = append(st.Agents, p.agent(a))
	}
	return st
}

func (p Probe) agent(name string) AgentStatus {
	a := AgentStatus{Agent: name, State: StateMissing, Scopes: scopesOf[name], Plugin: map[string]PluginStatus{}, MCP: map[string]MCPStatus{}}
	path, err := p.Look(name)
	if err != nil {
		return a
	}
	a.Found, a.Path = true, path
	switch name {
	case "claude":
		p.claude(&a)
	case "codex":
		p.codex(&a)
	}
	if a.Marketplace != nil && (a.Marketplace.Source == "directory" || a.Marketplace.Source == "local") && a.Marketplace.Where != "" && !p.Exists(a.Marketplace.Where) {
		a.Marketplace.Gone = true
	}
	a.State = StateOff
	rank := map[string]int{StateOff: 0, StateOK: 1, StateOutdated: 2, StateBroken: 3}
	for scope, pl := range a.Plugin {
		if pl.State == StateOK && Older(pl.Version, wantVersion) {
			pl.State = StateOutdated
			a.Plugin[scope] = pl
		}
		if rank[pl.State] > rank[a.State] {
			a.State = pl.State
		}
	}
	return a
}

func (p Probe) claude(a *AgentStatus) {
	if out, err := p.Output([]string{"claude", "plugin", "list", "--json"}); err == nil {
		a.Plugin = ParseClaudePlugins(out, p.Root)
	}
	if out, err := p.Output([]string{"claude", "plugin", "marketplace", "list", "--json"}); err == nil {
		a.Marketplace = ParseClaudeMarketplaces(out)
	}
	// Claude Code keeps its user and local servers in .claude.json (in
	// CLAUDE_CONFIG_DIR when set, else the home folder) and a project's in
	// its .mcp.json. Reading them starts no server, as claude mcp get does.
	file := ""
	if dir := p.Env("CLAUDE_CONFIG_DIR"); dir != "" {
		file = filepath.Join(dir, ".claude.json")
	} else if home, err := p.Home(); err == nil {
		file = filepath.Join(home, ".claude.json")
	}
	if b, err := p.Read(file); file != "" && err == nil {
		a.MCP = ParseClaudeMCPConfig(b, p.Root)
	}
	if p.Root != "" {
		if b, err := p.Read(filepath.Join(p.Root, ".mcp.json")); err == nil {
			if m, ok := mcpServer(b); ok {
				a.MCP["project"] = m
			}
		}
	}
}

func (p Probe) codex(a *AgentStatus) {
	toml := ""
	if home := p.codexHome(); home != "" {
		if b, err := p.Read(filepath.Join(home, "config.toml")); err == nil {
			toml = string(b)
		}
	}
	out, err := p.Output([]string{"codex", "plugin", "list", "--json"})
	switch {
	case err == nil:
		a.Plugin = ParseCodexPlugins(out)
	case CodexHasPlugin(toml):
		// Codex lists nothing when a marketplace fails to load.
		a.Plugin = map[string]PluginStatus{"user": {State: StateBroken, Error: errorOf(out)}}
	}
	a.Marketplace = ParseCodexMarketplace(toml)
	if m, ok := ParseCodexMCP(toml); ok {
		a.MCP["user"] = MCPStatus{Command: m.Command}
	}
}

func (p Probe) codexHome() string {
	if dir := p.Env("CODEX_HOME"); dir != "" {
		return dir
	}
	home, err := p.Home()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".codex")
}

// ParseClaudePlugins reads `claude plugin list --json` for the board
// plugin, by scope: the user's, and the project's and local one of the
// project at root.
func ParseClaudePlugins(out, root string) map[string]PluginStatus {
	got := map[string]PluginStatus{}
	var list []struct {
		ID          string   `json:"id"`
		Version     string   `json:"version"`
		Scope       string   `json:"scope"`
		ProjectPath string   `json:"projectPath"`
		Errors      []string `json:"errors"`
	}
	if json.Unmarshal([]byte(out), &list) != nil {
		return got
	}
	for _, e := range list {
		if e.ID != pluginName {
			continue
		}
		if e.Scope == "project" || e.Scope == "local" {
			if root == "" || e.ProjectPath == "" || !sameDir(e.ProjectPath, root) {
				continue
			}
		}
		s := PluginStatus{Version: e.Version, State: StateOK}
		if len(e.Errors) > 0 {
			s.State, s.Error = StateBroken, e.Errors[0]
		}
		got[e.Scope] = s
	}
	return got
}

// ParseClaudeMarketplaces reads `claude plugin marketplace list --json` for
// the stickypane marketplace.
func ParseClaudeMarketplaces(out string) *Marketplace {
	var list []struct {
		Name   string `json:"name"`
		Source string `json:"source"`
		Repo   string `json:"repo"`
		URL    string `json:"url"`
		Path   string `json:"path"`
	}
	if json.Unmarshal([]byte(out), &list) != nil {
		return nil
	}
	for _, m := range list {
		if m.Name != marketplaceName {
			continue
		}
		where := m.Repo
		for _, w := range []string{m.Path, m.URL} {
			if where == "" {
				where = w
			}
		}
		return &Marketplace{Source: m.Source, Where: where}
	}
	return nil
}

// ParseClaudeMCPConfig reads Claude Code's .claude.json for the stickypane
// server: the user's, and the local one of the project at root.
func ParseClaudeMCPConfig(b []byte, root string) map[string]MCPStatus {
	got := map[string]MCPStatus{}
	var cfg struct {
		MCPServers json.RawMessage `json:"mcpServers"`
		Projects   map[string]struct {
			MCPServers json.RawMessage `json:"mcpServers"`
		} `json:"projects"`
	}
	if json.Unmarshal(b, &cfg) != nil {
		return got
	}
	if m, ok := servers(cfg.MCPServers); ok {
		got["user"] = m
	}
	if root != "" {
		for dir, p := range cfg.Projects {
			if sameDir(dir, root) {
				if m, ok := servers(p.MCPServers); ok {
					got["local"] = m
				}
			}
		}
	}
	return got
}

// mcpServer reads a project's .mcp.json for the stickypane server.
func mcpServer(b []byte) (MCPStatus, bool) {
	var f struct {
		MCPServers json.RawMessage `json:"mcpServers"`
	}
	if json.Unmarshal(b, &f) != nil {
		return MCPStatus{}, false
	}
	return servers(f.MCPServers)
}

func servers(raw json.RawMessage) (MCPStatus, bool) {
	var all map[string]struct {
		Command string `json:"command"`
		URL     string `json:"url"`
	}
	if len(raw) == 0 || json.Unmarshal(raw, &all) != nil {
		return MCPStatus{}, false
	}
	s, ok := all["stickypane"]
	if !ok {
		return MCPStatus{}, false
	}
	if s.Command == "" {
		s.Command = s.URL
	}
	return MCPStatus{Command: s.Command}, true
}

// ParseCodexPlugins reads `codex plugin list --json` for the board plugin.
// Codex has one scope, the user's.
func ParseCodexPlugins(out string) map[string]PluginStatus {
	got := map[string]PluginStatus{}
	var list struct {
		Installed []struct {
			PluginID string `json:"pluginId"`
			Version  string `json:"version"`
			Enabled  bool   `json:"enabled"`
		} `json:"installed"`
	}
	if json.Unmarshal([]byte(out), &list) != nil {
		return got
	}
	for _, e := range list.Installed {
		if e.PluginID == pluginName && e.Enabled {
			got["user"] = PluginStatus{Version: e.Version, State: StateOK}
		}
	}
	return got
}

// ParseCodexMarketplace reads the [marketplaces.stickypane] table of
// Codex's config.toml.
func ParseCodexMarketplace(toml string) *Marketplace {
	var m *Marketplace
	in := false
	for _, line := range strings.Split(toml, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") {
			in = line == "[marketplaces."+marketplaceName+"]"
			if in && m == nil {
				m = &Marketplace{}
			}
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !in || !ok {
			continue
		}
		v = strings.Trim(strings.TrimSpace(v), `"'`)
		switch strings.TrimSpace(k) {
		case "source_type":
			m.Source = v
		case "source":
			m.Where = v
		}
	}
	return m
}

// CodexHasPlugin reads config.toml for the plugin's table, and takes
// `enabled = false` under it to mean it is off.
func CodexHasPlugin(toml string) bool {
	found := false
	for _, line := range strings.Split(toml, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") {
			if found {
				break
			}
			found = line == `[plugins."`+pluginName+`"]`
			continue
		}
		if found && strings.ReplaceAll(line, " ", "") == "enabled=false" {
			return false
		}
	}
	return found
}

// Older says whether version have is older than want. Either one not a
// release version (dev, a pseudo-version, "") is never older.
func Older(have, want string) bool {
	h, ok1 := semver(have)
	w, ok2 := semver(want)
	if !ok1 || !ok2 {
		return false
	}
	for i := range h {
		if h[i] != w[i] {
			return h[i] < w[i]
		}
	}
	return false
}

func semver(v string) ([3]int, bool) {
	var out [3]int
	parts := strings.Split(strings.TrimPrefix(v, "v"), ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, s := range parts {
		n, err := strconv.Atoi(s)
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

// errorOf is what an agent's failed command said, on one line.
func errorOf(s string) string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			out = append(out, strings.TrimPrefix(line, "Error: "))
		}
	}
	return strings.Join(out, " ")
}
