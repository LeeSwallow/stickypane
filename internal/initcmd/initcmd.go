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
	// Skill installs the guide as a Claude Code skill instead of adding it
	// to the instruction files. A skill is read only when a task calls for
	// it, so the guide does not take up room in every conversation.
	Skill bool
}

// skillPath is where a project's Claude Code skill for stickypane lives.
var skillPath = filepath.Join(".claude", "skills", "stickypane", "SKILL.md")

// skillHead says when the skill applies. It is all an agent sees of the
// skill until it decides to use it.
const skillHead = `---
name: stickypane
description: Put notes, checklists, kanban boards, charts, diagrams and questions on the user's stickypane board (a terminal pane next to you) and update them with one-line commands. Use when the user asks to show, track or ask something on the board, and when progress on tracked work changes.
---
`

// Skill returns the guide as a Claude Code skill file.
func Skill() string { return skillHead + GuideText() }

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

	if opts.Skill {
		path := filepath.Join(root, skillPath)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(Skill()), 0o644); err != nil {
			return err
		}
		fmt.Fprintf(out, "Wrote the stickypane skill to %s.\n", skillPath)
	} else if !opts.NoAgentDocs {
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
