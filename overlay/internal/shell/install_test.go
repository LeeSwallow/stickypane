package shell

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/LeeSwallow/stickypane/overlay/internal/install/model"
	"github.com/LeeSwallow/stickypane/overlay/internal/install/osys"
	installsvc "github.com/LeeSwallow/stickypane/overlay/internal/install/service"
)

// fakeInstaller records what it was asked; it never runs anything. An
// init makes .sticky/ in the folder, as the real one does.
type fakeInstaller struct {
	mu        sync.Mutex
	inits     []string
	connects  []string
	installed []model.Target
	err       error
}

func (f *fakeInstaller) Command(context.Context) installsvc.CommandState {
	return installsvc.CommandState{Links: []installsvc.Link{{Target: model.TargetUser, Path: "/h/.local/bin/stickypane", State: "other"}}}
}
func (f *fakeInstaller) Install(_ context.Context, t model.Target, _ bool) (installsvc.CommandState, error) {
	f.mu.Lock()
	f.installed = append(f.installed, t)
	f.mu.Unlock()
	return f.Command(nil), f.err
}
func (f *fakeInstaller) Uninstall(context.Context) (installsvc.CommandState, error) {
	return f.Command(nil), f.err
}
func (f *fakeInstaller) Connect(_ context.Context, dry, mcp bool) (installsvc.Result, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	what := "run"
	if dry {
		what = "plan"
	}
	f.connects = append(f.connects, what)
	return installsvc.Result{Command: "stickypane setup", Output: what}, f.err
}
func (f *fakeInstaller) Init(_ context.Context, dir string) (installsvc.Result, error) {
	f.mu.Lock()
	f.inits = append(f.inits, dir)
	f.mu.Unlock()
	if f.err != nil {
		return installsvc.Result{}, f.err
	}
	return installsvc.Result{Output: "made"}, os.MkdirAll(filepath.Join(dir, ".sticky"), 0o755)
}

func TestInitProjectRunsTheInstallersInit(t *testing.T) {
	r := newRig(t)
	dir := t.TempDir()
	if err := r.s.InitProject(dir); err != nil {
		t.Fatal(err)
	}
	if len(r.inst.inits) != 1 || r.inst.inits[0] != dir {
		t.Errorf("inits %q", r.inst.inits)
	}
}

// A window on a folder with no board says so, and makes one when asked,
// then starts its bridge again.
func TestAWindowOnAFolderWithoutABoardMakesOne(t *testing.T) {
	dir := t.TempDir()
	r := newRig(t)
	starts := 0
	var mu sync.Mutex
	r.s.cfg.Start = func(ctx context.Context, bin, project string) (Board, error) {
		mu.Lock()
		defer mu.Unlock()
		starts++
		if _, err := os.Stat(filepath.Join(project, ".sticky")); err != nil {
			return nil, errors.New("no board here")
		}
		return r.board, nil
	}
	id := r.s.OpenProject(dir)
	waitFor(t, "no board", func() bool { return r.s.Status(id).State == StateNoBoard })
	if err := r.s.MakeBoard(id); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "ready", func() bool { return r.s.Status(id).State == StateReady })
	real, _ := filepath.EvalSymlinks(dir)
	if len(r.inst.inits) != 1 || r.inst.inits[0] != real {
		t.Errorf("inits %q", r.inst.inits)
	}
	if res, err := r.s.InitHere(id); err != nil || res.Output != "made" || len(r.inst.inits) != 2 {
		t.Errorf("InitHere: %+v %v", res, err)
	}
	if err := r.s.MakeBoard(999); err == nil {
		t.Error("no such window")
	}
}

func TestSetupServiceSaysOccupiedAndCancelled(t *testing.T) {
	r := newRig(t)
	x := &SetupService{S: r.s}
	r.inst.err = installsvc.ErrOccupied
	a, err := x.InstallCommand("user", false)
	if err != nil || a.Occupied != "/h/.local/bin/stickypane" {
		t.Errorf("occupied: %+v %v", a, err)
	}
	r.inst.err = osys.ErrCancelled
	if a, err := x.InstallCommand("system", false); err != nil || !a.Cancelled {
		t.Errorf("cancelled: %+v %v", a, err)
	}
	r.inst.err = errors.New("disk full")
	if _, err := x.UninstallCommand(); err == nil {
		t.Error("a failure is an error")
	}
	if r.inst.installed[1] != model.TargetSystem {
		t.Errorf("targets %q", r.inst.installed)
	}
	r.inst.err = nil
	if p, err := x.ConnectPlan(true); err != nil || p.Output != "plan" {
		t.Errorf("plan %+v %v", p, err)
	}
	if p, err := x.Connect(true); err != nil || p.Output != "run" {
		t.Errorf("run %+v %v", p, err)
	}
}
