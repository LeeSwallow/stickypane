package shell

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/LeeSwallow/stickypane/desktop/internal/canvas"
	"github.com/LeeSwallow/stickypane/desktop/internal/index"
	installsvc "github.com/LeeSwallow/stickypane/desktop/internal/install/service"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// HomeService is what the home window calls (bound to the frontend).
type HomeService struct{ S *Shell }

// Ensure makes ~/.stickypane (or $STICKYPANE_HOME) and returns its path.
func (h *HomeService) Ensure() (string, error) { return h.S.Ensure() }

// Sessions is the home list.
func (h *HomeService) Sessions() (index.Groups, error) { return h.S.Sessions() }

// Restore brings back a project that was tidied away.
func (h *HomeService) Restore(id int64) error { return h.S.Restore(id) }

// PickFolder asks for a folder; "" when the person cancelled.
func (h *HomeService) PickFolder() (string, error) { return h.S.PickFolder() }

// Inspect says whether dir already has a board; it changes nothing.
func (h *HomeService) Inspect(dir string) Inspection { return h.S.Inspect(dir) }

// InitProject makes a board in dir and opens its window.
func (h *HomeService) InitProject(dir string) error {
	if err := h.S.InitProject(dir); err != nil {
		return err
	}
	h.S.OpenProject(dir)
	return nil
}

// OpenProject opens (or brings forward) the window of a project.
func (h *HomeService) OpenProject(path string) uint { return h.S.OpenProject(path) }

// StickypaneStatus says whether stickypane was found, and its version.
func (h *HomeService) StickypaneStatus() StickypaneState { return h.S.StickypaneStatus() }

// Notices are the lines the home window shows (a hotkey that could not be
// registered, say).
func (h *HomeService) Notices() []Notice { return h.S.Notices() }

// Quit ends the app (the title bar's menu: "Quit stickypane").
func (h *HomeService) Quit() { h.S.Quit() }

// Prefs and SetPrefs are the home window's settings in overlay.json.
func (h *HomeService) Prefs() Prefs           { return h.S.Prefs() }
func (h *HomeService) SetPrefs(p Prefs) error { return h.S.SetPrefs(p) }

// The parameters of the bridge's methods (docs/desktop/10 §10.7), mirrored
// from the main module's internal/surface/bridge, which this one cannot import.
type (
	ActParams struct {
		Note    string          `json:"note"`
		Version string          `json:"version"`
		Action  json.RawMessage `json:"action"` // a widget.Action, the note kind's own
	}
	// CreateParams is note.create. The app sends a shape's create options
	// (kinds.list names them) and never a body: the shape's own Go code
	// writes the file (RA12). Body stays for a caller that has the text.
	CreateParams struct {
		Tab     string                     `json:"tab"`
		Kind    string                     `json:"kind"`
		Title   string                     `json:"title"`
		Body    string                     `json:"body,omitempty"`
		Options map[string]json.RawMessage `json:"options,omitempty"`
	}
	ArrangeParams struct {
		Note  string  `json:"note"`
		Open  *bool   `json:"open,omitempty"`
		Size  *string `json:"size,omitempty"`
		Rows  *int    `json:"rows,omitempty"`
		Pin   *bool   `json:"pin,omitempty"`
		Color *string `json:"color,omitempty"`
		Title *string `json:"title,omitempty"`
	}
	OrderParams struct {
		Tab   string   `json:"tab"`
		Order []string `json:"order"`
	}
	MoveParams struct {
		Note string `json:"note"`
		To   string `json:"to"`
	}
	SendParams struct {
		Note    string          `json:"note"`
		Request json.RawMessage `json:"request"` // its title or part, or its number from 1
	}
	SettingsParams struct {
		Theme    *string `json:"theme,omitempty"`
		Style    *string `json:"style,omitempty"`
		View     *string `json:"view,omitempty"`
		Language *string `json:"language,omitempty"`
	}
	noteParams struct {
		Note string `json:"note"`
	}
)

