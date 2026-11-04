package shell

// Notice is a line for the home window. Code is translated there; Detail
// is the system's own words.
type Notice struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

// NoticeHotkey: the global hotkey could not be registered.
const NoticeHotkey = "hotkey_failed"

// NoticeStayInTray tells the home window that a closed window left the app
// running in the tray (shown once, with a Quit action).
const NoticeStayInTray = "stayInTray"

func dropNotice(ns []Notice, code string) []Notice {
	out := ns[:0]
	for _, n := range ns {
		if n.Code != code {
			out = append(out, n)
		}
	}
	return out
}

func (s *Shell) notice(n Notice) {
	s.mu.Lock()
	s.notices = append(dropNotice(s.notices, n.Code), n)
	s.mu.Unlock()
	s.cfg.Platform.Emit(NoticesEvent, s.Notices())
}

// Notices are the lines the home window shows.
func (s *Shell) Notices() []Notice {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Notice{}, s.notices...)
}

// trayHint says once, the first time a window is closed, that the app
// stays in the tray. If no window is left to say it in, the home window
// comes forward to say it.
func (s *Shell) trayHint(closing Window) {
	s.mu.Lock()
	if s.stopped || s.state.TrayHint {
		s.mu.Unlock()
		return
	}
	s.state.TrayHint = true
	var wins []Window
	if s.home != nil {
		wins = append(wins, s.home)
	}
	for _, p := range s.projects {
		if p.win != nil {
			wins = append(wins, p.win)
		}
	}
	s.mu.Unlock()
	s.keep("trayHint")
	s.notice(Notice{Code: NoticeStayInTray})
	for _, w := range wins { // asked outside the lock
		if w != closing && w.Visible() {
			return
		}
	}
	s.ShowHome()
}
