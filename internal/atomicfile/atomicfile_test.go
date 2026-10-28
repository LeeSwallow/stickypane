package atomicfile

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// leftovers fails when a temporary file is left in dir.
func leftovers(t *testing.T, dir string, want ...string) {
	t.Helper()
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var have []string
	for _, e := range ents {
		have = append(have, e.Name())
	}
	if len(have) != len(want) {
		t.Fatalf("folder holds %q, want %q", have, want)
	}
	for i := range want {
		if have[i] != want[i] {
			t.Fatalf("folder holds %q, want %q", have, want)
		}
	}
}

func TestWriteFileMakesAndReplaces(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.json")
	if err := WriteFile(p, []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := read(t, p); got != "one" {
		t.Fatalf("got %q", got)
	}
	if err := WriteFile(p, []byte("two"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := read(t, p); got != "two" {
		t.Fatalf("got %q", got)
	}
	leftovers(t, dir, "a.json")
	if runtime.GOOS != "windows" {
		// A new file takes perm; a file that is there keeps its mode.
		if fi, _ := os.Stat(p); fi.Mode().Perm() != 0o600 {
			t.Fatalf("mode %v, want 0600", fi.Mode().Perm())
		}
	}
}

func TestWriteFileWritesThroughALink(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(real, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "CLAUDE.md")
	if err := os.Symlink("AGENTS.md", link); err != nil {
		t.Skip("no symbolic links here:", err)
	}
	if err := WriteFile(link, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Lstat(link); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatal("the link was replaced by a file")
	}
	if got := read(t, real); got != "new" {
		t.Fatalf("target holds %q", got)
	}
}

func TestWriteFileIntoAMissingFolderFails(t *testing.T) {
	p := filepath.Join(t.TempDir(), "no", "a.md")
	if err := WriteFile(p, []byte("x"), 0o644); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("err = %v, want fs.ErrNotExist", err)
	}
}

func TestCreateNeverWritesOver(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "welcome.md")
	if err := Create(p, []byte("first"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Create(p, []byte("second"), 0o644); !errors.Is(err, fs.ErrExist) {
		t.Fatalf("err = %v, want fs.ErrExist", err)
	}
	if got := read(t, p); got != "first" {
		t.Fatalf("got %q", got)
	}
	leftovers(t, dir, "welcome.md")
}
