package shell

// The home window's size (08 §8.1).
const (
	HomeWidth  = 860
	HomeHeight = 620
)

// ShowHome shows the home window, making it the first time.
func (s *Shell) ShowHome() {
	s.mu.Lock()
	w := s.home
	s.mu.Unlock()
	if w == nil {
		nw := s.cfg.Platform.NewWindow(WindowOptions{
			Kind:   KindHome,
			Name:   nameHome,
			Title:  "stickypane",
			URL:    pageHome,
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

// Reopen is macOS asking the app to show itself again: the Dock icon
// clicked, or the app opened once more (`open -n`), while its windows are
// hidden. The board used last comes back when its window is open; the
// home only when no board is.
func (s *Shell) Reopen() {
	s.mu.Lock()
	var w Window
	if p := s.projects[s.state.Last]; p != nil && !p.closed {
		w = p.win
	}
	s.mu.Unlock()
	if w == nil {
		s.ShowHome()
		return
	}
	w.Show()
	w.Focus()
}
