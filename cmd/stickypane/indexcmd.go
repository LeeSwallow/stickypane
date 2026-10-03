package main

import (
	"io"
	"path/filepath"

	"github.com/LeeSwallow/stickypane/internal/api"
)

// indexCmd prints every note of the board in a line, grouped by tab: what
// an agent reads first instead of every note, and what a person scans on a
// board with many notes.
func indexCmd(e env, args []string) error {
	fs := newFlags("index")
	asJSON := fs.Bool("json", false, "print the index as JSON")
	if _, err := parse(fs, "index", args, 0, 0); err != nil {
		return err
	}
	a, err := open(e, false)
	if err != nil {
		return err
	}
	idx, err := a.Index()
	if err != nil {
		return err
	}
	if *asJSON {
		return printJSON(e.stdout, idx)
	}
	_, err = io.WriteString(e.stdout, api.IndexMarkdown(idx, filepath.Base(filepath.Dir(a.Store().Dir))))
	return err
}
