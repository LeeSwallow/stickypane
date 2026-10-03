// Command stickypane shows the notes in .sticky/ as a board.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/api"
	"github.com/LeeSwallow/stickypane/internal/app"
	"github.com/LeeSwallow/stickypane/internal/initcmd"
	"github.com/LeeSwallow/stickypane/internal/kinds"
	"github.com/LeeSwallow/stickypane/internal/mcp"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/theme"
	"github.com/LeeSwallow/stickypane/internal/widget/form"
	"github.com/LeeSwallow/stickypane/internal/widget/note"
)

// version is set by the release build.
var version = "dev"

const usage = `stickypane - a note board for you and your coding agent

Usage:
  stickypane [path]      open the board of the project at path (default: here)
  stickypane init        create .sticky/ and add the agent guide to
                         AGENTS.md or CLAUDE.md (--no-agent-docs skips that;
                         --skill installs it as a Claude Code skill instead)
  stickypane theme [name]  list the themes, or choose one ("auto" follows the
                         terminal's background)
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
                                 the note; a missing note is made. An item or
                                 card is named by its text, a part of it, or #2
  stickypane mv <name> <to>      rename a note, or move it: <to> is a new name, a
                                 folder (docs/) to make it a page of that book,
                                 or . for the top level
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
  stickypane mcp                 serve the notes over MCP on standard input/output
`

// waitEvery is how often wait looks at the form's file.
const waitEvery = 200 * time.Millisecond

// fileOf names a note's file in messages.
func fileOf(name string) string {
	if filepath.Ext(name) == "" {
		return name + ".md"
	}
	return name
}

const noBoard = "No .sticky directory found. Run `stickypane init` in your project first."

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
			skill := flags.Bool("skill", false, "install the guide as a Claude Code skill instead")
			if err := flags.Parse(args[1:]); err != nil {
				return 2
			}
			return report(stderr, initcmd.Run(".", initcmd.Options{NoAgentDocs: *noDocs, Skill: *skill}, stdout))
		case "guide":
			fmt.Fprint(stdout, initcmd.Guide())
			return 0
		case "version", "--version", "-v":
			fmt.Fprintln(stdout, "stickypane", version)
			return 0
		case "help", "--help", "-h":
			fmt.Fprint(stdout, usage)
			return 0
		case "list", "cat", "write", "answers", "wait", "mcp":
			return notes(args[0], args[1:], stdin, stdout, stderr)
		case "show", "hide":
			return manage(args[0], args[1:], stdout, stderr)
		case "todo", "card", "chart", "log", "set":
			return edit(args[0], args[1:], stdout, stderr)
		case "rm", "restore", "archive", "mv", "link":
			return manage(args[0], args[1:], stdout, stderr)
		case "theme":
			return themeCmd(args[1:], stdout, stderr)
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

// open returns the API for the notes folder of the project here, or the
// exit code to stop with.
func open(stderr io.Writer) (*api.API, int) {
	dir, err := store.Resolve(".")
	if errors.Is(err, store.ErrNotFound) {
		fmt.Fprintln(stderr, noBoard)
		return nil, 1
	}
	if err != nil {
		return nil, report(stderr, err)
	}
	return api.New(store.Open(dir), kinds.Default(note.Plain)), 0
}

