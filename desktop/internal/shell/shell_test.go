package shell

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/LeeSwallow/stickypane/desktop/internal/bridgeclient"
	"github.com/LeeSwallow/stickypane/desktop/internal/index"
)

// fakeWindow is a window that is never drawn.
type fakeWindow struct {
	mu                       sync.Mutex
	id                       uint
	opts                     WindowOptions
	bounds                   Rect
	visible, focused, onTop  bool
	closed                   bool
	shows, focuses           int
	onLost, onFocus, onMoved []func()
	onClosing                []func()
}

func (w *fakeWindow) ID() uint { return w.id }
func (w *fakeWindow) Show() {
	w.mu.Lock()
	w.visible = true
	w.shows++
	w.mu.Unlock()
}
func (w *fakeWindow) Hide() {
	w.mu.Lock()
	w.visible, w.focused = false, false
	w.mu.Unlock()
}
func (w *fakeWindow) Focus() {
	w.mu.Lock()
	w.focused = true
	w.focuses++
	w.mu.Unlock()
}
func (w *fakeWindow) Close() {
	w.mu.Lock()
	fs := w.onClosing
	w.closed, w.visible = true, false
	w.mu.Unlock()
	for _, f := range fs {
		f()
	}
}
func (w *fakeWindow) Visible() bool { w.mu.Lock(); defer w.mu.Unlock(); return w.visible }
func (w *fakeWindow) Focused() bool { w.mu.Lock(); defer w.mu.Unlock(); return w.focused }
func (w *fakeWindow) SetPinned(on bool) {
	w.mu.Lock()
	w.onTop = on
	w.mu.Unlock()
}
func (w *fakeWindow) Bounds() Rect { w.mu.Lock(); defer w.mu.Unlock(); return w.bounds }
func (w *fakeWindow) OnLostFocus(f func()) {
	w.mu.Lock()
	w.onLost = append(w.onLost, f)
	w.mu.Unlock()
}
func (w *fakeWindow) OnFocus(f func()) { w.mu.Lock(); w.onFocus = append(w.onFocus, f); w.mu.Unlock() }
func (w *fakeWindow) OnMoved(f func()) { w.mu.Lock(); w.onMoved = append(w.onMoved, f); w.mu.Unlock() }
func (w *fakeWindow) OnClosing(f func()) {
	w.mu.Lock()
	w.onClosing = append(w.onClosing, f)
	w.mu.Unlock()
}
func (w *fakeWindow) pinned() bool { w.mu.Lock(); defer w.mu.Unlock(); return w.onTop }
func (w *fakeWindow) fire(fs func(*fakeWindow) []func()) {
	w.mu.Lock()
	list := append([]func(){}, fs(w)...)
	w.mu.Unlock()
	for _, f := range list {
		f()
	}
}
func (w *fakeWindow) loseFocus() { w.fire(func(w *fakeWindow) []func() { return w.onLost }) }
func (w *fakeWindow) moveTo(r Rect) {
	w.mu.Lock()
	w.bounds = r
	w.mu.Unlock()
	w.fire(func(w *fakeWindow) []func() { return w.onMoved })
}

type event struct {
	name string
	data any
}

// fakePlatform records what the shell asked of the desktop.
type fakePlatform struct {
	mu        sync.Mutex
	next      uint
	windows   []*fakeWindow
	hotkeys   map[string]func()
	hotkeyErr error
	refuse    map[string]error // accelerators the desktop will not give
	events    []event
	tray      Tray
	quit      bool
}

