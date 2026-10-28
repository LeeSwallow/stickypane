package bridge

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// ErrTooOld: stickypane was found, but no copy of it has the bridge.
var ErrTooOld = errors.New("bridge: stickypane too old")

// probeTimeout bounds each `stickypane help` and `stickypane version`. It
// is long because the first run of a new program on macOS can wait for
// Gatekeeper's scan; a probe that runs out of time is not remembered.
var probeTimeout = 10 * time.Second

// Candidate is one stickypane program on this machine and what it said of
// itself.
type Candidate struct {
	Path    string `json:"path"`
	Version string `json:"version"`
	Bridge  bool   `json:"bridge"` // its help lists `stickypane bridge`
}

// TooOldError lists the copies of stickypane that were found, none of
// which has the bridge.
type TooOldError struct{ Found []Candidate }

func (e *TooOldError) Error() string {
	var parts []string
	for _, c := range e.Found {
		v := c.Version
		if v == "" {
			v = "(version unknown)"
		}
		parts = append(parts, fmt.Sprintf("stickypane %s at %s", v, c.Path))
	}
	return "Found " + strings.Join(parts, ", ") +
		", which is too old for the overlay (it has no `bridge`). Update stickypane."
}

func (e *TooOldError) Is(target error) bool { return target == ErrTooOld }

// searchDirs are where Find looks after PATH: a GUI app does not get the
// shell's PATH, so the usual homes of a command line program are tried.
var searchDirs = func() []string {
	dirs := []string{"/opt/homebrew/bin", "/usr/local/bin"}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, "go", "bin"))
	}
	return dirs
}

// executable is os.Executable; tests put the app in a fake bundle.
var executable = os.Executable

// BundledCLI returns the stickypane that ships inside the app, if it is
// there: Contents/Helpers/stickypane of the macOS bundle, or the
// stickypane next to the app's binary on Linux (docs/overlay/09 §9.1).
func BundledCLI() (path string, ok bool) {
	exe, err := executable()
	if err != nil {
		return "", false
	}
	if r, err := filepath.EvalSymlinks(exe); err == nil {
		exe = r
	}
	p := bundledPath(runtime.GOOS, exe)
	return p, isExecutable(p)
}

// bundledPath is where the app whose binary is exe keeps its stickypane.
func bundledPath(goos, exe string) string {
	dir := filepath.Dir(exe)
	switch goos {
	case "darwin":
		return filepath.Join(dir, "..", "Helpers", "stickypane")
	case "windows":
		return filepath.Join(dir, "stickypane.exe")
	}
	return filepath.Join(dir, "stickypane")
}

// Find returns the path of a stickypane program that has the bridge. It
// looks at the one bundled in the app (BundledCLI), then STICKYPANE_BIN,
// then every directory of PATH, then
// /opt/homebrew/bin, /usr/local/bin and ~/go/bin, and takes the first
// whose `stickypane help` lists `stickypane bridge`: an older copy
// earlier on PATH (Homebrew's, say) is passed over for a newer one later.
// An old copy is only ever run with help and version, never with bridge,
// which it would take for a folder and open its terminal board.
//
// A STICKYPANE_BIN that names no program is an error (ErrNotFound); copies
// that are all too old are a *TooOldError (ErrTooOld) naming them.
func Find() (string, error) {
	if p := os.Getenv("STICKYPANE_BIN"); p != "" && !isExecutable(p) {
		return "", fmt.Errorf("%w: STICKYPANE_BIN=%s", ErrNotFound, p)
	}
	paths := candidates()
	if len(paths) == 0 {
		return "", ErrNotFound
	}
	var old []Candidate
	for _, p := range paths {
		c := Probe(p)
		if c.Bridge {
			return p, nil
		}
		old = append(old, c)
	}
	return "", &TooOldError{Found: old}
}

// candidates are the stickypane programs on this machine, in the order
// Find prefers them, each once.
func candidates() []string {
	name := "stickypane"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	var out []string
	seen := map[string]bool{}
	add := func(p string) {
		if p == "" || !isExecutable(p) {
			return
		}
		key := filepath.Clean(p)
		if r, err := filepath.EvalSymlinks(p); err == nil {
			key = r
		}
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, p)
	}
	if p, ok := BundledCLI(); ok {
		add(p)
	}
	add(os.Getenv("STICKYPANE_BIN"))
	for _, d := range filepath.SplitList(os.Getenv("PATH")) {
		if d != "" {
			add(filepath.Join(d, name))
		}
	}
	for _, d := range searchDirs() {
		add(filepath.Join(d, name))
	}
	return out
}

// probes remembers what each program said, by path, size and time, so
// that Find runs help once per installed copy.
var probes = struct {
	sync.Mutex
	m map[string]probed
}{m: map[string]probed{}}

type probed struct {
	size int64
	mod  time.Time
	c    Candidate
}

func resetProbes() {
	probes.Lock()
	probes.m = map[string]probed{}
	probes.Unlock()
}

// Probe asks a stickypane program for its help and version. It never
// runs it with bridge.
func Probe(path string) Candidate {
	fi, err := os.Stat(path)
	if err != nil {
		return Candidate{Path: path}
	}
	probes.Lock()
	if p, ok := probes.m[path]; ok && p.size == fi.Size() && p.mod.Equal(fi.ModTime()) {
		probes.Unlock()
		return p.c
	}
	probes.Unlock()

	c := Candidate{Path: path}
	help, _, late := run(path, "help") // an old copy may exit non-zero; its words still count
	c.Bridge = strings.Contains(help, "stickypane bridge")
	v, err, lateV := run(path, "version")
	if err == nil {
		c.Version = strings.TrimPrefix(strings.TrimSpace(v), "stickypane ")
	}
	if late || lateV { // no answer yet: ask again next time
		return c
	}
	probes.Lock()
	probes.m[path] = probed{size: fi.Size(), mod: fi.ModTime(), c: c}
	probes.Unlock()
	return c
}

// run runs the program with one argument and no input, briefly. late
// says it ran out of time.
func run(path, arg string) (out string, err error, late bool) {
	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, arg)
	cmd.WaitDelay = time.Second
	b, err := cmd.CombinedOutput()
	return string(b), err, ctx.Err() != nil
}

func isExecutable(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.Mode().IsRegular() && (runtime.GOOS == "windows" || fi.Mode().Perm()&0o111 != 0)
}
