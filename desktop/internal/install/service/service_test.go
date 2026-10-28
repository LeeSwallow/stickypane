package service

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/LeeSwallow/stickypane/desktop/internal/install/model"
	"github.com/LeeSwallow/stickypane/desktop/internal/install/ports"
)

const (
	home    = "/home/min"
	bundled = "/Applications/Stickypane.app/Contents/Helpers/stickypane"
	brewBin = "/opt/homebrew/bin/stickypane"
	brewCsk = "/opt/homebrew/Caskroom/stickypane/0.1.1/stickypane"
	userLnk = home + "/.local/bin/stickypane"
	sysLnk  = "/usr/local/bin/stickypane"
)

// fakeFiles is a file system of paths: a program, a link (to its
// target) or a folder. Nothing touches the disk.
type fakeFiles struct {
	mu    sync.Mutex
	progs map[string]bool
	links map[string]string
	dirs  map[string]bool
	temps []string
	gone  []string
}

func newFiles() *fakeFiles {
	return &fakeFiles{progs: map[string]bool{bundled: true}, links: map[string]string{}, dirs: map[string]bool{}}
}

func (f *fakeFiles) Stat(p string) ports.Entry {
	f.mu.Lock()
	defer f.mu.Unlock()
	if t, ok := f.links[p]; ok {
		e := ports.Entry{Exists: true, Link: true}
		if f.progs[t] {
			e.Real, e.Exec = t, true
		}
		return e
	}
	if f.progs[p] {
		return ports.Entry{Exists: true, Real: p, Exec: true}
	}
	if f.dirs[p] {
		return ports.Entry{Exists: true, Real: p}
	}
	return ports.Entry{}
}
func (f *fakeFiles) MkdirAll(d string) error {
	f.mu.Lock()
	f.dirs[d] = true
	f.mu.Unlock()
	return nil
}
func (f *fakeFiles) Symlink(t, l string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.links[l]; ok || f.progs[l] {
		return errors.New("exists")
	}
	f.links[l] = t
	return nil
}
func (f *fakeFiles) Remove(p string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.links, p)
	delete(f.progs, p)
	f.gone = append(f.gone, p)
	return nil
}
func (f *fakeFiles) TempDir() (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d := "/tmp/stickypane-setup-1"
	f.dirs[d] = true
	f.temps = append(f.temps, d)
	return d, nil
}
func (f *fakeFiles) RemoveAll(d string) error {
	f.mu.Lock()
	delete(f.dirs, d)
	f.mu.Unlock()
	return nil
}

// fakeRunner answers the login shell with path, a version with versions,
// and records every other command.
type fakeRunner struct {
	mu       sync.Mutex
	path     string
	versions map[string]string
	out      string
	err      error
	ran      []ports.Cmd
}

func (r *fakeRunner) Run(_ context.Context, c ports.Cmd) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	switch {
	case len(c.Args) == 2 && c.Args[0] == "-ilc":
		return "motd\n" + model.Marker + r.path + model.Marker, nil
	case len(c.Args) == 1 && c.Args[0] == "version":
		return "stickypane " + r.versions[c.Name] + "\n", nil
	}
	r.ran = append(r.ran, c)
	return r.out, r.err
}

type fakeAdmin struct {
	scripts []string
	err     error
}

func (a *fakeAdmin) Run(_ context.Context, s string) error {
	a.scripts = append(a.scripts, s)
	return a.err
}

type rig struct {
	f *fakeFiles
	r *fakeRunner
	a *fakeAdmin
	s *Service
}

func newRig(path string, mod ...func(*Config)) rig {
	f, r, a := newFiles(), &fakeRunner{path: path, versions: map[string]string{bundled: "0.2.0", brewBin: "0.1.1"}}, &fakeAdmin{}
	cfg := Config{Files: f, Runner: r, Admin: a, Home: home, Shell: "/bin/zsh", PATH: []string{"/usr/bin", "/bin"}, Bundled: func() string { return bundled }}
	for _, m := range mod {
		m(&cfg)
	}
	return rig{f, r, a, New(cfg)}
}

var ctx = context.Background()

func TestInstallLinksIntoTheUsersBinWithoutAPassword(t *testing.T) {
	g := newRig(home + "/.local/bin:/usr/bin")
	st, err := g.s.Install(ctx, model.TargetUser, false)
	if err != nil {
		t.Fatal(err)
	}
	if g.f.links[userLnk] != bundled || !g.f.dirs[home+"/.local/bin"] || len(g.a.scripts) != 0 {
		t.Errorf("links %v dirs %v admin %v", g.f.links, g.f.dirs, g.a.scripts)
	}
	if st.First == nil || !st.First.Apps || st.Shadow != nil || !st.OnPath || st.Version != "0.2.0" {
		t.Errorf("state after install: %+v first %+v", st, st.First)
	}
	if st.Links[0].State != "apps" || st.Links[1].State != "none" {
		t.Errorf("links: %+v", st.Links)
	}
	// again: nothing changes
	if _, err := g.s.Install(ctx, model.TargetUser, false); err != nil || len(g.f.gone) != 0 {
		t.Errorf("a second install: %v, removed %v", err, g.f.gone)
	}
}

