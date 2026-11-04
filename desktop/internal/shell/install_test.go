package shell

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/LeeSwallow/stickypane/desktop/internal/install/model"
	"github.com/LeeSwallow/stickypane/desktop/internal/install/osys"
	installsvc "github.com/LeeSwallow/stickypane/desktop/internal/install/service"
)

// fakeInstaller records what it was asked; it never runs anything. An
// init makes .sticky/ in the folder, as the real one does.
type fakeInstaller struct {
	mu        sync.Mutex
	inits     []string
	langs     []string
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
func (f *fakeInstaller) Agents(_ context.Context, project string) (model.AgentsStatus, error) {
	return model.AgentsStatus{Plugin: "0.2.1", Project: project}, f.err
}
func (f *fakeInstaller) SetupAgent(_ context.Context, project string, r model.AgentRequest, dry bool) (installsvc.Result, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	what := "run"
	if dry {
		what = "plan"
	}
	f.connects = append(f.connects, what+" "+r.Agent+" "+r.Part+" "+r.Scope+" "+project)
	return installsvc.Result{Command: "stickypane setup", Output: what}, f.err
}
func (f *fakeInstaller) Init(_ context.Context, dir, lang string) (installsvc.Result, error) {
	f.mu.Lock()
	f.inits = append(f.inits, dir)
	f.langs = append(f.langs, lang)
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
	if r.inst.langs[0] != "" {
		t.Errorf("an app with no language of its own leaves it to the system, got %q", r.inst.langs[0])
	}
}

// A board made from the app gets its welcome note in the app's language.
func TestInitProjectPassesTheAppsLanguage(t *testing.T) {
	r := newRig(t)
	if err := r.s.SetPrefs(Prefs{Hotkey: DefaultHotkey, Language: "ko"}); err != nil {
		t.Fatal(err)
	}
	if err := r.s.InitProject(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if len(r.inst.langs) != 1 || r.inst.langs[0] != "ko" {
		t.Errorf("init was given %q, want the app's ko", r.inst.langs)
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
	req := model.AgentRequest{Agent: "codex", Part: "mcp", Scope: "user"}
	if p, err := x.AgentPlan(context.Background(), req); err != nil || p.Output != "plan" {
		t.Errorf("plan %+v %v", p, err)
	}
	if p, err := x.AgentRun(context.Background(), req); err != nil || p.Output != "run" {
		t.Errorf("run %+v %v", p, err)
	}
	if got := r.inst.connects; len(got) != 2 || got[0] != "plan codex mcp user " || got[1] != "run codex mcp user " {
		t.Errorf("a call from no window has no project: %q", got)
	}
}

// The project's scopes are those of the board the settings were opened
// from, or of the board's own window; none from the settings of the home.
func TestTheAgentsProjectIsTheBoardsTheSettingsCameFrom(t *testing.T) {
	r := newRig(t)
	a, pathA := readyProject(t, r, "a")
	if got := r.s.AgentsProject(a); got != pathA {
		t.Errorf("a board's window: %q, want %q", got, pathA)
	}
	sid := r.s.OpenSettings(a, "cli")
	if got := r.s.AgentsProject(sid); got != pathA {
		t.Errorf("the settings over a board: %q, want %q", got, pathA)
	}
	if again := r.s.OpenSettings(0, "cli"); again != sid || r.s.AgentsProject(sid) != "" {
		t.Errorf("the settings from the home: %q", r.s.AgentsProject(sid))
	}
}
