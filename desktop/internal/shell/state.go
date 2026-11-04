package shell

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/LeeSwallow/stickypane/desktop/internal/canvas"
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
	Hotkey   string `json:"hotkey"`
	Language string `json:"language,omitempty"` // the home window's: "", "ko", "en"
	Mode     string `json:"mode,omitempty"`     // the home window's: "", "light", "dark"
	Motion   string `json:"motion,omitempty"`   // every window's: "", "reduce", "off"
	// PinNew keeps a project window seen for the first time on top.
	PinNew  bool                   `json:"pinNew,omitempty"`
	Last    string                 `json:"last,omitempty"`    // the project the hotkey toggles
	Windows map[string]WindowState `json:"windows,omitempty"` // by project path
	// TrayHint is set once the person has been told, on closing a window,
	// that the app stays in the tray.
	TrayHint bool `json:"trayHint,omitempty"`
	// Projects are where each project's notes sit on this computer: their
	// places on the canvas and the frames of the notes torn off
	// (docs/desktop/03 §3.6.1), by project path.
	Projects map[string]canvas.Project `json:"projects,omitempty"`
	// Boards are what each board looked like when this computer last
	// opened it, for the home's cards (peek.go), by project path.
	Boards map[string]BoardPeek `json:"boards,omitempty"`
	// Hosts are the computers whose boards open over ssh (remote.go),
	// each with its boards' folders there. A remote board's key in the
	// maps above is ssh://<host><path>.
	Hosts []Host `json:"hosts,omitempty"`
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
		return State{Hotkey: DefaultHotkey}, fmt.Errorf("%s: %w", path, err)
	}
	if s.Hotkey == "" {
		s.Hotkey = DefaultHotkey
	}
	return s, nil
}

// emptyState is what makeStateFile writes: every setting at its default,
// so that a default changed later is not pinned by a first launch.
const emptyState = "{}\n"

// makeStateFile writes emptyState to path if nothing is there. It never
// replaces a file, whatever is in it, nor leaves half of one behind.
func makeStateFile(path string) (err error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, fs.ErrExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("make the app's state: %w", err)
	}
	if _, err = f.WriteString(emptyState); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return fmt.Errorf("make the app's state: %w", err)
	}
	if err = f.Close(); err != nil {
		_ = os.Remove(path)
		return fmt.Errorf("make the app's state: %w", err)
	}
	return nil
}

// stateTemp is the pattern of the temporary file a save writes first,
// beside the state file.
const stateTemp = ".overlay-*.json"

// Save writes the state file whole: to a temporary file, synced to disk,
// then renamed, so that a crash leaves the old file or the new one. A
// save that fails leaves no temporary file behind.
func (s State) Save(path string) error {
	if err := s.save(path); err != nil {
		return fmt.Errorf("save the app's state: %w", err)
	}
	return nil
}

func (s State) save(path string) (err error) {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, stateTemp)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil { // the temporary file goes; its own errors add nothing
			_ = tmp.Close()
			_ = os.Remove(tmp.Name())
		}
	}()
	if _, err := tmp.Write(append(b, '\n')); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
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
	c.Projects = make(map[string]canvas.Project, len(s.Projects))
	for k, v := range s.Projects {
		c.Projects[k] = v.Clone()
	}
	c.Boards = make(map[string]BoardPeek, len(s.Boards))
	for k, v := range s.Boards {
		c.Boards[k] = v
	}
	c.Hosts = cloneHosts(s.Hosts)
	return c
}
