package bridge

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// fakeStickypane writes a stickypane into dir that answers help and
// version as a release of the given version would, and logs every
// argument list it is run with to log.
func fakeStickypane(t *testing.T, dir, version string, hasBridge bool, log string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell scripts")
	}
	usage := `  stickypane init        make .sticky/`
	if hasBridge {
		usage += "\n  stickypane bridge              serve the board to the desktop app"
	}
	script := "#!/bin/sh\n" +
		"echo \"$@\" >> '" + log + "'\n" +
		"case \"$1\" in\n" +
		"help) /bin/cat <<'EOF'\nstickypane - a note board\n\nUsage:\n" + usage + "\nEOF\n;;\n" +
		"version) echo 'stickypane " + version + "' ;;\n" +
		"esac\n"
	p := filepath.Join(dir, "stickypane")
	if err := os.WriteFile(p, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func isolate(t *testing.T, path string, dirs ...string) {
	t.Helper()
	old := searchDirs
	searchDirs = func() []string { return dirs }
	oldExe := executable
	nowhere := filepath.Join(t.TempDir(), "app", "Contents", "MacOS", "stickypane-app")
	executable = func() (string, error) { return nowhere, nil }
	t.Cleanup(func() { searchDirs, executable = old, oldExe })
	t.Setenv("STICKYPANE_BIN", "")
	t.Setenv("PATH", path)
	resetProbes()
}

func TestFindPrefersAStickypaneWithTheBridge(t *testing.T) {
	log := filepath.Join(t.TempDir(), "runs.log")
	brew, gobin := t.TempDir(), t.TempDir()
	fakeStickypane(t, brew, "0.1.1", false, log)
	want := fakeStickypane(t, gobin, "0.2.0", true, log)
	isolate(t, brew, gobin) // the old one is first on PATH

	got, err := Find()
	if err != nil || got != want {
		t.Fatalf("Find = %q, %v; want %q", got, err, want)
	}
	c := Probe(got)
	if !c.Bridge || c.Version != "0.2.0" {
		t.Errorf("Probe = %+v", c)
	}
	runs, _ := os.ReadFile(log)
	if strings.Contains(string(runs), "bridge") {
		t.Errorf("an old stickypane was run with bridge:\n%s", runs)
	}
}

func TestFindRefusesAStickypaneTooOldForTheBridge(t *testing.T) {
	log := filepath.Join(t.TempDir(), "runs.log")
	brew := t.TempDir()
	old := fakeStickypane(t, brew, "0.1.1", false, log)
	isolate(t, brew)

	_, err := Find()
	if !errors.Is(err, ErrTooOld) {
		t.Fatalf("err = %v, want ErrTooOld", err)
	}
	var too *TooOldError
	if !errors.As(err, &too) || len(too.Found) != 1 || too.Found[0].Path != old || too.Found[0].Version != "0.1.1" {
		t.Fatalf("found %+v", too)
	}
	for _, w := range []string{old, "0.1.1", "too old", "bridge", "Update stickypane"} {
		if !strings.Contains(err.Error(), w) {
			t.Errorf("the error does not say %q: %s", w, err)
		}
	}
	runs, _ := os.ReadFile(log)
	if strings.Contains(string(runs), "bridge") {
		t.Errorf("an old stickypane was run with bridge:\n%s", runs)
	}
}

// The first run of a new program on macOS may wait for a malware scan; a
// probe that ran out of time is no answer and is asked again.
func TestASlowFirstProbeIsAskedAgain(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell scripts")
	}
	dir := t.TempDir()
	once := filepath.Join(dir, "scanned")
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = help ] && [ ! -f '" + once + "' ]; then /usr/bin/touch '" + once + "'; /bin/sleep 4; fi\n" +
		"case \"$1\" in\n" +
		"help) echo '  stickypane bridge   serve the board' ;;\n" +
		"version) echo 'stickypane 0.2.0' ;;\n" +
		"esac\n"
	p := filepath.Join(dir, "stickypane")
	if err := os.WriteFile(p, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	// The real first-run scan of this script happens here, untimed, so
	// that only the script's own sleep makes the first probe late.
	if err := exec.Command(p, "warm").Run(); err != nil {
		t.Fatal(err)
	}
	isolate(t, dir)
	old := probeTimeout
	probeTimeout = time.Second
	t.Cleanup(func() { probeTimeout = old })

	if c := Probe(p); c.Bridge {
		t.Fatalf("the slow first probe answered %+v", c)
	}
	if got, err := Find(); err != nil || got != p {
		t.Fatalf("Find after a slow first probe = %q, %v; want %q", got, err, p)
	}
}