func (p *fakePlatform) NewWindow(o WindowOptions) Window {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.next++
	b := o.Bounds
	w := &fakeWindow{id: p.next, opts: o, bounds: b, visible: !o.Hidden, onTop: o.Pinned}
	p.windows = append(p.windows, w)
	return w
}
func (p *fakePlatform) WorkArea() Rect { return Rect{X: 0, Y: 25, Width: 1600, Height: 1000} }
func (p *fakePlatform) RegisterHotkey(accel string, fn func()) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.hotkeyErr != nil {
		return p.hotkeyErr
	}
	if err := p.refuse[accel]; err != nil {
		return err
	}
	if p.hotkeys == nil {
		p.hotkeys = map[string]func(){}
	}
	p.hotkeys[accel] = fn
	return nil
}
func (p *fakePlatform) UnregisterHotkey(accel string) {
	p.mu.Lock()
	delete(p.hotkeys, accel)
	p.mu.Unlock()
}
func (p *fakePlatform) Emit(name string, data any) {
	p.mu.Lock()
	p.events = append(p.events, event{name, data})
	p.mu.Unlock()
}
func (p *fakePlatform) SetTray(t Tray) {
	p.mu.Lock()
	p.tray = t
	p.mu.Unlock()
}
func (p *fakePlatform) PickFolder() (string, error) { return "", nil }
func (p *fakePlatform) Quit()                       { p.mu.Lock(); p.quit = true; p.mu.Unlock() }

func (p *fakePlatform) projectWindows() []*fakeWindow {
	p.mu.Lock()
	defer p.mu.Unlock()
	var out []*fakeWindow
	for _, w := range p.windows {
		if w.opts.Kind == KindProject {
			out = append(out, w)
		}
	}
	return out
}
func (p *fakePlatform) homeWindows() []*fakeWindow {
	p.mu.Lock()
	defer p.mu.Unlock()
	var out []*fakeWindow
	for _, w := range p.windows {
		if w.opts.Kind == KindHome {
			out = append(out, w)
		}
	}
	return out
}
func (p *fakePlatform) emitted(name string) []any {
	p.mu.Lock()
	defer p.mu.Unlock()
	var out []any
	for _, e := range p.events {
		if e.name == name {
			out = append(out, e.data)
		}
	}
	return out
}

// fakeBoard is a bridge that answers from a table.
type fakeBoard struct {
	notes  chan bridgeclient.Notification
	mu     sync.Mutex
	calls  []string
	closed bool
}

func newFakeBoard() *fakeBoard { return &fakeBoard{notes: make(chan bridgeclient.Notification, 8)} }

func (b *fakeBoard) Call(ctx context.Context, method string, params, result any) error {
	b.mu.Lock()
	b.calls = append(b.calls, method)
	b.mu.Unlock()
	if r, ok := result.(*json.RawMessage); ok {
		*r = json.RawMessage(`{"method":"` + method + `"}`)
		if method == "board.load" {
			*r = json.RawMessage(`{"method":"board.load","title":"Alpha"}`)
		}
	}
	return nil
}
func (b *fakeBoard) Notifications() <-chan bridgeclient.Notification { return b.notes }
func (b *fakeBoard) Hello() bridgeclient.Hello {
	return bridgeclient.Hello{Protocol: 1, Version: "9.9.9"}
}
func (b *fakeBoard) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.closed {
		b.closed = true
		close(b.notes)
	}
	return nil
}

// fakeIndex is an index in memory.
type fakeIndex struct {
	mu      sync.Mutex
	groups  index.Groups
	restore []int64
	titles  map[string]string
}

func (ix *fakeIndex) Retitle(path, title string) error {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	if ix.titles == nil {
		ix.titles = map[string]string{}
	}
	ix.titles[path] = title
	return nil
}

func (ix *fakeIndex) Ingest() error                            { return nil }
func (ix *fakeIndex) Prune(time.Time) ([]index.Project, error) { return nil, nil }
func (ix *fakeIndex) Restore(id int64) error                   { ix.restore = append(ix.restore, id); return nil }
func (ix *fakeIndex) Sessions(time.Time) (index.Groups, error) { return ix.groups, nil }
func (ix *fakeIndex) Close() error                             { return nil }

type rig struct {
	t     *testing.T
	home  string
	p     *fakePlatform
	s     *Shell
	board *fakeBoard
	ix    *fakeIndex
	inst  *fakeInstaller
}

func newRig(t *testing.T, mod ...func(*Config)) *rig {
	t.Helper()
	r := &rig{t: t, home: t.TempDir(), p: &fakePlatform{}, board: newFakeBoard(), ix: &fakeIndex{}, inst: &fakeInstaller{}}
	cfg := Config{
		Home:     r.home,
		Platform: r.p,
		Find:     func() (string, error) { return "/bin/stickypane", nil },
		Start: func(ctx context.Context, bin, project string) (Board, error) {
			return r.board, nil
		},
		OpenIndex:     func(string) (Index, error) { return r.ix, nil },
		IndexDeadline: time.Second,
		Install:       r.inst,
	}
	for _, m := range mod {
		m(&cfg)
	}
	s, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	r.s = s
	t.Cleanup(s.Shutdown)
	return r
}