// ProjectService is what a project window calls. Each method acts on the
// board of the window that called it. Answers are the bridge's JSON as it
// is; the frontend reads them with its own types.
type ProjectService struct{ S *Shell }

// Quit ends the app (the title bar's menu: "Quit stickypane").
func (p *ProjectService) Quit() { p.S.Quit() }

// window is the id of the window a call came from.
func window(ctx context.Context) (uint, error) {
	if w, ok := ctx.Value(application.WindowKey).(application.Window); ok && w != nil {
		return w.ID(), nil
	}
	return 0, errors.New("no window made this call")
}

func (p *ProjectService) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	id, err := window(ctx)
	if err != nil {
		return nil, err
	}
	var out json.RawMessage
	err = p.S.Call(ctx, id, method, params, &out)
	return out, err
}

// Status is the window's project, its bridge's state and its pin.
func (p *ProjectService) Status(ctx context.Context) (Status, error) {
	id, err := window(ctx)
	if err != nil {
		return Status{}, err
	}
	return p.S.Status(id), nil
}

// Pin keeps the window on top and shown when it loses focus.
func (p *ProjectService) Pin(ctx context.Context, on bool) error {
	id, err := window(ctx)
	if err != nil {
		return err
	}
	return p.S.SetPin(id, on)
}

// MakeBoard makes a board in the window's folder, which had none, with
// the app's own stickypane init, and opens it.
func (p *ProjectService) MakeBoard(ctx context.Context) error {
	id, err := window(ctx)
	if err != nil {
		return err
	}
	return p.S.MakeBoard(id)
}

// InitHere runs stickypane init again in the window's project (the agent
// guide, after connecting the agents) and answers what it printed.
func (p *ProjectService) InitHere(ctx context.Context) (installsvc.Result, error) {
	id, err := window(ctx)
	if err != nil {
		return installsvc.Result{}, err
	}
	return p.S.InitHere(id)
}

// Load is board.load.
func (p *ProjectService) Load(ctx context.Context) (json.RawMessage, error) {
	return p.call(ctx, "board.load", nil)
}

// Act is note.act.
func (p *ProjectService) Act(ctx context.Context, a ActParams) (json.RawMessage, error) {
	return p.call(ctx, "note.act", a)
}

// Arrange is note.arrange: open, size, rows, pin, colour, title.
func (p *ProjectService) Arrange(ctx context.Context, a ArrangeParams) (json.RawMessage, error) {
	return p.call(ctx, "note.arrange", a)
}

// Order is tab.order.
func (p *ProjectService) Order(ctx context.Context, o OrderParams) (json.RawMessage, error) {
	return p.call(ctx, "tab.order", o)
}

// Move is note.move. The note's place on the canvas, and its torn-off
// window, go with it to its new name.
func (p *ProjectService) Move(ctx context.Context, m MoveParams) (json.RawMessage, error) {
	out, err := p.call(ctx, "note.move", m)
	if err != nil {
		return out, err
	}
	if to := movedTo(out); to != "" {
		if id, werr := window(ctx); werr == nil {
			p.S.Renamed(id, m.Note, to)
		}
	}
	return out, nil
}

// movedTo is the new name note.move answered with, or "".
func movedTo(out json.RawMessage) string {
	var moved struct {
		Note string `json:"note"`
	}
	if json.Unmarshal(out, &moved) != nil {
		return ""
	}
	return moved.Note
}

// Archive is note.archive.
func (p *ProjectService) Archive(ctx context.Context, note string) (json.RawMessage, error) {
	return p.call(ctx, "note.archive", noteParams{note})
}

// Trash is note.trash.
func (p *ProjectService) Trash(ctx context.Context, note string) (json.RawMessage, error) {
	return p.call(ctx, "note.trash", noteParams{note})
}

