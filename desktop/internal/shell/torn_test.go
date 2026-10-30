package shell

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sync"
	"testing"

	"github.com/LeeSwallow/stickypane/desktop/internal/bridgeclient"
	"github.com/LeeSwallow/stickypane/desktop/internal/canvas"
)

// loadBoard is a fake bridge whose board.load answers with the notes it
// is given.
type loadBoard struct {
	*fakeBoard
	mu   sync.Mutex
	load string
}

func (b *loadBoard) Call(ctx context.Context, method string, params, result any) error {
	b.fakeBoard.Call(ctx, method, params, result)
	b.mu.Lock()
	defer b.mu.Unlock()
	if r, ok := result.(*json.RawMessage); ok {
		switch method {
		case "board.load":
			if b.load != "" {
				*r = json.RawMessage(b.load)
			}
		}
	}
	return nil
}

func (b *loadBoard) answer(load string) {
	b.mu.Lock()
	b.load = load
	b.mu.Unlock()
}

const twoNotes = `{"title":"Alpha","tabs":[{"name":"work","notes":[{"name":"work/a.md"},{"name":"work/b.md"},{"name":"work/book/","pages":[{"name":"work/book/1.md"}]}]}]}`

// tornRig is a rig with a project open and ready, whose bridge loads the
// notes given.
func tornRig(t *testing.T) (*rig, *loadBoard, uint, string) {
	t.Helper()
	lb := &loadBoard{fakeBoard: newFakeBoard(), load: twoNotes}
	r := newRig(t, func(c *Config) {
		c.Start = func(context.Context, string, string) (Board, error) { return lb, nil }
	})
	path := filepath.Join(r.home, "alpha")
	id := r.s.OpenProject(path)
	waitFor(t, "ready", func() bool { return r.s.Status(id).State == StateReady })
	return r, lb, id, r.s.Status(id).Path
}

func (p *fakePlatform) noteWindows() []*fakeWindow {
	p.mu.Lock()
	defer p.mu.Unlock()
	var out []*fakeWindow
	for _, w := range p.windows {
		if w.opts.Kind == KindNote {
			out = append(out, w)
		}
	}
	return out
}

func (w *fakeWindow) isClosed() bool { w.mu.Lock(); defer w.mu.Unlock(); return w.closed }

// saved reads overlay.json as the next launch would.
func (r *rig) saved() State {
	r.t.Helper()
	st, err := LoadState(filepath.Join(r.home, StateFile))
	if err != nil {
		r.t.Fatal(err)
	}
	return st
}

func TestALayoutStartsAsACanvasWithSnap(t *testing.T) {
	r, _, id, _ := tornRig(t)
	l, err := r.s.Layout(id)
	if err != nil {
		t.Fatal(err)
	}
	if l.Mode != canvas.ModeCanvas || !l.Snap || l.Places == nil || l.Torn == nil {
		t.Errorf("layout = %+v, want canvas, snap, empty maps", l)
	}
	b, _ := json.Marshal(l)
	if string(b) != `{"mode":"canvas","snap":true,"places":{},"torn":{}}` {
		t.Errorf("json = %s", b)
	}
	if _, err := r.s.Layout(999); err == nil {
		t.Error("a window that is not a project's has a layout")
	}
}

func TestModeSnapAndPlacesAreKeptInOverlayJSON(t *testing.T) {
	r, _, id, path := tornRig(t)
	if err := r.s.SetMode(id, "grid"); err != nil {
		t.Fatal(err)
	}
	if err := r.s.SetMode(id, "tiles"); err == nil {
		t.Error("an unknown mode was taken")
	}
	if err := r.s.SetSnap(id, false); err != nil {
		t.Fatal(err)
	}
	if err := r.s.SetPlaces(id, map[string]canvas.Place{"work/a.md": {X: 40, Y: 80, W: 320, H: 240, Tilt: -1, Z: 3}}); err != nil {
		t.Fatal(err)
	}
	got := r.saved().Projects[path]
	want := canvas.Project{Mode: "grid", NoSnap: true, Places: map[string]canvas.Place{"work/a.md": {X: 40, Y: 80, W: 320, H: 240, Tilt: -1, Z: 3}}, Torn: map[string]canvas.Frame{}}
	if got.Mode != want.Mode || got.NoSnap != want.NoSnap || got.Places["work/a.md"] != want.Places["work/a.md"] {
		t.Errorf("saved %+v, want %+v", got, want)
	}
	l, _ := r.s.Layout(id)
	if l.Mode != "grid" || l.Snap {
		t.Errorf("layout = %+v", l)
	}
}

