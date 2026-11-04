package shell

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/LeeSwallow/stickypane/desktop/internal/index"
)

// openIndex opens the index in the background. Open retries while the
// database is locked and can take long; after IndexDeadline the home
// window is told it failed, and a late success is still taken.
func (s *Shell) openIndex() {
	done := make(chan struct{})
	go func() {
		defer close(done)
		ix, err := s.cfg.OpenIndex(s.cfg.Home)
		s.mu.Lock()
		if s.stopped {
			s.mu.Unlock()
			if ix != nil {
				ix.Close()
			}
			return
		}
		s.ix, s.ixErr = ix, err
		s.mu.Unlock()
		if err == nil {
			s.refreshTray()
		}
	}()
	select {
	case <-done:
	case <-time.After(s.cfg.IndexDeadline):
		s.mu.Lock()
		if s.ix == nil && s.ixErr == nil {
			s.ixErr = fmt.Errorf("index: not open after %v (is another app holding %s?)", s.cfg.IndexDeadline, filepath.Join(s.cfg.Home, "stickypane.db"))
		}
		s.mu.Unlock()
	case <-s.quit:
	}
	close(s.ixReady)
}

// index waits for the first open of the index to end.
func (s *Shell) index() (Index, error) {
	select {
	case <-s.ixReady:
	case <-time.After(s.cfg.IndexDeadline + time.Second): // Start was not called
		return nil, errors.New("index: not started")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ix != nil {
		return s.ix, nil
	}
	if s.ixErr != nil {
		return nil, s.ixErr
	}
	return nil, errors.New("index: not open")
}

// Ensure makes the home folder (that the app is used is what makes
// stickypane keep records there) and returns its path.
func (s *Shell) Ensure() (string, error) {
	return s.cfg.Home, os.MkdirAll(s.cfg.Home, 0o755)
}

// Sessions reads new events, tidies away folders that are gone, and
// returns the home list.
func (s *Shell) Sessions() (index.Groups, error) {
	ix, err := s.index()
	if err != nil {
		return index.Groups{}, err
	}
	g, err := sessions(ix)
	if err == nil {
		s.cfg.Platform.SetTray(trayOf(g))
	}
	return g, err
}

func sessions(ix Index) (index.Groups, error) {
	if err := ix.Ingest(); err != nil {
		return index.Groups{}, err
	}
	now := time.Now()
	if _, err := ix.Prune(now); err != nil {
		return index.Groups{}, err
	}
	return ix.Sessions(now)
}

// Restore brings back a project that was tidied away.
func (s *Shell) Restore(id int64) error {
	ix, err := s.index()
	if err != nil {
		return err
	}
	return ix.Restore(id)
}

// refreshTray rebuilds the tray from the index, if it is open.
func (s *Shell) refreshTray() {
	s.mu.Lock()
	ix := s.ix
	s.mu.Unlock()
	if ix == nil {
		return
	}
	if g, err := sessions(ix); err == nil {
		s.cfg.Platform.SetTray(trayOf(g))
	}
}