// waitFor polls cond until it holds or a second passes.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestOpenProjectReusesItsWindow(t *testing.T) {
	r := newRig(t)
	a := filepath.Join(r.home, "a")
	id1 := r.s.OpenProject(a)
	id2 := r.s.OpenProject(a + string(filepath.Separator)) // the same folder, written otherwise
	if id1 != id2 {
		t.Fatalf("second open made window %d, want %d", id2, id1)
	}
	ws := r.p.projectWindows()
	if len(ws) != 1 {
		t.Fatalf("%d project windows, want 1", len(ws))
	}
	w := ws[0]
	if !w.opts.Frameless {
		t.Error("a project window is frameless")
	}
	if w.bounds.Width != 840 || w.bounds.Height != 700 {
		t.Errorf("first size %dx%d, want 840x700 (70%% of a 1000 high screen)", w.bounds.Width, w.bounds.Height)
	}
	if w.focuses < 1 || w.shows < 1 {
		t.Error("opening again shows and focuses the window")
	}
	if id3 := r.s.OpenProject(filepath.Join(r.home, "b")); id3 == id1 {
		t.Error("another folder gets its own window")
	}
	if n := len(r.p.projectWindows()); n != 2 {
		t.Errorf("%d project windows, want 2", n)
	}
}

// /tmp and /private/tmp are one folder on macOS: one window, one path.
func TestOpenProjectSeesThroughSymlinks(t *testing.T) {
	r := newRig(t)
	real := filepath.Join(t.TempDir(), "real")
	if err := os.Mkdir(real, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skip("no symlinks here:", err)
	}
	id := r.s.OpenProject(link)
	if again := r.s.OpenProject(real); again != id {
		t.Errorf("the folder through its link and itself made windows %d and %d", id, again)
	}
	want, _ := filepath.EvalSymlinks(real)
	if got := r.s.Status(id).Path; got != want {
		t.Errorf("path %q, want the resolved %q", got, want)
	}
}

func TestOpenProjectAfterCloseMakesANewWindow(t *testing.T) {
	r := newRig(t)
	a := filepath.Join(r.home, "a")
	r.s.OpenProject(a)
	r.p.projectWindows()[0].Close()
	r.s.OpenProject(a)
	if n := len(r.p.projectWindows()); n != 2 {
		t.Errorf("%d windows made, want 2: a closed window is not reused", n)
	}
}

func TestWindowStateRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "overlay.json")
	want := State{
		Hotkey:   "Ctrl+Shift+K",
		Language: "ko",
		Mode:     "dark",
		Last:     "/a",
		Windows:  map[string]WindowState{"/a": {X: 10, Y: 20, Width: 900, Height: 600, Pinned: true}},
	}
	if err := want.Save(path); err != nil {
		t.Fatal(err)
	}
	got, err := LoadState(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Hotkey != want.Hotkey || got.Language != want.Language || got.Mode != want.Mode || got.Last != want.Last || got.Windows["/a"] != want.Windows["/a"] {
		t.Fatalf("got %+v, want %+v", got, want)
	}

	missing, err := LoadState(filepath.Join(t.TempDir(), "none.json"))
	if err != nil {
		t.Fatal(err)
	}
	if missing.Hotkey != DefaultHotkey {
		t.Errorf("no file: hotkey %q, want %q", missing.Hotkey, DefaultHotkey)
	}

	// Through the shell: a moved and pinned window opens where it was left.
	r := newRig(t)
	a := filepath.Join(r.home, "a")
	id := r.s.OpenProject(a)
	w := r.p.projectWindows()[0]
	w.moveTo(Rect{X: 40, Y: 50, Width: 800, Height: 500})
	if err := r.s.SetPin(id, true); err != nil {
		t.Fatal(err)
	}
	w.Close()
	r.s.Shutdown()

	p2 := &fakePlatform{}
	s2, err := New(Config{Home: r.home, Platform: p2, Find: func() (string, error) { return "", bridgeclient.ErrNotFound },
		OpenIndex: func(string) (Index, error) { return &fakeIndex{}, nil }})
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Shutdown()
	s2.OpenProject(a)
	w2 := p2.projectWindows()[0]
	if w2.bounds != (Rect{X: 40, Y: 50, Width: 800, Height: 500}) {
		t.Errorf("reopened at %+v", w2.bounds)
	}
	if !w2.pinned() {
		t.Error("a pinned window opens pinned")
	}
	if _, err := os.Stat(filepath.Join(r.home, "overlay.json")); err != nil {
		t.Errorf("overlay.json: %v", err)
	}
}

