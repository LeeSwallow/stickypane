package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseNameStatusAndNumstat(t *testing.T) {
	t.Parallel()
	names := ParseNameStatus([]byte("A\x00added.txt\x00R086\x00old.txt\x00new.txt\x00M\x00image.bin\x00"))
	want := []Change{{Status: "A", Path: "added.txt"}, {Status: "R", Old: "old.txt", Path: "new.txt"}, {Status: "M", Path: "image.bin"}}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("ParseNameStatus = %+v", names)
	}
	counts := ParseNumstat([]byte("2\t0\tadded.txt\x001\t1\t\x00old.txt\x00new.txt\x00-\t-\timage.bin\x00"))
	if counts["added.txt"].Add != 2 || counts["new.txt"].Del != 1 || !counts["image.bin"].Binary || len(counts) != 3 {
		t.Errorf("ParseNumstat = %+v", counts)
	}
}

// repo makes a repository with a commit on main and a worktree on feat
// that renames, edits, deletes and adds files, makes one binary and
// changes a mode; then leaves a change and a new file uncommitted.
func repo(t *testing.T) (main, wt string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	dir := t.TempDir()
	main, wt = filepath.Join(dir, "repo"), filepath.Join(dir, "wt")
	git := func(in string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = in
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	write := func(in, name, text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(in, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	os.Mkdir(main, 0o755)
	git(main, "init", "-q", "-b", "main")
	write(main, "moved.txt", "one\ntwo\nthree\nfour\nfive\n")
	write(main, "image.bin", "\x00\x01bin")
	write(main, "run.sh", "echo hi\n")
	write(main, "gone.txt", "bye\n")
	write(main, "shared.go", "package x\n")
	git(main, "add", "-A")
	git(main, "commit", "-q", "-m", "base")
	git(main, "worktree", "add", "-q", "-b", "feat", wt)
	git(wt, "mv", "moved.txt", "moved-to.txt")
	write(wt, "image.bin", "\x00\x02bin")
	os.Chmod(filepath.Join(wt, "run.sh"), 0o755)
	git(wt, "rm", "-q", "gone.txt")
	write(wt, "shared.go", "package x\n\nvar y = 1")
	git(wt, "add", "-A")
	git(wt, "commit", "-q", "-m", "change")
	// The base moves on after the branch left it: that is not the branch's.
	write(main, "later.txt", "later\n")
	git(main, "add", "-A")
	git(main, "commit", "-q", "-m", "later")
	write(wt, "moved-to.txt", "one\ntwo\nthree\nfour\nfive\nsix\n")
	write(wt, "new.txt", "fresh\n")
	return main, wt
}

func TestChangesSinceTheBase(t *testing.T) {
	_, wt := repo(t)
	list, err := Changes(wt, "main")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]Change{}
	for _, c := range list {
		got[c.Path] = c
	}
	if len(list) != 5 || got["later.txt"].Path != "" {
		t.Fatalf("Changes = %+v", list)
	}
	if c := got["moved-to.txt"]; c.Status != "R" || c.Old != "moved.txt" {
		t.Errorf("rename = %+v", c)
	}
	if c := got["image.bin"]; !c.Binary {
		t.Errorf("binary = %+v", c)
	}
	if c := got["gone.txt"]; c.Status != "D" || c.Del != 1 {
		t.Errorf("deleted = %+v", c)
	}
	if c := got["shared.go"]; c.Add != 2 || c.Del != 0 {
		t.Errorf("edit = %+v", c)
	}
	text, cut, err := Diff(wt, "main", got["run.sh"])
	if err != nil || cut || !strings.Contains(text, "old mode 100644\nnew mode 100755") {
		t.Errorf("Diff(run.sh) = %q, %v, %v", text, cut, err)
	}
	text, _, err = Diff(wt, "main", got["moved-to.txt"])
	if err != nil || !strings.Contains(text, "rename from moved.txt") {
		t.Errorf("Diff(rename) = %q, %v", text, err)
	}
	text, _, _ = Diff(wt, "main", got["shared.go"])
	if !strings.Contains(text, `\ No newline at end of file`) {
		t.Errorf("Diff(shared.go) = %q", text)
	}
}

func TestPendingChanges(t *testing.T) {
	_, wt := repo(t)
	list, err := Pending(wt)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].Path != "moved-to.txt" || list[0].Where != Uncommitted || list[0].Add != 1 ||
		list[1].Path != "new.txt" || list[1].Where != Untracked {
		t.Fatalf("Pending = %+v", list)
	}
	text, _, err := Diff(wt, "", list[1])
	if err != nil || !strings.Contains(text, "+fresh") || !strings.Contains(text, "new file mode") {
		t.Errorf("Diff(untracked) = %q, %v", text, err)
	}
	text, _, err = Diff(wt, "", list[0])
	if err != nil || !strings.Contains(text, "+six") {
		t.Errorf("Diff(uncommitted) = %q, %v", text, err)
	}
}

func TestADiffIsReadNoFurtherThanItsCap(t *testing.T) {
	c := &capped{max: 4}
	c.Write([]byte("ab"))
	c.Write([]byte("cdef"))
	if c.buf.String() != "abcd" || !c.cut {
		t.Errorf("capped = %q, %v", c.buf.String(), c.cut)
	}
}

func TestOriginHead(t *testing.T) {
	main, wt := repo(t)
	if h := OriginHead(wt); h != "" {
		t.Errorf("no remote: OriginHead = %q", h)
	}
	for _, args := range [][]string{{"update-ref", "refs/remotes/origin/main", "main"}, {"symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main"}} {
		if out, err := exec.Command("git", append([]string{"-C", main}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
	}
	if h := OriginHead(wt); h != "origin/main" {
		t.Errorf("OriginHead = %q", h)
	}
}

// Without gh, or outside a repository gh knows, there is no pull request
// and no wait.
func TestPullRequestWithoutGh(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if _, ok := PullRequest(t.TempDir()); ok {
		t.Error("a pull request without gh")
	}
}
