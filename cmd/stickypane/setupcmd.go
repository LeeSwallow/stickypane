package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/usecase/initcmd"
)

// The agents' commands and records, replaced in tests.
var (
	setupRun    = initcmd.RunCommand
	setupCheck  = initcmd.QuietCommand
	setupStatus = func(root string, agents []string) initcmd.Status {
		return initcmd.ReadStatus(initcmd.SystemProbe(root), agents)
	}
)

// agentNames are the agents as people know them.
var agentNames = map[string]string{"claude": "Claude Code", "codex": "Codex"}

// setupCmd installs the board plugin for the agents found on the PATH,
// through their own commands: where it is missing, older than this
// stickypane's, or broken by a marketplace whose folder is gone. It makes
// no board. Every choice is a flag; in a terminal, the ones not given are
// asked, and the plan is confirmed before anything runs. --status says
// where each agent stands and runs nothing.
func setupCmd(e env, args []string) error {
	fs := newFlags("setup")
	scope := fs.String("scope", "", "user, project or local (Claude Code)")
	mcp := fs.Bool("mcp", false, "also register stickypane as an MCP server")
	plugin := fs.Bool("plugin", true, "install, update or remove the board plugin")
	agents := fs.String("agents", "", "claude, codex or both, comma-separated (default: those found)")
	yes := fs.Bool("yes", false, "run without asking")
	undo := fs.Bool("undo", false, "uninstall the plugin (and the MCP server with --mcp)")
	dry := fs.Bool("dry-run", false, "print the plan and run nothing")
	command := fs.String("command", "", "the stickypane the MCP server runs (default: stickypane on the PATH)")
	status := fs.Bool("status", false, "say where each agent stands, and run nothing")
	asJSON := fs.Bool("json", false, "with --status, print it as JSON")
	if _, err := parse(fs, "setup", args, 0, 0); err != nil {
		return err
	}
	given := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { given[f.Name] = true })
	if *asJSON && !*status {
		return usagef("setup", "--json goes with --status")
	}
	if *scope != "" && !slices.Contains([]string{"user", "project", "local"}, *scope) {
		return usagef("setup", "--scope is user, project or local, not %q", *scope)
	}
	if *command != "" && !filepath.IsAbs(*command) {
		return usagef("setup", "--command is a full path, not %q", *command)
	}
	if !*plugin && !*mcp {
		return usagef("setup", "--plugin=false leaves only the MCP server: add --mcp")
	}
	opts := initcmd.SetupOptions{Scope: *scope, MCP: *mcp, NoPlugin: !*plugin, Undo: *undo, Command: *command}
	if *agents != "" {
		for _, a := range strings.Split(*agents, ",") {
			a = strings.TrimSpace(a)
			if a != "claude" && a != "codex" {
				return usagef("setup", "--agents names claude or codex, not %q", a)
			}
			opts.Agents = append(opts.Agents, a)
		}
	}
	wanted := opts.Agents
	if len(wanted) == 0 {
		wanted = []string{"claude", "codex"}
	}
	// The project's scopes are the project's around the current folder, or
	// the folder itself, as the agents take it.
	root := initcmd.ProjectRoot(".")
	if root == "" {
		root, _ = filepath.Abs(".")
	}
	st := setupStatus(root, wanted)

	if *status {
		if *asJSON {
			enc := json.NewEncoder(e.stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(st)
		}
		printStatus(e, st)
		return nil
	}

	ask := interactive(e) && !*yes && !*dry
	in := bufio.NewReader(e.stdin)
	if ask && !given["scope"] && !*undo {
		opts.Scope = choose(e, in, "Install for every project (user), this repository shared (project) or just you here (local)?", []string{"user", "project", "local"})
	}
	if ask && !given["mcp"] {
		opts.MCP = choose(e, in, "Also register stickypane as an MCP server, for agents that prefer tools?", []string{"no", "yes"}) == "yes"
	}

	found := slices.ContainsFunc(st.Agents, func(a initcmd.AgentStatus) bool { return a.Found })
	if !found {
		fmt.Fprintf(e.stdout, "No agent found: no %s command on the PATH. Install Claude Code or Codex first, or give any other agent the guide with `stickypane init` in a project.\n", strings.Join(wanted, " or "))
		return nil
	}
	steps := initcmd.PlanSetup(opts, st)
	if len(steps) == 0 {
		fmt.Fprintln(e.stdout, "Nothing to do: the board plugin is set up for every agent found.")
	} else {
		fmt.Fprintln(e.stdout, "stickypane setup runs:")
		for _, s := range steps {
			fmt.Fprintf(e.stdout, "  %s\n", strings.Join(s.Cmd, " "))
		}
	}
	switch {
	case *dry || len(steps) == 0 && *undo:
		return nil
	case len(steps) > 0 && !*yes && !ask:
		fmt.Fprintln(e.stdout, "Run `stickypane setup --yes` to do it, with the same flags.")
		return nil
	case len(steps) > 0 && ask && choose(e, in, "Run these?", []string{"yes", "no"}) != "yes":
		return nil
	}
	if err := initcmd.RunSetup(steps, setupRun, setupCheck, e.stdout); err != nil {
		return err
	}
	if !*undo {
		// setup teaches the agents and makes no board: it may run from a
		// folder that is no project at all, such as / or $HOME.
		fmt.Fprintln(e.stdout, "Next: `stickypane init` in a project makes its board; in a git repository the agent's first `stickypane show` does too.")
	}
	return nil
}

