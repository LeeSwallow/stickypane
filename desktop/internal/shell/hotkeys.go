package shell

import (
	"errors"
	"fmt"

	"github.com/LeeSwallow/stickypane/desktop/internal/hotkey"
)

func (s *Shell) registerHotkey(accel string) {
	if accel == "" {
		return
	}
	s.hotkeyMu.Lock()
	defer s.hotkeyMu.Unlock()
	if err := s.cfg.Platform.RegisterHotkey(accel, s.Toggle); err != nil {
		s.notice(Notice{Code: NoticeHotkey, Detail: fmt.Sprintf("%s: %v", accel, err)})
		return
	}
	s.mu.Lock()
	s.hotkey = accel
	s.mu.Unlock()
}

// SetHotkey changes the hotkey, or registers it again when it is the one
// that could not be had, and keeps it in the state. If the new one cannot
// be had the old one is kept; if the old one cannot be had back either,
// the home window is told there is no hotkey.
func (s *Shell) SetHotkey(accel string) error {
	s.hotkeyMu.Lock()
	defer s.hotkeyMu.Unlock()
	s.mu.Lock()
	old := s.hotkey // registered now; "" when none is
	s.mu.Unlock()
	if old == accel {
		return nil
	}
	if old != "" {
		s.cfg.Platform.UnregisterHotkey(old)
	}
	if err := s.cfg.Platform.RegisterHotkey(accel, s.Toggle); err != nil {
		if old != "" {
			if rerr := s.cfg.Platform.RegisterHotkey(old, s.Toggle); rerr != nil {
				s.mu.Lock()
				s.hotkey = ""
				s.mu.Unlock()
				s.notice(Notice{Code: NoticeHotkey, Detail: fmt.Sprintf("%s: %v; %s: %v", accel, err, old, rerr)})
			}
		}
		return err
	}
	s.mu.Lock()
	s.hotkey = accel
	s.state.Hotkey = accel
	s.notices = dropNotice(s.notices, NoticeHotkey)
	s.mu.Unlock()
	s.cfg.Platform.Emit(NoticesEvent, s.Notices())
	return s.save()
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