// The desktop reports a window a point smaller than it was asked for;
// taken as a move, the window would shrink a little at every launch.
func TestAPointOfRoundingIsNotAMove(t *testing.T) {
	r := newRig(t)
	id := r.s.OpenProject(filepath.Join(r.home, "a"))
	w := r.p.projectWindows()[0]
	w.moveTo(Rect{X: 315, Y: 172, Width: 839, Height: 645})
	w.moveTo(Rect{X: 315, Y: 172, Width: 838, Height: 644})
	r.s.SetPin(id, false) // saves
	st, err := LoadState(filepath.Join(r.home, StateFile))
	if err != nil {
		t.Fatal(err)
	}
	if got := st.Windows[cleanPath(filepath.Join(r.home, "a"))]; got.Width != 839 || got.Height != 645 {
		t.Errorf("saved %+v, want 839x645 kept", got)
	}
}

func TestWindowShowsMissingStickypane(t *testing.T) {
	r := newRig(t, func(c *Config) { c.Find = func() (string, error) { return "", bridgeclient.ErrNotFound } })
	id := r.s.OpenProject(filepath.Join(r.home, "a"))
	waitFor(t, "status missing", func() bool { return r.s.Status(id).State == StateMissing })
	if got := r.p.emitted(StatusEvent(id)); len(got) == 0 {
		t.Error("the window is told its status changed")
	}
	st := r.s.StickypaneStatus()
	if st.Found {
		t.Error("StickypaneStatus says found")
	}
}

func TestWindowSaysStickypaneIsTooOld(t *testing.T) {
	old := []bridgeclient.Candidate{{Path: "/opt/homebrew/bin/stickypane", Version: "0.1.1"}}
	started := false
	r := newRig(t, func(c *Config) {
		c.Find = func() (string, error) { return "", &bridgeclient.TooOldError{Found: old} }
		c.Start = func(context.Context, string, string) (Board, error) { started = true; return nil, nil }
	})
	id := r.s.OpenProject(filepath.Join(r.home, "a"))
	waitFor(t, "status too old", func() bool { return r.s.Status(id).State == StateTooOld })
	if e := r.s.Status(id).Error; !strings.Contains(e, "/opt/homebrew/bin/stickypane") || !strings.Contains(e, "Update stickypane") {
		t.Errorf("the window says %q", e)
	}
	if started {
		t.Error("an old stickypane was started")
	}
	st := r.s.StickypaneStatus()
	if !st.Found || st.Bridge || st.Path != old[0].Path || st.Version != "0.1.1" || len(st.Candidates) != 1 || st.Error == "" {
		t.Errorf("StickypaneStatus = %+v", st)
	}
}

func TestStickypaneStatusNamesTheOneWithTheBridge(t *testing.T) {
	r := newRig(t, func(c *Config) {
		c.Find = func() (string, error) { return "/home/go/bin/stickypane", nil }
		c.Probe = func(p string) bridgeclient.Candidate {
			return bridgeclient.Candidate{Path: p, Version: "0.2.0", Bridge: true}
		}
	})
	st := r.s.StickypaneStatus()
	if !st.Found || !st.Bridge || st.Path != "/home/go/bin/stickypane" || st.Version != "0.2.0" || st.Error != "" {
		t.Errorf("StickypaneStatus = %+v", st)
	}
}

