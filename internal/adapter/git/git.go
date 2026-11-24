// Package git reads what git says about the worktrees of a repository. It
// runs git and changes nothing; where git is missing or the folder is not
// in a repository, it answers nothing.
package git

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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
// first. Where dir is in no repository, the error is ErrNoRepository.
func Worktrees(dir string) ([]Worktree, error) {
	out, err := run(dir, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return Parse(out), nil
}

// Main is the path of the main worktree of the repository whose worktree
// is at dir, or "" when there is none: no git, no repository, a bare one,
// or a .git file that git's own records do not point back to. A .git file
// is only a line of text, one an archive could carry: it is believed only
// when it names <main>/.git/worktrees/<id> and that folder's gitdir file
// names this .git again.
func Main(dir string) string {
	dotgit := filepath.Join(dir, ".git")
	fi, err := os.Lstat(dotgit)
	if err != nil || !(fi.IsDir() || fi.Mode().IsRegular()) {
		return ""
	}
	list, err := Worktrees(dir)
	if err != nil || len(list) == 0 || list[0].Bare {
		return ""
	}
	main := list[0].Path
	if fi.Mode().IsRegular() && !linked(dotgit, main) {
		return ""
	}
	return main
}

// linked says whether the .git file dotgit names a worktree folder of the
// repository at main, and that folder names dotgit back.
func linked(dotgit, main string) bool {
	gitdir, ok := pointer(dotgit, "gitdir: ")
	if !ok || !Same(filepath.Dir(gitdir), filepath.Join(main, ".git", "worktrees")) {
		return false
	}
	back, ok := pointer(filepath.Join(gitdir, "gitdir"), "")
	return ok && Same(back, dotgit)
}

// pointer reads the path in the one-line file at p after prefix, taken
// relative to p's folder when it is not absolute.
func pointer(p, prefix string) (string, bool) {
	b, err := os.ReadFile(p)
	if err != nil {
		return "", false
	}
	v, ok := strings.CutPrefix(strings.TrimSpace(string(b)), prefix)
	if !ok || v == "" {
		return "", false
	}
	if !filepath.IsAbs(v) {
		v = filepath.Join(filepath.Dir(p), v)
	}
	return v, true
}

// Same says whether a and b are the same file or folder.
func Same(a, b string) bool {
	fa, err1 := os.Stat(a)
	fb, err2 := os.Stat(b)
	return err1 == nil && err2 == nil && os.SameFile(fa, fb)
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

// Local lists the variables by which git is told which repository it is in
// (git rev-parse --local-env-vars). git sets them for a hook or a rebase
// --exec; a git started here must find the repository from its folder.
var Local = []string{
	"GIT_ALTERNATE_OBJECT_DIRECTORIES", "GIT_CONFIG", "GIT_CONFIG_PARAMETERS",
	"GIT_CONFIG_COUNT", "GIT_OBJECT_DIRECTORY", "GIT_DIR", "GIT_WORK_TREE",
	"GIT_IMPLICIT_WORK_TREE", "GIT_GRAFT_FILE", "GIT_INDEX_FILE",
	"GIT_NO_REPLACE_OBJECTS", "GIT_REPLACE_REF_BASE", "GIT_PREFIX",
	"GIT_SHALLOW_FILE", "GIT_COMMON_DIR",
}

// Environ is this process's environment without the variables in Local.
func Environ() []string {
	var env []string
	for _, kv := range os.Environ() {
		if name, _, _ := strings.Cut(kv, "="); !slices.Contains(Local, name) {
			env = append(env, kv)
		}
	}
	return env
}

// ErrNoRepository is git saying the folder is in no repository.
var ErrNoRepository = errors.New("not a git repository")

// command is git run in dir. No fsmonitor: a repository's config must not
// start a program here. Paths come back as they are, not with their bytes
// escaped.
func command(dir string, args ...string) *exec.Cmd {
	cmd := exec.Command("git", append([]string{"-c", "core.fsmonitor=false", "-c", "core.quotePath=false", "-C", dir}, args...)...)
	cmd.Env = Environ()
	return cmd
}

func run(dir string, args ...string) ([]byte, error) {
	// git speaks English (LC_ALL=C), so that its complaint can be read.
	cmd := command(dir, args...)
	cmd.Env = append(cmd.Env, "LC_ALL=C")
	var out, complaint bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &complaint
	err := cmd.Run()
	if err != nil && strings.Contains(complaint.String(), "not a git repository") {
		err = ErrNoRepository
	}
	return out.Bytes(), err
}
