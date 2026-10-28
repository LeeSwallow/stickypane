package shell

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitProjectRefusesHomeAndRoot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	r := newRig(t)
	for _, dir := range []string{home, "/"} {
		if err := r.s.InitProject(dir); err == nil {
			t.Errorf("InitProject(%q) did not refuse", dir)
		}
	}
}

func TestInspectSaysWhetherABoardIsThere(t *testing.T) {
	r := newRig(t)
	dir := t.TempDir()
	if got := r.s.Inspect(dir); got.Exists {
		t.Errorf("Inspect of an empty folder: %+v", got)
	}
	if err := os.Mkdir(filepath.Join(dir, ".sticky"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := r.s.Inspect(dir); !got.Exists {
		t.Errorf("Inspect with .sticky: %+v", got)
	}
}