func TestBridgeStartFailureIsShownInTheWindow(t *testing.T) {
	r := newRig(t, func(c *Config) {
		c.Start = func(context.Context, string, string) (Board, error) { return nil, bridgeclient.ErrProtocol }
	})
	id := r.s.OpenProject(filepath.Join(r.home, "a"))
	waitFor(t, "status failed", func() bool { return r.s.Status(id).State == StateFailed })
	if r.s.Status(id).Error == "" {
		t.Error("a failure says why")
	}
}

func TestBoardChangedReachesTheWindow(t *testing.T) {
	r := newRig(t)
	id := r.s.OpenProject(filepath.Join(r.home, "a"))
	waitFor(t, "ready", func() bool { return r.s.Status(id).State == StateReady })
	r.board.notes <- bridgeclient.Notification{Method: "board.changed", Params: json.RawMessage(`{"notes":["x"]}`)}
	r.board.notes <- bridgeclient.Notification{Method: bridgeclient.MethodRestarted}
	waitFor(t, "board.changed event", func() bool { return len(r.p.emitted(ChangedEvent(id))) == 1 })
	waitFor(t, "bridge.restarted event", func() bool { return len(r.p.emitted(RestartedEvent(id))) == 1 })
	if got := r.p.emitted(ChangedEvent(id))[0]; string(got.(json.RawMessage)) != `{"notes":["x"]}` {
		t.Errorf("board.changed carries %v", got)
	}

	var res json.RawMessage
	if err := r.s.Call(context.Background(), id, "note.arrange", map[string]any{"note": "x"}, &res); err != nil {
		t.Fatal(err)
	}
	if string(res) != `{"method":"note.arrange"}` {
		t.Errorf("call answered %s", res)
	}
	r.p.projectWindows()[0].Close()
	waitFor(t, "bridge closed", func() bool { r.board.mu.Lock(); defer r.board.mu.Unlock(); return r.board.closed })
	if err := r.s.Call(context.Background(), id, "board.load", nil, &res); err == nil {
		t.Error("a call from a closed window fails")
	}
}

func TestBoardLoadNamesTheProjectInTheIndex(t *testing.T) {
	r := newRig(t)
	r.s.Start()
	a := filepath.Join(r.home, "a")
	id := r.s.OpenProject(a)
	waitFor(t, "ready", func() bool { return r.s.Status(id).State == StateReady })
	if _, err := r.s.Sessions(); err != nil { // the index is open
		t.Fatal(err)
	}
	var res json.RawMessage
	if err := r.s.Call(context.Background(), id, "board.load", nil, &res); err != nil {
		t.Fatal(err)
	}
	r.ix.mu.Lock()
	defer r.ix.mu.Unlock()
	if len(r.ix.titles) != 1 || r.ix.titles[r.s.Status(id).Path] != "Alpha" {
		t.Fatalf("titles = %v", r.ix.titles)
	}
}

func TestSecondLaunchOpensTheProject(t *testing.T) {
	r := newRig(t)
	r.s.Launch([]string{"/Applications/Stickypane.app/Contents/MacOS/stickypane-app", "--project=rel/b"}, "/work")
	ws := r.p.projectWindows()
	if len(ws) != 1 {
		t.Fatalf("%d project windows, want 1", len(ws))
	}
	if got := r.s.Status(ws[0].id).Path; got != filepath.Clean("/work/rel/b") {
		t.Errorf("opened %q", got)
	}
	r.s.Launch([]string{"x", "--project", "/abs/c"}, "/work")
	if got := r.s.Status(r.p.projectWindows()[1].id).Path; got != "/abs/c" {
		t.Errorf("opened %q", got)
	}
	// Without --project the home window comes forward.
	r.s.Launch([]string{"x"}, "/work")
	hs := r.p.homeWindows()
	if len(hs) != 1 || !hs[0].Visible() || hs[0].focuses == 0 {
		t.Error("a launch without a project shows the home window")
	}
	if len(r.p.projectWindows()) != 2 {
		t.Error("no more project windows")
	}
}

func TestHomeWindowIsOneAndSized(t *testing.T) {
	r := newRig(t)
	r.s.ShowHome()
	r.s.ShowHome()
	hs := r.p.homeWindows()
	if len(hs) != 1 {
		t.Fatalf("%d home windows", len(hs))
	}
	if hs[0].bounds.Width != 860 || hs[0].bounds.Height != 620 {
		t.Errorf("home %dx%d, want 860x620", hs[0].bounds.Width, hs[0].bounds.Height)
	}
	// Changed in A5c: the home draws its own title bar like a project window;
	// on macOS the platform keeps the native traffic lights (hidden-inset).
	if !hs[0].opts.Frameless {
		t.Error("the home window draws its own title bar")
	}
}