func TestInstallSaysWhenTheUsersBinIsNotOnThePath(t *testing.T) {
	g := newRig("/usr/bin:/bin")
	st, err := g.s.Install(ctx, model.TargetUser, false)
	if err != nil {
		t.Fatal(err)
	}
	if st.OnPath || st.RCFile != home+"/.zshrc" || st.RCLine != `export PATH="$HOME/.local/bin:$PATH"` || st.First != nil {
		t.Errorf("state: %+v", st)
	}
}

func TestAnOlderCopyEarlierOnThePathIsNamedWithItsFix(t *testing.T) {
	g := newRig("/opt/homebrew/bin:" + home + "/.local/bin")
	g.f.progs[brewCsk] = true
	g.f.links[brewBin] = brewCsk
	g.r.versions[brewBin] = "0.1.1"
	st, err := g.s.Install(ctx, model.TargetUser, false)
	if err != nil {
		t.Fatal(err)
	}
	if st.First == nil || st.First.Path != brewBin || st.First.Apps || st.First.Version != "0.1.1" {
		t.Fatalf("first: %+v", st.First)
	}
	if st.Shadow == nil || st.Shadow.Kind != "cask" || st.Shadow.Command != "brew uninstall --cask stickypane" || st.Shadow.Before != home+"/.local/bin" || st.Shadow.After != "/opt/homebrew/bin" {
		t.Errorf("shadow: %+v", st.Shadow)
	}
	if _, ok := g.f.links[brewBin]; !ok {
		t.Error("Homebrew's copy was touched")
	}
}

func TestAProgramThatIsNotTheAppsIsReplacedOnlyWhenAsked(t *testing.T) {
	g := newRig(home + "/.local/bin")
	g.f.progs[userLnk] = true // a stickypane put there by hand
	if _, err := g.s.Install(ctx, model.TargetUser, false); !errors.Is(err, ErrOccupied) || !g.f.progs[userLnk] {
		t.Fatalf("without asking: %v, %v", err, g.f.progs)
	}
	if _, err := g.s.Install(ctx, model.TargetUser, true); err != nil || g.f.links[userLnk] != bundled {
		t.Fatalf("asked: %v, %v", err, g.f.links)
	}
}

func TestALinkToAnotherCopyOfTheAppIsReplacedWithoutAsking(t *testing.T) {
	g := newRig(home + "/.local/bin")
	old := "/Users/min/Downloads/Stickypane.app/Contents/Helpers/stickypane"
	g.f.progs[old] = true
	g.f.links[userLnk] = old
	if _, err := g.s.Install(ctx, model.TargetUser, false); err != nil || g.f.links[userLnk] != bundled {
		t.Fatalf("%v %v", err, g.f.links)
	}
}

func TestTheSystemFolderAsksAnAdministrator(t *testing.T) {
	g := newRig("/usr/local/bin")
	if _, err := g.s.Install(ctx, model.TargetSystem, false); err != nil {
		t.Fatal(err)
	}
	if len(g.a.scripts) != 1 || g.a.scripts[0] != model.LinkScript(bundled, sysLnk) || len(g.f.links) != 0 {
		t.Errorf("admin %q links %v", g.a.scripts, g.f.links)
	}
	g.a.err = errors.New("cancelled")
	if _, err := g.s.Install(ctx, model.TargetSystem, false); err == nil {
		t.Error("a cancelled password dialog is not an error")
	}
}

