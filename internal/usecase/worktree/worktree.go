// Package worktree keeps the board in step with the git worktrees of its
// repository. Every worktree is recorded, open or closed, in the registry;
// its folder on the board, worktrees/<name>/, is made at the first write
// into it, with an overview page, and marked closed, never deleted, when
// the worktree is gone. Git is right where the board and git disagree.
// Where git is missing or the board is not the main worktree's, nothing
// happens.
package worktree

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/LeeSwallow/stickypane/internal/adapter/env"
	"github.com/LeeSwallow/stickypane/internal/adapter/git"
	"github.com/LeeSwallow/stickypane/internal/adapter/registry"
	"github.com/LeeSwallow/stickypane/internal/adapter/store"
	"github.com/LeeSwallow/stickypane/internal/core/doc"
	"github.com/LeeSwallow/stickypane/internal/core/i18n"
	"github.com/LeeSwallow/stickypane/internal/core/when"
)

const (
	// Tab is the folder of the worktrees' folders in the notes folder.
	Tab = "worktrees"
	// Overview is the first page of a worktree's folder: what it is for,
	// for people, with its branch and state in the front matter.
	Overview = "00-개요.md"
)

// Worktree is a worktree as the board shows it. Running and Waiting are
// worked out each time, never stored.
type Worktree struct {
	Name    string `json:"name"`
	Path    string `json:"path,omitempty"` // "" when only its folder is known
	Branch  string `json:"branch,omitempty"`
	State   string `json:"state"`   // open or closed
	Running bool   `json:"running"` // git reports it locked, as Claude Code does while its agent runs
	Waiting int    `json:"waiting"` // forms in its folder not answered yet
	Folder  bool   `json:"folder"`  // its folder is on the board
}

// Board is a board and where it keeps its records.
type Board struct {
	Store   *store.Store
	Records registry.Writer
	Now     func() time.Time
}

// repo is what git says of the board's repository.
type repo struct {
	root, common string
	live         []git.Worktree // the other worktrees git still has
	locked       map[string]bool
}

// repo reads git, and is false where the board is not the main worktree's
// of a repository.
func (b Board) repo() (repo, bool) {
	root := filepath.Dir(b.Store.Dir)
	list, err := git.Worktrees(root)
	if err != nil || len(list) == 0 || list[0].Bare || !sameDir(list[0].Path, root) {
		return repo{}, false
	}
	common, err := git.CommonDir(root)
	if err != nil {
		return repo{}, false
	}
	r := repo{root: root, common: common, locked: map[string]bool{}}
	for _, w := range list[1:] {
		if !w.Prunable {
			r.live = append(r.live, w)
			r.locked[w.Path] = w.Locked
		}
	}
	return r, true
}

// folder is what a worktree's overview page says.
type folder struct{ branch, state string }

// folders reads the overview of every folder in Tab that has one with a
// state: the folders the board made. Others are a person's, and left alone.
func (b Board) folders() map[string]folder {
	out := map[string]folder{}
	ents, _ := os.ReadDir(b.Store.Path(Tab))
	for _, e := range ents {
		raw, err := os.ReadFile(b.Store.Path(Tab + "/" + e.Name() + "/" + Overview))
		if err != nil {
			continue
		}
		d := doc.Parse(raw)
		if state, ok := d.Get("state"); ok {
			branch, _ := d.Get("branch")
			out[e.Name()] = folder{branch, state}
		}
	}
	return out
}

// names gives each live worktree its folder name: the one a record or a
// folder has for it (by branch, or by path when detached), else the last
// element of its path, with a short hash of the path when that is taken.
func names(live []git.Worktree, records []registry.Worktree, folders map[string]folder) []registry.Worktree {
	taken := map[string]bool{}
	for _, r := range records {
		taken[r.Name] = true
	}
	byName := make([]string, 0, len(folders))
	for n := range folders {
		taken[n] = true
		byName = append(byName, n)
	}
	slices.Sort(byName)
	var out []registry.Worktree
	for _, w := range live {
		name := ""
		for _, r := range records {
			if (w.Branch != "" && r.Branch == w.Branch) || (w.Branch == "" && r.Branch == "" && r.Path == w.Path) {
				name = r.Name
				break
			}
		}
		for _, n := range byName {
			if name == "" && w.Branch != "" && folders[n].branch == w.Branch {
				name = n
			}
		}
		if name == "" {
			name = filepath.Base(w.Path)
			if taken[name] {
				sum := sha1.Sum([]byte(w.Path))
				name += "-" + hex.EncodeToString(sum[:])[:4]
			}
			taken[name] = true
		}
		out = append(out, registry.Worktree{Name: name, Path: w.Path, Branch: w.Branch, State: registry.StateOpen})
	}
	return out
}

// merge brings the record old in step with the live worktrees, and says
// what changed.
func merge(old, live []registry.Worktree, project string) ([]registry.Worktree, []registry.Event) {
	out := slices.Clone(old)
	var events []registry.Event
	event := func(typ string, w registry.Worktree, from string) {
		events = append(events, registry.Event{Type: typ, Project: project, Name: w.Name, Path: w.Path, From: from})
	}
	alive := map[string]bool{}
	for _, l := range live {
		alive[l.Name] = true
		i := slices.IndexFunc(out, func(r registry.Worktree) bool { return r.Name == l.Name })
		switch {
		case i < 0:
			out = append(out, l)
			event(registry.WorktreeOpen, l, "")
			continue
		case out[i].State == registry.StateClosed:
			event(registry.WorktreeOpen, l, "")
		case out[i].Path != l.Path:
			event(registry.WorktreeMove, l, out[i].Path)
		}
		out[i] = l
	}
	for i, r := range out {
		if r.State == registry.StateOpen && !alive[r.Name] {
			out[i].State = registry.StateClosed
			event(registry.WorktreeClose, out[i], "")
		}
	}
	return out, events
}

