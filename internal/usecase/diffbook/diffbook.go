// Package diffbook is what the git worktrees of a repository changed, for
// the diff note's book and `stickypane diff`: which base each is compared
// with, the files each changed, the files two of them changed, and one
// file's diff when its page is shown. It reads git through
// internal/adapter/git and writes nothing. It reads `git worktree list`
// itself, so it works without the board's record of worktrees.
package diffbook

import (
	"path/filepath"
	"slices"
	"sort"

	"github.com/LeeSwallow/stickypane/internal/adapter/git"
	"github.com/LeeSwallow/stickypane/internal/core/widget/diff"
)

// Book is what the pages of the diff note are made from: the list, and the
// chosen worktree with its base and its changed files, those committed
// since the base first, then those not committed yet.
type Book struct {
	Overview diff.Overview
	Path     string // the chosen worktree's folder; "" when there is none
	Base     string
	Changes  []git.Change
}

// Base is what the worktrees are compared with when nothing chose one:
// the branch origin/HEAD names, else the branch of the main worktree, else
// none ("").
func Base(list []git.Worktree) string {
	if len(list) == 0 {
		return ""
	}
	if b := git.OriginHead(list[0].Path); b != "" {
		return b
	}
	return list[0].Branch
}

// Name is how a worktree is called: its branch, or its folder's name when
// its HEAD is detached.
func Name(w git.Worktree) string {
	if w.Branch != "" {
		return w.Branch
	}
	return filepath.Base(w.Path)
}

// Read reads the worktrees of the repository dir is in. chosen names the
// worktree whose files are the pages; "", or a name git no longer has, is
// the worktree dir is in. base is what every worktree is compared with;
// "" is Base. A worktree whose changes cannot be read says why in its row.
func Read(dir, chosen, base string) (Book, error) {
	list, err := git.Worktrees(dir)
	if err != nil {
		return Book{}, err
	}
	if base == "" {
		base = Base(list)
	}
	var b Book
	by := map[string][]string{}
	changed := map[string][]git.Change{}
	current := ""
	for _, w := range list {
		if w.Bare {
			continue
		}
		row := diff.Worktree{Name: Name(w), Path: w.Path, Base: base, Closed: w.Prunable, Current: git.Same(w.Path, dir)}
		if row.Current {
			current = row.Name
		}
		if !row.Closed && base != "" {
			changes, err := git.Changes(w.Path, base)
			if err != nil {
				row.Err = err.Error()
			}
			changed[row.Name] = changes
			for _, c := range changes {
				row.Files, row.Add, row.Del = row.Files+1, row.Add+c.Add, row.Del+c.Del
				by[c.Path] = append(by[c.Path], row.Name)
			}
		}
		b.Overview.Worktrees = append(b.Overview.Worktrees, row)
	}
	for path, names := range by {
		if len(names) > 1 {
			b.Overview.Overlaps = append(b.Overview.Overlaps, diff.Overlap{Path: path, By: names})
		}
	}
	sort.Slice(b.Overview.Overlaps, func(i, j int) bool { return b.Overview.Overlaps[i].Path < b.Overview.Overlaps[j].Path })

	b.Overview.Chosen = current
	for _, w := range b.Overview.Worktrees {
		if w.Name == chosen && !w.Closed {
			b.Overview.Chosen = chosen
		}
	}
	for _, w := range b.Overview.Worktrees {
		if w.Name == b.Overview.Chosen {
			b.Path, b.Base = w.Path, w.Base
		}
	}
	if b.Path == "" {
		return b, nil
	}
	b.Changes = changed[b.Overview.Chosen]
	pending, err := git.Pending(b.Path)
	if err != nil {
		return b, err
	}
	b.Changes = append(b.Changes, pending...)
	return b, nil
}

// Diff is the diff of one of the book's changed files, read only now.
func (b Book) Diff(c git.Change) (text string, cut bool, err error) {
	return git.Diff(b.Path, b.Base, c)
}

// Change is the book's changed file at path: its commits since the base,
// else its edits not committed yet; with uncommitted, only those edits.
func (b Book) Change(path string, uncommitted bool) (git.Change, bool) {
	i := slices.IndexFunc(b.Changes, func(c git.Change) bool {
		return c.Path == path && (!uncommitted || c.Where != git.Committed)
	})
	if i < 0 {
		return git.Change{}, false
	}
	return b.Changes[i], true
}

// PR is the pull request of the worktree at path, or nil when gh does not
// say: no gh, no network, no pull request.
func PR(path string) *diff.PR {
	pr, ok := git.PullRequest(path)
	if !ok {
		return nil
	}
	return &diff.PR{Title: pr.Title, State: pr.State, URL: pr.URL}
}