func TestTearOffOpensANoteWindowOnTop(t *testing.T) {
	r, _, id, path := tornRig(t)
	nid, err := r.s.TearOff(id, "work/a.md")
	if err != nil {
		t.Fatal(err)
	}
	ws := r.p.noteWindows()
	if len(ws) != 1 || ws[0].id != nid {
		t.Fatalf("note windows %v, id %d", ws, nid)
	}
	w := ws[0]
	if w.opts.URL != "/?window=note&note=work%2Fa.md" || !w.opts.Frameless || !w.opts.Placed || !w.pinned() || w.opts.Title != "a" {
		t.Errorf("options %+v pinned %v", w.opts, w.pinned())
	}
	// the middle of a 1600x1000 work area that starts 25 down
	if w.bounds != (Rect{X: 620, Y: 315, Width: NoteWidth, Height: NoteHeight}) {
		t.Errorf("bounds %+v", w.bounds)
	}
	if again, _ := r.s.TearOff(id, "work/a.md"); again != nid || len(r.p.noteWindows()) != 1 {
		t.Error("tearing off a torn note again made another window")
	}
	second, _ := r.s.TearOff(id, "work/b.md")
	if b := r.p.noteWindows()[1].bounds; b.X != 620+noteCascade || b.Y != 315+noteCascade || second == nid {
		t.Errorf("the second note opens one step on: %+v", b)
	}
	if f := r.saved().Projects[path].Torn["work/a.md"]; f != (canvas.Frame{ScreenX: 620, ScreenY: 315, W: NoteWidth, H: NoteHeight, Pinned: true}) {
		t.Errorf("saved frame %+v", f)
	}
	if len(r.p.emitted(CanvasEvent(id))) < 2 {
		t.Error("the board window was not told of the notes torn off")
	}
	if _, err := r.s.TearOff(id, " "); err == nil {
		t.Error("a note without a name was torn off")
	}
}

func TestATornNoteSpeaksForItsNote(t *testing.T) {
	r, lb, id, path := tornRig(t)
	nid, _ := r.s.TearOff(id, "work/a.md")
	st := r.s.Status(nid)
	if st.Window != nid || st.Note != "work/a.md" || st.Path != path || st.State != StateReady || !st.Pinned || st.Version != "9.9.9" {
		t.Errorf("status %+v", st)
	}
	var out json.RawMessage
	if err := r.s.Call(context.Background(), nid, "note.act", nil, &out); err != nil {
		t.Errorf("a torn note's window does not reach its bridge: %v", err)
	}
	if l, err := r.s.Layout(nid); err != nil || l.Torn["work/a.md"].W != NoteWidth {
		t.Errorf("a torn note's window reads its project's layout: %+v %v", l, err)
	}
	// its pin is its own
	if err := r.s.SetPin(nid, false); err != nil {
		t.Fatal(err)
	}
	w := r.p.noteWindows()[0]
	if w.pinned() || r.s.Status(nid).Pinned || r.s.Status(id).Pinned != false {
		t.Error("unpinning the torn note did not unpin it")
	}
	if r.saved().Projects[path].Torn["work/a.md"].Pinned {
		t.Error("the unpin was not saved")
	}
	// it hears what its board hears
	lb.notes <- bridgeclient.Notification{Method: "board.changed", Params: json.RawMessage(`{"n":1}`)}
	lb.notes <- bridgeclient.Notification{Method: bridgeclient.MethodRestarted}
	waitFor(t, "events to the torn note", func() bool {
		return len(r.p.emitted(ChangedEvent(nid))) == 1 && len(r.p.emitted(RestartedEvent(nid))) == 1 && len(r.p.emitted(ChangedEvent(id))) == 1
	})
}

