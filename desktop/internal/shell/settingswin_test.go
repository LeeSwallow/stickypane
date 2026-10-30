package shell

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/desktop/internal/canvas"
	"github.com/LeeSwallow/stickypane/desktop/internal/hotkey"
)

func (p *fakePlatform) settingsWindows() []*fakeWindow {
	p.mu.Lock()
	defer p.mu.Unlock()
	var out []*fakeWindow
	for _, w := range p.windows {
		if w.opts.Kind == KindSettings && !w.closed {
			out = append(out, w)
		}
	}
	return out
}

// readyProject opens a project and waits for its bridge.
func readyProject(t *testing.T, r *rig, name string) (uint, string) {
	t.Helper()
	dir := filepath.Join(r.home, name)
	id := r.s.OpenProject(dir)
	waitFor(t, "the bridge", func() bool { return r.s.Status(id).State == StateReady })
	return id, r.s.Status(id).Path
}

func TestSettingsIsOneWindowForTheBoardItWasOpenedFrom(t *testing.T) {
	r := newRig(t)
	a, pathA := readyProject(t, r, "a")
	sid := r.s.OpenSettings(a, "app")
	ws := r.p.settingsWindows()
	if len(ws) != 1 {
		t.Fatalf("%d settings windows, want 1", len(ws))
	}
	if u := ws[0].opts.URL; !strings.Contains(u, "window=settings") || !strings.Contains(u, "section=app") {
		t.Errorf("settings URL %q", u)
	}
	if ws[0].opts.Bounds.Width != SettingsWidth || !ws[0].opts.Frameless {
		t.Errorf("settings window %+v", ws[0].opts)
	}
	// its calls to a board go to the board it was opened from
	if st := r.s.Status(sid); st.Path != pathA || st.State != StateReady {
		t.Errorf("settings window status %+v, want board %s", st, pathA)
	}
	var out json.RawMessage
	if err := r.s.Call(context.Background(), sid, "settings.get", nil, &out); err != nil {
		t.Errorf("a call from the settings window: %v", err)
	}

	// opened again from another board: the same window, the new board
	b, pathB := readyProject(t, r, "b")
	if again := r.s.OpenSettings(b, "nonsense"); again != sid {
		t.Errorf("a second settings window %d, want %d", again, sid)
	}
	if st := r.s.Status(sid); st.Path != pathB {
		t.Errorf("settings now on %q, want %q", st.Path, pathB)
	}
	if len(r.p.emitted(SettingsEvent)) == 0 {
		t.Error("the settings window was not told its board changed")
	}

	// the board closes: the settings window has no board
	r.s.mu.Lock()
	wb := r.s.byWin[b].win
	r.s.mu.Unlock()
	wb.Close()
	if st := r.s.Status(sid); st.State != StateClosed {
		t.Errorf("settings status after its board closed: %+v", st)
	}
	if err := r.s.Call(context.Background(), sid, "settings.get", nil, &out); err == nil {
		t.Error("a call with no board did not fail")
	}

	// closed, it is made again
	ws[0].Close()
	if again := r.s.OpenSettings(0, "about"); again == sid {
		t.Error("a closed settings window came back")
	}
}

func TestSettingsFromHomeHasNoBoard(t *testing.T) {
	r := newRig(t)
	sid := r.s.OpenSettings(0, "board")
	if st := r.s.Status(sid); st.State != StateClosed {
		t.Errorf("status %+v, want closed", st)
	}
	if r.s.boardWindow(sid) != 0 {
		t.Error("the settings window has a board window")
	}
}

func TestOpenAtLaunchOpensSettingsForTheLastBoard(t *testing.T) {
	r := newRig(t)
	r.s.OpenAtLaunch("nothing")
	if len(r.p.settingsWindows()) != 0 {
		t.Fatal("a settings window for an unknown name")
	}
	_, path := readyProject(t, r, "a")
	r.s.OpenAtLaunch("settings:cli")
	ws := r.p.settingsWindows()
	if len(ws) != 1 || !strings.Contains(ws[0].opts.URL, "section=cli") {
		t.Fatalf("settings windows %v", ws)
	}
	if st := r.s.Status(ws[0].id); st.Path != path {
		t.Errorf("settings on %q, want %q", st.Path, path)
	}
}

