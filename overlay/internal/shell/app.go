// Package shell is the overlay app's desktop side: the home window, one
// window per project with its bridge, the tray, the global hotkey, pins,
// and the second launch of a single-instance app (docs/overlay/08 §8.1,
// §8.5; ADR-0007). The desktop is reached through Platform and Window, so
// that everything here runs in tests without a display; windows.go and
// tray.go put Wails behind them.
package shell

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LeeSwallow/stickypane/overlay/internal/bridge"
	"github.com/LeeSwallow/stickypane/overlay/internal/canvas"
	"github.com/LeeSwallow/stickypane/overlay/internal/index"
)

// Sizes of the windows (08 §8.1).
const (
	HomeWidth     = 860
	HomeHeight    = 620
	ProjectWidth  = 840
	ProjectHeight = 0.70 // of the screen's work area
	trayRecent    = 5
)

// The states of a project window's bridge.
const (
	StateLoading = "loading" // the bridge is starting
	StateReady   = "ready"
	StateMissing = "missing" // no stickypane program was found
	StateTooOld  = "too_old" // stickypane was found, but none has the bridge
	StateFailed  = "failed"  // it was found but its bridge would not start
	StateClosed  = "closed"  // no such window
)

// NoticeHotkey: the global hotkey could not be registered.
const NoticeHotkey = "hotkey_failed"

// NoticesEvent tells the home window its notices changed.
const NoticesEvent = "home.notices"

// ChangedEvent is the Wails event that carries a project window's
// board.changed; RestartedEvent its bridge.restarted; StatusEvent a change
// of its Status.
func ChangedEvent(window uint) string {
	return "board.changed:" + strconv.FormatUint(uint64(window), 10)
}
func RestartedEvent(window uint) string {
	return "bridge.restarted:" + strconv.FormatUint(uint64(window), 10)
}
func StatusEvent(window uint) string {
	return "project.status:" + strconv.FormatUint(uint64(window), 10)
}

// Board is a running bridge: *bridge.Client.
type Board interface {
	Call(ctx context.Context, method string, params, result any) error
	Notifications() <-chan bridge.Notification
	Hello() bridge.Hello
	Close() error
}

// Index is the index of boards on this machine: *index.Index.
type Index interface {
	Ingest() error
	Prune(now time.Time) ([]index.Project, error)
	Restore(id int64) error
	Retitle(path, title string) error
	Sessions(now time.Time) (index.Groups, error)
	Close() error
}

// Config is what New needs. Find, Start and OpenIndex default to the real
// bridge and index.
type Config struct {
	Home          string // ~/.stickypane, or $STICKYPANE_HOME
	Platform      Platform
	Find          func() (string, error)             // a stickypane with the bridge
	Probe         func(path string) bridge.Candidate // what a stickypane says of itself
	Start         func(ctx context.Context, bin, project string) (Board, error)
	OpenIndex     func(home string) (Index, error)
	IndexDeadline time.Duration // how long Sessions waits for the index to open
}

// Notice is a line for the home window. Code is translated there; Detail
// is the system's own words.
type Notice struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

// Status is what a project window knows of itself.
type Status struct {
	Window  uint   `json:"window"`
	Path    string `json:"path"`
	State   string `json:"state"`
	Error   string `json:"error,omitempty"`
	Pinned  bool   `json:"pinned"`
	Version string `json:"version,omitempty"` // the bridge's stickypane
	// Note is the note a torn-off note's window shows; "" in a board's.
	Note string `json:"note,omitempty"`
}

// project is one open project window.
type project struct {
	path   string
	win    Window // nil while it is being made
	board  Board
	state  string
	err    string
	cancel context.CancelFunc
	closed bool
	torn   map[string]*tornWin // its notes torn off into windows, by note
}

// Shell is the app. Its methods are safe for concurrent use; none holds
// the lock while it calls the platform, whose calls wait for the UI thread.
type Shell struct {
	cfg       Config
	statePath string
	saveMu    sync.Mutex // one write of overlay.json at a time
	hotkeyMu  sync.Mutex // one change of the hotkey at a time

	mu       sync.Mutex
	state    State
	home     Window
	projects map[string]*project // by clean path
	byWin    map[uint]*project
	// tornByWin are the windows of torn-off notes (torn.go)
	tornByWin map[uint]*tornWin
	notices   []Notice
	hotkey    string // registered accelerator, "" when none
	hello     bridge.Hello
	started   bool

	ix      Index
	ixErr   error
	ixReady chan struct{} // closed when the first open ended or timed out
	quit    chan struct{}
	stopped bool
}

