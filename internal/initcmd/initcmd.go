// Package initcmd sets a project up for stickypane: the notes folder, a
// welcome note, and a short guide for coding agents.
package initcmd

import (
	"github.com/LeeSwallow/stickypane/internal/env"

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

// Options adjusts what Run does. The common case needs neither.
type Options struct {
	NoAgentDocs bool // leave AGENTS.md, CLAUDE.md and .claude/ alone
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

// ProjectRoot returns the top of the git repository that start is in, or ""
// outside one. A repository at the home folder does not count: a board there
// would be found from every folder under it.
func ProjectRoot(start string) string {
	dir, err := filepath.Abs(start)
	if err != nil {
		return ""
	}
	home, _ := os.UserHomeDir()
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			if home != "" && sameDir(dir, home) {
				return ""
			}
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// MakeBoard makes the notes folder of the project at root, with a welcome
// note, unless it is there. It returns the folder and whether it was made.
func MakeBoard(root string) (string, bool, error) {
	// A project set up before the folder was renamed keeps its folder: a
	// second, empty one would hide its notes.
	name := store.DirName
	if fi, err := os.Stat(filepath.Join(root, store.LegacyDirName)); err == nil && fi.IsDir() {
		if _, err := os.Stat(filepath.Join(root, store.DirName)); errors.Is(err, fs.ErrNotExist) {
			name = store.LegacyDirName
		}
	}
	dir := filepath.Join(root, name)
	switch _, err := os.Stat(dir); {
	case err == nil:
		return dir, false, nil
	case !errors.Is(err, fs.ErrNotExist):
		return "", false, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", false, err
	}
	if err := os.WriteFile(filepath.Join(dir, "welcome.md"), []byte(welcome), 0o644); err != nil {
		return "", false, err
	}
	// Deleted notes wait in the trash to be restored. They are the user's
	// own undo history and do not belong in the repository.
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(store.TrashDir+"/\n"), 0o644); err != nil {
		return "", false, err
	}
	return dir, true, nil
}

// Run prepares the project at root: the notes folder, then the guide where
// the project's agents will read it. It is safe to run again: an existing
// notes folder is left as it is, and the guide is replaced in place.
//
// Where the guide goes is decided from what is there. An agent that has the
// board plugin installed learns the board from it, so its instruction file
// is left alone, and a guide an earlier run put there is taken out.
// Otherwise Claude Code gets the guide in CLAUDE.md, or as a skill when the
// project has a .claude folder but no CLAUDE.md; Codex and other agents get
// it in AGENTS.md. When no file and no plugin is there, AGENTS.md is made.
func Run(root string, opts Options, out io.Writer) error {
	dir, made, err := MakeBoard(root)
	if err != nil {
		return err
	}
	if made {
		fmt.Fprintf(out, "Made %s/ with a welcome note.\n", filepath.Base(dir))
	} else {
		fmt.Fprintf(out, "%s/ already exists.\n", filepath.Base(dir))
	}
	if !opts.NoAgentDocs {
		var r report
		if err := teach(root, opts, &r); err != nil {
			return err
		}
		if line := r.String(); line != "" {
			fmt.Fprintln(out, line)
		}
	}
	fmt.Fprintln(out, NextStep())
	return nil
}

// NextStep is the one thing to do once the board is there: open it.
// Inside tmux, Zellij, WezTerm or Windows Terminal it is that tool's
// split command.
func NextStep() string {
	if split := env.Detect().Pane.Split("stickypane"); split != "" {
		return "Next: `" + split + "` opens the board next to your agent."
	}
	return "Next: run `stickypane` in a pane next to your agent."
}

// report collects what teach did, to say it in one line.
type report struct {
	added, current, removed []string
	plugin                  bool
}

func (r report) String() string {
	var parts []string
	if len(r.added) > 0 {
		parts = append(parts, "added the agent guide to "+and(r.added))
	}
	if len(r.current) > 0 {
		parts = append(parts, "the agent guide in "+and(r.current)+" is up to date")
	}
	if len(r.removed) > 0 {
		parts = append(parts, "took the agent guide out of "+and(r.removed)+": the board plugin teaches it")
	} else if r.plugin {
		parts = append(parts, "the board plugin teaches your agent")
	}
	if len(parts) == 0 {
		return ""
	}
	line := strings.Join(parts, "; ") + "."
	return strings.ToUpper(line[:1]) + line[1:]
}

func and(names []string) string {
	if len(names) == 1 {
		return names[0]
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}

// teach puts the guide where the project's agents read it, as Run says.
func teach(root string, opts Options, r *report) error {
	claudePlugin, codexPlugin := plugins(root)
	r.plugin = claudePlugin || codexPlugin
	has := func(name string) bool {
		_, err := os.Stat(filepath.Join(root, name))
		return err == nil
	}

	wrote := false
	switch {
	case claudePlugin:
		if err := strip(root, "CLAUDE.md", r); err != nil {
			return err
		}
		if has(skillPath) {
			path := filepath.Join(root, skillPath)
			if err := os.Remove(path); err != nil {
				return err
			}
			_ = os.Remove(filepath.Dir(path)) // only when nothing else is in it
			r.removed = append(r.removed, filepath.ToSlash(skillPath))
		}
	case opts.Skill || (!has("CLAUDE.md") && has(".claude")):
		if err := put(root, skillPath, func(string) string { return Skill() }, r); err != nil {
			return err
		}
		wrote = true
	case has("CLAUDE.md"):
		if err := put(root, "CLAUDE.md", inject, r); err != nil {
			return err
		}
		wrote = true
	}

	switch {
	case codexPlugin:
		return strip(root, "AGENTS.md", r)
	case opts.Skill:
		return nil
	case has("AGENTS.md") || (!wrote && !r.plugin):
		return put(root, "AGENTS.md", inject, r)
	}
	return nil
}

// put rewrites the file at name, relative to root, with change.
func put(root, name string, change func(string) string, r *report) error {
	path := filepath.Join(root, name)
	old, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	text := change(string(old))
	if text == string(old) {
		r.current = append(r.current, filepath.ToSlash(name))
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		return err
	}
	r.added = append(r.added, filepath.ToSlash(name))
	return nil
}

// strip takes the guide out of the file at name, relative to root, and
// removes the file when nothing else was in it.
func strip(root, name string, r *report) error {
	path := filepath.Join(root, name)
	old, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	content := string(old)
	start, end, ok := find(content)
	if !ok {
		return nil
	}
	r.removed = append(r.removed, name)
	before := strings.TrimRight(content[:start], "\n")
	after := strings.TrimLeft(content[end:], "\n")
	switch {
	case before == "" && after == "":
		return os.Remove(path)
	case before == "":
		return os.WriteFile(path, []byte(after), 0o644)
	case after == "":
		return os.WriteFile(path, []byte(before+"\n"), 0o644)
	}
	return os.WriteFile(path, []byte(before+"\n\n"+after), 0o644)
}

// find returns where an earlier guide starts and ends in content.
//
// An earlier guide is an end marker together with the nearest start marker
// before it. Pairing them this way means a stray marker never makes init
// delete the user's own text or add a second guide on the next run.
func find(content string) (start, end int, ok bool) {
	for from := 0; ; {
		rel := strings.Index(content[from:], endMark)
		if rel < 0 {
			return 0, 0, false
		}
		end := from + rel
		if start := strings.LastIndex(content[:end], startMark); start >= 0 {
			return start, end + len(endMark), true
		}
		from = end + len(endMark)
	}
}

// inject puts the guide into content: in place of an earlier guide when one
// is there, otherwise at the end after a blank line.
func inject(content string) string {
	block := strings.TrimRight(guide, "\n")
	if start, end, ok := find(content); ok {
		return content[:start] + block + content[end:]
	}
	if content != "" {
		content = strings.TrimRight(content, "\n") + "\n\n"
	}
	return content + block + "\n"
}
