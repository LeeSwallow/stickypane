package shell

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/LeeSwallow/stickypane/desktop/internal/bridgeclient"
	"github.com/LeeSwallow/stickypane/desktop/internal/remote"
)

// A project window's size the first time; after that, where it was left
// (08 §8.1).
const (
	ProjectWidth  = 720
	ProjectHeight = 640
	// On a screen too small for it, the first size leaves screenMargin
	// of the work area free, and is never under the least size.
	screenMargin       = 48
	projectLeastWidth  = 400
	projectLeastHeight = 360
)

// The states of a project window's bridge.
const (
	StateLoading = "loading" // the bridge is starting
	StateReady   = "ready"
	StateMissing = "missing"  // no stickypane program was found
	StateTooOld  = "too_old"  // stickypane was found, but none has the bridge
	StateFailed  = "failed"   // it was found but its bridge would not start
	StateNoBoard = "no_board" // the folder has no .sticky/: the window offers to make one
	StateClosed  = "closed"   // no such window
)

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
	// Remote is the host's name of a board on another computer; "" for
	// one on this computer.
	Remote string `json:"remote,omitempty"`
}

// boardOf is the project of a board's own window (not a torn-off
// note's); ErrNoWindow when no board window has that id.
func (s *Shell) boardOf(window uint) (*project, error) {
	s.mu.Lock()
	p := s.byWin[window]
	s.mu.Unlock()
	if p == nil {
		return nil, noWindow(window)
	}
	return p, nil
}

// cleanPath makes one key of the ways a folder is written: absolute,
// clean, and through its symlinks (/tmp is /private/tmp on macOS) when
// the folder exists. A remote board's key is only made clean: its folder
// is on another computer.
func cleanPath(p string) string {
	if remote.IsRemote(p) {
		if name, dir, ok := remote.ParseKey(p); ok {
			return remote.Key(name, dir)
		}
		return p
	}
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
			raise(w)
			return w.ID()
		}
		return 0 // it is being made by another caller
	}
	p := &project{path: path, state: StateLoading, torn: map[string]*tornWin{}}
	s.projects[path] = p
	ws, placed := s.state.Windows[path]
	if !placed && s.state.PinNew {
		// a board seen for the first time takes the setting "keep new
		// windows on top", and keeps it as its own pin
		s.remember(p, func(w *WindowState) { w.Pinned = true })
		ws = s.state.Windows[path]
	}
	s.mu.Unlock()

	bounds := Rect{X: ws.X, Y: ws.Y, Width: ws.Width, Height: ws.Height}
	if !placed || ws.Width <= 0 || ws.Height <= 0 {
		bounds = firstSize(s.cfg.Platform.WorkArea())
	}
	title := filepath.Base(path)
	if name, dir, ok := remote.ParseKey(path); ok {
		title = filepath.Base(dir) + " — " + name
	}
	w := s.cfg.Platform.NewWindow(WindowOptions{
		Kind:      KindProject,
		Name:      prefixProject + path,
		Title:     title,
		URL:       pageProject,
		Bounds:    bounds,
		Placed:    placed && ws.Width > 0,
		Frameless: true,
		Pinned:    ws.Pinned,
	})
	id := w.ID()
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	p.win, p.ctx, p.cancel = w, ctx, cancel
	s.byWin[id] = p
	s.mu.Unlock()

	w.OnLostFocus(func() { s.lostFocus(p) })
	w.OnFocus(func() { s.focused(p) })
	w.OnMoved(func() { s.moved(p) })
	w.OnClosing(func() { s.closing(p) })
	raise(w)
	go s.connect(ctx, p)
	s.reopenTorn(p)
	return id
}

// firstSize is a project window's size the first time: ProjectWidth by
// ProjectHeight, within the screen's work area.
func firstSize(area Rect) Rect {
	w, h := ProjectWidth, ProjectHeight
	if area.Width > 0 && w > area.Width-screenMargin {
		w = max(area.Width-screenMargin, projectLeastWidth)
	}
	if area.Height > 0 && h > area.Height-screenMargin {
		h = max(area.Height-screenMargin, projectLeastHeight)
	}
	return Rect{Width: w, Height: h}
}

