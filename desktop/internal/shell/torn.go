package shell

// The free canvas and the notes torn off it (docs/desktop/03 §3.6.1,
// ADR-0017). Where notes sit is this computer's: the shell keeps it in
// overlay.json by project path, and the rules of it are internal/canvas.
// A torn-off note is a small window of its own over the same bridge as its
// project's window; it lives while that window does, and comes back with
// it.

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/LeeSwallow/stickypane/desktop/internal/canvas"
)

// The first size of a torn-off note's window, and how far each next one
// opens from the one before.
const (
	NoteWidth   = 360
	NoteHeight  = 420
	noteCascade = 28
)

// CanvasEvent tells a project window its layout changed outside it: a note
// torn off, put back or gone.
func CanvasEvent(window uint) string {
	return "canvas.changed:" + strconv.FormatUint(uint64(window), 10)
}

// NoteURL is the page of a torn-off note's window.
func NoteURL(note string) string {
	return "/?window=note&note=" + url.QueryEscape(note)
}

// tornWin is one torn-off note's window.
type tornWin struct {
	p      *project
	note   string
	win    Window // nil while it is being made
	quiet  bool   // closed by the app, not the person: its frame stays
	closed bool
}

// Layout is a project's canvas as a window reads it.
type Layout struct {
	Mode   string                  `json:"mode"`
	Snap   bool                    `json:"snap"`
	Places map[string]canvas.Place `json:"places"`
	Torn   map[string]canvas.Frame `json:"torn"`
}

func layoutOf(c canvas.Project) Layout {
	c = c.Clone() // never nil maps: the page reads {} and not null
	return Layout{Mode: c.ModeOf(), Snap: !c.NoSnap, Places: c.Places, Torn: c.Torn}
}

// owner is the project of a window, its own or a torn-off note's, and the
// torn-off note when it is one. The caller holds mu.
func (s *Shell) owner(window uint) (*project, *tornWin) {
	if p := s.byWin[window]; p != nil {
		return p, nil
	}
	if tw := s.tornByWin[window]; tw != nil && !tw.closed {
		return tw.p, tw
	}
	return nil, nil
}

// edit changes a project's layout in the state. The caller holds mu.
func (s *Shell) edit(p *project, f func(canvas.Project) canvas.Project) {
	next := f(s.state.Projects[p.path])
	if next.Empty() {
		delete(s.state.Projects, p.path)
		return
	}
	if s.state.Projects == nil {
		s.state.Projects = map[string]canvas.Project{}
	}
	s.state.Projects[p.path] = next
}

// emitLayout tells a project's window its layout.
func (s *Shell) emitLayout(p *project) {
	s.mu.Lock()
	if p.closed || p.win == nil {
		s.mu.Unlock()
		return
	}
	id := p.win.ID()
	l := layoutOf(s.state.Projects[p.path])
	s.mu.Unlock()
	s.cfg.Platform.Emit(CanvasEvent(id), l)
}

// Layout is the canvas of a window's project.
func (s *Shell) Layout(window uint) (Layout, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, _ := s.owner(window)
	if p == nil {
		return Layout{}, fmt.Errorf("no project window %d", window)
	}
	return layoutOf(s.state.Projects[p.path]), nil
}

// change edits the layout of a window's project and saves it.
func (s *Shell) change(window uint, f func(canvas.Project) canvas.Project) error {
	s.mu.Lock()
	p, _ := s.owner(window)
	if p == nil {
		s.mu.Unlock()
		return fmt.Errorf("no project window %d", window)
	}
	s.edit(p, f)
	s.mu.Unlock()
	return s.save()
}

// SetMode switches the project's board window between canvas and grid.
func (s *Shell) SetMode(window uint, mode string) error {
	if !canvas.ValidMode(mode) {
		return fmt.Errorf("no layout %q: canvas or grid", mode)
	}
	return s.change(window, func(c canvas.Project) canvas.Project {
		c = c.Clone()
		c.Mode = mode
		return c
	})
}

// SetSnap makes notes land on the canvas's grid, or anywhere.
func (s *Shell) SetSnap(window uint, on bool) error {
	return s.change(window, func(c canvas.Project) canvas.Project {
		c = c.Clone()
		c.NoSnap = !on
		return c
	})
}

// SetPlaces keeps where notes are on the canvas.
func (s *Shell) SetPlaces(window uint, places map[string]canvas.Place) error {
	return s.change(window, func(c canvas.Project) canvas.Project { return c.SetPlaces(places) })
}

// TearOff opens a note of a window's project in a window of its own, or
// brings that window forward, and returns its id.
func (s *Shell) TearOff(window uint, note string) (uint, error) {
	if strings.TrimSpace(note) == "" {
		return 0, fmt.Errorf("which note?")
	}
	s.mu.Lock()
	p, _ := s.owner(window)
	if p == nil || p.closed {
		s.mu.Unlock()
		return 0, fmt.Errorf("no project window %d", window)
	}
	if tw := p.torn[note]; tw != nil {
		w := tw.win
		s.mu.Unlock()
		if w == nil {
			return 0, nil // being made by another caller
		}
		w.Show()
		w.Focus()
		return w.ID(), nil
	}
	f, ok := s.state.Projects[p.path].Torn[note]
	if !ok {
		f = canvas.Frame{Pinned: true} // a sticky note stays on top until unpinned
	}
	tw := &tornWin{p: p, note: note}
	p.torn[note] = tw
	n := len(p.torn) - 1
	s.mu.Unlock()
	id := s.openTorn(tw, f, n)
	s.save()
	s.emitLayout(p)
	return id, nil
}