func TestOldStickypaneBinIsPassedOver(t *testing.T) {
	log := filepath.Join(t.TempDir(), "runs.log")
	a, b := t.TempDir(), t.TempDir()
	old := fakeStickypane(t, a, "0.1.1", false, log)
	want := fakeStickypane(t, b, "0.2.0", true, log)
	isolate(t, "", b)
	t.Setenv("STICKYPANE_BIN", old)
	if got, err := Find(); err != nil || got != want {
		t.Fatalf("Find = %q, %v; want %q", got, err, want)
	}
}

// fakeBundle lays out an app as it is installed on this OS and returns its
// binary and the directory its stickypane belongs in.
func fakeBundle(t *testing.T) (exe, cliDir string) {
	t.Helper()
	root := t.TempDir()
	if runtime.GOOS == "darwin" {
		app := filepath.Join(root, "Stickypane.app", "Contents")
		exe = filepath.Join(app, "MacOS", "stickypane-app")
		cliDir = filepath.Join(app, "Resources", "bin")
	} else {
		exe = filepath.Join(root, "stickypane-app", "stickypane-app")
		cliDir = filepath.Dir(exe)
	}
	for _, d := range []string{filepath.Dir(exe), cliDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return exe, cliDir
}

func TestFindPrefersTheStickypaneInTheApp(t *testing.T) {
	log := filepath.Join(t.TempDir(), "runs.log")
	exe, cliDir := fakeBundle(t)
	want, _ := filepath.EvalSymlinks(fakeStickypane(t, cliDir, "0.3.0", true, log))
	env, path := t.TempDir(), t.TempDir()
	isolate(t, path)
	executable = func() (string, error) { return exe, nil }
	t.Setenv("STICKYPANE_BIN", fakeStickypane(t, env, "0.2.0", true, log))
	fakeStickypane(t, path, "0.2.0", true, log)

	if got, ok := BundledCLI(); !ok || filepath.Clean(got) != want {
		t.Fatalf("BundledCLI = %q, %v; want %q", got, ok, want)
	}
	if got, err := Find(); err != nil || filepath.Clean(got) != want {
		t.Fatalf("Find = %q, %v; want the bundled %q", got, err, want)
	}
}

func TestWithoutABundledStickypaneFindLooksFurther(t *testing.T) {
	log := filepath.Join(t.TempDir(), "runs.log")
	exe, _ := fakeBundle(t)
	path := t.TempDir()
	want := fakeStickypane(t, path, "0.2.0", true, log)
	isolate(t, path)
	executable = func() (string, error) { return exe, nil }

	if got, ok := BundledCLI(); ok {
		t.Fatalf("BundledCLI = %q, true in an app without one", got)
	}
	if got, err := Find(); err != nil || got != want {
		t.Fatalf("Find = %q, %v; want %q", got, err, want)
	}
}

func TestBundledPath(t *testing.T) {
	for _, c := range []struct{ goos, exe, want string }{
		{"darwin", "/Applications/Stickypane.app/Contents/MacOS/stickypane-app",
			"/Applications/Stickypane.app/Contents/Resources/bin/stickypane"},
		{"linux", "/opt/stickypane-app/stickypane-app", "/opt/stickypane-app/stickypane"},
	} {
		if got := bundledPath(c.goos, c.exe); got != filepath.FromSlash(c.want) {
			t.Errorf("bundledPath(%s, %s) = %s, want %s", c.goos, c.exe, got, c.want)
		}
	}
}
