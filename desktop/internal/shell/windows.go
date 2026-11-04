package shell

import (
	"net/url"
	"runtime"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// The kinds of window.
const (
	KindHome    = "home"
	KindProject = "project"
	KindNote    = "note" // a note torn off its board (torn.go)
)

// The page each kind of window opens: the frontend tells them apart by
// ?window= (src/lib/shell.ts windowKind), so these are its words too.
const (
	pageHome     = "/"
	pageProject  = "/?window=project"
	pageNote     = "/?window=note"     // &note=<name>[&fit=1]
	pageSettings = "/?window=settings" // &section=<section>
)

// The names Wails keeps the windows by. A board's adds its project's key,
// a torn-off note's the key and the note.
const (
	nameHome      = "home"
	nameSettings  = "settings"
	prefixProject = "project:"
	prefixNote    = "note:"
)

// fallbackArea is the work area taken when the desktop names no screen.
var fallbackArea = Rect{Width: 1440, Height: 900}

// The smallest a torn-off note's window gets.
const (
	NoteMinWidth  = 200
	NoteMinHeight = 140
)

// How a frameless window gets its close, minimise and zoom buttons
// (docs/desktop/03 §3.11.6). The page reads it from ?chrome= to leave room
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
func WithChrome(page, chrome string) string { return withQuery(page, "chrome", chrome) }

// WithSystemLanguage adds syslang=<lang> to a window's page URL: what
// "auto" comes to here (SystemLanguage), so that a window set to auto
// speaks what the tray, the menu bar and the board's bridge speak.
func WithSystemLanguage(page, lang string) string { return withQuery(page, "syslang", lang) }

func withQuery(page, key, value string) string {
	u, err := url.Parse(page)
	if err != nil {
		return page
	}
	q := u.Query()
	q.Set(key, value)
	u.RawQuery = q.Encode()
	return u.String()
}

// AppMenuRoles are the macOS menu bar's menus: the app menu carries
// "Quit" with Cmd+Q; Edit makes copy and paste work in text fields.
var AppMenuRoles = []application.Role{application.AppMenu, application.EditMenu, application.WindowMenu}

// AppMenu is the menu bar of the app, in lang ("ko" or "en").
func AppMenu(lang string) *application.Menu {
	m := application.NewMenu()
	for _, r := range AppMenuRoles {
		m.AddRole(r)
	}
	if lang == "ko" {
		relabel(m, menuKo)
	}
	return m
}

// relabel gives every item of m, and of its submenus, the label to has
// for its English one.
func relabel(m *application.Menu, to map[string]string) {
	for i := 0; ; i++ {
		it := m.ItemAt(i)
		if it == nil {
			return
		}
		if l, ok := to[it.Label()]; ok {
			it.SetLabel(l)
		}
		if sub := it.GetSubmenu(); sub != nil {
			relabel(sub, to)
		}
	}
}

// menuKo is the menu bar in Korean, in macOS's own words: Wails writes
// its role items in English.
var menuKo = map[string]string{
	"About stickypane":      "stickypane에 관하여",
	"Services":              "서비스",
	"Hide stickypane":       "stickypane 가리기",
	"Hide Others":           "기타 가리기",
	"Show All":              "모두 보기",
	"Quit stickypane":       "stickypane 종료",
	"Edit":                  "편집",
	"Undo":                  "실행 취소",
	"Redo":                  "실행 복귀",
	"Cut":                   "오려두기",
	"Copy":                  "복사하기",
	"Paste":                 "붙여넣기",
	"Paste and Match Style": "붙여넣기 및 스타일 일치",
	"Delete":                "삭제",
	"Select All":            "전체 선택",
	"Speech":                "말하기",
	"Start Speaking":        "말하기 시작",
	"Stop Speaking":         "말하기 멈추기",
	"Window":                "윈도우",
	"Minimize":              "최소화",
	"Zoom":                  "확대/축소",
	"Bring All to Front":    "모두 앞으로 가져오기",
	"Close":                 "닫기",
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

// raise shows a window and gives it the focus.
func raise(w Window) {
	w.Show()
	w.Focus()
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
	// SetLanguage says the app's own words — the menu bar, the folder
	// picker — in lang ("ko" or "en").
	SetLanguage(lang string)
	PickFolder() (string, error)
	Quit()
}

// Wails is the Platform of a Wails application.
type Wails struct {
	App  *application.App
	tray *wailsTray

	mu   sync.Mutex
	lang string
}

// NewWails puts app behind Platform. The tray is made by StartTray once
// the app runs.
func NewWails(app *application.App) *Wails { return &Wails{App: app} }

func (p *Wails) NewWindow(o WindowOptions) Window {
	o.URL = WithSystemLanguage(o.URL, SystemLanguage())
	w := p.App.Window.NewWithOptions(webviewOptions(o, runtime.GOOS))
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

// webviewOptions are the Wails options of a window on goos: the place,
// size, page and chrome WindowOptions ask for.
func webviewOptions(o WindowOptions, goos string) application.WebviewWindowOptions {
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
		if o.Kind == KindSettings {
			opts.MinWidth, opts.MinHeight = SettingsMinWidth, SettingsMinHeight
		}
		chrome := ChromeFor(goos)
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
	return opts
}

func (p *Wails) WorkArea() Rect {
	s := p.App.Screen.GetPrimary()
	if s == nil {
		return fallbackArea
	}
	a := s.WorkArea
	return Rect{X: a.X, Y: a.Y, Width: a.Width, Height: a.Height}
}

func (p *Wails) RegisterHotkey(accel string, fn func()) error {
	return p.App.GlobalShortcut.Register(accel, fn)
}

func (p *Wails) UnregisterHotkey(accel string) { p.App.GlobalShortcut.Unregister(accel) }

func (p *Wails) Emit(name string, data any) { p.App.Event.Emit(name, data) }

func (p *Wails) SetLanguage(lang string) {
	p.mu.Lock()
	same := p.lang == lang
	p.lang = lang
	p.mu.Unlock()
	if same {
		return
	}
	// the menu bar is the main thread's; this may be called from any other
	application.InvokeAsync(func() { p.App.Menu.Set(AppMenu(lang)) })
}

// StartLanguage sets the menu bar in lang before the app runs.
func (p *Wails) StartLanguage(lang string) {
	p.mu.Lock()
	p.lang = lang
	p.mu.Unlock()
	p.App.Menu.Set(AppMenu(lang))
}

func (p *Wails) PickFolder() (string, error) {
	p.mu.Lock()
	lang := p.lang
	p.mu.Unlock()
	message, button := pickWords(lang)
	return p.App.Dialog.OpenFile().
		CanChooseDirectories(true).
		CanChooseFiles(false).
		CanCreateDirectories(true).
		SetMessage(message).
		SetButtonText(button).
		PromptForSingleSelection()
}

// pickWords are the folder picker's message and button, in lang.
func pickWords(lang string) (message, button string) {
	return words(lang, "Pick the project folder for the board", "보드를 둘 프로젝트 폴더를 골라 주세요"),
		words(lang, "Choose", "선택")
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

// wailsAutostart is Wails' AutostartManager behind Autostart.
type wailsAutostart struct{ m *application.AutostartManager }

// WailsAutostart is the app's opening at login, or nil when Wails has none.
func WailsAutostart(app *application.App) Autostart {
	if app == nil || app.Autostart == nil {
		return nil
	}
	return wailsAutostart{app.Autostart}
}

func (a wailsAutostart) Enabled() (bool, error) { return a.m.IsEnabled() }
func (a wailsAutostart) SetEnabled(on bool) error {
	if on {
		return a.m.Enable()
	}
	return a.m.Disable()
}