// printStatus says where each agent stands, a few lines each.
func printStatus(e env, st initcmd.Status) {
	fmt.Fprintf(e.stdout, "This stickypane carries the board plugin %s.\n", st.Plugin)
	for _, a := range st.Agents {
		name := agentNames[a.Agent]
		if !a.Found {
			fmt.Fprintf(e.stdout, "\n%s: not found (no %s command on the PATH)\n", name, a.Agent)
			continue
		}
		fmt.Fprintf(e.stdout, "\n%s: %s\n", name, map[string]string{
			initcmd.StateOff:      "not connected",
			initcmd.StateOK:       "connected",
			initcmd.StateOutdated: "an update is waiting",
			initcmd.StateBroken:   "broken",
		}[a.State])
		var plugin, mcp []string
		for _, s := range a.Scopes {
			if p, ok := a.Plugin[s]; ok {
				word := p.Version
				if p.State != initcmd.StateOK {
					word += " (" + p.State + ")"
				}
				plugin = append(plugin, s+" "+strings.TrimSpace(word))
			}
			if m, ok := a.MCP[s]; ok {
				mcp = append(mcp, s+" "+m.Command)
			}
		}
		fmt.Fprintf(e.stdout, "  plugin       %s\n", orNone(plugin))
		fmt.Fprintf(e.stdout, "  MCP server   %s\n", orNone(mcp))
		if m := a.Marketplace; m != nil {
			line := m.Source + " " + m.Where
			if m.Gone {
				line += " (the folder is gone)"
			}
			fmt.Fprintf(e.stdout, "  marketplace  %s\n", line)
		}
		for _, s := range a.Scopes {
			if p, ok := a.Plugin[s]; ok && p.Error != "" {
				fmt.Fprintf(e.stdout, "  error        %s\n", p.Error)
				break
			}
		}
		switch {
		case a.Marketplace != nil && a.Marketplace.Gone:
			fmt.Fprintf(e.stdout, "  `stickypane setup --agents %s` points the marketplace at GitHub again and installs the plugin anew.\n", a.Agent)
		case a.State == initcmd.StateOutdated || a.State == initcmd.StateBroken:
			fmt.Fprintf(e.stdout, "  `stickypane setup --agents %s` updates it.\n", a.Agent)
		}
	}
}

func orNone(parts []string) string {
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ", ")
}

// choose asks a question with options, the first being the default, and
// returns the answer: an option, or the first letters of one.
func choose(e env, in *bufio.Reader, question string, options []string) string {
	fmt.Fprintf(e.stdout, "%s [%s] ", question, strings.Join(options, "/"))
	line, _ := in.ReadString('\n')
	line = strings.ToLower(strings.TrimSpace(line))
	for _, o := range options {
		if line != "" && strings.HasPrefix(o, line) {
			return o
		}
	}
	return options[0]
}

// interactive reports whether standard input is a terminal someone types
// into, as opposed to a pipe or a test's buffer.
func interactive(e env) bool {
	f, ok := e.stdin.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}
