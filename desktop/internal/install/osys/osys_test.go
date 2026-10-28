package osys

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/desktop/internal/install/ports"
)

func TestAppleScriptQuotesTheScript(t *testing.T) {
	got := AppleScript(`ln -sfn '/a "b"\c' /usr/local/bin/stickypane`)
	want := `do shell script "ln -sfn '/a \"b\"\\c' /usr/local/bin/stickypane" with administrator privileges`
	if got != want {
		t.Errorf("AppleScript =\n%s\nwant\n%s", got, want)
	}
}

func TestRunnerRunsWithThePathGiven(t *testing.T) {
	dir := t.TempDir()
	out, err := Runner{}.Run(context.Background(), ports.Cmd{Dir: dir, Path: "/x:/usr/bin:/bin", Name: "/bin/sh", Args: []string{"-c", `printf '%s %s' "$PATH" "$(pwd -P)"`}})
	real, _ := filepath.EvalSymlinks(dir)
	if err != nil || out != "/x:/usr/bin:/bin "+real {
		t.Errorf("%q %v", out, err)
	}
}

// Files only ever act on paths given; here, a temporary folder.
func TestFilesSeeLinksProgramsAndDanglingLinks(t *testing.T) {
	dir := t.TempDir()
	prog := filepath.Join(dir, "stickypane")
	if err := os.WriteFile(prog, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	var f Files
	link := filepath.Join(dir, "bin", "stickypane")
	if err := f.MkdirAll(filepath.Dir(link)); err != nil {
		t.Fatal(err)
	}
	if err := f.Symlink(prog, link); err != nil {
		t.Fatal(err)
	}
	real, _ := filepath.EvalSymlinks(prog)
	if e := f.Stat(link); !e.Exists || !e.Link || !e.Exec || e.Real != real {
		t.Errorf("link: %+v", e)
	}
	if e := f.Stat(prog); !e.Exists || e.Link || !e.Exec {
		t.Errorf("program: %+v", e)
	}
	if err := f.Remove(prog); err != nil {
		t.Fatal(err)
	}
	if e := f.Stat(link); !e.Exists || !e.Link || e.Exec || e.Real != "" {
		t.Errorf("dangling: %+v", e)
	}
	if e := f.Stat(filepath.Join(dir, "none")); e.Exists {
		t.Errorf("nothing: %+v", e)
	}
	tmp, err := f.TempDir()
	if err != nil || !strings.Contains(filepath.Base(tmp), "stickypane-setup-") {
		t.Fatalf("%q %v", tmp, err)
	}
	if err := f.RemoveAll(tmp); err != nil {
		t.Fatal(err)
	}
}
