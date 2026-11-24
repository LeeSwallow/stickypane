package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/LeeSwallow/stickypane/internal/adapter/atomicfile"
	"github.com/LeeSwallow/stickypane/internal/adapter/filelock"
)

// WorktreesFile is where the worktrees of every repository are, in the
// home folder: a path is a fact of one computer, so it is kept here and
// never in a board.
const WorktreesFile = "worktrees.json"

// The events of a worktree, with its Name and Path (and From, the path it
// left, for a move).
const (
	WorktreeOpen  = "worktree.open"
	WorktreeClose = "worktree.close"
	WorktreeMove  = "worktree.move"
)

// The states of a Worktree.
const (
	StateOpen   = "open"
	StateClosed = "closed"
)

// Worktree is one worktree of a repository, as the board last saw it.
// Name is its folder on the board, worktrees/<name>/.
type Worktree struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Branch string `json:"branch,omitempty"`
	State  string `json:"state"`
}

// worktrees is the file WorktreesFile: each board's worktrees, by the
// absolute path of its repository's common git folder.
type worktrees struct {
	Version int                   `json:"version"`
	Boards  map[string][]Worktree `json:"boards"`
}

func (w Writer) readWorktrees() worktrees {
	var f worktrees
	if b, err := os.ReadFile(filepath.Join(w.Home, WorktreesFile)); err == nil {
		_ = json.Unmarshal(b, &f)
	}
	if f.Boards == nil {
		f.Boards = map[string][]Worktree{}
	}
	return f
}

// Worktrees is the record of board's worktrees, or nil when there is none
// or records are not kept.
func (w Writer) Worktrees(board string) []Worktree {
	if !enabled(w.Home) {
		return nil
	}
	return w.readWorktrees().Boards[board]
}

// UpdateWorktrees hands change the record of board's worktrees and keeps
// what it returns, with the events, under the home folder's lock so that
// two processes do not lose each other's change. Where records are not
// kept change still runs, with no record, and nothing is written.
func (w Writer) UpdateWorktrees(board string, change func(old []Worktree) ([]Worktree, []Event)) {
	if !enabled(w.Home) {
		change(nil)
		return
	}
	unlock, err := filelock.Lock(w.Home, 5*time.Second)
	if err != nil {
		change(w.Worktrees(board))
		return
	}
	defer unlock()
	f := w.readWorktrees()
	list, events := change(f.Boards[board])
	if len(events) == 0 {
		return
	}
	f.Version = 1
	f.Boards[board] = list
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil || atomicfile.WriteFile(filepath.Join(w.Home, WorktreesFile), append(b, '\n'), atomicfile.FileMode) != nil {
		return
	}
	for _, e := range events {
		w.Append(e)
	}
}