// DefaultHome is $STICKYPANE_HOME, else ~/.stickypane.
func DefaultHome() string {
	if h := os.Getenv("STICKYPANE_HOME"); h != "" {
		return h
	}
	h, err := os.UserHomeDir()
	if err != nil || h == "" {
		return ".stickypane"
	}
	return filepath.Join(h, ".stickypane")
}

// New reads the state and returns the shell; Start registers the hotkey
// and opens the index.
func New(cfg Config) (*Shell, error) {
	if cfg.Home == "" {
		cfg.Home = DefaultHome()
	}
	if cfg.Find == nil {
		cfg.Find = bridge.Find
	}
	if cfg.Probe == nil {
		cfg.Probe = bridge.Probe
	}
	if cfg.Start == nil {
		cfg.Start = func(ctx context.Context, bin, project string) (Board, error) {
			c, err := bridge.Start(ctx, bin, project)
			if err != nil {
				return nil, err
			}
			return c, nil
		}
	}
	if cfg.OpenIndex == nil {
		cfg.OpenIndex = func(home string) (Index, error) {
			ix, err := index.Open(home)
			if err != nil {
				return nil, err
			}
			return ix, nil
		}
	}
	if cfg.IndexDeadline == 0 {
		cfg.IndexDeadline = 10 * time.Second
	}
	path := filepath.Join(cfg.Home, StateFile)
	st, err := LoadState(path)
	s := &Shell{
		cfg:       cfg,
		statePath: path,
		state:     st,
		projects:  map[string]*project{},
		byWin:     map[uint]*project{},
		tornByWin: map[uint]*tornWin{},
		ixReady:   make(chan struct{}),
		quit:      make(chan struct{}),
	}
	if err != nil { // a damaged file costs the window places, not the app
		s.notices = append(s.notices, Notice{Code: "state_unreadable", Detail: err.Error()})
	}
	return s, nil
}

// Start runs once the app runs: it registers the hotkey and opens the
// index off the caller's thread, bounded by IndexDeadline.
func (s *Shell) Start() {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return
	}
	s.started = true
	accel := s.state.Hotkey
	s.mu.Unlock()

	s.registerHotkey(accel)
	go s.openIndex()
}

func (s *Shell) registerHotkey(accel string) {
	if accel == "" {
		return
	}
	s.hotkeyMu.Lock()
	defer s.hotkeyMu.Unlock()
	if err := s.cfg.Platform.RegisterHotkey(accel, s.Toggle); err != nil {
		s.notice(Notice{Code: NoticeHotkey, Detail: fmt.Sprintf("%s: %v", accel, err)})
		return
	}
	s.mu.Lock()
	s.hotkey = accel
	s.mu.Unlock()
}

// SetHotkey changes the hotkey, or registers it again when it is the one
// that could not be had, and keeps it in the state. If the new one cannot
// be had the old one is kept; if the old one cannot be had back either,
// the home window is told there is no hotkey.
func (s *Shell) SetHotkey(accel string) error {
	s.hotkeyMu.Lock()
	defer s.hotkeyMu.Unlock()
	s.mu.Lock()
	old := s.hotkey // registered now; "" when none is
	s.mu.Unlock()
	if old == accel {
		return nil
	}
	if old != "" {
		s.cfg.Platform.UnregisterHotkey(old)
	}
	if err := s.cfg.Platform.RegisterHotkey(accel, s.Toggle); err != nil {
		if old != "" {
			if rerr := s.cfg.Platform.RegisterHotkey(old, s.Toggle); rerr != nil {
				s.mu.Lock()
				s.hotkey = ""
				s.mu.Unlock()
				s.notice(Notice{Code: NoticeHotkey, Detail: fmt.Sprintf("%s: %v; %s: %v", accel, err, old, rerr)})
			}
		}
		return err
	}
	s.mu.Lock()
	s.hotkey = accel
	s.state.Hotkey = accel
	s.notices = dropNotice(s.notices, NoticeHotkey)
	s.mu.Unlock()
	s.cfg.Platform.Emit(NoticesEvent, s.Notices())
	return s.save()
}