// openTorn makes a torn-off note's window: where it was left, or the n-th
// step down and right of the middle of the screen.
func (s *Shell) openTorn(tw *tornWin, f canvas.Frame, n int) uint {
	b := Rect{X: f.ScreenX, Y: f.ScreenY, Width: f.W, Height: f.H}
	if !f.Placed() {
		a := s.cfg.Platform.WorkArea()
		b = Rect{
			X:      a.X + (a.Width-NoteWidth)/2 + n*noteCascade,
			Y:      a.Y + (a.Height-NoteHeight)/2 + n*noteCascade,
			Width:  NoteWidth,
			Height: NoteHeight,
		}
	}
	w := s.cfg.Platform.NewWindow(WindowOptions{
		Kind:      KindNote,
		Name:      "note:" + tw.p.path + "#" + tw.note,
		Title:     strings.TrimSuffix(path.Base(tw.note), ".md"),
		URL:       NoteURL(tw.note),
		Bounds:    b,
		Placed:    true,
		Frameless: true,
		Pinned:    f.Pinned,
	})
	id := w.ID()
	s.mu.Lock()
	tw.win = w
	if tw.p.closed { // the project's window went meanwhile: so does this one
		tw.quiet = true
		s.mu.Unlock()
		w.Close()
		return 0
	}
	s.tornByWin[id] = tw
	f.ScreenX, f.ScreenY, f.W, f.H = b.X, b.Y, b.Width, b.Height
	s.edit(tw.p, func(c canvas.Project) canvas.Project { return c.Tear(tw.note, f) })
	s.mu.Unlock()
	w.OnMoved(func() { s.tornMoved(tw) })
	w.OnClosing(func() { s.tornClosing(tw) })
	w.Show()
	w.Focus()
	return id
}

// frameOf is a torn-off note's frame now, keeping its pin. The caller
// holds mu.
func (s *Shell) frameOf(tw *tornWin, b Rect) (canvas.Frame, bool) {
	f, ok := s.state.Projects[tw.p.path].Torn[tw.note]
	if !ok || b.Width <= 0 || b.Height <= 0 {
		return f, false
	}
	if near(f.ScreenX, b.X) && near(f.ScreenY, b.Y) && near(f.W, b.Width) && near(f.H, b.Height) {
		return f, false
	}
	f.ScreenX, f.ScreenY, f.W, f.H = b.X, b.Y, b.Width, b.Height
	return f, true
}

// tornMoved keeps where a torn-off note's window is; it is written with
// the next save (a close, a quit).
func (s *Shell) tornMoved(tw *tornWin) {
	b := tw.win.Bounds()
	s.mu.Lock()
	defer s.mu.Unlock()
	if tw.closed || s.stopped {
		return
	}
	if f, ok := s.frameOf(tw, b); ok {
		s.edit(tw.p, func(c canvas.Project) canvas.Project { return c.Tear(tw.note, f) })
	}
}

// tornClosing is a torn-off note's window going. Closed by the person, the
// note is back on the board; closed by the app (its project's window
// closed, the app quit), its frame stays for the next time.
func (s *Shell) tornClosing(tw *tornWin) {
	b := tw.win.Bounds() // still there: this runs before the window is gone
	s.mu.Lock()
	if tw.closed {
		s.mu.Unlock()
		return
	}
	tw.closed = true
	delete(s.tornByWin, tw.win.ID())
	if tw.p.torn[tw.note] == tw {
		delete(tw.p.torn, tw.note)
	}
	keep := tw.quiet || s.stopped
	if keep {
		if f, ok := s.frameOf(tw, b); ok {
			s.edit(tw.p, func(c canvas.Project) canvas.Project { return c.Tear(tw.note, f) })
		}
	} else {
		s.edit(tw.p, func(c canvas.Project) canvas.Project { return c.PutBack(tw.note) })
	}
	s.mu.Unlock()
	s.save()
	if !keep {
		s.emitLayout(tw.p)
	}
}

// PutBack puts a torn-off note back on its board and closes its window.
// From a torn-off note's own window, note "" is that note.
func (s *Shell) PutBack(window uint, note string) error {
	s.mu.Lock()
	p, self := s.owner(window)
	if p == nil {
		s.mu.Unlock()
		return fmt.Errorf("no project window %d", window)
	}
	if note == "" && self != nil {
		note = self.note
	}
	s.edit(p, func(c canvas.Project) canvas.Project { return c.PutBack(note) })
	tw := p.torn[note]
	if tw != nil {
		tw.quiet = true // its frame is already forgotten
	}
	s.mu.Unlock()
	if tw != nil && tw.win != nil {
		tw.win.Close()
	}
	err := s.save()
	s.emitLayout(p)
	// the board comes forward, where the note now is
	s.mu.Lock()
	board, closed := p.win, p.closed
	s.mu.Unlock()
	if board != nil && !closed {
		board.Show()
		board.Focus()
	}
	return err
}

