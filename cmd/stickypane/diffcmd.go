package main

import (
	"fmt"
	"text/tabwriter"

	"github.com/LeeSwallow/stickypane/internal/adapter/git"
	"github.com/LeeSwallow/stickypane/internal/core/i18n"
	"github.com/LeeSwallow/stickypane/internal/usecase/diffbook"
)

// diffCmd is `diff list` and `diff show`: what the git worktrees of the
// repository here changed, read from git as the board's diff note reads
// it. It needs no board.
func diffCmd(verb string) func(env, []string) error {
	return func(e env, args []string) error {
		fs := newFlags("diff " + verb)
		asJSON := fs.Bool("json", false, "print it as JSON")
		var uncommitted *bool
		if verb == "show" {
			uncommitted = fs.Bool("uncommitted", false, "the edits not committed yet, not the commits")
		}
		least, most := 0, 1
		if verb == "show" {
			least, most = 2, 2
		}
		words, err := parse(fs, "diff "+verb, args, least, most)
		if err != nil {
			return err
		}
		chosen := ""
		if len(words) > 0 {
			chosen = words[0]
		}
		book, err := diffbook.Read(".", chosen, "")
		if err != nil {
			return err
		}
		if chosen != "" && book.Overview.Chosen != chosen {
			return i18n.Err("diff.unknown", map[string]any{"Name": chosen})
		}
		switch {
		case verb == "show":
			c, ok := book.Change(words[1], *uncommitted)
			if !ok {
				return i18n.Err("diff.nofile", map[string]any{"Name": chosen, "File": words[1]})
			}
			text, cut, err := book.Diff(c)
			if err != nil {
				return err
			}
			if *asJSON {
				return printJSON(e.stdout, struct {
					git.Change
					Diff string `json:"diff"`
					Cut  bool   `json:"cut,omitempty"`
				}{c, text, cut})
			}
			_, err = fmt.Fprint(e.stdout, text)
			return err
		case chosen != "":
			if *asJSON {
				return printJSON(e.stdout, book.Changes)
			}
			tw := tabwriter.NewWriter(e.stdout, 0, 0, 2, ' ', 0)
			for _, c := range book.Changes {
				fmt.Fprintf(tw, "%s\t%s\t+%d\t-%d\t%s\n", c.Status, c.Path, c.Add, c.Del, c.Where)
			}
			return tw.Flush()
		}
		if *asJSON {
			return printJSON(e.stdout, book.Overview)
		}
		tw := tabwriter.NewWriter(e.stdout, 0, 0, 2, ' ', 0)
		for _, w := range book.Overview.Worktrees {
			state := "open"
			switch {
			case w.Closed:
				state = "closed"
			case w.Err != "":
				state = w.Err
			}
			fmt.Fprintf(tw, "%s\t%s\t%d\t+%d\t-%d\t%s\n", w.Name, w.Base, w.Files, w.Add, w.Del, state)
		}
		tw.Flush()
		for _, o := range book.Overview.Overlaps {
			fmt.Fprintf(e.stdout, "! %s: %v\n", o.Path, o.By)
		}
		return nil
	}
}
