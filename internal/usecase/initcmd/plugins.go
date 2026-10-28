package initcmd

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// pluginName is the board plugin as Claude Code and Codex name it.
const pluginName = "board@stickypane"

// plugins says whether Claude Code and Codex have the board plugin
// installed for the project at root. It reads the agents' own records of
// what is installed and never writes them.
func plugins(root string) (claude, codex bool) {
	return claudeHasPlugin(root), codexHasPlugin()
}

// agentHome is the folder an agent keeps its settings in: the folder env
// names, or name under the home folder.
func agentHome(env, name string) string {
	if dir := os.Getenv(env); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, name)
}

// claudeHasPlugin reads Claude Code's list of installed plugins. A plugin
// installed for the user counts everywhere; one installed for a project
// counts in that project only.
func claudeHasPlugin(root string) bool {
	home := agentHome("CLAUDE_CONFIG_DIR", ".claude")
	if home == "" {
		return false
	}
	b, err := os.ReadFile(filepath.Join(home, "plugins", "installed_plugins.json"))
	if err != nil {
		return false
	}
	var list struct {
		Plugins map[string][]struct {
			Scope       string `json:"scope"`
			ProjectPath string `json:"projectPath"`
		} `json:"plugins"`
	}
	if json.Unmarshal(b, &list) != nil {
		return false
	}
	for _, p := range list.Plugins[pluginName] {
		if p.Scope == "user" || (p.ProjectPath != "" && sameDir(p.ProjectPath, root)) {
			return true
		}
	}
	return false
}

// codexHasPlugin reads Codex's config.toml for the plugin's table, and takes
// `enabled = false` under it to mean it is off.
func codexHasPlugin() bool {
	home := agentHome("CODEX_HOME", ".codex")
	if home == "" {
		return false
	}
	f, err := os.Open(filepath.Join(home, "config.toml"))
	if err != nil {
		return false
	}
	defer f.Close()
	found := false
	scan := bufio.NewScanner(f)
	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
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

// sameDir says whether a and b are the same folder, following links.
func sameDir(a, b string) bool {
	clean := func(p string) string {
		if abs, err := filepath.Abs(p); err == nil {
			p = abs
		}
		if real, err := filepath.EvalSymlinks(p); err == nil {
			p = real
		}
		return p
	}
	return clean(a) == clean(b)
}
