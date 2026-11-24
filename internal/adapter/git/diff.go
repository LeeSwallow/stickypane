package git

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Where a change is: committed since the base, or not yet committed.
const (
	Committed   = ""
	Uncommitted = "uncommitted" // a tracked file changed since HEAD
	Untracked   = "untracked"   // a file git does not track yet
)

// Change is one changed file: how it changed, its paths and its counts of
// lines added and removed.
type Change struct {
	Status string `json:"status"`        // "A", "D", "M", "R", "C" or "T"
	Old    string `json:"old,omitempty"` // the path before, for a rename or a copy
	Path   string `json:"path"`
	Add    int    `json:"add"`
	Del    int    `json:"del"`
	Binary bool   `json:"binary,omitempty"` // git counts no lines: --numstat said "-"
	Where  string `json:"where,omitempty"`  // Committed, Uncommitted or Untracked
}

// diffFlags make git's diff the same whatever a repository's config says:
// no color, no program of the repository's run on a file, a/ and b/
// prefixes, renames found, paths from the top.
var diffFlags = []string{"--no-color", "--no-ext-diff", "--no-textconv", "--src-prefix=a/", "--dst-prefix=b/", "-M", "--no-relative"}

// OriginHead is the branch origin/HEAD names, "origin/main", or "" when
// the repository has no such remote branch.
func OriginHead(dir string) string {
	out, err := run(dir, "symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// Changes are the files changed in the worktree at dir since base:
// `git diff base...HEAD`, so a base that moved on since the branch left it
// counts only what the branch did.
func Changes(dir, base string) ([]Change, error) {
	if err := CheckBase(dir, base); err != nil {
		return nil, err
	}
	return changes(dir, base+"...HEAD", Committed)
}

// ErrBadBase is a base that is not a commit git knows. A base comes from
// sticky.json or a note's front matter: text anyone may write, so it is
// never handed to git as an option.
var ErrBadBase = errors.New("not a commit")

// CheckBase says whether base names a commit in the repository at dir. A
// base that starts with "-" is refused here, so rev-parse needs no
// --end-of-options, which it learned only in git 2.30.
func CheckBase(dir, base string) error {
	if base == "" || strings.HasPrefix(base, "-") {
		return fmt.Errorf("base %q: %w", base, ErrBadBase)
	}
	if _, err := run(dir, "rev-parse", "--quiet", "--verify", base+"^{commit}"); err != nil {
		return fmt.Errorf("base %q: %w", base, ErrBadBase)
	}
	return nil
}

// diffArgs are the words of a `git diff` the board runs: the flags that
// keep the repository's programs out, mode (--name-status, --numstat or
// none), the revision after --end-of-options and the paths after --.
func diffArgs(mode, what string, paths ...string) []string {
	args := append([]string{"diff"}, diffFlags...)
	if mode != "" {
		args = append(args, mode, "-z")
	}
	return append(append(append(args, "--end-of-options", what), "--"), paths...)
}

// Pending are the changes not committed yet: the tracked files changed
// since HEAD, then the files git does not track.
func Pending(dir string) ([]Change, error) {
	list, err := changes(dir, "HEAD", Uncommitted)
	if err != nil {
		return nil, err
	}
	out, err := run(dir, "ls-files", "-z", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	for _, p := range strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00") {
		if p != "" {
			list = append(list, Change{Status: "A", Path: p, Where: Untracked})
		}
	}
	return list, nil
}

// changes reads what `git diff <what>` changed, by name and by count.
func changes(dir, what, where string) ([]Change, error) {
	names, err := run(dir, diffArgs("--name-status", what)...)
	if err != nil {
		return nil, err
	}
	counts, err := run(dir, diffArgs("--numstat", what)...)
	if err != nil {
		return nil, err
	}
	list := ParseNameStatus(names)
	byPath := ParseNumstat(counts)
	for i := range list {
		list[i].Where = where
		if c, ok := byPath[list[i].Path]; ok {
			list[i].Add, list[i].Del, list[i].Binary = c.Add, c.Del, c.Binary
		}
	}
	return list, nil
}

// ParseNameStatus reads `git diff --name-status -z`: a status, then one
// path, or two for a rename or a copy ("R086", old, new).
func ParseNameStatus(out []byte) []Change {
	f := strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00")
	var list []Change
	for i := 0; i+1 < len(f); {
		status := f[i]
		c := Change{Status: status[:min(len(status), 1)], Path: f[i+1]}
		i += 2
		if (c.Status == "R" || c.Status == "C") && i < len(f) {
			c.Old, c.Path = c.Path, f[i]
			i++
		}
		list = append(list, c)
	}
	return list
}

// ParseNumstat reads `git diff --numstat -z` by the path each count is
// for: "added\tremoved\tpath", or "added\tremoved\t" then the old and new
// path of a rename. "-" for both counts is a binary file.
func ParseNumstat(out []byte) map[string]Change {
	f := strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00")
	counts := map[string]Change{}
	for i := 0; i < len(f); i++ {
		parts := strings.SplitN(f[i], "\t", 3)
		if len(parts) < 3 {
			continue
		}
		path := parts[2]
		if path == "" && i+2 < len(f) { // a rename: the paths follow
			path = f[i+2]
			i += 2
		}
		c := Change{Binary: parts[0] == "-" && parts[1] == "-"}
		c.Add, _ = strconv.Atoi(parts[0])
		c.Del, _ = strconv.Atoi(parts[1])
		counts[path] = c
	}
	return counts
}

// MaxDiff is how much of one file's diff is read; the rest is left out.
const MaxDiff = 1 << 20

// Diff is the unified diff of one changed file in the worktree at dir,
// read no further than MaxDiff; cut says that some was left out.
func Diff(dir, base string, c Change) (text string, cut bool, err error) {
	paths := []string{c.Path}
	if c.Old != "" {
		paths = []string{c.Old, c.Path}
	}
	var args []string
	switch c.Where {
	case Untracked:
		// Exit status 1 is "they differ", which a new file always does.
		args = append(append([]string{"diff"}, diffFlags...), "--no-index", "--", os.DevNull, c.Path)
	case Uncommitted:
		args = diffArgs("", "HEAD", paths...)
	default:
		if err := CheckBase(dir, base); err != nil {
			return "", false, err
		}
		args = diffArgs("", base+"...HEAD", paths...)
	}
	cmd := command(dir, args...)
	out := &capped{max: MaxDiff}
	cmd.Stdout = out
	err = cmd.Run()
	var exit *exec.ExitError
	if c.Where == Untracked && errors.As(err, &exit) && exit.ExitCode() == 1 {
		err = nil
	}
	return out.buf.String(), out.cut, err
}

// capped keeps the first max bytes written to it and drops the rest.
type capped struct {
	buf bytes.Buffer
	max int
	cut bool
}

func (c *capped) Write(p []byte) (int, error) {
	if room := c.max - c.buf.Len(); room < len(p) {
		c.buf.Write(p[:max(room, 0)])
		c.cut = true
		return len(p), nil
	}
	return c.buf.Write(p)
}

// PR is the pull request of a branch as gh tells it.
type PR struct {
	Title string `json:"title"`
	State string `json:"state"`
	URL   string `json:"url"`
}

// PRWait is how long PullRequest waits for gh, which asks the network.
const PRWait = 3 * time.Second

// PullRequest asks gh for the pull request of the branch checked out at
// dir. ok is false when gh is missing, not signed in, finds none, or does
// not answer within PRWait: the caller then shows nothing.
func PullRequest(dir string) (pr PR, ok bool) {
	ctx, cancel := context.WithTimeout(context.Background(), PRWait)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", "pr", "view", "--json", "title,state,url")
	cmd.Dir = dir
	cmd.Env = append(Environ(), "GH_PROMPT_DISABLED=1", "GH_NO_UPDATE_NOTIFIER=1", "NO_COLOR=1")
	out, err := cmd.Output()
	if err != nil || json.Unmarshal(out, &pr) != nil || pr.Title == "" {
		return PR{}, false
	}
	return pr, true
}