func TestHotkeyTogglesTheLastProjectWindow(t *testing.T) {
	r := newRig(t)
	r.s.Start()
	key := r.p.hotkeys[DefaultHotkey]
	if key == nil {
		t.Fatalf("hotkey %q not registered: %v", DefaultHotkey, r.p.hotkeys)
	}
	r.s.OpenProject(filepath.Join(r.home, "a"))
	r.s.OpenProject(filepath.Join(r.home, "b"))
	b := r.p.projectWindows()[1]
	key()
	if b.Visible() {
		t.Fatal("the hotkey hides the focused last window")
	}
	key()
	if !b.Visible() || !b.Focused() {
		t.Fatal("the hotkey brings it back, focused")
	}
	a := r.p.projectWindows()[0]
	if a.shows != 1 {
		t.Errorf("the other window was touched: %d shows", a.shows)
	}
}

func TestHotkeyFailureIsANoticeInHome(t *testing.T) {
	r := newRig(t, func(c *Config) {})
	r.p.hotkeyErr = errors.New("taken by another app")
	r.s.Start()
	ns := r.s.Notices()
	if len(ns) != 1 || ns[0].Code != NoticeHotkey || ns[0].Detail == "" {
		t.Fatalf("notices %+v", ns)
	}
	if len(r.p.emitted(NoticesEvent)) == 0 {
		t.Error("the home window is told")
	}
}

func TestSavingTheSameHotkeyRetriesAFailedOne(t *testing.T) {
	r := newRig(t)
	r.p.hotkeyErr = errors.New("taken by another app")
	r.s.Start()
	r.p.mu.Lock()
	r.p.hotkeyErr = nil // the other app let go
	r.p.mu.Unlock()
	if err := r.s.SetPrefs(Prefs{Hotkey: DefaultHotkey}); err != nil {
		t.Fatal(err)
	}
	if r.p.hotkeys[DefaultHotkey] == nil {
		t.Error("the unchanged hotkey was not registered again")
	}
	if len(r.s.Notices()) != 0 {
		t.Errorf("notices %+v", r.s.Notices())
	}
}

func TestAHotkeyThatCannotBeHadKeepsTheOld(t *testing.T) {
	r := newRig(t)
	r.s.Start()
	r.p.refuse = map[string]error{"Ctrl+K": errors.New("taken")}
	if err := r.s.SetHotkey("Ctrl+K"); err == nil {
		t.Fatal("a refused hotkey is an error")
	}
	if r.p.hotkeys[DefaultHotkey] == nil || r.s.Prefs().Hotkey != DefaultHotkey {
		t.Errorf("the old hotkey is lost: %v, prefs %+v", r.p.hotkeys, r.s.Prefs())
	}

	// If the old one cannot be had back either, the home window says so.
	r.p.mu.Lock()
	r.p.refuse[DefaultHotkey] = errors.New("taken meanwhile")
	r.p.mu.Unlock()
	if err := r.s.SetHotkey("Ctrl+K"); err == nil {
		t.Fatal("a refused hotkey is an error")
	}
	if ns := r.s.Notices(); len(ns) != 1 || ns[0].Code != NoticeHotkey {
		t.Errorf("notices %+v", ns)
	}
}

func TestConcurrentSavesLeaveTheLastState(t *testing.T) {
	r := newRig(t)
	id := r.s.OpenProject(filepath.Join(r.home, "a"))
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(on bool) {
			defer wg.Done()
			r.s.SetPin(id, on)
		}(i%2 == 0)
	}
	wg.Wait()
	if err := r.s.SetPin(id, true); err != nil {
		t.Fatal(err)
	}
	st, err := LoadState(filepath.Join(r.home, StateFile))
	if err != nil {
		t.Fatal(err)
	}
	if !st.Windows[r.s.Status(id).Path].Pinned {
		t.Error("the file does not hold the last pin")
	}
	left, _ := filepath.Glob(filepath.Join(r.home, ".overlay-*"))
	if len(left) != 0 {
		t.Errorf("temporary files left: %v", left)
	}
}

