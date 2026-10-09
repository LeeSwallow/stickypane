package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/LeeSwallow/stickypane/internal/api"
	"github.com/LeeSwallow/stickypane/internal/initcmd"
	"github.com/LeeSwallow/stickypane/internal/kinds"
	"github.com/LeeSwallow/stickypane/internal/mcp"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/widget/note"
)

// open returns the API for the notes folder of the project here. With
// create, a command that puts something on the board makes the folder at
// the top of the repository when there is none, so an agent can start a
// board without a setup step; it says so on stderr.
func open(e env, create bool) (*api.API, error) {
	dir, err := store.Resolve(".")
	if errors.Is(err, store.ErrNotFound) && create {
		if root := initcmd.ProjectRoot("."); root != "" {
			if dir, _, err = initcmd.MakeBoard(root); err == nil {
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
	return api.New(store.Open(dir), kinds.Default(note.Plain)), nil
}

// printJSON writes v as JSON without escaping HTML.
func printJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// listCmd lists the notes, as a table or as JSON.
func listCmd(e env, args []string) error {
	fs := newFlags("list")
	asJSON := fs.Bool("json", false, "print the list as JSON")
	if _, err := parse(fs, "list", args, 0, 0); err != nil {
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
	words, err := parse(newFlags("cat"), "cat", args, 1, 1)
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
	fs := newFlags("write")
	var opts api.Options
	fs.StringVar(&opts.Type, "type", "", "note, board, checklist, log, chart or form")
	fs.StringVar(&opts.Title, "title", "", "the note's title")
	fs.StringVar(&opts.Size, "size", "", "page, half or card")
	fs.BoolVar(&opts.Open, "open", false, "put the note on the screen now")
	words, err := parse(fs, "write", args, 1, 1)
	if err != nil {
		return err
	}
	a, err := open(e, true)
	if err != nil {
		return err
	}
	body, err := io.ReadAll(e.stdin)
	if err != nil {
		return err
	}
	file, err := a.Write(words[0], opts, body)
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
	a, err := open(e, false)
	if err != nil {
		return err
	}
	server := &mcp.Server{API: a, Guide: initcmd.GuideText(), Version: fullVersion()}
	return server.Serve(e.ctx, e.stdin, e.stdout)
}