func TestSetSettingTellsTheBoardsWindows(t *testing.T) {
	r := newRig(t)
	a, _ := readyProject(t, r, "a")
	sid := r.s.OpenSettings(a, "board")
	before := len(r.p.emitted(StatusEvent(a)))
	r.s.Touched(sid)
	if len(r.p.emitted(StatusEvent(a))) == before {
		t.Error("the board window was not told to read its board again")
	}
}

func TestResetPlacesForgetsThePlacesOnly(t *testing.T) {
	r := newRig(t)
	a, path := readyProject(t, r, "a")
	if err := r.s.SetMode(a, canvas.ModeGrid); err != nil {
		t.Fatal(err)
	}
	if err := r.s.SetPlaces(a, map[string]canvas.Place{"x.md": {X: 40, Y: 40, W: 300, H: 200, Z: 1}}); err != nil {
		t.Fatal(err)
	}
	if err := r.s.ResetPlaces(a); err != nil {
		t.Fatal(err)
	}
	st, _ := LoadState(filepath.Join(r.home, StateFile))
	got := st.Projects[path]
	if len(got.Places) != 0 || got.Mode != canvas.ModeGrid {
		t.Errorf("after reset %+v, want no places and the grid kept", got)
	}
	if len(r.p.emitted(CanvasEvent(a))) == 0 {
		t.Error("the board window was not told its layout")
	}
	// from the settings window it is the board's
	sid := r.s.OpenSettings(a, "board")
	if r.s.boardWindow(sid) != a {
		t.Errorf("boardWindow(settings) = %d, want %d", r.s.boardWindow(sid), a)
	}
}

func TestCheckHotkey(t *testing.T) {
	r := newRig(t)
	r.s.Start()
	r.p.mu.Lock()
	r.p.refuse = map[string]error{"Ctrl+Option+K": errors.New("held by another app")}
	r.p.mu.Unlock()

	if c := r.s.CheckHotkey("cmd+space"); c.OK || c.Code != hotkey.CodeReserved {
		t.Errorf("Cmd+Space: %+v", c)
	}
	if c := r.s.CheckHotkey("K"); c.OK || c.Code != hotkey.CodeModifier {
		t.Errorf("K: %+v", c)
	}
	if c := r.s.CheckHotkey("ctrl+alt+k"); c.OK || c.Code != CodeTaken || c.Accel != "Ctrl+Option+K" {
		t.Errorf("a key held elsewhere: %+v", c)
	}
	if c := r.s.CheckHotkey("option+space"); !c.OK || c.Accel != "Option+Space" {
		t.Errorf("the hotkey in use: %+v", c)
	}
	c := r.s.CheckHotkey("Ctrl+Shift+J")
	if !c.OK || c.Accel != "Ctrl+Shift+J" {
		t.Errorf("a free hotkey: %+v", c)
	}
	r.p.mu.Lock()
	_, left := r.p.hotkeys["Ctrl+Shift+J"]
	_, kept := r.p.hotkeys[DefaultHotkey]
	r.p.mu.Unlock()
	if left || !kept {
		t.Errorf("trying a hotkey changed what is registered (left %v, kept %v)", left, kept)
	}
}

func TestSetPrefsTellsEveryWindowAndPinsNewWindows(t *testing.T) {
	r := newRig(t)
	if err := r.s.SetPrefs(Prefs{Hotkey: DefaultHotkey, Mode: "dark", PinNew: true}); err != nil {
		t.Fatal(err)
	}
	ev := r.p.emitted(PrefsEvent)
	if len(ev) != 1 || ev[0].(Prefs).Mode != "dark" || !ev[0].(Prefs).PinNew {
		t.Fatalf("prefs events %v", ev)
	}
	id := r.s.OpenProject(filepath.Join(r.home, "new"))
	if !r.s.Status(id).Pinned || !r.p.projectWindows()[0].pinned() {
		t.Error("a new window is not on top with PinNew")
	}
	st, _ := LoadState(filepath.Join(r.home, StateFile))
	if !st.PinNew {
		t.Error("PinNew was not saved")
	}
}

