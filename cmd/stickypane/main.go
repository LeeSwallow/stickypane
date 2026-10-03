// Command stickypane shows the notes in .sticky/ as a board.
//
// The files of this package: main.go starts the program and opens the
// board, commands.go is the table of commands, errors.go turns errors into
// exit codes, flags.go parses flags between words, and notes.go, edit.go,
// settings.go and kinds.go hold the commands themselves.
package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/app"
	"github.com/LeeSwallow/stickypane/internal/initcmd"
	"github.com/LeeSwallow/stickypane/internal/kinds"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/theme"
)

// version is set by the release build.
var version = "dev"

const usage = `stickypane - a note board for you and your coding agent

Usage:
  stickypane [path]      open the board of the project at path (default: here);
                         in a git repository without one, set it up first
  stickypane init        make .sticky/ and put the agent guide where your agents
                         read it: CLAUDE.md, AGENTS.md or a .claude skill, or
                         nowhere when the board plugin is installed. Safe to run
                         again. --skill: a skill instead of the instruction
                         files; --no-agent-docs: the folder only
  stickypane setup [--scope user|project|local] [--mcp] [--agents claude,codex]
                         [--undo] [--dry-run] [--yes]
                         install the board plugin for Claude Code and Codex
                         through their own commands; asks in a terminal
  stickypane theme [name]  list the themes, or choose one ("auto" follows the
                         terminal's background)
  stickypane language [code]  list the languages of the screen, or choose one
                         ("auto" follows LANG); the agent guide stays English
  stickypane kinds [shape]  list the shapes a note can take, or show one in
                         full: the command, a file to copy, how it behaves
  stickypane env [--json]  say what the board found about where it runs: the
                         system, the shell, how to open a pane beside the
                         agent, the screen's language and how scripts run
  stickypane guide       print the agent guide
  stickypane version     print the version

For scripts and agents that would rather not edit the files themselves:
  stickypane list [--json]       list the notes
  stickypane show <name|path>    put a note on the screen; a path to any file or
                                 folder of the project is linked onto the board
  stickypane hide <name>         fold a note away
  stickypane cat <name>          print a note's file
  stickypane write <name> [--type T] [--title X] [--open] [--size S]
                                 create or replace a note from standard input
  stickypane todo <name> add|check|uncheck <item>
  stickypane card <name> add|move <card> [--to <column>]
  stickypane chart <name> set|add <label> <number>
  stickypane log <name> [--time] <text>
  stickypane set <name> key=value ...
                                 change one thing without reading or rewriting
                                 the note; a missing note is made, and so is the
                                 board in a git repository without one. An item
                                 or card is named by its text, a part of it, or #2
  stickypane mv <name> <to>      rename a note, or move it: <to> is a new name, a
                                 folder (deploy/) to move it into that tab or
                                 book, or . for the top level
  stickypane link <path> [name]  show a file or a folder of the project on the
                                 board without copying it (a symbolic link)
  stickypane rm <name>           move a note, a page or a folder to .trash/
  stickypane restore <name>      bring back what rm removed last under that name
  stickypane archive <name>      move a note out of sight into archive/
  stickypane answers <name> [--json]
                                 print what the user answered in a form
  stickypane wait <name> [--timeout 5m] [--json]
                                 wait until the user presses a button of a form,
                                 then print the answers (exit code 3 on timeout)
  stickypane api <name> [request] [--env E] [--all]
                                 (experimental) list the requests of a .http
                                 note, or send one (by its name, a part of it
                                 or #2) or all, and print the response, its
                                 checks and captures
  stickypane watch [--note N] [--type T] [--json] [--exec CMD] [--once] [--timeout 10m]
                                 print what happens on the board as it happens:
                                 note.created, item.ticked, card.moved,
                                 form.submitted, log.appended, chart.changed...;
                                 --exec runs CMD for each, with STICKY_* set
  stickypane mcp                 serve the notes over MCP on standard input/output
`

// waitEvery is how often wait looks at the form's file.
const waitEvery = 200 * time.Millisecond

// graceAfterSignal is how long a command may take to stop after Ctrl-C or
// SIGTERM before the program leaves anyway: reading standard input, as the
// MCP server does, cannot be interrupted.
const graceAfterSignal = 2 * time.Second

// fileOf names a note's file in messages.
func fileOf(name string) string {
	if filepath.Ext(name) == "" {
		return name + ".md"
	}
	return name
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ctx.Done()
		stop() // a second signal stops the program at once
		time.Sleep(graceAfterSignal)
		os.Exit(exitInterrupted)
	}()
	os.Exit(runCtx(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run executes the command line and returns the exit code; see exitCode.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	return runCtx(context.Background(), args, stdin, stdout, stderr)
}

// runCtx is run with a context that ends when the user interrupts.
func runCtx(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	e := env{ctx: ctx, stdin: stdin, stdout: stdout, stderr: stderr}
	if len(args) > 0 {
		if c, ok := lookup(args[0]); ok {
			return exitCode(c.run(e, args[1:]), stderr)
		}
		if strings.HasPrefix(args[0], "-") || len(args) > 1 {
			return exitCode(usageError{msg: "unknown command " + args[0]}, stderr)
		}
	}
	start := "."
	if len(args) == 1 {
		start = args[0]
	}
	return exitCode(openBoard(e, start), stderr)
}

// openBoard opens the board of the project at start. In a repository
// without one, it sets it up first, as init would, so that init is a step
// nobody has to know about.
func openBoard(e env, start string) error {
	dir, err := store.Resolve(start)
	var setup bytes.Buffer
	if errors.Is(err, store.ErrNotFound) {
		root := initcmd.ProjectRoot(start)
		if root == "" {
			return errNoBoard
		}
		if err := initcmd.Run(root, initcmd.Options{}, &setup); err != nil {
			return err
		}
		dir, err = store.Resolve(root)
	}
	if err != nil {
		return err
	}
	// What was set up, without the last line: the next step was this one.
	lines := strings.Split(strings.TrimRight(setup.String(), "\n"), "\n")
	done := strings.Join(lines[:len(lines)-1], " ")
	if err := runBoard(e.ctx, dir, done); err != nil {
		return err
	}
	if done != "" {
		_, err = io.WriteString(e.stdout, strings.Join(lines[:len(lines)-1], "\n")+"\n")
	}
	return err
}

// runBoard is the terminal UI; tests put a stand-in here.
var runBoard = board

// board runs the terminal UI until the user quits or ctx ends. Markdown
// starts with dark colors and switches when the terminal reports a light
// background. status, when there is one, is shown on the status line at the
// start.
func board(ctx context.Context, dir, status string) error {
	st := store.Open(dir)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	watch, watchErr := st.Watch(ctx)

	th := theme.NewHolder(theme.Pick(st.Theme(), true))
	m := app.New(st, kinds.Default(kinds.Markdown(th)), watch, th)
	if status != "" {
		m.SetStatus(status)
	}
	if watchErr != nil {
		m.SetStatus(app.Message("WatchUnavailable"))
	}
	_, err := tea.NewProgram(m, tea.WithContext(ctx)).Run()
	if errors.Is(err, tea.ErrProgramKilled) && ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}