func TestUninstallTakesAwayOnlyTheAppsLinks(t *testing.T) {
	g := newRig(home + "/.local/bin")
	g.f.links[userLnk] = bundled
	g.f.links[sysLnk] = bundled
	g.f.progs[brewBin] = true
	st, err := g.s.Uninstall(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := g.f.links[userLnk]; ok || len(g.a.scripts) != 1 || g.a.scripts[0] != model.UnlinkScript(sysLnk) || !g.f.progs[brewBin] {
		t.Errorf("links %v admin %q progs %v", g.f.links, g.a.scripts, g.f.progs)
	}
	if st.Links[0].State != "none" {
		t.Errorf("after: %+v", st.Links)
	}
	g = newRig(home + "/.local/bin")
	g.f.progs[userLnk] = true
	if _, err := g.s.Uninstall(ctx); err != nil || !g.f.progs[userLnk] || len(g.f.gone) != 0 {
		t.Errorf("a program not the app's was removed: %v %v", err, g.f.gone)
	}
}

func TestNoBundledCopyNoInstall(t *testing.T) {
	g := newRig("/usr/bin", func(c *Config) { c.Bundled = func() string { return "" } })
	if _, err := g.s.Install(ctx, model.TargetUser, false); !errors.Is(err, ErrNoBundled) || len(g.f.links) != 0 {
		t.Errorf("%v %v", err, g.f.links)
	}
}

func TestConnectRunsTheBundledSetupInAFolderOfItsOwn(t *testing.T) {
	g := newRig("/opt/homebrew/bin:/usr/bin")
	g.r.out = "stickypane setup runs:\n  claude plugin install board@stickypane --scope user\n"
	res, err := g.s.Connect(ctx, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.r.ran) != 1 {
		t.Fatalf("ran %+v", g.r.ran)
	}
	c := g.r.ran[0]
	want := "setup --scope user --mcp --command " + bundled + " --dry-run"
	if c.Name != bundled || strings.Join(c.Args, " ") != want || c.Dir != "/tmp/stickypane-setup-1" {
		t.Errorf("ran %+v", c)
	}
	if !strings.HasPrefix(c.Path, "/opt/homebrew/bin:/usr/bin:/bin:"+home+"/.local/bin") {
		t.Errorf("PATH %q", c.Path)
	}
	if res.Command != "stickypane "+want || !strings.Contains(res.Output, "claude plugin install") {
		t.Errorf("result %+v", res)
	}
	if g.f.dirs["/tmp/stickypane-setup-1"] {
		t.Error("the folder was left behind")
	}
}

func TestConnectRegistersTheLinkedCommandWhenThereIsOne(t *testing.T) {
	g := newRig(home + "/.local/bin")
	g.f.links[userLnk] = bundled
	if _, err := g.s.Connect(ctx, false, true); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(g.r.ran[0].Args, " "); got != "setup --scope user --mcp --command "+userLnk+" --yes" {
		t.Errorf("args %q", got)
	}
}

func TestConnectRefusesAFolderThatIsHome(t *testing.T) {
	g := newRig("/usr/bin")
	g.f.temps = nil
	g.s.cfg.Files = homeTemp{g.f}
	if _, err := g.s.Connect(ctx, true, false); !errors.Is(err, ErrUnsafeDir) || len(g.r.ran) != 0 {
		t.Errorf("%v %v", err, g.r.ran)
	}
}

// homeTemp hands out the home folder as a temporary one.
type homeTemp struct{ *fakeFiles }

func (homeTemp) TempDir() (string, error) { return home, nil }
func (homeTemp) RemoveAll(string) error   { panic("the home folder would be removed") }

func TestInitRunsInTheProjectAndRefusesHomeAndRoot(t *testing.T) {
	g := newRig("/usr/bin")
	g.r.out = "Made .sticky/"
	res, err := g.s.Init(ctx, "/work/p", "")
	if err != nil || res.Output != "Made .sticky/" {
		t.Fatalf("%v %+v", err, res)
	}
	if c := g.r.ran[0]; c.Dir != "/work/p" || c.Name != bundled || strings.Join(c.Args, " ") != "init" || len(c.Env) != 0 {
		t.Errorf("ran %+v", c)
	}
	for _, d := range []string{home, "/", filepath.Join(home, ".")} {
		if _, err := g.s.Init(ctx, d, ""); !errors.Is(err, ErrUnsafeDir) {
			t.Errorf("Init(%q) = %v", d, err)
		}
	}
	if len(g.r.ran) != 1 {
		t.Errorf("ran %+v", g.r.ran)
	}
	// the app's language reaches init through the environment, so the
	// welcome note is in it and the board's own setting stays unset
	if _, err := g.s.Init(ctx, "/work/q", "ko"); err != nil {
		t.Fatal(err)
	}
	if c := g.r.ran[1]; strings.Join(c.Args, " ") != "init" || strings.Join(c.Env, " ") != "STICKYPANE_LANG=ko" {
		t.Errorf("ran %+v", c)
	}
}

func TestTheLoginPathIsReadOnceAndFallsBackToTheApps(t *testing.T) {
	g := newRig("/a:/b")
	if p := g.s.LoginPath(ctx); strings.Join(p, ":") != "/a:/b" {
		t.Errorf("login path %q", p)
	}
	g.r.path = "/changed"
	if p := g.s.LoginPath(ctx); strings.Join(p, ":") != "/a:/b" {
		t.Errorf("read again: %q", p)
	}
	g2 := newRig("", func(c *Config) { c.Runner = silent{} })
	if p := g2.s.LoginPath(ctx); strings.Join(p, ":") != "/usr/bin:/bin" {
		t.Errorf("fallback %q", p)
	}
}

type silent struct{}

func (silent) Run(context.Context, ports.Cmd) (string, error) { return "", errors.New("no shell") }
