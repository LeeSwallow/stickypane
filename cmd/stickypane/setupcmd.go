package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/core/i18n"
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
// where each agent stands and runs nothing. It speaks the person's
// language; the commands it runs and --json stay as they are.
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
		return usageSay("setup", "setup.jsonNeedsStatus", nil)
	}
	if *scope != "" && !slices.Contains([]string{"user", "project", "local"}, *scope) {
		return usageSay("setup", "setup.badScope", map[string]any{"Value": strconv.Quote(*scope)})
	}
	if *command != "" && !filepath.IsAbs(*command) {
		return usageSay("setup", "setup.badCommand", map[string]any{"Value": strconv.Quote(*command)})
	}
	if !*plugin && !*mcp {
		return usageSay("setup", "setup.pluginNeedsMCP", nil)
	}
	opts := initcmd.SetupOptions{Scope: *scope, MCP: *mcp, NoPlugin: !*plugin, Undo: *undo, Command: *command}
	if *agents != "" {
		for _, a := range strings.Split(*agents, ",") {
			a = strings.TrimSpace(a)
			if a != "claude" && a != "codex" {
				return usageSay("setup", "setup.badAgent", map[string]any{"Value": strconv.Quote(a)})
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
	tr := helpStrings()

	if *status {
		if *asJSON {
			enc := json.NewEncoder(e.stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(st)
		}
		printStatus(e, tr, st)
		return nil
	}

	ask := interactive(e) && !*yes && !*dry
	in := bufio.NewReader(e.stdin)
	if ask && !given["scope"] && !*undo {
		opts.Scope = choose(e, in, tr.Say("setup.askScope", nil), []string{"user", "project", "local"})
	}
	if ask && !given["mcp"] {
		opts.MCP = choose(e, in, tr.Say("setup.askMCP", nil), []string{"no", "yes"}) == "yes"
	}

	found := slices.ContainsFunc(st.Agents, func(a initcmd.AgentStatus) bool { return a.Found })
	if !found {
		fmt.Fprintln(e.stdout, tr.Say("setup.noAgent", map[string]any{"Commands": tr.List(wanted, true)}))
		return nil
	}
	steps := initcmd.PlanSetup(opts, st)
	if len(steps) == 0 {
		fmt.Fprintln(e.stdout, tr.Say("setup.nothing", nil))
	} else {
		fmt.Fprintln(e.stdout, tr.Say("setup.runs", nil))
		for _, s := range steps {
			fmt.Fprintf(e.stdout, "  %s\n", strings.Join(s.Cmd, " "))
		}
	}
	switch {
	case *dry || len(steps) == 0 && *undo:
		return nil
	case len(steps) > 0 && !*yes && !ask:
		fmt.Fprintln(e.stdout, tr.Say("setup.toRun", nil))
		return nil
	case len(steps) > 0 && ask && choose(e, in, tr.Say("setup.askRun", nil), []string{"yes", "no"}) != "yes":
		return nil
	}
	if err := initcmd.RunSetup(steps, setupRun, setupCheck, e.stdout, tr); err != nil {
		return err
	}
	if !*undo {
		// setup teaches the agents and makes no board: it may run from a
		// folder that is no project at all, such as / or $HOME.
		fmt.Fprintln(e.stdout, tr.Say("setup.next", nil))
	}
	return nil
}

// printStatus says where each agent stands, a few lines each, in tr's
// language; versions, scopes, commands and paths stay as they are.
func printStatus(e env, tr i18n.Strings, st initcmd.Status) {
	labels := []string{tr.Say("setup.plugin", nil), tr.Say("setup.mcp", nil), tr.Say("setup.marketplace", nil), tr.Say("setup.error", nil)}
	width := 0
	for _, l := range labels {
		width = max(width, ansi.StringWidth(l))
	}
	row := func(label, value string) {
		fmt.Fprintf(e.stdout, "  %s%s  %s\n", label, strings.Repeat(" ", width-ansi.StringWidth(label)), value)
	}
	fmt.Fprintln(e.stdout, tr.Say("setup.carries", map[string]any{"Version": st.Plugin}))
	for _, a := range st.Agents {
		name := agentNames[a.Agent]
		if !a.Found {
			fmt.Fprintf(e.stdout, "\n%s\n", tr.Say("setup.notFound", map[string]any{"Name": name, "Command": a.Agent}))
			continue
		}
		fmt.Fprintf(e.stdout, "\n%s: %s\n", name, tr.Say("setup.state."+a.State, nil))
		var plugin, mcp []string
		for _, s := range a.Scopes {
			if p, ok := a.Plugin[s]; ok {
				word := p.Version
				if p.State != initcmd.StateOK {
					word += " (" + tr.Say("setup.word."+p.State, nil) + ")"
				}
				plugin = append(plugin, s+" "+strings.TrimSpace(word))
			}
			if m, ok := a.MCP[s]; ok {
				mcp = append(mcp, s+" "+m.Command)
			}
		}
		row(labels[0], orNone(tr, plugin))
		row(labels[1], orNone(tr, mcp))
		if m := a.Marketplace; m != nil {
			line := m.Source + " " + m.Where
			if m.Gone {
				line += " " + tr.Say("setup.folderGone", nil)
			}
			row(labels[2], line)
		}
		for _, s := range a.Scopes {
			if p, ok := a.Plugin[s]; ok && p.Error != "" {
				row(labels[3], p.Error)
				break
			}
		}
		switch {
		case a.Marketplace != nil && a.Marketplace.Gone:
			fmt.Fprintf(e.stdout, "  %s\n", tr.Say("setup.fixGone", map[string]any{"Agent": a.Agent}))
		case a.State == initcmd.StateOutdated || a.State == initcmd.StateBroken:
			fmt.Fprintf(e.stdout, "  %s\n", tr.Say("setup.fixUpdate", map[string]any{"Agent": a.Agent}))
		}
	}
}

func orNone(tr i18n.Strings, parts []string) string {
	if len(parts) == 0 {
		return tr.Say("setup.none", nil)
	}
	return strings.Join(parts, ", ")
}

// choose asks a question with options, the first being the default, and
// returns the answer: an option, or the first letters of one. The options
// are what is typed, so they stay as they are in every language.
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
