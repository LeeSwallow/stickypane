package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/httpfile"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// apiCmd lists the requests of a .http note, or sends one or all of them and
// prints their logs. It is the agent's way to send what it wrote, without
// the user's yes: the agent could send the same with curl.
func apiCmd(e env, args []string) error {
	fs := newFlags("api")
	envName := fs.String("env", "", "the environment to use")
	all := fs.Bool("all", false, "send every request, in order")
	words, err := parse(fs, "api", args, 1, 2)
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
	st := store.Open(dir)
	name := words[0]
	if filepath.Ext(name) == "" {
		for _, ext := range []string{".http", ".rest"} {
			if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(name+ext))); err == nil {
				name += ext
				break
			}
		}
	}
	data, err := st.Read(name)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", name, err)
	}
	f := httpfile.Parse(string(data))
	if len(f.Requests) == 0 {
		return fmt.Errorf("%s has no requests yet", name)
	}

	var parts []int
	switch {
	case *all:
		for i := range f.Requests {
			parts = append(parts, i)
		}
	case len(words) == 2:
		titles := make([]string, len(f.Requests))
		for i, r := range f.Requests {
			titles[i] = r.Title()
		}
		i, err := widget.Pick(titles, words[1], "request")
		if err != nil {
			return err
		}
		parts = []int{i}
	default:
		for i, r := range f.Requests {
			fmt.Fprintf(e.stdout, "#%d  %-6s %s\n", i+1, r.Method, r.Title())
		}
		return nil
	}

	failed := 0
	for n, i := range parts {
		res, err := httpfile.SendNote(e.ctx, st, dir, name, i, *envName)
		if err != nil {
			return err
		}
		if n > 0 {
			fmt.Fprintln(e.stdout)
		}
		fmt.Fprint(e.stdout, strings.TrimPrefix(httpfile.Log(res), "$ "))
		if res.Failed() {
			failed++
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d requests failed; the last one's log is in %s", failed, len(parts), httpfile.LogName(name))
	}
	return nil
}