// Sync reads git, records what changed, and marks the board's worktree
// folders closed or open again. It answers every worktree the board knows
// of; nil where there is no repository.
func (b Board) Sync() ([]Worktree, error) {
	r, ok := b.repo()
	if !ok {
		return nil, nil
	}
	folders := b.folders()
	var all []registry.Worktree
	b.Records.UpdateWorktrees(r.common, func(old []registry.Worktree) ([]registry.Worktree, []registry.Event) {
		var events []registry.Event
		all, events = merge(old, names(r.live, old, folders), r.root)
		return all, events
	})
	open := map[string]bool{}
	for _, w := range all {
		open[w.Name] = w.State == registry.StateOpen
	}
	var first error
	for name, f := range folders {
		op := state{registry.StateOpen, ""}
		switch {
		case f.state == registry.StateOpen && !open[name]:
			op = state{registry.StateClosed, b.Now().Format(when.Stamp)}
		case f.state == registry.StateClosed && open[name]:
		default:
			continue
		}
		if err := b.Store.Apply(Tab+"/"+name+"/"+Overview, op); err != nil && first == nil {
			first = err
		}
		folders[name] = folder{f.branch, op.state}
	}
	return b.view(r, all, folders), first
}

// List is what Sync would answer, worked out without writing anything:
// for a screen that reads the board each time it changes, and leaves the
// records and the folders to Sync.
func (b Board) List() []Worktree {
	r, ok := b.repo()
	if !ok {
		return nil
	}
	folders := b.folders()
	old := b.Records.Worktrees(r.common)
	all, _ := merge(old, names(r.live, old, folders), r.root)
	return b.view(r, all, folders)
}

// view is what is known of every worktree, in the record's order, then the
// folders without a record.
func (b Board) view(r repo, all []registry.Worktree, folders map[string]folder) []Worktree {
	out := []Worktree{}
	seen := map[string]bool{}
	add := func(w Worktree) {
		seen[w.Name] = true
		if _, err := os.Stat(b.Store.Path(Tab + "/" + w.Name)); err == nil {
			w.Folder = true
			w.Waiting = b.waiting(w.Name)
		}
		out = append(out, w)
	}
	for _, w := range all {
		add(Worktree{Name: w.Name, Path: w.Path, Branch: w.Branch, State: w.State, Running: w.State == registry.StateOpen && r.locked[w.Path]})
	}
	byName := make([]string, 0, len(folders))
	for n := range folders {
		byName = append(byName, n)
	}
	slices.Sort(byName)
	for _, n := range byName {
		if !seen[n] {
			add(Worktree{Name: n, Branch: folders[n].branch, State: folders[n].state})
		}
	}
	return out
}

// waiting counts the forms in a worktree's folder that are not answered.
func (b Board) waiting(name string) int {
	n := 0
	ents, _ := os.ReadDir(b.Store.Path(Tab + "/" + name))
	for _, e := range ents {
		if !store.IsMarkdown(e.Name()) {
			continue
		}
		raw, err := os.ReadFile(b.Store.Path(Tab + "/" + name + "/" + e.Name()))
		if err != nil {
			continue
		}
		d := doc.Parse(raw)
		if s, _ := d.Get("submitted"); d.Type() == "form" && s == "" {
			n++
		}
	}
	return n
}

// NewFolder is the store's hook (store.Store.NewFolder): a folder made in
// Tab for a live worktree starts with its overview page.
func (b Board) NewFolder(path string) map[string][]byte {
	name, ok := strings.CutPrefix(path, Tab+"/")
	if !ok || strings.Contains(name, "/") {
		return nil
	}
	r, ok := b.repo()
	if !ok {
		return nil
	}
	for _, w := range names(r.live, b.Records.Worktrees(r.common), b.folders()) {
		if w.Name == name {
			return map[string][]byte{Overview: b.overview(w)}
		}
	}
	return nil
}

// overview is the first page of a worktree's folder, in the board's
// language. It does not carry the path: that is a fact of one computer.
func (b Board) overview(w registry.Worktree) []byte {
	tr := i18n.Pick(b.Store.Language(), env.Detect().Locale)
	text := tr.Say("worktree.overviewDetached", nil)
	if w.Branch != "" {
		text = tr.Say("worktree.overview", map[string]any{"Branch": w.Branch})
	}
	d := doc.Parse([]byte("# " + w.Name + "\n\n" + text + "\n"))
	d = d.Set("branch", w.Branch).Set("state", registry.StateOpen).Set("created", b.Now().Format(when.Stamp))
	return d.Bytes()
}

// state is the op that marks an overview open or closed: closed is the
// time, or "" to take it away.
type state struct{ state, closed string }

func (o state) Apply(d doc.Document) (doc.Document, error) {
	d = d.Set("state", o.state)
	if o.closed == "" {
		return d.Unset("closed"), nil
	}
	return d.Set("closed", o.closed), nil
}

func sameDir(a, b string) bool {
	fa, err1 := os.Stat(a)
	fb, err2 := os.Stat(b)
	return err1 == nil && err2 == nil && os.SameFile(fa, fb)
}
