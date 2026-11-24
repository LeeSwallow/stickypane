package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/LeeSwallow/stickypane/internal/adapter/registry"
	"github.com/LeeSwallow/stickypane/internal/adapter/store"
	"github.com/LeeSwallow/stickypane/internal/core/i18n"
	"github.com/LeeSwallow/stickypane/internal/core/kinds"
	"github.com/LeeSwallow/stickypane/internal/core/widget/note"
	"github.com/LeeSwallow/stickypane/internal/surface/bridge"
	"github.com/LeeSwallow/stickypane/internal/surface/mcp"
	"github.com/LeeSwallow/stickypane/internal/usecase/api"
	"github.com/LeeSwallow/stickypane/internal/usecase/initcmd"
	"github.com/LeeSwallow/stickypane/internal/usecase/worktree"
)

// open returns the API for the notes folder of the project here. With
// create, a command that puts something on the board makes the folder at
// the top of the repository when there is none, so an agent can start a
// board without a setup step; it says so on stderr.
func open(e env, create bool) (*api.API, error) {
	dir, err := store.Resolve(".")
	if errors.Is(err, store.ErrNotFound) && create {
		if root := initcmd.ProjectRoot("."); root != "" {
			if dir, _, err = initcmd.MakeBoard(root, ""); err == nil {
				fmt.Fprintf(e.stderr, "stickypane: made %s for this project\n", dir)
			}
		}
	}
	if errors.Is(err, store.ErrNotFound) {
		return nil, errNoBoard
	}
	if err != nil {
		return nil, err
	}
	if changes[e.cmd] {
		registry.Touch(filepath.Dir(dir), e.cmd)
	}
	st, _ := followWorktrees(dir, e.now)
	return api.NewWithClock(st, kinds.Default(note.Plain), e.now), nil
}

// followWorktrees opens the board dir in step with its repository's git
// worktrees: what changed is recorded and the folders closed or opened
// again (best effort: a command does not fail for it), and a first write
// into worktrees/<name>/ makes that worktree's folder.
func followWorktrees(dir string, now func() time.Time) (*store.Store, []worktree.Worktree) {
	st := store.Open(dir)
	b := worktree.Board{Store: st, Records: registry.Default(), Now: now}
	list, _ := b.Sync()
	st.NewFolder = b.NewFolder
	return st, list
}

// worktreeCmd lists the worktrees of the repository as the board follows
// them, or shows one.
func worktreeCmd(verb string) func(env, []string) error {
	return func(e env, args []string) error {
		fs := newFlags("worktree " + verb)
		asJSON := fs.Bool("json", false, "print it as JSON")
		n := 0
		if verb == "show" {
			n = 1
		}
		words, err := parse(fs, "worktree "+verb, args, n, n)
		if err != nil {
			return err
		}
		dir, err := store.Resolve(".")
		if errors.Is(err, store.ErrNotFound) {
			return errNoBoard
		}
		if err != nil {
			return err
		}
		_, list := followWorktrees(dir, e.now)
		if verb == "show" {
			i := slices.IndexFunc(list, func(w worktree.Worktree) bool { return w.Name == words[0] })
			if i < 0 {
				return i18n.Err("worktree.unknown", map[string]any{"Name": words[0]})
			}
			if *asJSON {
				return printJSON(e.stdout, list[i])
			}
			w := list[i]
			fmt.Fprintf(e.stdout, "name: %s\nstate: %s\nbranch: %s\npath: %s\nfolder: %s\nwaiting: %d\n",
				w.Name, shownState(w), w.Branch, w.Path, worktree.Tab+"/"+w.Name+"/", w.Waiting)
			return nil
		}
		if *asJSON {
			return printJSON(e.stdout, list)
		}
		tw := tabwriter.NewWriter(e.stdout, 0, 0, 2, ' ', 0)
		for _, w := range list {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%s\n", w.Name, shownState(w), w.Branch, w.Waiting, w.Path)
		}
		return tw.Flush()
	}
}

// shownState is a worktree's state with running for a locked open one.
func shownState(w worktree.Worktree) string {
	if w.Running {
		return "running"
	}
	return w.State
}

