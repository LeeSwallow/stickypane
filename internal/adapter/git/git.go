// Package git reads what git says about the worktrees of a repository. It
// runs git and changes nothing; where git is missing or the folder is not
// in a repository, it answers nothing.
package git

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
)

// Worktree is one entry of `git worktree list --porcelain`.
type Worktree struct {
	Path     string
	Branch   string // the branch without refs/heads/; "" when detached or bare
	Bare     bool
	Locked   bool // git keeps it: a tool such as Claude Code holds it while an agent runs
	Prunable bool // its folder is gone; git prune would forget it
}

// Parse reads the output of `git worktree list --porcelain`: one entry a
// paragraph, the main worktree first.
func Parse(out []byte) []Worktree {
	var list []Worktree
	for _, para := range strings.Split(string(out), "\n\n") {
		var w Worktree
		for _, line := range strings.Split(para, "\n") {
			key, value, _ := strings.Cut(line, " ")
			switch key {
			case "worktree":
				w.Path = value
			case "branch":
				w.Branch = strings.TrimPrefix(value, "refs/heads/")
			case "bare":
				w.Bare = true
			case "locked":
				w.Locked = true
			case "prunable":
				w.Prunable = true
			}
		}
		if w.Path != "" {
			list = append(list, w)
		}
	}
	return list
}

// Worktrees lists the worktrees of the repository dir is in, the main one
// first.
func Worktrees(dir string) ([]Worktree, error) {
	out, err := run(dir, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return Parse(out), nil
}

// Main is the path of the main worktree of the repository dir is in, or ""
// when there is none: no git, no repository, or a bare one.
func Main(dir string) string {
	list, err := Worktrees(dir)
	if err != nil || len(list) == 0 || list[0].Bare {
		return ""
	}
	return list[0].Path
}

// CommonDir is the absolute path of the folder every worktree of dir's
// repository shares: the main worktree's .git.
func CommonDir(dir string) (string, error) {
	out, err := run(dir, "rev-parse", "--git-common-dir")
	if err != nil {
		return "", err
	}
	p := strings.TrimSpace(string(out))
	if !filepath.IsAbs(p) {
		p = filepath.Join(dir, p)
	}
	return filepath.Clean(p), nil
}

func run(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	return out.Bytes(), err
}
