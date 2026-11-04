package shell

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	installsvc "github.com/LeeSwallow/stickypane/desktop/internal/install/service"
	"github.com/LeeSwallow/stickypane/desktop/internal/remote"
)

// Inspection says whether a folder already has a board.
type Inspection struct {
	Exists bool   `json:"exists"` // dir has a .sticky/
	Path   string `json:"path"`   // dir, cleaned
}

// Inspect looks at dir without changing anything.
func (s *Shell) Inspect(dir string) Inspection {
	dir = filepath.Clean(dir)
	fi, err := os.Stat(filepath.Join(dir, ".sticky"))
	return Inspection{Exists: err == nil && fi.IsDir(), Path: dir}
}

// userHomeDir is the person's home folder; a test replaces it.
var userHomeDir = os.UserHomeDir

// resolvePath cleans p and follows its symlinks; when that fails, only cleans.
func resolvePath(p string) string {
	p = filepath.Clean(p)
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return filepath.Clean(r)
	}
	return p
}

// samePath compares two resolved paths; macOS volumes are case-insensitive.
func samePath(a, b string) bool {
	if runtime.GOOS == "darwin" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// refuseInit stops a board being made in the home folder or the root, where
// every project below would find it.
func refuseInit(dir string) error {
	d := resolvePath(dir)
	if d == string(filepath.Separator) {
		return errors.New("will not make a board in the root folder")
	}
	if h, err := userHomeDir(); err == nil && samePath(d, resolvePath(h)) {
		return errors.New("will not make a board in the home folder: every project below it would find it")
	}
	return nil
}

// InitProject makes a board in dir with the app's own `stickypane init`.
func (s *Shell) InitProject(dir string) error { return s.InitProjectIn(dir, "") }

// InitProjectIn makes a board in dir with its welcome note in lang, the
// language the window that asked draws in ("ko", "en"); "" is the app's
// language setting.
func (s *Shell) InitProjectIn(dir, lang string) error {
	if err := refuseInit(dir); err != nil {
		return err
	}
	if lang == "" {
		lang = s.Prefs().Language
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	_, err := s.cfg.Install.Init(ctx, dir, lang)
	return err
}

// MakeBoard makes a board in the folder of a window that found none
// (StateNoBoard), and starts its bridge again.
func (s *Shell) MakeBoard(window uint) error { return s.MakeBoardIn(window, "") }

// MakeBoardIn is MakeBoard with the welcome note in lang (InitProjectIn).
func (s *Shell) MakeBoardIn(window uint, lang string) error {
	s.mu.Lock()
	p := s.byWin[window]
	s.mu.Unlock()
	if p == nil {
		return fmt.Errorf("no project window %d", window)
	}
	if remote.IsRemote(p.path) {
		return fmt.Errorf("%w: make its board there, with stickypane init", ErrRemote)
	}
	if err := s.InitProjectIn(p.path, lang); err != nil {
		return err
	}
	if !s.setState(p, StateLoading, "", nil) {
		return nil
	}
	go s.connect(p.ctx, p)
	return nil
}

// InitHere runs `stickypane init` again in a window's project: it puts the
// agent guide where agents read it, or takes it out where the plugin
// teaches them now. It is safe to run again.
func (s *Shell) InitHere(window uint) (installsvc.Result, error) {
	s.mu.Lock()
	p := s.byWin[window]
	s.mu.Unlock()
	if p == nil {
		return installsvc.Result{}, fmt.Errorf("no project window %d", window)
	}
	if remote.IsRemote(p.path) {
		return installsvc.Result{}, fmt.Errorf("%w: run stickypane init there", ErrRemote)
	}
	if err := refuseInit(p.path); err != nil {
		return installsvc.Result{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	return s.cfg.Install.Init(ctx, p.path, s.Prefs().Language)
}

// PickFolder asks for a folder.
func (s *Shell) PickFolder() (string, error) { return s.cfg.Platform.PickFolder() }
