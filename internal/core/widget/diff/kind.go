// Package diff is the note that shows what git worktrees changed: a book
// whose first page lists the worktrees and whose other pages are the
// changed files of one of them, each drawn as its diff. It only reads: the
// board hands it git's output, and nothing in it runs git or writes.
package diff

import (
	"github.com/LeeSwallow/stickypane/internal/core/doc"
	"github.com/LeeSwallow/stickypane/internal/core/widget"
)

// Kind registers the diff note: a Markdown note with "type: diff". The
// board turns it into the book of pages; anywhere else it is drawn from its
// own body, read as a unified diff.
var Kind = widget.Kind{
	Name:     "diff",
	Label:    "Diff (view only)",
	Icon:     "±",
	Blurb:    "What the git worktrees changed: a page for each changed file.",
	Command:  "stickypane note write changes --type diff",
	Usage:    "A note with type: diff is, on the board, a book: the first page lists the repository's git worktrees, how many files each changed since its base, and the files two of them changed; enter opens one and b sets the base. Each changed file is then a page of its diff, turned with , and . (one at a time). worktree: in the front matter pins one worktree and skips the list; base: sets the base. Nothing on it can be changed; stickypane diff list and diff show print the same.",
	Example:  "diff --git a/main.go b/main.go\n--- a/main.go\n+++ b/main.go\n@@ -1,2 +1,2 @@\n package main\n-var gap = 1\n+var gap = 2\n",
	Size:     func(doc.Document) string { return widget.SizeHalf },
	Template: func(title string, _ widget.Env) []byte { return widget.NewFile("diff", title, "") },
	Parse:    func(d doc.Document, e widget.Env) widget.Widget { return NewPage(d.Body, e) },
	Data:     func(d doc.Document, _ widget.Env) any { return struct{ Files []File }{Parse(d.Body)} },
}

// ListKind is the kind of the book's first page, the list of worktrees. It
// is not registered: no file is a list; the board makes the page.
var ListKind = widget.Kind{
	Name: "diff",
	Icon: "±",
	Hint: []string{"j k", "choose", "enter", "open", "b", "base"},
	Keys: "j k up down enter b",
}
