package main

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LeeSwallow/stickypane/internal/adapter/store"
)

// grammarPairs are the forms before <noun> <verb> <target> (#149) and the
// forms that replace them, in an order where each step has what it needs.
var grammarPairs = []struct {
	stdin    string
	old, new []string
}{
	{"- [ ] a\n", []string{"write", "plan", "--type", "checklist"}, []string{"note", "write", "plan", "--type", "checklist"}},
	{"", []string{"todo", "plan", "add", "x"}, []string{"todo", "add", "plan", "x"}},
	{"", []string{"todo", "plan", "check", "x"}, []string{"todo", "check", "plan", "x"}},
	{"", []string{"todo", "plan", "uncheck", "x"}, []string{"todo", "uncheck", "plan", "x"}},
	{"", []string{"card", "work", "add", "login", "--to", "Doing"}, []string{"card", "add", "work", "login", "--to", "Doing"}},
	{"", []string{"card", "work", "move", "login", "--to", "Done"}, []string{"card", "move", "work", "login", "--to", "Done"}},
	{"", []string{"chart", "tokens", "add", "input", "1200"}, []string{"chart", "add", "tokens", "input", "1200"}},
	{"", []string{"chart", "tokens", "set", "output", "-5"}, []string{"chart", "set", "tokens", "output", "-5"}},
	{"", []string{"log", "worklog", "--time", "x"}, []string{"log", "add", "worklog", "--time", "x"}},
	{"", []string{"say", "chat", "x"}, []string{"chat", "add", "chat", "x"}},
	{"", []string{"say", "chat", "y", "--as", "ci"}, []string{"chat", "add", "chat", "y", "--as", "ci"}},
	{"", []string{"cat", "plan"}, []string{"note", "cat", "plan"}},
	{"", []string{"show", "plan"}, []string{"note", "show", "plan"}},
	{"", []string{"hide", "plan"}, []string{"note", "hide", "plan"}},
	{"", []string{"set", "plan", "size=1/2"}, []string{"note", "set", "plan", "size=1/2"}},
	{"", []string{"list"}, []string{"note", "list"}},
	{"", []string{"link", "README.md"}, []string{"note", "link", "README.md"}},
	{"", []string{"mv", "plan", "roadmap"}, []string{"note", "mv", "plan", "roadmap"}},
	{"", []string{"archive", "roadmap"}, []string{"note", "archive", "roadmap"}},
	{"", []string{"rm", "worklog"}, []string{"note", "rm", "worklog"}},
	{"", []string{"restore", "worklog"}, []string{"note", "restore", "worklog"}},
}

// Every old form and its new form give the same output and leave the same
// files, step by step.
func TestOldAndNewFormsDoTheSame(t *testing.T) {
	at := time.Date(2026, 10, 3, 14, 2, 5, 0, time.Local)
	old, new := project(t), project(t)
	for _, root := range []string{old, new} {
		if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# hi\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	step := func(root, stdin string, args []string) string {
		t.Chdir(root)
		var out, errOut bytes.Buffer
		e := env{ctx: context.Background(), stdin: strings.NewReader(stdin), stdout: &out, stderr: &errOut, now: func() time.Time { return at }}
		code := runEnv(e, args)
		if code != 0 {
			t.Errorf("%v: code %d, %s", args, code, errOut.String())
		}
		return out.String() + errOut.String()
	}
	for _, p := range grammarPairs {
		a, b := step(old, p.stdin, p.old), step(new, p.stdin, p.new)
		if a != b {
			t.Errorf("%v says %q, %v says %q", p.old, a, p.new, b)
		}
		if a, b := tree(t, old), tree(t, new); a != b {
			t.Fatalf("after %v and %v the boards differ:\n%s\n---\n%s", p.old, p.new, a, b)
		}
	}
}

// tree is every file of the board under root, with its contents.
func tree(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	dir := filepath.Join(root, store.DirName)
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		var body []byte
		if d.Type()&fs.ModeSymlink != 0 {
			to, err := os.Readlink(path)
			body = []byte("-> " + to)
			if err != nil {
				return err
			}
		} else if body, err = os.ReadFile(path); err != nil {
			return err
		}
		b.WriteString("== " + rel + "\n" + string(body) + "\n")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// A shape command whose first word is a verb of the noun reads in the new
// order, even when the second word is a verb too.
func TestTheNewOrderWinsWhenANoteIsNamedLikeAVerb(t *testing.T) {
	root := project(t)
	if code, _, errOut := exec(t, "todo", "add", "check", "x"); code != 0 {
		t.Fatal(errOut)
	}
	if _, err := os.Stat(filepath.Join(root, store.DirName, "check.md")); err != nil {
		t.Errorf("todo add check x adds x to the note check: %v", err)
	}
}
