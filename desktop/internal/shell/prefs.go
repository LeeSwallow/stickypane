package shell

// Prefs are the home window's settings in overlay.json.
type Prefs struct {
	Hotkey   string `json:"hotkey"`
	Language string `json:"language"`
	Mode     string `json:"mode"`
	// Motion is how much the windows move: "" as the system says,
	// "reduce" (fades only) or "off".
	Motion string `json:"motion"`
	// PinNew keeps a board's window on top the first time it opens.
	PinNew bool `json:"pinNew"`
}

// Prefs returns the home window's settings.
func (s *Shell) Prefs() Prefs {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Prefs{Hotkey: s.state.Hotkey, Language: s.state.Language, Mode: s.state.Mode, Motion: s.state.Motion, PinNew: s.state.PinNew}
}

// SetPrefs keeps the app's settings; a new hotkey is registered. Every
// window hears them (PrefsEvent).
func (s *Shell) SetPrefs(p Prefs) error {
	defer s.emitPrefs()
	s.mu.Lock()
	s.state.Language, s.state.Mode, s.state.PinNew = p.Language, p.Mode, p.PinNew
	switch p.Motion {
	case "reduce", "off":
		s.state.Motion = p.Motion
	default:
		s.state.Motion = ""
	}
	// A new hotkey, or the same one that could not be had: try it.
	change := p.Hotkey != "" && p.Hotkey != s.hotkey
	s.mu.Unlock()
	if change {
		if err := s.SetHotkey(p.Hotkey); err != nil {
			s.keep("SetPrefs")
			return err
		}
		return nil
	}
	return s.save()
}

// emitPrefs tells every window the app's settings.
func (s *Shell) emitPrefs() { s.cfg.Platform.Emit(PrefsEvent, s.Prefs()) }