// reopenTorn opens again the notes a project had torn off when its window
// last closed.
func (s *Shell) reopenTorn(p *project) {
	s.mu.Lock()
	frames := s.state.Projects[p.path].Torn
	var notes []string
	for n := range frames {
		if p.torn[n] == nil {
			notes = append(notes, n)
		}
	}
	sort.Strings(notes)
	tws := make([]*tornWin, len(notes))
	for i, n := range notes {
		tws[i] = &tornWin{p: p, note: n}
		p.torn[n] = tws[i]
	}
	s.mu.Unlock()
	for i, tw := range tws {
		s.openTorn(tw, frames[tw.note], i)
	}
}

// closeTorn closes a project's torn-off notes, keeping their frames.
func (s *Shell) closeTorn(p *project) {
	s.mu.Lock()
	var wins []Window
	for _, tw := range p.torn {
		tw.quiet = true
		if tw.win != nil {
			wins = append(wins, tw.win)
		}
	}
	s.mu.Unlock()
	for _, w := range wins {
		w.Close()
	}
}

// windowIDs are the windows that show a project: its own and its torn-off
// notes'. The caller holds mu.
func (s *Shell) windowIDs(p *project) []uint {
	var ids []uint
	if p.win != nil {
		ids = append(ids, p.win.ID())
	}
	for _, tw := range p.torn {
		if tw.win != nil && !tw.closed {
			ids = append(ids, tw.win.ID())
		}
	}
	return ids
}

// noteNames are the notes a board.load answered with, pages and books
// included; ok is false when the answer is not a board.
func noteNames(result any) (names []string, ok bool) {
	raw, isRaw := result.(*json.RawMessage)
	if !isRaw || raw == nil {
		return nil, false
	}
	type note struct {
		Name  string `json:"name"`
		Pages []note `json:"pages"`
	}
	var b struct {
		Tabs *[]struct {
			Notes []note `json:"notes"`
		} `json:"tabs"`
	}
	if json.Unmarshal(*raw, &b) != nil || b.Tabs == nil {
		return nil, false
	}
	var walk func([]note)
	walk = func(ns []note) {
		for _, n := range ns {
			names = append(names, n.Name)
			walk(n.Pages)
		}
	}
	for _, t := range *b.Tabs {
		walk(t.Notes)
	}
	return names, true
}

// keepNotes forgets the places of notes a board.load no longer has, and
// closes the windows of the torn-off ones (archived, deleted, moved away).
func (s *Shell) keepNotes(p *project, result any) {
	names, ok := noteNames(result)
	if !ok {
		return
	}
	s.mu.Lock()
	cur := s.state.Projects[p.path]
	next, gone := cur.Keep(names)
	if len(next.Places) == len(cur.Places) && len(gone) == 0 {
		s.mu.Unlock()
		return
	}
	s.edit(p, func(canvas.Project) canvas.Project { return next })
	var wins []Window
	for _, n := range gone {
		if tw := p.torn[n]; tw != nil {
			tw.quiet = true
			if tw.win != nil {
				wins = append(wins, tw.win)
			}
		}
	}
	s.mu.Unlock()
	for _, w := range wins {
		w.Close()
	}
	s.save()
	if len(gone) > 0 {
		s.emitLayout(p)
	}
}

// Renamed moves a note's place and frame after note.move gave it a new
// name; its torn-off window, if any, now shows the new name.
func (s *Shell) Renamed(window uint, from, to string) {
	if from == to || to == "" {
		return
	}
	s.mu.Lock()
	p, _ := s.owner(window)
	if p == nil {
		s.mu.Unlock()
		return
	}
	s.edit(p, func(c canvas.Project) canvas.Project { return c.Rename(from, to) })
	var tid uint
	if tw := p.torn[from]; tw != nil {
		delete(p.torn, from)
		tw.note = to
		p.torn[to] = tw
		if tw.win != nil {
			tid = tw.win.ID()
		}
	}
	s.mu.Unlock()
	s.save()
	s.emitLayout(p)
	if tid != 0 {
		s.cfg.Platform.Emit(StatusEvent(tid), s.Status(tid))
	}
}

// pinTorn keeps a torn-off note's window on top, or lets others over it.
// The caller holds mu; pinTorn lets it go.
func (s *Shell) pinTorn(tw *tornWin, on bool) error {
	f, ok := s.state.Projects[tw.p.path].Torn[tw.note]
	if ok {
		f.Pinned = on
		s.edit(tw.p, func(c canvas.Project) canvas.Project { return c.Tear(tw.note, f) })
	}
	w := tw.win
	s.mu.Unlock()
	if w == nil {
		return nil
	}
	w.SetPinned(on)
	s.cfg.Platform.Emit(StatusEvent(w.ID()), s.Status(w.ID()))
	return s.save()
}
