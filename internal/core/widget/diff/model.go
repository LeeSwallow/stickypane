package diff

// File is one file of a diff, as git wrote it: its paths, how it changed,
// and its hunks of lines.
type File struct {
	Old, New   string // the paths, without a/ and b/; "" for /dev/null
	Status     string // "A" added, "D" deleted, "R" renamed, "C" copied, else "M"
	OldMode    string // set with NewMode when the mode changed: "100644"
	NewMode    string
	Similarity string // how alike a renamed or copied file is: "86%"
	Binary     bool   // git compared bytes, not lines
	Hunks      []Hunk
}

// Hunk is one @@ block: its header line as git wrote it, then its lines.
type Hunk struct {
	Header string
	Lines  []Line
}

// Line is a line of a hunk. Op is ' ' for a line kept, '+' added, '-'
// removed, and '\\' for git's "\ No newline at end of file", which belongs
// to the line before it.
type Line struct {
	Op   byte
	Text string
}

// Path is the file's name now, or its last name when it was deleted.
func (f File) Path() string {
	if f.New != "" {
		return f.New
	}
	return f.Old
}

// Counts are the lines added and removed. A "\ No newline" line is
// neither.
func (f File) Counts() (add, del int) {
	for _, h := range f.Hunks {
		for _, l := range h.Lines {
			switch l.Op {
			case '+':
				add++
			case '-':
				del++
			}
		}
	}
	return add, del
}

// Worktree is a row of the list page: a worktree of the repository and
// what changed in it since its base.
type Worktree struct {
	Name    string `json:"name"` // its branch, or its folder's name when HEAD is detached
	Path    string `json:"path"`
	Base    string `json:"base,omitempty"` // what it is compared with; "" when none was found
	Files   int    `json:"files"`
	Add     int    `json:"add"`
	Del     int    `json:"del"`
	Closed  bool   `json:"closed,omitempty"`  // git lists it, but its folder is gone
	Current bool   `json:"current,omitempty"` // the board runs in it
	Err     string `json:"error,omitempty"`   // why its changes could not be read
}

// Overlap is a file that more than one worktree changed: where a merge
// may conflict.
type Overlap struct {
	Path string   `json:"path"`
	By   []string `json:"by"`
}

// PR is the pull request of a worktree's branch: only its title, state
// and address.
type PR struct {
	Title string `json:"title"`
	State string `json:"state"` // as gh says it: OPEN, MERGED, CLOSED
	URL   string `json:"url"`
}

// Overview is what the list page shows: the worktrees, the files two of
// them changed, and the pull request of the chosen one.
type Overview struct {
	Worktrees []Worktree `json:"worktrees"`
	Overlaps  []Overlap  `json:"overlaps,omitempty"`
	Chosen    string     `json:"chosen,omitempty"` // the worktree whose files are the pages
	PR        *PR        `json:"pr,omitempty"`
}