func TestClosingATornNotePutsItBack(t *testing.T) {
	r, _, id, path := tornRig(t)
	r.s.TearOff(id, "work/a.md")
	w := r.p.noteWindows()[0]
	w.moveTo(Rect{X: 5, Y: 50, Width: 300, Height: 200})
	w.Close() // the person closed it
	if _, ok := r.saved().Projects[path].Torn["work/a.md"]; ok {
		t.Error("a note closed by the person stays torn off")
	}
	if l, _ := r.s.Layout(id); len(l.Torn) != 0 {
		t.Errorf("layout %+v", l)
	}
	ev := r.p.emitted(CanvasEvent(id))
	if last := ev[len(ev)-1].(Layout); len(last.Torn) != 0 {
		t.Errorf("the board was told %+v", last)
	}
	if st := r.s.Status(w.id); st.State != StateClosed {
		t.Errorf("a closed torn note's window still has a status: %+v", st)
	}
}

func TestPutBackClosesTheWindow(t *testing.T) {
	r, _, id, path := tornRig(t)
	nid, _ := r.s.TearOff(id, "work/a.md")
	r.s.TearOff(id, "work/b.md")
	// from its own window, "" is its own note
	if err := r.s.PutBack(nid, ""); err != nil {
		t.Fatal(err)
	}
	// from the board, by name
	if err := r.s.PutBack(id, "work/b.md"); err != nil {
		t.Fatal(err)
	}
	for _, w := range r.p.noteWindows() {
		if !w.isClosed() {
			t.Errorf("window %d is still open", w.id)
		}
	}
	if torn := r.saved().Projects[path].Torn; len(torn) != 0 {
		t.Errorf("still torn: %v", torn)
	}
}

func TestTornNotesCloseAndComeBackWithTheirBoard(t *testing.T) {
	r, _, id, path := tornRig(t)
	r.s.TearOff(id, "work/a.md")
	w := r.p.noteWindows()[0]
	w.moveTo(Rect{X: 900, Y: 60, Width: 280, Height: 330})
	r.p.projectWindows()[0].Close()
	if !w.isClosed() {
		t.Fatal("the torn note outlived its board's window")
	}
	if f := r.saved().Projects[path].Torn["work/a.md"]; f != (canvas.Frame{ScreenX: 900, ScreenY: 60, W: 280, H: 330, Pinned: true}) {
		t.Errorf("the frame kept is %+v", f)
	}
	id2 := r.s.OpenProject(path)
	ws := r.p.noteWindows()
	if len(ws) != 2 {
		t.Fatalf("%d note windows, want the torn note back", len(ws))
	}
	if b := ws[1].bounds; b != (Rect{X: 900, Y: 60, Width: 280, Height: 330}) {
		t.Errorf("it came back at %+v", b)
	}
	waitFor(t, "ready", func() bool { return r.s.Status(id2).State == StateReady })
	if st := r.s.Status(ws[1].id); st.Note != "work/a.md" || st.State != StateReady {
		t.Errorf("status %+v", st)
	}
}

func TestANoteGoneFromTheBoardClosesItsWindow(t *testing.T) {
	r, lb, id, path := tornRig(t)
	r.s.SetPlaces(id, map[string]canvas.Place{"work/a.md": {W: 200, H: 100}, "work/b.md": {W: 200, H: 100}, "work/book/1.md": {W: 200, H: 100}})
	r.s.TearOff(id, "work/a.md")
	r.s.TearOff(id, "work/book/1.md")
	var out json.RawMessage
	// a load that is not a board changes nothing
	lb.answer(`{"title":"Alpha"}`)
	r.s.Call(context.Background(), id, "board.load", nil, &out)
	if n := len(r.saved().Projects[path].Places); n != 3 {
		t.Fatalf("%d places after a load without tabs", n)
	}
	// a archived; the book's page is still there
	lb.answer(`{"title":"Alpha","tabs":[{"name":"work","notes":[{"name":"work/b.md"},{"name":"work/book/","pages":[{"name":"work/book/1.md"}]}]}]}`)
	if err := r.s.Call(context.Background(), id, "board.load", nil, &out); err != nil {
		t.Fatal(err)
	}
	ws := r.p.noteWindows()
	if !ws[0].isClosed() || ws[1].isClosed() {
		t.Errorf("closed: a %v, page %v; want a only", ws[0].isClosed(), ws[1].isClosed())
	}
	p := r.saved().Projects[path]
	if _, ok := p.Places["work/a.md"]; ok || len(p.Places) != 2 {
		t.Errorf("places %v", p.Places)
	}
	if _, ok := p.Torn["work/a.md"]; ok || len(p.Torn) != 1 {
		t.Errorf("torn %v", p.Torn)
	}
}