// printJSON writes v as JSON without escaping HTML.
func printJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// listCmd lists the notes, as a table or as JSON.
func listCmd(e env, args []string) error {
	fs := newFlags("note list")
	asJSON := fs.Bool("json", false, "print the list as JSON")
	if _, err := parse(fs, "note list", args, 0, 0); err != nil {
		return err
	}
	a, err := open(e, false)
	if err != nil {
		return err
	}
	infos, err := a.List()
	if err != nil {
		return err
	}
	if *asJSON {
		return printJSON(e.stdout, infos)
	}
	w := tabwriter.NewWriter(e.stdout, 0, 0, 2, ' ', 0)
	for _, n := range infos {
		state := "closed"
		if n.Open {
			state = "open"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", n.Name, n.Type, n.Size, state, n.Title)
	}
	return w.Flush()
}

// catCmd prints a note's file.
func catCmd(e env, args []string) error {
	words, err := parse(newFlags("note cat"), "note cat", args, 1, 1)
	if err != nil {
		return err
	}
	a, err := open(e, false)
	if err != nil {
		return err
	}
	b, err := a.Cat(words[0])
	if err != nil {
		return err
	}
	_, err = e.stdout.Write(b)
	return err
}

// writeCmd creates or replaces a note from standard input.
func writeCmd(e env, args []string) error {
	fs := newFlags("note write")
	var opts api.Options
	fs.StringVar(&opts.Type, "type", "", "note, board, checklist, log, chart or form")
	fs.StringVar(&opts.Title, "title", "", "the note's title")
	fs.StringVar(&opts.Size, "size", "", "the width: 1/4, 1/3, 1/2, 2/3, 3/4 or 1 of the screen (or card, half, page)")
	fs.BoolVar(&opts.Open, "open", false, "put the note on the screen now")
	var given map[string]string
	fs.Func("opt", "a create option of the shape, name=value; the shape writes the note and nothing is read from standard input", func(v string) error {
		name, value, ok := strings.Cut(v, "=")
		if !ok || name == "" {
			return errors.New("an option is name=value")
		}
		if given == nil {
			given = map[string]string{}
		}
		if before, ok := given[name]; ok {
			value = before + "\n" + value // repeated, it is several lines
		}
		given[name] = value
		return nil
	})
	words, err := parse(fs, "note write", args, 1, 1)
	if err != nil {
		return err
	}
	a, err := open(e, true)
	if err != nil {
		return err
	}
	name := words[0]
	var body []byte
	if given != nil {
		var ext string
		if body, ext, err = a.Make(opts.Type, opts.Title, given); err != nil {
			return err
		}
		if ext != ".md" {
			// A file of its own, a script or a page: its extension says
			// what it is, and it has no front matter for a type or title.
			if filepath.Ext(name) == "" {
				name += ext
			}
			opts.Type, opts.Title = "", ""
		}
	} else if body, err = io.ReadAll(e.stdin); err != nil {
		return err
	}
	file, err := a.Write(name, opts, body)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(e.stdout, "wrote", file)
	return err
}

// answersCmd prints what the user answered in a form; with wait, it first
// waits until a button is pressed.
func answersCmd(wait bool) func(env, []string) error {
	cmd := "answers"
	if wait {
		cmd = "wait"
	}
	return func(e env, args []string) error {
		fs := newFlags(cmd)
		asJSON := fs.Bool("json", false, "print the answers as JSON")
		timeout := new(time.Duration)
		if wait {
			timeout = fs.Duration("timeout", 0, "give up after this long, such as 5m (default: wait for good)")
		}
		words, err := parse(fs, cmd, args, 1, 1)
		if err != nil {
			return err
		}
		if *timeout < 0 {
			return usagef(cmd, "--timeout cannot be negative")
		}
		a, err := open(e, false)
		if err != nil {
			return err
		}
		name := words[0]
		ctx := e.ctx
		if *timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, *timeout)
			defer cancel()
		}
		got, err := a.Answers(name)
		if wait {
			got, err = a.Wait(ctx, name, waitEvery)
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return timeoutError{fmt.Sprintf("no button was pressed on %s within %s", fileOf(name), *timeout)}
		}
		if err != nil {
			return err
		}
		if *asJSON {
			return printJSON(e.stdout, got)
		}
		_, err = io.WriteString(e.stdout, got.String())
		return err
	}
}

// mcpCmd serves the notes over MCP on standard input and output.
func mcpCmd(e env, args []string) error {
	if _, err := parse(newFlags("mcp"), "mcp", args, 0, 0); err != nil {
		return err
	}
	// The server starts without a board, as an agent's MCP client starts it
	// in any project: a tool that changes the board makes it, as the
	// command line does, and one that reads it says there is none.
	a, err := open(e, false)
	if errors.Is(err, errNoBoard) {
		a, err = api.NewWithClock(nil, kinds.Default(note.Plain), e.now), nil
	}
	if err != nil {
		return err
	}
	defer live(registry.KindMCP)()
	server := &mcp.Server{API: a, Guide: initcmd.GuideText(), Version: fullVersion()}
	server.Open = func(write bool) (*api.API, error) { return open(e, write) }
	return server.Serve(e.ctx, e.stdin, e.stdout)
}

// bridgeCmd serves the board to the desktop app, which runs it as its
// child: one JSON object a line on standard input and output.
func bridgeCmd(e env, args []string) error {
	if _, err := parse(newFlags("bridge"), "bridge", args, 0, 0); err != nil {
		return err
	}
	a, err := open(e, false)
	if err != nil {
		return err
	}
	prune()
	defer live(registry.KindBridge)()
	bridge.Version = fullVersion()
	return bridge.Serve(e.ctx, a, kinds.Default(note.Plain), e.stdin, e.stdout)
}

// pageAddCmd adds a page to a book, named and made as the board's N makes
// it (api.CreateFrom).
func pageAddCmd(e env, args []string) error {
	words, err := parse(newFlags("page add"), "page add", args, 1, 2)
	if err != nil {
		return err
	}
	if !strings.Contains(strings.Trim(words[0], "/"), "/") {
		return usagef("page add", "%s is not a book: a book is a folder in a tab, such as deploy/docs", words[0])
	}
	a, err := open(e, false)
	if err != nil {
		return err
	}
	text, body := "", []byte(nil)
	if len(words) > 1 {
		text, body = words[1], []byte(words[1]+"\n")
	}
	file, err := a.CreateFrom(words[0], text, "", api.Options{}, body)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(e.stdout, "added", file)
	return err
}

// pageListCmd lists the pages of a book.
func pageListCmd(e env, args []string) error {
	fs := newFlags("page list")
	asJSON := fs.Bool("json", false, "print the pages as JSON")
	words, err := parse(fs, "page list", args, 1, 1)
	if err != nil {
		return err
	}
	a, err := open(e, false)
	if err != nil {
		return err
	}
	pages, err := a.Pages(words[0])
	if err != nil {
		return err
	}
	if *asJSON {
		return printJSON(e.stdout, pages)
	}
	w := tabwriter.NewWriter(e.stdout, 0, 0, 2, ' ', 0)
	for _, p := range pages {
		fmt.Fprintf(w, "%s\t%s\t%s\n", p.Name, p.Type, p.Title)
	}
	return w.Flush()
}
