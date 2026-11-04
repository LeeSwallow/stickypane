package initcmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The fixtures in testdata/agents are what Claude Code 2.1 and Codex 0.166
// printed, run with a home folder of their own, with the paths shortened.
func fixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "agents", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// fakeProbe answers commands from outputs (a command line to what it
// printed; a missing one fails) and files from files.
type fakeProbe struct {
	found   []string
	outputs map[string]string
	fails   map[string]string
	files   map[string]string
	dirs    []string
	root    string
}

func (f fakeProbe) probe() Probe {
	return Probe{
		Look: func(name string) (string, error) {
			for _, a := range f.found {
				if a == name {
					return "/bin/" + name, nil
				}
			}
			return "", errors.New("not found")
		},
		Output: func(cmd []string) (string, error) {
			line := strings.Join(cmd, " ")
			if out, ok := f.fails[line]; ok {
				return out, errors.New("exit status 1")
			}
			out, ok := f.outputs[line]
			if !ok {
				return "", errors.New("exit status 1")
			}
			return out, nil
		},
		Read: func(path string) ([]byte, error) {
			if s, ok := f.files[path]; ok {
				return []byte(s), nil
			}
			return nil, os.ErrNotExist
		},
		Exists: func(path string) bool {
			for _, d := range f.dirs {
				if d == path {
					return true
				}
			}
			return false
		},
		Env:  func(key string) string { return map[string]string{"CLAUDE_CONFIG_DIR": "/c", "CODEX_HOME": "/x"}[key] },
		Home: func() (string, error) { return "/home", nil },
		Root: f.root,
	}
}

func setWant(t *testing.T, v string) {
	old := wantVersion
	wantVersion = v
	t.Cleanup(func() { wantVersion = old })
}

func TestStatusOfAnAgentNotOnThePath(t *testing.T) {
	st := ReadStatus(fakeProbe{}.probe(), []string{"claude", "codex"})
	for _, a := range st.Agents {
		if a.Found || a.State != StateMissing {
			t.Errorf("%s: %+v", a.Agent, a)
		}
	}
}

func TestStatusOfClaudeCode(t *testing.T) {
	setWant(t, "0.2.1")
	f := fakeProbe{
		found: []string{"claude"},
		outputs: map[string]string{
			"claude plugin list --json":             fixture(t, "claude-plugin-list-ok.json"),
			"claude plugin marketplace list --json": fixture(t, "claude-marketplace-list-github.json"),
		},
		files: map[string]string{"/c/.claude.json": `{"mcpServers":{"stickypane":{"type":"stdio","command":"/usr/local/bin/stickypane","args":["mcp"]}}}`},
	}
	a := ReadStatus(f.probe(), []string{"claude"}).Agents[0]
	if a.State != StateOK || a.Plugin["user"].Version != "0.2.1" || a.Marketplace == nil || a.Marketplace.Where != "LeeSwallow/stickypane" || a.Marketplace.Gone {
		t.Errorf("ok: %+v %+v", a, a.Marketplace)
	}
	if a.MCP["user"].Command != "/usr/local/bin/stickypane" || len(a.MCP) != 1 {
		t.Errorf("mcp: %+v", a.MCP)
	}
	// A newer stickypane: the plugin is older.
	setWant(t, "0.3.0")
	if a := ReadStatus(f.probe(), []string{"claude"}).Agents[0]; a.State != StateOutdated || a.Plugin["user"].State != StateOutdated {
		t.Errorf("outdated: %+v", a)
	}
}

// The case seen for real: the marketplace was added from a checkout that
// moved, and Claude Code fails to load the plugin.
func TestStatusOfAMarketplaceWhoseFolderIsGone(t *testing.T) {
	setWant(t, "0.2.1")
	f := fakeProbe{
		found: []string{"claude"},
		outputs: map[string]string{
			"claude plugin list --json":             fixture(t, "claude-plugin-list-broken.json"),
			"claude plugin marketplace list --json": fixture(t, "claude-marketplace-list-folder.json"),
		},
	}
	a := ReadStatus(f.probe(), []string{"claude"}).Agents[0]
	if a.State != StateBroken || !strings.Contains(a.Plugin["user"].Error, "cache-miss") {
		t.Errorf("broken: %+v", a)
	}
	if m := a.Marketplace; m == nil || m.Source != "directory" || m.Where != "/Users/min/Projects/stickypane" || !m.Gone {
		t.Errorf("marketplace: %+v", m)
	}
	f.dirs = []string{"/Users/min/Projects/stickypane"}
	if m := ReadStatus(f.probe(), []string{"claude"}).Agents[0].Marketplace; m.Gone {
		t.Errorf("a folder that is there is not gone: %+v", m)
	}
	// and setup puts it right
	plan(t, SetupOptions{}, ReadStatus(fakeProbe{found: f.found, outputs: f.outputs}.probe(), []string{"claude"}),
		"claude plugin marketplace remove stickypane",
		"claude plugin marketplace add LeeSwallow/stickypane",
		"claude plugin install board@stickypane --scope user")
}

