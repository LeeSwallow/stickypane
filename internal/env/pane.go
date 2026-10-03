package env

import "strings"

// Pane is the tool that can open a pane next to the agent's, or none.
type Pane struct {
	Name  string // "tmux", "zellij", "wezterm", "windows-terminal", or ""
	split string // the command, with %s for what runs in the new pane
}

// The pane tools, in the order they are looked for: a multiplexer inside a
// terminal is the one to split, so tmux and Zellij come before the
// terminals that can split themselves.
var panes = []struct {
	name, split string
	is          func(getenv func(string) string) bool
}{
	{"tmux", "tmux split-window -h %s", func(g func(string) string) bool { return g("TMUX") != "" }},
	{"zellij", "zellij run --direction right -- %s", func(g func(string) string) bool { return g("ZELLIJ") != "" }},
	{"wezterm", "wezterm cli split-pane --right -- %s", func(g func(string) string) bool {
		return g("WEZTERM_PANE") != "" || g("TERM_PROGRAM") == "WezTerm"
	}},
	{"windows-terminal", "wt -w 0 split-pane -V %s", func(g func(string) string) bool { return g("WT_SESSION") != "" }},
}

func detectPane(getenv func(string) string) Pane {
	for _, p := range panes {
		if p.is(getenv) {
			return Pane{Name: p.name, split: p.split}
		}
	}
	return Pane{}
}

// Split returns the command that opens command in a pane to the right, or
// "" when no tool here can.
func (p Pane) Split(command string) string {
	if p.split == "" {
		return ""
	}
	return strings.ReplaceAll(p.split, "%s", command)
}