func TestAMovedNoteTakesItsPlaceAlong(t *testing.T) {
	r, _, id, path := tornRig(t)
	r.s.SetPlaces(id, map[string]canvas.Place{"work/a.md": {X: 30, W: 200, H: 100}})
	nid, _ := r.s.TearOff(id, "work/a.md")
	// ProjectService.Move reads the new name from note.move's answer and
	// tells the shell (the service needs a Wails window in its context)
	if got := movedTo(json.RawMessage(`{"note":"done/a.md"}`)); got != "done/a.md" {
		t.Fatalf("movedTo = %q", got)
	}
	if got := movedTo(json.RawMessage(`[]`)); got != "" {
		t.Errorf("movedTo of a list = %q", got)
	}
	r.s.Renamed(id, "work/a.md", "done/a.md")
	p := r.saved().Projects[path]
	if p.Places["done/a.md"].X != 30 || p.Torn["done/a.md"].W != NoteWidth {
		t.Errorf("after the move: %+v", p)
	}
	if st := r.s.Status(nid); st.Note != "done/a.md" {
		t.Errorf("the torn note shows %q", st.Note)
	}
	if len(r.p.emitted(StatusEvent(nid))) == 0 {
		t.Error("the torn note was not told its new name")
	}
}

func TestQuittingKeepsTornNotesForTheNextLaunch(t *testing.T) {
	r, _, id, path := tornRig(t)
	r.s.TearOff(id, "work/a.md")
	w := r.p.noteWindows()[0]
	w.mu.Lock()
	w.bounds = Rect{X: 11, Y: 22, Width: 333, Height: 244} // moved, not yet told
	w.mu.Unlock()
	r.s.Shutdown()
	w.Close() // the app closes its windows as it goes
	if f := r.saved().Projects[path].Torn["work/a.md"]; f != (canvas.Frame{ScreenX: 11, ScreenY: 22, W: 333, H: 244, Pinned: true}) {
		t.Errorf("after quitting the frame is %+v", f)
	}
}

func (w *fakeWindow) blur() { w.mu.Lock(); w.focused = false; w.mu.Unlock() }

// Tearing a note off moves the focus to its window; the board it came
// from stays on screen beside it, so the note is seen to leave the board
// rather than the board to vanish (#62). Focus going anywhere else still
// hides an unpinned board.
func TestTearingOffLeavesTheBoardShown(t *testing.T) {
	r, _, id, _ := tornRig(t)
	board := r.p.projectWindows()[0]
	if _, err := r.s.TearOff(id, "work/a.md"); err != nil {
		t.Fatal(err)
	}
	board.loseFocus() // to the torn-off note
	if !board.Visible() {
		t.Fatal("the board hid when its own torn-off note took the focus")
	}
	r.p.noteWindows()[0].blur()
	board.loseFocus() // to another app
	if board.Visible() {
		t.Error("an unpinned board stays when the focus leaves the app's windows of it")
	}
}

// Putting a note back shows its board, where the note now is.
func TestPutBackShowsTheBoard(t *testing.T) {
	r, _, id, _ := tornRig(t)
	board := r.p.projectWindows()[0]
	nid, _ := r.s.TearOff(id, "work/a.md")
	board.Hide()
	if err := r.s.PutBack(nid, ""); err != nil {
		t.Fatal(err)
	}
	if !board.Visible() || !board.Focused() {
		t.Error("putting a note back leaves its board hidden")
	}
}