// The plugin and the MCP server at the project's scopes count in that
// project only.
func TestStatusByScope(t *testing.T) {
	setWant(t, "0.2.1")
	f := fakeProbe{
		found:   []string{"claude"},
		outputs: map[string]string{"claude plugin list --json": fixture(t, "claude-plugin-list-scopes.json")},
		files: map[string]string{
			"/c/.claude.json":               `{"mcpServers":{"stickypane":{"command":"/opt/user/stickypane"}},"projects":{"/Users/min/work/app":{"mcpServers":{"stickypane":{"command":"/opt/local/stickypane"}}}}}`,
			"/Users/min/work/app/.mcp.json": `{"mcpServers":{"stickypane":{"type":"stdio","command":"/usr/local/bin/stickypane","args":["mcp"]}}}`,
		},
		root: "/Users/min/work/app",
	}
	a := ReadStatus(f.probe(), []string{"claude"}).Agents[0]
	if _, ok := a.Plugin["user"]; ok || a.Plugin["project"].Version != "0.2.1" || a.Plugin["local"].State != StateOK {
		t.Errorf("plugin in the project: %+v", a.Plugin)
	}
	if a.MCP["user"].Command != "/opt/user/stickypane" || a.MCP["local"].Command != "/opt/local/stickypane" || a.MCP["project"].Command != "/usr/local/bin/stickypane" {
		t.Errorf("mcp in the project: %+v", a.MCP)
	}
	f.root = "/Users/min/other"
	a = ReadStatus(f.probe(), []string{"claude"}).Agents[0]
	if len(a.Plugin) != 0 || a.State != StateOff || len(a.MCP) != 1 {
		t.Errorf("in another project: %+v", a)
	}
}

func TestStatusOfCodex(t *testing.T) {
	setWant(t, "0.2.1")
	f := fakeProbe{
		found:   []string{"codex"},
		outputs: map[string]string{"codex plugin list --json": fixture(t, "codex-plugin-list-local.json")},
		files:   map[string]string{"/x/config.toml": fixture(t, "codex-config-local.toml")},
		dirs:    []string{"/Users/min/Projects/stickypane"},
	}
	a := ReadStatus(f.probe(), []string{"codex"}).Agents[0]
	if a.State != StateOK || a.Plugin["user"].Version != "0.2.1" || a.MCP["user"].Command != "/usr/local/bin/stickypane" || a.Marketplace.Source != "local" || a.Marketplace.Gone {
		t.Errorf("ok: %+v %+v", a, a.Marketplace)
	}
	if len(a.Scopes) != 1 {
		t.Errorf("codex scopes: %v", a.Scopes)
	}
	// The folder moved: Codex lists nothing and says why.
	f.dirs = nil
	f.outputs = nil
	f.fails = map[string]string{"codex plugin list --json": fixture(t, "codex-plugin-list-broken.txt")}
	a = ReadStatus(f.probe(), []string{"codex"}).Agents[0]
	if a.State != StateBroken || !a.Marketplace.Gone || !strings.Contains(a.Plugin["user"].Error, "does not contain a supported manifest") {
		t.Errorf("broken: %+v %+v", a, a.Marketplace)
	}
	// Nothing installed.
	f.fails = nil
	f.outputs = map[string]string{"codex plugin list --json": fixture(t, "codex-plugin-list-empty.json")}
	f.files = nil
	if a := ReadStatus(f.probe(), []string{"codex"}).Agents[0]; a.State != StateOff || a.Marketplace != nil {
		t.Errorf("off: %+v", a)
	}
}

func TestParseTheAgentsLists(t *testing.T) {
	if m := ParseClaudeMarketplaces(fixture(t, "claude-marketplace-list-github.json")); m == nil || m.Source != "github" {
		t.Errorf("github: %+v", m)
	}
	if m := ParseClaudeMarketplaces("[]"); m != nil {
		t.Errorf("none: %+v", m)
	}
	if got := ParseClaudePlugins("No plugins installed.", ""); len(got) != 0 {
		t.Errorf("text is not a list: %+v", got)
	}
	if got := ParseCodexPlugins(fixture(t, "codex-plugin-list-empty.json")); len(got) != 0 {
		t.Errorf("codex empty: %+v", got)
	}
}

func TestOlder(t *testing.T) {
	for _, c := range []struct {
		have, want string
		older      bool
	}{
		{"0.2.1", "0.3.0", true},
		{"0.2.1", "0.2.1", false},
		{"0.10.0", "0.9.9", false},
		{"v0.2.0", "0.2.1", true},
		{"0.2.1", "dev", false},
		{"", "0.2.1", false},
		{"0.2.1-rc1", "0.3.0", false},
	} {
		if got := Older(c.have, c.want); got != c.older {
			t.Errorf("Older(%q, %q) = %v", c.have, c.want, got)
		}
	}
}