// Restore is note.restore.
func (p *ProjectService) Restore(ctx context.Context, note string) (json.RawMessage, error) {
	return p.call(ctx, "note.restore", noteParams{note})
}

// Create is note.create.
func (p *ProjectService) Create(ctx context.Context, c CreateParams) (json.RawMessage, error) {
	return p.call(ctx, "note.create", c)
}

// Run is note.run: it answers when the script is done.
func (p *ProjectService) Run(ctx context.Context, note string) (json.RawMessage, error) {
	return p.call(ctx, "note.run", noteParams{note})
}

// Send is note.send: it answers when the request is done.
func (p *ProjectService) Send(ctx context.Context, s SendParams) (json.RawMessage, error) {
	return p.call(ctx, "note.send", s)
}

// Open is note.open: the note in the person's editor.
func (p *ProjectService) Open(ctx context.Context, note string) (json.RawMessage, error) {
	return p.call(ctx, "note.open", noteParams{note})
}

// Kinds is kinds.list, with the examples in language: the language the
// window draws in ("ko", "en"). "" asks for the board's.
func (p *ProjectService) Kinds(ctx context.Context, language string) (json.RawMessage, error) {
	if language == "" {
		return p.call(ctx, "kinds.list", nil)
	}
	return p.call(ctx, "kinds.list", map[string]string{"language": language})
}

// Themes is themes.list: every theme's palette.
func (p *ProjectService) Themes(ctx context.Context) (json.RawMessage, error) {
	return p.call(ctx, "themes.list", nil)
}

// Index is board.index.
func (p *ProjectService) Index(ctx context.Context) (json.RawMessage, error) {
	return p.call(ctx, "board.index", nil)
}

// Settings is settings.get.
func (p *ProjectService) Settings(ctx context.Context) (json.RawMessage, error) {
	return p.call(ctx, "settings.get", nil)
}

// SetSetting is settings.set.
func (p *ProjectService) SetSetting(ctx context.Context, s SettingsParams) (json.RawMessage, error) {
	return p.call(ctx, "settings.set", s)
}

// CanvasService is what a project window, and a torn-off note's window,
// call for where notes sit on this computer (docs/desktop/03 §3.6.1): the
// canvas's layout and the notes torn off it. Each acts on the project of
// the window that called it.
type CanvasService struct{ S *Shell }

// Layout is the project's canvas: its mode, snap, places and torn-off notes.
func (c *CanvasService) Layout(ctx context.Context) (Layout, error) {
	id, err := window(ctx)
	if err != nil {
		return Layout{}, err
	}
	return c.S.Layout(id)
}

// SetMode is canvas or grid.
func (c *CanvasService) SetMode(ctx context.Context, mode string) error {
	id, err := window(ctx)
	if err != nil {
		return err
	}
	return c.S.SetMode(id, mode)
}

// SetSnap makes notes land on the canvas's grid, or anywhere.
func (c *CanvasService) SetSnap(ctx context.Context, on bool) error {
	id, err := window(ctx)
	if err != nil {
		return err
	}
	return c.S.SetSnap(id, on)
}

// Place keeps where notes are on the canvas, by note name.
func (c *CanvasService) Place(ctx context.Context, places map[string]canvas.Place) error {
	id, err := window(ctx)
	if err != nil {
		return err
	}
	return c.S.SetPlaces(id, places)
}

// TearOff opens a note in a window of its own and returns its id.
func (c *CanvasService) TearOff(ctx context.Context, note string) (uint, error) {
	id, err := window(ctx)
	if err != nil {
		return 0, err
	}
	return c.S.TearOff(id, note)
}

// PutBack puts a torn-off note back on the board and closes its window;
// from that window, note "" is its own.
func (c *CanvasService) PutBack(ctx context.Context, note string) error {
	id, err := window(ctx)
	if err != nil {
		return err
	}
	return c.S.PutBack(id, note)
}