func TestFirstSizeFitsTheScreen(t *testing.T) {
	if got := firstSize(Rect{Width: 1600, Height: 1000}); got.Width != ProjectWidth || got.Height != ProjectHeight {
		t.Errorf("on a large screen %+v", got)
	}
	if got := firstSize(Rect{Width: 700, Height: 600}); got.Width != 652 || got.Height != 552 {
		t.Errorf("on a small screen %+v", got)
	}
}

func TestReopenBringsBackTheLastBoard(t *testing.T) {
	r := newRig(t)
	r.s.Reopen()
	if len(r.p.homeWindows()) != 1 {
		t.Fatal("with no board, reopen shows the home")
	}
	r2 := newRig(t)
	r2.s.OpenProject(filepath.Join(r2.home, "a"))
	w := r2.p.projectWindows()[0]
	w.Hide()
	r2.s.Reopen()
	if !w.Visible() || len(r2.p.homeWindows()) != 0 {
		t.Errorf("reopen: board visible %v, home windows %d", w.Visible(), len(r2.p.homeWindows()))
	}
}

type fakeAutostart struct{ on bool }

func (f *fakeAutostart) Enabled() (bool, error)   { return f.on, nil }
func (f *fakeAutostart) SetEnabled(on bool) error { f.on = on; return nil }

func TestAutostart(t *testing.T) {
	r := newRig(t)
	if st := r.s.AutostartState(); st.Supported {
		t.Error("supported with no autostart")
	}
	if _, err := r.s.SetAutostart(true); !errors.Is(err, ErrNoAutostart) {
		t.Errorf("SetAutostart with none: %v", err)
	}
	a := &fakeAutostart{}
	r2 := newRig(t, func(c *Config) { c.Autostart = a })
	st, err := r2.s.SetAutostart(true)
	if err != nil || !st.Supported || !st.Enabled || !a.on {
		t.Errorf("SetAutostart(true) = %+v, %v", st, err)
	}
}

func TestPeekOfABoard(t *testing.T) {
	raw := json.RawMessage(`{
	  "title": "Alpha",
	  "settings": {"theme": "nord"},
	  "palette": {"Name": "nord", "Dark": true, "Notes": ["#1","#2","#3","#4","#5","#6"]},
	  "tabs": [
	    {"name": "work", "notes": [
	      {"name": "a.md", "data": {"columns": [
	        {"cards": [{"who": ["@claude"], "stalled": true}, {"who": ["sam", "claude"]}]},
	        {"cards": [{"who": ["codex"]}]}
	      ]}},
	      {"name": "b.md", "data": {"markdown": "x"}}
	    ]},
	    {"name": "log", "notes": [{"name": "book", "pages": [{"name": "1"}, {"name": "2"}]}]}
	  ]
	}`)
	pk, ok := peekOf(raw)
	if !ok {
		t.Fatal("not read")
	}
	if pk.Notes != 3 || pk.Stalled != 1 || pk.Theme != "nord" || !pk.Dark || len(pk.Colors) != 6 {
		t.Errorf("peek %+v", pk)
	}
	if strings.Join(pk.Agents, ",") != "claude,codex,sam" {
		t.Errorf("agents %v", pk.Agents)
	}
	if _, ok := peekOf(json.RawMessage(`{"title":"x"}`)); ok {
		t.Error("an answer with no tabs read as a board")
	}
}

func TestBoardLoadKeepsAPeek(t *testing.T) {
	r := newRig(t)
	a, path := readyProject(t, r, "a")
	r.s.peek(path, &[]json.RawMessage{json.RawMessage(`{"tabs":[{"notes":[{"name":"x"}]}]}`)}[0])
	if pk, ok := r.s.Peeks()[path]; !ok || pk.Notes != 1 {
		t.Errorf("peeks %v", r.s.Peeks())
	}
	_ = a
	st, _ := LoadState(filepath.Join(r.home, StateFile))
	if st.Boards[path].Notes != 1 {
		t.Error("the peek was not saved")
	}
}