func dropNotice(ns []Notice, code string) []Notice {
	out := ns[:0]
	for _, n := range ns {
		if n.Code != code {
			out = append(out, n)
		}
	}
	return out
}

func (s *Shell) notice(n Notice) {
	s.mu.Lock()
	s.notices = append(dropNotice(s.notices, n.Code), n)
	s.mu.Unlock()
	s.cfg.Platform.Emit(NoticesEvent, s.Notices())
}

// Notices are the lines the home window shows.
func (s *Shell) Notices() []Notice {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Notice{}, s.notices...)
}

// openIndex opens the index in the background. Open retries while the
// database is locked and can take long; after IndexDeadline the home
// window is told it failed, and a late success is still taken.
func (s *Shell) openIndex() {
	done := make(chan struct{})
	go func() {
		defer close(done)
		ix, err := s.cfg.OpenIndex(s.cfg.Home)
		s.mu.Lock()
		if s.stopped {
			s.mu.Unlock()
			if ix != nil {
				ix.Close()
			}
			return
		}
		s.ix, s.ixErr = ix, err
		s.mu.Unlock()
		if err == nil {
			s.refreshTray()
		}
	}()
	select {
	case <-done:
	case <-time.After(s.cfg.IndexDeadline):
		s.mu.Lock()
		if s.ix == nil && s.ixErr == nil {
			s.ixErr = fmt.Errorf("index: not open after %v (is another app holding %s?)", s.cfg.IndexDeadline, filepath.Join(s.cfg.Home, "stickypane.db"))
		}
		s.mu.Unlock()
	case <-s.quit:
	}
	close(s.ixReady)
}

// index waits for the first open of the index to end.
func (s *Shell) index() (Index, error) {
	select {
	case <-s.ixReady:
	case <-time.After(s.cfg.IndexDeadline + time.Second): // Start was not called
		return nil, errors.New("index: not started")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ix != nil {
		return s.ix, nil
	}
	if s.ixErr != nil {
		return nil, s.ixErr
	}
	return nil, errors.New("index: not open")
}

// Ensure makes the home folder (that the app is used is what makes
// stickypane keep records there) and returns its path.
func (s *Shell) Ensure() (string, error) {
	return s.cfg.Home, os.MkdirAll(s.cfg.Home, 0o755)
}

// Sessions reads new events, tidies away folders that are gone, and
// returns the home list.
func (s *Shell) Sessions() (index.Groups, error) {
	ix, err := s.index()
	if err != nil {
		return index.Groups{}, err
	}
	g, err := sessions(ix)
	if err == nil {
		s.cfg.Platform.SetTray(trayOf(g))
	}
	return g, err
}

func sessions(ix Index) (index.Groups, error) {
	if err := ix.Ingest(); err != nil {
		return index.Groups{}, err
	}
	now := time.Now()
	if _, err := ix.Prune(now); err != nil {
		return index.Groups{}, err
	}
	return ix.Sessions(now)
}

// Restore brings back a project that was tidied away.
func (s *Shell) Restore(id int64) error {
	ix, err := s.index()
	if err != nil {
		return err
	}
	return ix.Restore(id)
}

// refreshTray rebuilds the tray from the index, if it is open.
func (s *Shell) refreshTray() {
	s.mu.Lock()
	ix := s.ix
	s.mu.Unlock()
	if ix == nil {
		return
	}
	if g, err := sessions(ix); err == nil {
		s.cfg.Platform.SetTray(trayOf(g))
	}
}

// StickypaneState is whether a stickypane was found, whether it has the
// bridge the overlay needs, and what it says of itself.
type StickypaneState struct {
	Found       bool               `json:"found"`
	Bridge      bool               `json:"bridge"` // false: every copy found is too old
	Path        string             `json:"path,omitempty"`
	Version     string             `json:"version,omitempty"`
	Protocol    int                `json:"protocol"`             // the bridge's, 0 until a bridge has said hello
	AppProtocol int                `json:"appProtocol"`          // the one this app speaks
	Candidates  []bridge.Candidate `json:"candidates,omitempty"` // the copies found, when all are too old
	Error       string             `json:"error,omitempty"`
}

// StickypaneStatus finds the stickypane the app would use. The protocol is
// known once a project window's bridge has said hello.
func (s *Shell) StickypaneStatus() StickypaneState {
	st := StickypaneState{AppProtocol: bridge.Protocol}
	bin, err := s.cfg.Find()
	var old *bridge.TooOldError
	switch {
	case errors.As(err, &old):
		st.Found, st.Candidates, st.Error = true, old.Found, err.Error()
		if len(old.Found) > 0 {
			st.Path, st.Version = old.Found[0].Path, old.Found[0].Version
		}
		return st
	case err != nil:
		st.Error = err.Error()
		return st
	}
	c := s.cfg.Probe(bin)
	st.Found, st.Bridge, st.Path, st.Version = true, true, bin, c.Version
	s.mu.Lock()
	st.Protocol = s.hello.Protocol
	s.mu.Unlock()
	return st
}

// Inspection says whether a folder already has a board.
type Inspection struct {
	Exists bool   `json:"exists"` // dir has a .sticky/
	Path   string `json:"path"`   // dir, cleaned
}

// Inspect looks at dir without changing anything.
func (s *Shell) Inspect(dir string) Inspection {
	dir = filepath.Clean(dir)
	fi, err := os.Stat(filepath.Join(dir, ".sticky"))
	return Inspection{Exists: err == nil && fi.IsDir(), Path: dir}
}

// userHomeDir is the person's home folder; a test replaces it.
var userHomeDir = os.UserHomeDir

// resolvePath cleans p and follows its symlinks; when that fails, only cleans.
func resolvePath(p string) string {
	p = filepath.Clean(p)
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return filepath.Clean(r)
	}
	return p
}

