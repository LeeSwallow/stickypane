// Package initcmd sets a project up for stickypane: the notes folder, a
// welcome note, and a short guide for coding agents.
package initcmd

import (
	_ "embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/store"
)

//go:embed guide.md
var guide string

//go:embed welcome.md
var welcome string

// The guide lives between these markers so that a later run can replace it.
const (
	startMark = "<!-- stickypane:start -->"
	endMark   = "<!-- stickypane:end -->"
)

// agentFiles are the instruction files coding agents read.
var agentFiles = []string{"AGENTS.md", "CLAUDE.md"}

// Options adjusts what Run does.
type Options struct {
	NoAgentDocs bool // leave AGENTS.md and CLAUDE.md alone
}

// Guide returns the text Run adds to agent instruction files, wrapped in the
// markers that let a later run find and replace it.
func Guide() string { return guide }

// GuideText returns the guide without its markers, for readers that are not
// an instruction file.
func GuideText() string {
	text := strings.TrimPrefix(guide, startMark+"\n")
	return strings.TrimSuffix(strings.TrimRight(text, "\n"), endMark)
}

// Run prepares the project at root. It is safe to run again: an existing
// notes folder is left as it is, and the guide is replaced in place.
func Run(root string, opts Options, out io.Writer) error {
	dir := filepath.Join(root, store.DirName)
	switch _, err := os.Stat(dir); {
	case errors.Is(err, fs.ErrNotExist):
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, "welcome.md"), []byte(welcome), 0o644); err != nil {
			return err
		}
		fmt.Fprintf(out, "Created %s/ with a welcome note.\n", store.DirName)
	case err != nil:
		return err
	default:
		fmt.Fprintf(out, "%s/ already exists.\n", store.DirName)
	}

	if !opts.NoAgentDocs {
		var targets []string
		for _, name := range agentFiles {
			if _, err := os.Stat(filepath.Join(root, name)); err == nil {
				targets = append(targets, name)
			}
		}
		if len(targets) == 0 {
			targets = agentFiles[:1]
		}
		for _, name := range targets {
			path := filepath.Join(root, name)
			old, err := os.ReadFile(path)
			if err != nil && !errors.Is(err, fs.ErrNotExist) {
				return err
			}
			if err := os.WriteFile(path, []byte(inject(string(old), guide)), 0o644); err != nil {
				return err
			}
			fmt.Fprintf(out, "Added the stickypane guide to %s.\n", name)
		}
	}
	fmt.Fprintln(out, "Run `stickypane` in a pane next to your agent.")
	return nil
}

// inject puts the guide into content: in place of an earlier guide when one
// is there, otherwise at the end after a blank line.
//
// An earlier guide is an end marker together with the nearest start marker
// before it. Pairing them this way means a stray marker never makes inject
// delete the user's own text or add a second guide on the next run.
func inject(content, guide string) string {
	block := strings.TrimRight(guide, "\n")
	for from := 0; ; {
		rel := strings.Index(content[from:], endMark)
		if rel < 0 {
			break
		}
		end := from + rel
		if start := strings.LastIndex(content[:end], startMark); start >= 0 {
			return content[:start] + block + content[end+len(endMark):]
		}
		from = end + len(endMark)
	}
	if content != "" {
		content = strings.TrimRight(content, "\n") + "\n\n"
	}
	return content + block + "\n"
}
