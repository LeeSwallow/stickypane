package shell

import (
	"os"
	"path/filepath"
	"runtime"
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

func TestInitProjectRefusesASymlinkToHome(t *testing.T) {
	home := t.TempDir()
	old := userHomeDir
	userHomeDir = func() (string, error) { return home, nil }
	t.Cleanup(func() { userHomeDir = old })
	link := filepath.Join(t.TempDir(), "h")
	if err := os.Symlink(home, link); err != nil {
		t.Fatal(err)
	}
	r := newRig(t)
	if err := r.s.InitProject(link); err == nil {
		t.Error("InitProject through a symlink to home did not refuse")
	}
	if runtime.GOOS == "darwin" && !samePath("/Users/Min", "/users/min") {
		t.Error("samePath is case-sensitive on macOS")
	}
	if runtime.GOOS != "darwin" && samePath("/a/B", "/a/b") {
		t.Error("samePath is case-insensitive off macOS")
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