// connect starts the project's bridge and passes its notifications on.
func (s *Shell) connect(ctx context.Context, p *project) {
	if remote.IsRemote(p.path) {
		b, err := s.connectRemote(ctx, p)
		if err != nil {
			s.setState(p, StateFailed, err.Error(), nil)
			return
		}
		s.serve(p, b)
		return
	}
	bin, err := s.cfg.Find() // only ever a stickypane that has the bridge
	if errors.Is(err, bridgeclient.ErrTooOld) {
		s.setState(p, StateTooOld, err.Error(), nil)
		return
	}
	if err != nil {
		s.setState(p, StateMissing, err.Error(), nil)
		return
	}
	b, err := s.cfg.Start(ctx, bin, p.path)
	if err != nil {
		state := StateFailed
		if fi, serr := os.Stat(p.path); serr == nil && fi.IsDir() && !s.Inspect(p.path).Exists {
			state = StateNoBoard // a folder, but no board in it
		}
		s.setState(p, state, err.Error(), nil)
		return
	}
	s.serve(p, b)
}

// serve keeps a started bridge as the project's and passes its
// notifications on until it ends.
func (s *Shell) serve(p *project, b Board) {
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
			case bridgeclient.MethodRestarted:
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
	p, tw := s.ownerOf(window)
	if p == nil {
		return Status{Window: window, State: StateClosed}
	}
	st := Status{Window: window, Path: p.path, State: p.state, Error: p.err, Pinned: s.state.Windows[p.path].Pinned, Remote: remoteName(p.path)}
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
	p, _ := s.ownerOf(window)
	var b Board
	state := StateClosed
	if p != nil {
		b, state = p.board, p.state
	}
	s.mu.Unlock()
	if b == nil {
		return fmt.Errorf("%w (%s)", ErrNoBoard, state)
	}
	far := remote.IsRemote(p.path)
	if far && method == bridgeclient.MethodOpen {
		return fmt.Errorf("%w: its notes open in an editor there, not here", ErrRemote)
	}
	err := b.Call(ctx, method, params, result)
	if err == nil && method == bridgeclient.MethodBoardLoad {
		if !far { // the index is of this computer's boards
			s.retitle(p.path, result)
		}
		s.keepNotes(p, result)
		s.peek(p.path, result)
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
		if err := ix.Retitle(path, b.Title); err != nil {
			s.cfg.Logf("retitle %s: %v", path, err) // the next load tries again
		}
	}
}

// SetPin pins or unpins a project window: a pinned window stays on top
// and does not hide when it loses focus.
func (s *Shell) SetPin(window uint, on bool) error {
	s.mu.Lock()
	p, tw := s.owner(window)
	if p == nil {
		s.mu.Unlock()
		return noWindow(window)
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
	if pinned || closed {
		return
	}
	if s.focusInProject(p) {
		return
	}
	if d := s.cfg.FocusSettle; d > 0 {
		// the desktop may say the board lost the focus before it says
		// which window has it now
		go func() {
			time.Sleep(d)
			if !s.focusInProject(p) && !w.Focused() {
				w.Hide()
			}
		}()
		return
	}
	w.Hide()
}

// focusInProject says whether one of the project's torn-off notes has the
// focus: going to one of them leaves the board on screen beside it.
func (s *Shell) focusInProject(p *project) bool {
	s.mu.Lock()
	var wins []Window
	for _, tw := range p.torn {
		if tw.win != nil && !tw.closed {
			wins = append(wins, tw.win)
		}
	}
	s.mu.Unlock()
	for _, w := range wins {
		if w.Focused() {
			return true
		}
	}
	return false
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
	s.detachSettings(p)
	s.keep("closing")
	s.trayHint(p.win)
}
