package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	osexec "os/exec"
	"slices"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/initcmd"
)

// The agents' commands and records, replaced in tests.
var (
	setupRun   = initcmd.RunCommand
	setupCheck = initcmd.QuietCommand
	setupLook  = osexec.LookPath
	setupHas   = initcmd.InstalledPlugins
)

// setupCmd installs the board plugin for the agents found on the PATH,
// through their own commands, then sets the project up. Every choice is a
// flag; in a terminal, the ones not given are asked, and the plan is
// confirmed before anything runs.
func setupCmd(e env, args []string) error {
	fs := newFlags("setup")
	scope := fs.String("scope", "", "user, project or local (Claude Code)")
	mcp := fs.Bool("mcp", false, "also register stickypane as an MCP server")
	agents := fs.String("agents", "", "claude, codex or both, comma-separated (default: those found)")
	yes := fs.Bool("yes", false, "run without asking")
	undo := fs.Bool("undo", false, "remove what setup adds")
	dry := fs.Bool("dry-run", false, "print the plan and run nothing")
	if _, err := parse(fs, "setup", args, 0, 0); err != nil {
		return err
	}
	given := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { given[f.Name] = true })
	ask := interactive(e) && !*yes && !*dry
	in := bufio.NewReader(e.stdin)

	if ask && !given["scope"] && !*undo {
		*scope = choose(e, in, "Install for every project (user), this repository shared (project) or just you here (local)?", []string{"user", "project", "local"})
	}
	if *scope != "" && !slices.Contains([]string{"user", "project", "local"}, *scope) {
		return usagef("setup", "--scope is user, project or local, not %q", *scope)
	}
	if ask && !given["mcp"] {
		*mcp = choose(e, in, "Also register stickypane as an MCP server, for agents that prefer tools?", []string{"no", "yes"}) == "yes"
	}
	opts := initcmd.SetupOptions{Scope: *scope, MCP: *mcp, Undo: *undo}
	if *agents != "" {
		for _, a := range strings.Split(*agents, ",") {
			a = strings.TrimSpace(a)
			if a != "claude" && a != "codex" {
				return usagef("setup", "--agents names claude or codex, not %q", a)
			}
			opts.Agents = append(opts.Agents, a)
		}
	}

	root := initcmd.ProjectRoot(".")
	if root == "" {
		root = "."
	}
	steps := initcmd.PlanSetup(opts, setupLook, setupHas(root))
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
	if *undo {
		return nil
	}
	// With the plugin in place, init leaves the instruction files alone
	// and takes out a guide an earlier run put there.
	return initcmd.Run(root, initcmd.Options{}, e.stdout)
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
