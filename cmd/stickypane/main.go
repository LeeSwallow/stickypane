// Command stickypane shows the notes in .stickypane/ as a board.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/app"
	"github.com/LeeSwallow/stickypane/internal/initcmd"
	"github.com/LeeSwallow/stickypane/internal/kinds"
	"github.com/LeeSwallow/stickypane/internal/store"
)

// version is set by the release build.
var version = "dev"

const usage = `stickypane - a sticky-note board for you and your coding agent

Usage:
  stickypane [path]      open the board of the project at path (default: here)
  stickypane init        create .stickypane/ and add the agent guide to
                         AGENTS.md or CLAUDE.md (--no-agent-docs skips that)
  stickypane guide       print the agent guide
  stickypane version     print the version
`

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

// run executes the command line and returns the exit code: 0 on success,
// 1 when something failed, 2 when the arguments make no sense.
func run(args []string, stdout, stderr io.Writer) int {
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
			if err := initcmd.Run(".", initcmd.Options{NoAgentDocs: *noDocs}, stdout); err != nil {
				fmt.Fprintln(stderr, "stickypane:", err)
				return 1
			}
			return 0
		case "guide":
			fmt.Fprint(stdout, initcmd.Guide())
			return 0
		case "version", "--version", "-v":
			fmt.Fprintln(stdout, "stickypane", version)
			return 0
		case "help", "--help", "-h":
			fmt.Fprint(stdout, usage)
			return 0
		}
		if strings.HasPrefix(args[0], "-") || len(args) > 1 {
			fmt.Fprint(stderr, usage)
			return 2
		}
		start = args[0]
	}

	dir, err := store.Resolve(start)
	if errors.Is(err, store.ErrNotFound) {
		fmt.Fprintln(stderr, "No .stickypane directory found. Run `stickypane init` in your project first.")
		return 1
	}
	if err != nil {
		fmt.Fprintln(stderr, "stickypane:", err)
		return 1
	}
	if err := board(dir); err != nil {
		fmt.Fprintln(stderr, "stickypane:", err)
		return 1
	}
	return 0
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
