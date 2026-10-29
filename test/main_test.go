// Package test holds the checks of the repository as a whole: the README
// against the program's help, the plugin's files against each other and
// the changelog, the hook scripts, the links of every Markdown file, and
// where a time layout may be written. They read files and run the built
// program, and reach no package's internals; the unit tests stay beside
// the code they test. Run them with go test ./test/...
package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

// repoRoot is the repository, one level up from this package.
var repoRoot = ".."

// pluginRoot is the plugin, ${CLAUDE_PLUGIN_ROOT} once installed: the
// folder both marketplaces at the repository's root point at.
var pluginRoot = filepath.Join(repoRoot, "plugin")

// The program, built once for the tests that run it.
var (
	buildOnce sync.Once
	buildDir  string
	binPath   string
	buildOut  []byte
	buildErr  error
)

func TestMain(m *testing.M) {
	code := m.Run()
	if buildDir != "" {
		os.RemoveAll(buildDir)
	}
	os.Exit(code)
}

// program builds cmd/stickypane, once, and returns the binary's path.
func program(t *testing.T) string {
	t.Helper()
	buildOnce.Do(func() {
		buildDir, buildErr = os.MkdirTemp("", "stickypane-test-")
		if buildErr != nil {
			return
		}
		binPath = filepath.Join(buildDir, "stickypane")
		if runtime.GOOS == "windows" {
			binPath += ".exe"
		}
		cmd := exec.Command("go", "build", "-o", binPath, "./cmd/stickypane")
		cmd.Dir = repoRoot
		buildOut, buildErr = cmd.CombinedOutput()
	})
	if buildErr != nil {
		t.Fatalf("building stickypane: %v\n%s", buildErr, buildOut)
	}
	return binPath
}

// run runs the built program in an empty folder with an empty home, in the
// language given, and returns what it printed.
func run(t *testing.T, lang string, args ...string) string {
	t.Helper()
	bin := program(t)
	home := t.TempDir()
	cmd := exec.Command(bin, args...)
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), "HOME="+home, "USERPROFILE="+home, "STICKYPANE_LANG="+lang)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("stickypane %v: %v", args, err)
	}
	return string(out)
}
