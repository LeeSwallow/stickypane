package shell

import (
	"net/url"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// The kinds of window.
const (
	KindHome    = "home"
	KindProject = "project"
	KindNote    = "note" // a note torn off its board (torn.go)
)

// The smallest a torn-off note's window gets.
const (
	NoteMinWidth  = 200
	NoteMinHeight = 140
)

// How a frameless window gets its close, minimise and zoom buttons
// (docs/overlay/03 §3.11.6). The page reads it from ?chrome= to leave room
// for them or to draw its own.
const (
	ChromeMac = "mac" // macOS: the native traffic lights over our title bar (hidden-inset)
	ChromeOwn = "own" // elsewhere: the title bar draws close and minimise itself
)

// ChromeFor is the chrome of frameless windows on an OS (runtime.GOOS).
func ChromeFor(goos string) string {
	if goos == "darwin" {
		return ChromeMac
	}
	return ChromeOwn
}

// WithChrome adds chrome=<c> to a window's page URL.
func WithChrome(page, chrome string) string {
	u, err := url.Parse(page)
	if err != nil {
		return page
	}
	q := u.Query()
	q.Set("chrome", chrome)
	u.RawQuery = q.Encode()
	return u.String()
}

// AppMenuRoles are the macOS menu bar's menus: the app menu carries
// "Quit" with Cmd+Q; Edit makes copy and paste work in text fields.
var AppMenuRoles = []application.Role{application.AppMenu, application.EditMenu, application.WindowMenu}

// AppMenu is the menu bar of the app.
func AppMenu() *application.Menu {
	m := application.NewMenu()
	for _, r := range AppMenuRoles {
		m.AddRole(r)
	}
	return m
}

// Rect is a window's place and size, in the screen's points.
type Rect struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// WindowOptions is what a new window needs. Placed false centres it.
type WindowOptions struct {
	Kind      string
	Name      string
	Title     string
	URL       string
	Bounds    Rect
	Placed    bool
	Frameless bool
	Pinned    bool
	Hidden    bool
}

// Window is one window on the desktop. Its callbacks run on their own
// goroutines, never on the UI thread.
type Window interface {
	ID() uint
	Show()
	Hide()
	Focus()
	Close()
	Visible() bool
	Focused() bool
	SetPinned(on bool) // always on top
	Bounds() Rect
	OnLostFocus(func())
	OnFocus(func())
	OnMoved(func())   // moved or resized
	OnClosing(func()) // the window is about to go, for good
}

// Platform is the desktop: windows, the hotkey, events to the frontend,
// the tray and dialogs.
type Platform interface {
	NewWindow(o WindowOptions) Window
	WorkArea() Rect // of the primary screen
	RegisterHotkey(accel string, fn func()) error
	UnregisterHotkey(accel string)
	Emit(name string, data any)
	SetTray(t Tray)
	PickFolder() (string, error)
	Quit()
}

// Wails is the Platform of a Wails application.
type Wails struct {
	App  *application.App
	tray *wailsTray
}

// NewWails puts app behind Platform. The tray is made by StartTray once
// the app runs.
func NewWails(app *application.App) *Wails { return &Wails{App: app} }

func (p *Wails) NewWindow(o WindowOptions) Window {
	opts := application.WebviewWindowOptions{
		Name:      o.Name,
		Title:     o.Title,
		URL:       o.URL,
		Width:     o.Bounds.Width,
		Height:    o.Bounds.Height,
		Frameless: o.Frameless,
		Hidden:    o.Hidden,
		// Pin is AlwaysOnTop; hiding on focus loss is the shell's, so that
		// a pin can turn it off (Wails' HideOnFocusLost is fixed at creation).
		AlwaysOnTop: o.Pinned,
	}
	if o.Kind == KindNote {
		// A torn-off note is a bare sticky: no title bar or traffic lights
		// on any system; the page draws its own pin and put-back.
		opts.URL = WithChrome(o.URL, ChromeOwn)
		opts.MinWidth, opts.MinHeight = NoteMinWidth, NoteMinHeight
	} else if o.Frameless {
		chrome := ChromeFor(runtime.GOOS)
		opts.URL = WithChrome(o.URL, chrome)
		if chrome == ChromeMac {
			// A titled window with a transparent, hidden-inset title bar: the
			// traffic lights stay, the page draws everything else and is the
			// drag region (--wails-draggable).
			opts.Frameless = false
			opts.Mac.TitleBar = application.MacTitleBarHiddenInset
		}
	}
	if o.Placed {
		opts.InitialPosition = application.WindowXY
		opts.X, opts.Y = o.Bounds.X, o.Bounds.Y
	} else {
		opts.InitialPosition = application.WindowCentered
	}
	w := p.App.Window.NewWithOptions(opts)
	ww := &wailsWindow{w: w}
	if o.Kind == KindHome {
		// The home window hides instead of closing: the app lives on in the
		// tray, and the window comes back from there.
		w.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
			e.Cancel()
			w.Hide()
		})
	}
	return ww
}

func (p *Wails) WorkArea() Rect {
	s := p.App.Screen.GetPrimary()
	if s == nil {
		return Rect{Width: 1440, Height: 900}
	}
	a := s.WorkArea
	return Rect{X: a.X, Y: a.Y, Width: a.Width, Height: a.Height}
}

func (p *Wails) RegisterHotkey(accel string, fn func()) error {
	return p.App.GlobalShortcut.Register(accel, fn)
}

func (p *Wails) UnregisterHotkey(accel string) { p.App.GlobalShortcut.Unregister(accel) }

func (p *Wails) Emit(name string, data any) { p.App.Event.Emit(name, data) }

func (p *Wails) PickFolder() (string, error) {
	return p.App.Dialog.OpenFile().
		CanChooseDirectories(true).
		CanChooseFiles(false).
		CanCreateDirectories(true).
		PromptForSingleSelection()
}

func (p *Wails) Quit() { p.App.Quit() }

// wailsWindow is a Wails window behind Window.
type wailsWindow struct{ w *application.WebviewWindow }

func (x *wailsWindow) ID() uint      { return x.w.ID() }
func (x *wailsWindow) Show()         { x.w.Show() }
func (x *wailsWindow) Hide()         { x.w.Hide() }
func (x *wailsWindow) Focus()        { x.w.Focus() }
func (x *wailsWindow) Close()        { x.w.Close() }
func (x *wailsWindow) Visible() bool { return x.w.IsVisible() }
func (x *wailsWindow) Focused() bool { return x.w.IsFocused() }
func (x *wailsWindow) SetPinned(on bool) {
	x.w.SetAlwaysOnTop(on)
}
func (x *wailsWindow) Bounds() Rect {
	b := x.w.Bounds()
	return Rect{X: b.X, Y: b.Y, Width: b.Width, Height: b.Height}
}
func (x *wailsWindow) on(t events.WindowEventType, f func()) {
	x.w.OnWindowEvent(t, func(*application.WindowEvent) { f() })
}
func (x *wailsWindow) OnLostFocus(f func()) { x.on(events.Common.WindowLostFocus, f) }
func (x *wailsWindow) OnFocus(f func())     { x.on(events.Common.WindowFocus, f) }
func (x *wailsWindow) OnMoved(f func()) {
	x.on(events.Common.WindowDidMove, f)
	x.on(events.Common.WindowDidResize, f)
}

// OnClosing runs f as a hook, before the window is gone, so that its
// place can still be read.
func (x *wailsWindow) OnClosing(f func()) {
	x.w.RegisterHook(events.Common.WindowClosing, func(*application.WindowEvent) { f() })
}