// themeCmd lists the themes or chooses one.
func themeCmd(args []string, stdout, stderr io.Writer) int {
	if len(args) > 1 {
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
	st := store.Open(dir)
	list := func(w io.Writer) {
		current := st.Theme()
		if current == "" {
			current = "auto"
		}
		mark := func(name string) string {
			if strings.EqualFold(name, current) {
				return "* "
			}
			return "  "
		}
		fmt.Fprintf(w, "%sauto  (follows the terminal: %s or %s)\n", mark("auto"), theme.DefaultDark, theme.DefaultLight)
		for _, t := range theme.All() {
			kind := "dark"
			if !t.Dark {
				kind = "light"
			}
			fmt.Fprintf(w, "%s%s  (%s)\n", mark(t.Name), t.Name, kind)
		}
	}
	if len(args) == 0 {
		list(stdout)
		return 0
	}
	name := strings.TrimSpace(args[0])
	if strings.EqualFold(name, "auto") {
		name = "auto"
	} else if t, ok := theme.Lookup(name); ok {
		name = t.Name
	} else {
		fmt.Fprintf(stderr, "stickypane: there is no theme %q. The themes are:\n", args[0])
		list(stderr)
		return 1
	}
	if err := st.SetTheme(name); err != nil {
		return report(stderr, err)
	}
	fmt.Fprintln(stdout, "theme:", name)
	return 0
}

// manage runs the commands that remove, restore and move notes.
func manage(cmd string, args []string, stdout, stderr io.Writer) int {
	least, most := 1, 1
	switch cmd {
	case "mv":
		least, most = 2, 2
	case "link":
		most = 2
	}
	if len(args) < least || len(args) > most {
		fmt.Fprint(stderr, usage)
		return 2
	}
	a, code := open(stderr)
	if a == nil {
		return code
	}
	var out string
	var err error
	switch cmd {
	case "rm":
		out, err = a.Remove(args[0])
	case "restore":
		out, err = a.Restore(args[0])
	case "archive":
		out, err = a.Archive(args[0])
	case "show":
		out, err = a.Show(args[0])
	case "hide":
		out, err = a.Hide(args[0])
	case "link":
		name := ""
		if len(args) == 2 {
			name = args[1]
		}
		out, err = a.Link(args[0], name)
	default:
		out, err = a.Move(args[0], args[1])
	}
	if err != nil {
		return report(stderr, err)
	}
	fmt.Fprintln(stdout, out)
	return 0
}

// edit runs the commands that change one thing in a note.
func edit(cmd string, args []string, stdout, stderr io.Writer) int {
	// --to and --time may come anywhere after the note's name.
	var to string
	stamp := false
	var words []string
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--to" && i+1 < len(args):
			to = args[i+1]
			i++
		case strings.HasPrefix(args[i], "--to="):
			to = strings.TrimPrefix(args[i], "--to=")
		case args[i] == "--time" && cmd == "log":
			stamp = true
		default:
			words = append(words, args[i])
		}
	}
	bad := func() int {
		fmt.Fprint(stderr, usage)
		return 2
	}
	if len(words) < 2 {
		return bad()
	}
	name, rest := words[0], words[1:]
	a, code := open(stderr)
	if a == nil {
		return code
	}
	var out string
	var err error
	switch cmd {
	case "todo":
		if len(rest) < 2 {
			return bad()
		}
		out, err = a.Todo(name, rest[0], strings.Join(rest[1:], " "))
	case "card":
		if len(rest) < 2 || (rest[0] == "move" && to == "") {
			return bad()
		}
		out, err = a.Card(name, rest[0], strings.Join(rest[1:], " "), to)
	case "chart":
		if len(rest) < 3 {
			return bad()
		}
		out, err = a.Chart(name, rest[0], strings.Join(rest[1:len(rest)-1], " "), rest[len(rest)-1])
	case "log":
		line := strings.Join(rest, " ")
		if stamp {
			line = time.Now().Format("15:04") + " " + line
		}
		out, err = a.Log(name, line)
	default: // set
		out, err = a.Set(name, rest)
	}
	if err != nil {
		return report(stderr, err)
	}
	fmt.Fprintln(stdout, out)
	return 0
}

// notes runs the commands that read and write notes without the board.
func notes(cmd string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("stickypane "+cmd, flag.ContinueOnError)
	flags.SetOutput(stderr)
	asJSON, timeout := new(bool), new(time.Duration)
	var opts api.Options
	switch cmd {
	case "list":
		asJSON = flags.Bool("json", false, "print the list as JSON")
	case "answers", "wait":
		asJSON = flags.Bool("json", false, "print the answers as JSON")
		if cmd == "wait" {
			timeout = flags.Duration("timeout", 0, "give up after this long, such as 5m (default: wait for good)")
		}
	case "write":
		flags.StringVar(&opts.Type, "type", "", "note, board, checklist, log, chart or form")
		flags.StringVar(&opts.Title, "title", "", "the note's title")
		flags.StringVar(&opts.Size, "size", "", "page, half or card")
		flags.BoolVar(&opts.Open, "open", false, "put the note on the screen now")
	}
	// The note name comes first, flags after it.
	var name string
	needsName := cmd != "list" && cmd != "mcp"
	if needsName && len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		name, args = args[0], args[1:]
	}
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *timeout < 0 {
		fmt.Fprintln(stderr, "stickypane: --timeout cannot be negative")
		return 2
	}
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
	case "cat":
		b, err := a.Cat(name)
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
	case "answers", "wait":
		ctx := context.Background()
		if *timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, *timeout)
			defer cancel()
		}
		var got form.Answers
		if cmd == "wait" {
			got, err = a.Wait(ctx, name, waitEvery)
		} else {
			got, err = a.Answers(name)
		}
		if errors.Is(err, context.DeadlineExceeded) {
			fmt.Fprintf(stderr, "stickypane: no button was pressed on %s within %s\n", fileOf(name), *timeout)
			return 3
		}
		if err != nil {
			return report(stderr, err)
		}
		if *asJSON {
			enc := json.NewEncoder(stdout)
			enc.SetEscapeHTML(false)
			return report(stderr, enc.Encode(got))
		}
		fmt.Fprint(stdout, got.String())
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

	th := theme.NewHolder(theme.Pick(st.Theme(), true))
	m := app.New(st, kinds.Default(kinds.Markdown(th)), watch, th)
	if watchErr != nil {
		m.SetStatus("File watching is unavailable. Press r to refresh.")
	}
	_, err := tea.NewProgram(m).Run()
	return err
}