func TestPinKeepsTheWindowOnFocusLoss(t *testing.T) {
	r := newRig(t)
	id := r.s.OpenProject(filepath.Join(r.home, "a"))
	w := r.p.projectWindows()[0]
	w.loseFocus()
	if w.Visible() {
		t.Fatal("an unpinned window hides when it loses focus")
	}
	w.Show()
	if err := r.s.SetPin(id, true); err != nil {
		t.Fatal(err)
	}
	if !w.pinned() {
		t.Error("pin keeps the window on top")
	}
	w.loseFocus()
	if !w.Visible() {
		t.Error("a pinned window stays when it loses focus")
	}
	if !r.s.Status(id).Pinned {
		t.Error("status says pinned")
	}
}

func TestTrayListsActiveAndRecent(t *testing.T) {
	r := newRig(t)
	row := func(id int64, p string) index.Row {
		return index.Row{Project: index.Project{ID: id, Path: p, Title: filepath.Base(p)}}
	}
	r.ix.groups = index.Groups{
		Active: []index.Row{row(1, "/p1"), row(2, "/p2")},
		Days: []index.Day{
			{Date: "2026-10-28", Rows: []index.Row{row(3, "/p3"), row(4, "/p4")}},
			{Date: "2026-10-27", Rows: []index.Row{row(5, "/p5"), row(6, "/p6")}},
		},
	}
	r.s.Start()
	waitFor(t, "tray", func() bool { r.p.mu.Lock(); defer r.p.mu.Unlock(); return len(r.p.tray.Recent) > 0 })
	r.p.mu.Lock()
	tr := r.p.tray
	r.p.mu.Unlock()
	if tr.Active != 2 {
		t.Errorf("active %d, want 2", tr.Active)
	}
	if len(tr.Recent) != 5 || tr.Recent[0].Path != "/p1" || tr.Recent[4].Path != "/p5" {
		t.Errorf("recent %+v", tr.Recent)
	}
}

func TestIndexOpenIsBoundedByADeadline(t *testing.T) {
	block := make(chan struct{})
	defer close(block)
	r := newRig(t, func(c *Config) {
		c.OpenIndex = func(string) (Index, error) { <-block; return &fakeIndex{}, nil }
		c.IndexDeadline = 50 * time.Millisecond
	})
	r.s.Start()
	start := time.Now()
	_, err := r.s.Sessions()
	if err == nil {
		t.Fatal("Sessions succeeds without an index")
	}
	if time.Since(start) > time.Second {
		t.Errorf("Sessions waited %v", time.Since(start))
	}
}

func TestEnsureHonoursStickypaneHome(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sp")
	t.Setenv("STICKYPANE_HOME", dir)
	if got := DefaultHome(); got != dir {
		t.Errorf("DefaultHome %q, want %q", got, dir)
	}
	r := newRig(t, func(c *Config) { c.Home = dir })
	got, err := r.s.Ensure()
	if err != nil || got != dir {
		t.Fatalf("Ensure %q, %v", got, err)
	}
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		t.Errorf("home not made: %v", err)
	}
}

func TestMotionIsKeptAndOnlyItsWords(t *testing.T) {
	r := newRig(t)
	r.s.Start()
	for _, c := range []struct{ in, want string }{{"off", "off"}, {"reduce", "reduce"}, {"wild", ""}, {"", ""}} {
		if err := r.s.SetPrefs(Prefs{Hotkey: DefaultHotkey, Motion: c.in}); err != nil {
			t.Fatal(err)
		}
		if got := r.s.Prefs().Motion; got != c.want {
			t.Errorf("motion %q kept as %q, want %q", c.in, got, c.want)
		}
	}
	if err := r.s.SetPrefs(Prefs{Hotkey: DefaultHotkey, Motion: "off"}); err != nil {
		t.Fatal(err)
	}
	st, err := LoadState(filepath.Join(r.home, StateFile))
	if err != nil || st.Motion != "off" {
		t.Errorf("overlay.json motion %q, %v", st.Motion, err)
	}
}