// samePath compares two resolved paths; macOS volumes are case-insensitive.
func samePath(a, b string) bool {
	if runtime.GOOS == "darwin" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// refuseInit stops a board being made in the home folder or the root, where
// every project below would find it.
func refuseInit(dir string) error {
	d := resolvePath(dir)
	if d == string(filepath.Separator) {
		return errors.New("will not make a board in the root folder")
	}
	if h, err := userHomeDir(); err == nil && samePath(d, resolvePath(h)) {
		return errors.New("will not make a board in the home folder: every project below it would find it")
	}
	return nil
}

// InitProject makes a board in dir with `stickypane init`.
func (s *Shell) InitProject(dir string) error {
	if err := refuseInit(dir); err != nil {
		return err
	}
	bin, err := s.cfg.Find()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "init")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("stickypane init: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// PickFolder asks for a folder.
func (s *Shell) PickFolder() (string, error) { return s.cfg.Platform.PickFolder() }

// ShowHome shows the home window, making it the first time.
func (s *Shell) ShowHome() {
	s.mu.Lock()
	w := s.home
	s.mu.Unlock()
	if w == nil {
		nw := s.cfg.Platform.NewWindow(WindowOptions{
			Kind:   KindHome,
			Name:   "home",
			Title:  "stickypane",
			URL:    "/",
			Bounds: Rect{Width: HomeWidth, Height: HomeHeight},
			// our title bar, with the OS's window buttons (03 §3.11.6)
			Frameless: true,
		})
		s.mu.Lock()
		if s.home == nil {
			s.home = nw
		}
		w = s.home
		s.mu.Unlock()
		if w != nw { // another caller made it first
			nw.Close()
		} else {
			// The home hides when closed; the first time, it says (in itself,
			// shown again) that the app stays in the tray.
			nw.OnClosing(func() { go s.trayHint(nw) })
		}
	}
	w.Show()
	w.Focus()
}

// cleanPath makes one key of the ways a folder is written: absolute,
// clean, and through its symlinks (/tmp is /private/tmp on macOS) when
// the folder exists.
func cleanPath(p string) string {
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	if real, err := filepath.EvalSymlinks(p); err == nil {
		return real
	}
	return filepath.Clean(p)
}

// OpenProject shows the window of the project at path, making it and
// starting its bridge if it is not open, and returns its id.
func (s *Shell) OpenProject(path string) uint {
	path = cleanPath(path)
	s.mu.Lock()
	s.state.Last = path
	if p := s.projects[path]; p != nil {
		w := p.win
		s.mu.Unlock()
		if w != nil {
			w.Show()
			w.Focus()
			return w.ID()
		}
		return 0 // it is being made by another caller
	}
	p := &project{path: path, state: StateLoading, torn: map[string]*tornWin{}}
	s.projects[path] = p
	ws, placed := s.state.Windows[path]
	s.mu.Unlock()

	bounds := Rect{X: ws.X, Y: ws.Y, Width: ws.Width, Height: ws.Height}
	if !placed || ws.Width <= 0 || ws.Height <= 0 {
		area := s.cfg.Platform.WorkArea()
		h := int(float64(area.Height) * ProjectHeight)
		if h <= 0 {
			h = 600
		}
		bounds = Rect{Width: ProjectWidth, Height: h}
	}
	w := s.cfg.Platform.NewWindow(WindowOptions{
		Kind:      KindProject,
		Name:      "project:" + path,
		Title:     filepath.Base(path),
		URL:       "/?window=project",
		Bounds:    bounds,
		Placed:    placed && ws.Width > 0,
		Frameless: true,
		Pinned:    ws.Pinned,
	})
	id := w.ID()
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	p.win, p.cancel = w, cancel
	s.byWin[id] = p
	s.mu.Unlock()

	w.OnLostFocus(func() { s.lostFocus(p) })
	w.OnFocus(func() { s.focused(p) })
	w.OnMoved(func() { s.moved(p) })
	w.OnClosing(func() { s.closing(p) })
	w.Show()
	w.Focus()
	go s.connect(ctx, p)
	s.reopenTorn(p)
	return id
}

// connect starts the project's bridge and passes its notifications on.
func (s *Shell) connect(ctx context.Context, p *project) {
	bin, err := s.cfg.Find() // only ever a stickypane that has the bridge
	if errors.Is(err, bridge.ErrTooOld) {
		s.setState(p, StateTooOld, err.Error(), nil)
		return
	}
	if err != nil {
		s.setState(p, StateMissing, err.Error(), nil)
		return
	}
	b, err := s.cfg.Start(ctx, bin, p.path)
	if err != nil {
		s.setState(p, StateFailed, err.Error(), nil)
		return
	}
	if !s.setState(p, StateReady, "", b) { // closed meanwhile
		b.Close()
		return
	}
	s.mu.Lock()
	s.hello = b.Hello()
	s.mu.Unlock()
	go s.refreshTray()
	for n := range b.Notifications() {
		// the project's window and its torn-off notes hear the same
		s.mu.Lock()
		ids := s.windowIDs(p)
		s.mu.Unlock()
		for _, id := range ids {
			switch n.Method {
			case bridge.MethodRestarted:
				s.cfg.Platform.Emit(RestartedEvent(id), nil)
			default:
				s.cfg.Platform.Emit(ChangedEvent(id), n.Params)
			}
		}
	}
}

// setState records a project's bridge state and tells its window. It says
// false, and keeps nothing, when the window has closed.
func (s *Shell) setState(p *project, state, why string, b Board) bool {
	s.mu.Lock()
	if p.closed {
		s.mu.Unlock()
		return false
	}
	p.state, p.err, p.board = state, why, b
	ids := s.windowIDs(p)
	s.mu.Unlock()
	for _, id := range ids {
		s.cfg.Platform.Emit(StatusEvent(id), s.Status(id))
	}
	return true
}

// Status is what a window knows of its project.
func (s *Shell) Status(window uint) Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, tw := s.owner(window)
	if p == nil {
		return Status{Window: window, State: StateClosed}
	}
	st := Status{Window: window, Path: p.path, State: p.state, Error: p.err, Pinned: s.state.Windows[p.path].Pinned}
	if tw != nil {
		st.Note, st.Pinned = tw.note, s.state.Projects[p.path].Torn[tw.note].Pinned
	}
	if p.board != nil {
		st.Version = p.board.Hello().Version
	}
	return st
}

// Call sends one request to the bridge of a window's project.
func (s *Shell) Call(ctx context.Context, window uint, method string, params, result any) error {
	s.mu.Lock()
	p, _ := s.owner(window)
	var b Board
	state := StateClosed
	if p != nil {
		b, state = p.board, p.state
	}
	s.mu.Unlock()
	if b == nil {
		return fmt.Errorf("no board in this window (%s)", state)
	}
	err := b.Call(ctx, method, params, result)
	if err == nil && method == "board.load" {
		s.retitle(p.path, result)
		s.keepNotes(p, result)
	}
	return err
}

// retitle gives the index the title a board.load answered with: the app
// learns a project's name from its bridge, never from its files. An index
// not open yet is not waited for; the next load names it.
func (s *Shell) retitle(path string, result any) {
	raw, ok := result.(*json.RawMessage)
	if !ok || raw == nil {
		return
	}
	var b struct {
		Title string `json:"title"`
	}
	if json.Unmarshal(*raw, &b) != nil || b.Title == "" {
		return
	}
	s.mu.Lock()
	ix := s.ix
	s.mu.Unlock()
	if ix != nil {
		ix.Retitle(path, b.Title)
	}
}

// SetPin pins or unpins a project window: a pinned window stays on top
// and does not hide when it loses focus.
func (s *Shell) SetPin(window uint, on bool) error {
	s.mu.Lock()
	p, tw := s.owner(window)
	if p == nil {
		s.mu.Unlock()
		return fmt.Errorf("no project window %d", window)
	}
	if tw != nil {
		return s.pinTorn(tw, on) // unlocks
	}
	s.remember(p, func(ws *WindowState) { ws.Pinned = on })
	w := p.win
	s.mu.Unlock()
	w.SetPinned(on)
	s.cfg.Platform.Emit(StatusEvent(window), s.Status(window))
	return s.save()
}

// remember changes the saved state of a project's window. The caller
// holds mu.
func (s *Shell) remember(p *project, f func(*WindowState)) {
	if s.state.Windows == nil {
		s.state.Windows = map[string]WindowState{}
	}
	ws := s.state.Windows[p.path]
	f(&ws)
	s.state.Windows[p.path] = ws
}

func (s *Shell) lostFocus(p *project) {
	s.mu.Lock()
	pinned := s.state.Windows[p.path].Pinned
	w := p.win
	closed := p.closed
	s.mu.Unlock()
	if !pinned && !closed {
		w.Hide()
	}
}

func (s *Shell) focused(p *project) {
	s.mu.Lock()
	if !p.closed {
		s.state.Last = p.path
	}
	s.mu.Unlock()
}

func (s *Shell) moved(p *project) {
	b := p.win.Bounds()
	if b.Width <= 0 || b.Height <= 0 {
		return
	}
	s.mu.Lock()
	if !p.closed {
		s.place(p, b)
	}
	s.mu.Unlock()
}

// place keeps where a window is. The desktop reports a window up to a
// point off what it was asked for (rounding to pixels), so a change of a
// point or less is not a move: else the window would shrink at every
// launch. The caller holds mu.
func (s *Shell) place(p *project, b Rect) {
	s.remember(p, func(ws *WindowState) {
		if near(ws.X, b.X) && near(ws.Y, b.Y) && near(ws.Width, b.Width) && near(ws.Height, b.Height) {
			return
		}
		ws.X, ws.Y, ws.Width, ws.Height = b.X, b.Y, b.Width, b.Height
	})
}

func near(a, b int) bool { return a-b <= 1 && b-a <= 1 }

// closing ends a project window's bridge and keeps where it was.
func (s *Shell) closing(p *project) {
	s.mu.Lock()
	if p.closed {
		s.mu.Unlock()
		return
	}
	p.closed = true
	delete(s.byWin, p.win.ID())
	if s.projects[p.path] == p {
		delete(s.projects, p.path)
	}
	b, cancel := p.board, p.cancel
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if b != nil {
		b.Close()
	}
	s.closeTorn(p)
	s.save()
	s.trayHint(p.win)
}

// NoticeStayInTray tells the home window that a closed window left the app
// running in the tray (shown once, with a Quit action).
const NoticeStayInTray = "stayInTray"

// trayHint says once, the first time a window is closed, that the app
// stays in the tray. If no window is left to say it in, the home window
// comes forward to say it.
func (s *Shell) trayHint(closing Window) {
	s.mu.Lock()
	if s.stopped || s.state.TrayHint {
		s.mu.Unlock()
		return
	}
	s.state.TrayHint = true
	var wins []Window
	if s.home != nil {
		wins = append(wins, s.home)
	}
	for _, p := range s.projects {
		if p.win != nil {
			wins = append(wins, p.win)
		}
	}
	s.mu.Unlock()
	s.save()
	s.notice(Notice{Code: NoticeStayInTray})
	for _, w := range wins { // asked outside the lock
		if w != closing && w.Visible() {
			return
		}
	}
	s.ShowHome()
}

// Quit ends the app: every window, every bridge, the tray.
func (s *Shell) Quit() { s.cfg.Platform.Quit() }

// Toggle is the hotkey: it hides the last project window when it is in
// front, and otherwise brings it forward (opening it again if it was
// closed). With no project yet it shows the home window.
func (s *Shell) Toggle() {
	s.mu.Lock()
	last := s.state.Last
	var w Window
	if p := s.projects[last]; p != nil {
		w = p.win
	}
	s.mu.Unlock()
	switch {
	case w != nil && w.Visible() && w.Focused():
		w.Hide()
	case w != nil:
		w.Show()
		w.Focus()
	case last != "":
		s.OpenProject(last)
	default:
		s.ShowHome()
	}
}

// Launch handles the arguments of a launch, the first or a second one
// passed on by the single instance: --project=<path> (relative to dir)
// opens that project, anything else shows the home window.
func (s *Shell) Launch(args []string, dir string) {
	path := ProjectArg(args)
	if path == "" {
		s.ShowHome()
		return
	}
	if !filepath.IsAbs(path) && dir != "" {
		path = filepath.Join(dir, path)
	}
	s.OpenProject(path)
}

// ProjectArg is the value of --project in args (args[0] is the program).
func ProjectArg(args []string) string {
	for i := 1; i < len(args); i++ {
		a := args[i]
		for _, pre := range []string{"--project=", "-project="} {
			if strings.HasPrefix(a, pre) {
				return strings.TrimPrefix(a, pre)
			}
		}
		if (a == "--project" || a == "-project") && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

// save writes the state file.
// Saves are one at a time, each taking the state when its turn comes, so
// an older state never lands after a newer one.
func (s *Shell) save() error {
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	s.mu.Lock()
	st := s.state.clone()
	s.mu.Unlock()
	return st.Save(s.statePath)
}

// Prefs are the home window's settings in overlay.json.
type Prefs struct {
	Hotkey   string `json:"hotkey"`
	Language string `json:"language"`
	Mode     string `json:"mode"`
	// Motion is how much the windows move: "" as the system says,
	// "reduce" (fades only) or "off".
	Motion string `json:"motion"`
}

// Prefs returns the home window's settings.
func (s *Shell) Prefs() Prefs {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Prefs{Hotkey: s.state.Hotkey, Language: s.state.Language, Mode: s.state.Mode, Motion: s.state.Motion}
}

// SetPrefs keeps the home window's settings; a new hotkey is registered.
func (s *Shell) SetPrefs(p Prefs) error {
	s.mu.Lock()
	s.state.Language, s.state.Mode = p.Language, p.Mode
	switch p.Motion {
	case "reduce", "off":
		s.state.Motion = p.Motion
	default:
		s.state.Motion = ""
	}
	// A new hotkey, or the same one that could not be had: try it.
	change := p.Hotkey != "" && p.Hotkey != s.hotkey
	s.mu.Unlock()
	if change {
		if err := s.SetHotkey(p.Hotkey); err != nil {
			s.save()
			return err
		}
		return nil
	}
	return s.save()
}

// Shutdown closes every bridge and the index and saves the state. It is
// safe to call more than once.
func (s *Shell) Shutdown() {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}
	s.stopped = true
	close(s.quit)
	var boards []Board
	var cancels []context.CancelFunc
	var open []*project
	var torn []*tornWin
	for _, p := range s.projects {
		if p.win != nil {
			open = append(open, p)
		}
		for _, tw := range p.torn {
			if tw.win != nil && !tw.closed {
				torn = append(torn, tw)
			}
		}
		if p.board != nil {
			boards = append(boards, p.board)
		}
		if p.cancel != nil {
			cancels = append(cancels, p.cancel)
		}
	}
	ix := s.ix
	s.ix = nil
	s.mu.Unlock()
	for _, p := range open { // where each window is now; the platform is asked outside the lock
		if b := p.win.Bounds(); b.Width > 0 && b.Height > 0 {
			s.mu.Lock()
			s.place(p, b)
			s.mu.Unlock()
		}
	}
	for _, tw := range torn { // and each torn-off note's, kept for the next launch
		b := tw.win.Bounds()
		s.mu.Lock()
		if f, ok := s.frameOf(tw, b); ok {
			s.edit(tw.p, func(c canvas.Project) canvas.Project { return c.Tear(tw.note, f) })
		}
		s.mu.Unlock()
	}
	for _, c := range cancels {
		c()
	}
	for _, b := range boards {
		b.Close()
	}
	if ix != nil {
		ix.Close()
	}
	s.save()
}
