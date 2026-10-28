package shell

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// DefaultHotkey toggles the last project window (ADR-0007).
const DefaultHotkey = "Option+Space"

// StateFile is the name of the app's own settings, in the home folder.
const StateFile = "overlay.json"

// WindowState is where a project window was left, and whether it was
// pinned. A zero Width means it was never placed: the window opens at its
// first size, centred.
type WindowState struct {
	X      int  `json:"x"`
	Y      int  `json:"y"`
	Width  int  `json:"width"`
	Height int  `json:"height"`
	Pinned bool `json:"pinned"`
}

// State is ~/.stickypane/overlay.json: the app's settings that are not a
// board's. Notes and their settings are the bridge's; this is windows and
// the home screen.
type State struct {
	Hotkey   string                 `json:"hotkey"`
	Language string                 `json:"language,omitempty"` // the home window's: "", "ko", "en"
	Mode     string                 `json:"mode,omitempty"`     // the home window's: "", "light", "dark"
	Motion   string                 `json:"motion,omitempty"`   // every window's: "", "reduce", "off"
	Last     string                 `json:"last,omitempty"`     // the project the hotkey toggles
	Windows  map[string]WindowState `json:"windows,omitempty"`  // by project path
	// TrayHint is set once the person has been told, on closing a window,
	// that the app stays in the tray.
	TrayHint bool `json:"trayHint,omitempty"`
}

// LoadState reads the state file; a missing file is the defaults.
func LoadState(path string) (State, error) {
	s := State{Hotkey: DefaultHotkey}
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return State{Hotkey: DefaultHotkey}, err
	}
	if s.Hotkey == "" {
		s.Hotkey = DefaultHotkey
	}
	return s, nil
}

// Save writes the state file whole: to a temporary file, synced to disk,
// then renamed, so that a crash leaves the old file or the new one.
func (s State) Save(path string) error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".overlay-*.json")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(append(b, '\n')); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// clone copies the state so that it can be saved outside the lock.
func (s State) clone() State {
	c := s
	c.Windows = make(map[string]WindowState, len(s.Windows))
	for k, v := range s.Windows {
		c.Windows[k] = v
	}
	return c
}
