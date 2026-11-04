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
	"strings"

	"github.com/LeeSwallow/stickypane/desktop/internal/canvas"
)

// The first size of a torn-off note's window, and how far each next one
// opens from the one before.
const (
	NoteWidth   = 360
	NoteHeight  = 420
	noteCascade = 28
	noteGap     = 16
)

// NoteURL is the page of a torn-off note's window; fit asks it to take
// its note's height, the first time it opens.
func NoteURL(note string, fit bool) string {
	u := pageNote + "&note=" + url.QueryEscape(note)
	if fit {
		u += "&fit=1"
	}
	return u
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
		return Layout{}, noWindow(window)
	}
	return layoutOf(s.state.Projects[p.path]), nil
}

// change edits the layout of a window's project and saves it.
func (s *Shell) change(window uint, f func(canvas.Project) canvas.Project) error {
	s.mu.Lock()
	p, _ := s.owner(window)
	if p == nil {
		s.mu.Unlock()
		return noWindow(window)
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
		return 0, ErrNoNote
	}
	s.mu.Lock()
	p, _ := s.owner(window)
	if p == nil || p.closed {
		s.mu.Unlock()
		return 0, noWindow(window)
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
	id := s.openTorn(tw, f, cascade(s.cfg.Platform.WorkArea(), n))
	s.save()
	s.emitLayout(p)
	return id, nil
}

// TearOffAll opens each of the notes given, a tab's, in a window of its
// own: the ones never torn off before lie side by side across the screen,
// the others go back where they were left. Notes already torn off stay as
// they are. It returns the windows it opened.
func (s *Shell) TearOffAll(window uint, notes []string) ([]uint, error) {
	s.mu.Lock()
	p, _ := s.owner(window)
	if p == nil || p.closed {
		s.mu.Unlock()
		return nil, noWindow(window)
	}
	type tear struct {
		tw *tornWin
		f  canvas.Frame
	}
	var tears []tear
	fresh := 0
	for _, note := range notes {
		if strings.TrimSpace(note) == "" || p.torn[note] != nil {
			continue
		}
		f, ok := s.state.Projects[p.path].Torn[note]
		if !ok {
			f = canvas.Frame{Pinned: true}
		}
		if !f.Placed() {
			fresh++
		}
		tw := &tornWin{p: p, note: note}
		p.torn[note] = tw
		tears = append(tears, tear{tw, f})
	}
	s.mu.Unlock()
	if len(tears) == 0 {
		return []uint{}, nil
	}
	slots := tile(s.cfg.Platform.WorkArea(), fresh)
	ids := []uint{}
	i := 0
	for _, t := range tears {
		var at Rect
		if !t.f.Placed() {
			at = slots[i]
			i++
		}
		if id := s.openTorn(t.tw, t.f, at); id != 0 {
			ids = append(ids, id)
		}
	}
	err := s.save()
	s.emitLayout(p)
	return ids, err
}

// PutBackAll puts the notes given back on their board, closing their
// windows; for a note not torn off it changes nothing.
func (s *Shell) PutBackAll(window uint, notes []string) error {
	s.mu.Lock()
	p, _ := s.owner(window)
	if p == nil {
		s.mu.Unlock()
		return noWindow(window)
	}
	var wins []Window
	for _, note := range notes {
		s.edit(p, func(c canvas.Project) canvas.Project { return c.PutBack(note) })
		if tw := p.torn[note]; tw != nil {
			tw.quiet = true // its frame is already forgotten
			if tw.win != nil {
				wins = append(wins, tw.win)
			}
		}
	}
	board, closed := p.win, p.closed
	s.mu.Unlock()
	for _, w := range wins {
		w.Close()
	}
	err := s.save()
	s.emitLayout(p)
	if board != nil && !closed {
		board.Show()
		board.Focus()
	}
	return err
}

// cascade is the n-th step down and right of the middle of the work area,
// where a note torn off by itself opens.
func cascade(a Rect, n int) Rect {
	return Rect{
		X:      a.X + (a.Width-NoteWidth)/2 + n*noteCascade,
		Y:      a.Y + (a.Height-NoteHeight)/2 + n*noteCascade,
		Width:  NoteWidth,
		Height: NoteHeight,
	}
}

// tile lays n note windows out side by side over the work area, in rows
// from the top left, the block of them in the middle. When more come than
// the screen holds, the next round lies one cascade step on.
func tile(a Rect, n int) []Rect {
	if n <= 0 {
		return nil
	}
	cols := max(1, (a.Width-noteGap)/(NoteWidth+noteGap))
	rows := max(1, (a.Height-noteGap)/(NoteHeight+noteGap))
	used := min(n, cols)
	usedRows := min(rows, (n+cols-1)/cols)
	x0 := a.X + (a.Width-used*NoteWidth-(used-1)*noteGap)/2
	y0 := a.Y + (a.Height-usedRows*NoteHeight-(usedRows-1)*noteGap)/2
	out := make([]Rect, n)
	for i := range out {
		k := i % (cols * rows)
		step := i / (cols * rows) * noteCascade
		out[i] = Rect{
			X:      x0 + k%cols*(NoteWidth+noteGap) + step,
			Y:      y0 + k/cols*(NoteHeight+noteGap) + step,
			Width:  NoteWidth,
			Height: NoteHeight,
		}
	}
	return out
}

// openTorn makes a torn-off note's window: where it was left, or at the
// place given.
func (s *Shell) openTorn(tw *tornWin, f canvas.Frame, at Rect) uint {
	b := Rect{X: f.ScreenX, Y: f.ScreenY, Width: f.W, Height: f.H}
	if !f.Placed() {
		b = at
	}
	w := s.cfg.Platform.NewWindow(WindowOptions{
		Kind:      KindNote,
		Name:      prefixNote + tw.p.path + "#" + tw.note,
		Title:     strings.TrimSuffix(path.Base(tw.note), ".md"),
		URL:       NoteURL(tw.note, !f.Placed()),
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
		return noWindow(window)
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
	a := s.cfg.Platform.WorkArea()
	for i, tw := range tws {
		s.openTorn(tw, frames[tw.note], cascade(a, i))
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
