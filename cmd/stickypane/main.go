// Command stickypane shows the notes in .stickypane/ as a board.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/api"
	"github.com/LeeSwallow/stickypane/internal/app"
	"github.com/LeeSwallow/stickypane/internal/initcmd"
	"github.com/LeeSwallow/stickypane/internal/kinds"
	"github.com/LeeSwallow/stickypane/internal/mcp"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/widget/note"
)

// version is set by the release build.
var version = "dev"

const usage = `stickypane - a note board for you and your coding agent

Usage:
  stickypane [path]      open the board of the project at path (default: here)
  stickypane init        create .stickypane/ and add the agent guide to
                         AGENTS.md or CLAUDE.md (--no-agent-docs skips that)
  stickypane guide       print the agent guide
  stickypane version     print the version

For scripts and agents that would rather not edit the files themselves:
  stickypane list [--json]       list the notes
  stickypane show <name>         print a note's file
  stickypane write <name> [--type T] [--title X] [--open] [--size S]
                                 create or replace a note from standard input
  stickypane mcp                 serve the notes over MCP on standard input/output
`

const noBoard = "No .stickypane directory found. Run `stickypane init` in your project first."

func main() { os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }

// run executes the command line and returns the exit code: 0 on success,
// 1 when something failed, 2 when the arguments make no sense.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	start := "."
	if len(args) > 0 {
		switch args[0] {
		case "init":
			flags := flag.NewFlagSet("stickypane init", flag.ContinueOnError)
			flags.SetOutput(stderr)
			noDocs := flags.Bool("no-agent-docs", false, "leave AGENTS.md and CLAUDE.md alone")
			if err := flags.Parse(args[1:]); err != nil {
				return 2
			}
			return report(stderr, initcmd.Run(".", initcmd.Options{NoAgentDocs: *noDocs}, stdout))
		case "guide":
			fmt.Fprint(stdout, initcmd.Guide())
			return 0
		case "version", "--version", "-v":
			fmt.Fprintln(stdout, "stickypane", version)
			return 0
		case "help", "--help", "-h":
			fmt.Fprint(stdout, usage)
			return 0
		case "list", "show", "write", "mcp":
			return notes(args[0], args[1:], stdin, stdout, stderr)
		}
		if strings.HasPrefix(args[0], "-") || len(args) > 1 {
			fmt.Fprint(stderr, usage)
			return 2
		}
		start = args[0]
	}

	dir, err := store.Resolve(start)
	if errors.Is(err, store.ErrNotFound) {
		fmt.Fprintln(stderr, noBoard)
		return 1
	}
	if err != nil {
		return report(stderr, err)
	}
	return report(stderr, board(dir))
}

// report prints an error, if any, and returns the exit code for it.
func report(stderr io.Writer, err error) int {
	if err != nil {
		fmt.Fprintln(stderr, "stickypane:", err)
		return 1
	}
	return 0
}

// notes runs the commands that read and write notes without the board.
func notes(cmd string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("stickypane "+cmd, flag.ContinueOnError)
	flags.SetOutput(stderr)
	asJSON := new(bool)
	var opts api.Options
	switch cmd {
	case "list":
		asJSON = flags.Bool("json", false, "print the list as JSON")
	case "write":
		flags.StringVar(&opts.Type, "type", "", "note, board, checklist or log")
		flags.StringVar(&opts.Title, "title", "", "the note's title")
		flags.StringVar(&opts.Size, "size", "", "page, half or card")
		flags.BoolVar(&opts.Open, "open", false, "put the note on the screen now")
	}
	// The note name comes first, flags after it.
	var name string
	if (cmd == "show" || cmd == "write") && len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		name, args = args[0], args[1:]
	}
	if err := flags.Parse(args); err != nil {
		return 2
	}
	needsName := cmd == "show" || cmd == "write"
	if flags.NArg() > 0 || (needsName && name == "") {
		fmt.Fprint(stderr, usage)
		return 2
	}

	dir, err := store.Resolve(".")
	if errors.Is(err, store.ErrNotFound) {
		fmt.Fprintln(stderr, noBoard)
		return 1
	}
	if err != nil {
		return report(stderr, err)
	}
	a := api.New(store.Open(dir), kinds.Default(note.Plain))

	switch cmd {
	case "list":
		infos, err := a.List()
		if err != nil {
			return report(stderr, err)
		}
		if *asJSON {
			enc := json.NewEncoder(stdout)
			enc.SetEscapeHTML(false)
			return report(stderr, enc.Encode(infos))
		}
		w := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
		for _, n := range infos {
			state := "closed"
			if n.Open {
				state = "open"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", n.Name, n.Type, n.Size, state, n.Title)
		}
		return report(stderr, w.Flush())
	case "show":
		b, err := a.Show(name)
		if err != nil {
			return report(stderr, err)
		}
		_, err = stdout.Write(b)
		return report(stderr, err)
	case "write":
		body, err := io.ReadAll(stdin)
		if err != nil {
			return report(stderr, err)
		}
		file, err := a.Write(name, opts, body)
		if err != nil {
			return report(stderr, err)
		}
		fmt.Fprintln(stdout, "wrote", file)
		return 0
	default: // mcp
		server := &mcp.Server{API: a, Guide: initcmd.GuideText(), Version: version}
		return report(stderr, server.Serve(stdin, stdout))
	}
}

// board runs the terminal UI until the user quits. Markdown starts with dark
// colors and switches when the terminal reports a light background.
func board(dir string) error {
	st := store.Open(dir)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	watch, watchErr := st.Watch(ctx)

	theme := &kinds.Theme{Dark: true}
	m := app.New(st, kinds.Default(kinds.Markdown(theme)), watch)
	m.OnBackground = func(dark bool) { theme.Dark = dark }
	if watchErr != nil {
		m.SetStatus("File watching is unavailable. Press r to refresh.")
	}
	_, err := tea.NewProgram(m).Run()
	return err
}
