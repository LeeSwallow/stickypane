package shell

import (
	"errors"
	"fmt"
)

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
