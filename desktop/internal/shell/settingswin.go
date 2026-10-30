package shell

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/LeeSwallow/stickypane/desktop/internal/canvas"
	"github.com/LeeSwallow/stickypane/desktop/internal/hotkey"
)

// The settings window (docs/desktop/03 §3.12): one window, its own, with
// the board it was opened from. Its calls to a board go to that board's
// bridge; the app's settings are overlay.json's, and every window hears
// when they change.

// KindSettings is the settings window's kind.
const KindSettings = "settings"

// The settings window's first and smallest size.
const (
	SettingsWidth     = 720
	SettingsHeight    = 580
	SettingsMinWidth  = 600
	SettingsMinHeight = 460
)

// PrefsEvent carries the app's settings (Prefs) to every window when they
// change; SettingsEvent tells the settings window its board changed.
const (
	PrefsEvent    = "prefs.changed"
	SettingsEvent = "settings.board"
)

// The sections of the settings window.
var settingsSections = map[string]bool{"board": true, "app": true, "cli": true, "about": true}

// settingsWin is the settings window and the project it was opened from;
// p is nil when it was opened from the home, or its board closed.
type settingsWin struct {
	win Window
	p   *project
}

// OpenSettings shows the settings window on a section, for the board of
// the window from (0, or the home's id: no board). An open one comes
// forward and takes the new board.
func (s *Shell) OpenSettings(from uint, section string) uint {
	if !settingsSections[section] {
		section = "board"
	}
	s.mu.Lock()
	p := s.byWin[from]
	if p != nil && p.closed {
		p = nil
	}
	if sw := s.settings; sw != nil && sw.win != nil {
		sw.p = p
		w := sw.win
		s.mu.Unlock()
		s.cfg.Platform.Emit(SettingsEvent, map[string]string{"section": section})
		w.Show()
		w.Focus()
		return w.ID()
	}
	sw := &settingsWin{p: p}
	s.settings = sw
	s.mu.Unlock()

	w := s.cfg.Platform.NewWindow(WindowOptions{
		Kind:      KindSettings,
		Name:      "settings",
		Title:     "Settings",
		URL:       "/?window=settings&section=" + url.QueryEscape(section),
		Bounds:    Rect{Width: SettingsWidth, Height: SettingsHeight},
		Frameless: true,
	})
	s.mu.Lock()
	sw.win = w
	s.mu.Unlock()
	w.OnClosing(func() {
		s.mu.Lock()
		if s.settings == sw {
			s.settings = nil
		}
		s.mu.Unlock()
	})
	w.Show()
	w.Focus()
	return w.ID()
}

// OpenAtLaunch opens what a launch asks for by name: "settings" or
// "settings:<section>", for the last board opened (STICKYPANE_APP_OPEN,
// to look at a window without pressing anything). Anything else is
// nothing.
func (s *Shell) OpenAtLaunch(what string) {
	name, section, _ := strings.Cut(strings.TrimSpace(what), ":")
	if name != "settings" {
		return
	}
	s.mu.Lock()
	var from uint
	if p := s.projects[s.state.Last]; p != nil && p.win != nil {
		from = p.win.ID()
	}
	s.mu.Unlock()
	s.OpenSettings(from, section)
}

// ownerOf is the project a window acts on: its own, a torn-off note's,
// or the board the settings window was opened from. The caller holds mu.
func (s *Shell) ownerOf(window uint) (*project, *tornWin) {
	if sw := s.settings; sw != nil && sw.win != nil && sw.win.ID() == window {
		if sw.p == nil || sw.p.closed {
			return nil, nil
		}
		return sw.p, nil
	}
	return s.owner(window)
}

// boardWindow is the board window a call acts on: the window itself, or,
// from the settings window, the board it was opened from (0 when none).
func (s *Shell) boardWindow(window uint) uint {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sw := s.settings; sw != nil && sw.win != nil && sw.win.ID() == window {
		if sw.p == nil || sw.p.closed || sw.p.win == nil {
			return 0
		}
		return sw.p.win.ID()
	}
	return window
}

// detachSettings lets the settings window know its board closed. The
// caller does not hold mu.
func (s *Shell) detachSettings(p *project) {
	s.mu.Lock()
	sw := s.settings
	hit := sw != nil && sw.p == p
	if hit {
		sw.p = nil
	}
	s.mu.Unlock()
	if hit {
		s.cfg.Platform.Emit(SettingsEvent, map[string]string{})
	}
}

