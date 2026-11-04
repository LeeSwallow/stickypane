package shell

import (
	"net/url"

	"github.com/LeeSwallow/stickypane/desktop/internal/canvas"
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

// The sections of the settings window.
var settingsSections = map[string]bool{"board": true, "app": true, "hosts": true, "cli": true, "about": true}

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