// Touched tells every window of a window's project to read its board
// again (a setting changed from the settings window).
func (s *Shell) Touched(window uint) {
	s.mu.Lock()
	p, _ := s.ownerOf(window)
	var ids []uint
	if p != nil {
		ids = s.windowIDs(p)
	}
	s.mu.Unlock()
	for _, id := range ids {
		s.cfg.Platform.Emit(StatusEvent(id), s.Status(id))
	}
}

// layoutChanged tells a project's board window its layout, after the
// settings window changed it.
func (s *Shell) layoutChanged(window uint) {
	s.mu.Lock()
	p, _ := s.ownerOf(window)
	s.mu.Unlock()
	if p != nil {
		s.emitLayout(p)
	}
}

// ResetPlaces forgets where every note of a window's project sits on the
// canvas; the canvas places them again from the grid (03 §3.6.2). The
// mode, the snap and the torn-off notes stay.
func (s *Shell) ResetPlaces(window uint) error {
	if err := s.change(window, func(c canvas.Project) canvas.Project {
		c = c.Clone()
		c.Places = nil
		return c
	}); err != nil {
		return err
	}
	s.layoutChanged(window)
	return nil
}

// HotkeyCheck is whether a hotkey can be had: OK with it written one
// way, or the reason it cannot (a hotkey.Code*, or "taken": another
// program holds it).
type HotkeyCheck struct {
	OK     bool   `json:"ok"`
	Accel  string `json:"accel"`
	Code   string `json:"code,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// CodeTaken: the desktop would not give the hotkey (another program has it).
const CodeTaken = "taken"

// CheckHotkey says whether accel can be the global hotkey, trying it with
// the desktop when the rules allow it. It changes nothing.
func (s *Shell) CheckHotkey(accel string) HotkeyCheck {
	n, err := hotkey.Check(accel)
	var he *hotkey.Error
	if errors.As(err, &he) {
		return HotkeyCheck{Accel: he.Accel, Code: he.Code, Detail: he.Detail}
	}
	if err != nil {
		return HotkeyCheck{Accel: accel, Code: hotkey.CodeKey, Detail: err.Error()}
	}
	s.hotkeyMu.Lock()
	defer s.hotkeyMu.Unlock()
	s.mu.Lock()
	now := s.hotkey
	s.mu.Unlock()
	if same, _ := hotkey.Normalize(now); now != "" && same == n {
		return HotkeyCheck{OK: true, Accel: n}
	}
	if err := s.cfg.Platform.RegisterHotkey(n, func() {}); err != nil {
		return HotkeyCheck{Accel: n, Code: CodeTaken, Detail: err.Error()}
	}
	s.cfg.Platform.UnregisterHotkey(n)
	return HotkeyCheck{OK: true, Accel: n}
}

// Autostart is opening the app at login: the desktop's (Wails'
// AutostartManager); nil where it cannot.
type Autostart interface {
	Enabled() (bool, error)
	SetEnabled(on bool) error
}

// AutostartState says whether the app can open at login here, and whether it does.
type AutostartState struct {
	Supported bool   `json:"supported"`
	Enabled   bool   `json:"enabled"`
	Error     string `json:"error,omitempty"`
}

// ErrNoAutostart: this desktop cannot open the app at login.
var ErrNoAutostart = errors.New("opening at login is not supported here")

// AutostartState says whether the app opens at login.
func (s *Shell) AutostartState() AutostartState {
	if s.cfg.Autostart == nil {
		return AutostartState{}
	}
	on, err := s.cfg.Autostart.Enabled()
	if err != nil {
		return AutostartState{Supported: true, Error: err.Error()}
	}
	return AutostartState{Supported: true, Enabled: on}
}

// SetAutostart makes the app open at login, or not.
func (s *Shell) SetAutostart(on bool) (AutostartState, error) {
	if s.cfg.Autostart == nil {
		return AutostartState{}, ErrNoAutostart
	}
	if err := s.cfg.Autostart.SetEnabled(on); err != nil {
		return s.AutostartState(), fmt.Errorf("open at login: %w", err)
	}
	return s.AutostartState(), nil
}

// About is what the settings' 정보 shows: the app's version and the
// stickypane the app uses.
type About struct {
	App        string          `json:"app"`
	Stickypane StickypaneState `json:"stickypane"`
}

// About is the app's version and its stickypane's.
func (s *Shell) About() About {
	return About{App: s.cfg.AppVersion, Stickypane: s.StickypaneStatus()}
}

// emitPrefs tells every window the app's settings.
func (s *Shell) emitPrefs() { s.cfg.Platform.Emit(PrefsEvent, s.Prefs()) }
