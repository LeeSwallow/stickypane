package store

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Debounce is how long changes are collected before one signal is sent.
const Debounce = 100 * time.Millisecond

// watched returns what has to be watched besides the notes folder itself:
// the folders that are books, and the files that linked notes point to,
// which change without anything in the notes folder changing.
func (s *Store) watched() []string {
	var paths []string
	linked := func(path string) {
		if fi, err := os.Lstat(path); err == nil && fi.Mode()&os.ModeSymlink != 0 {
			if target, err := filepath.EvalSymlinks(path); err == nil {
				paths = append(paths, target)
			}
		}
	}
	entries, _ := os.ReadDir(s.Dir)
	for _, e := range entries {
		name := e.Name()
		path := filepath.Join(s.Dir, name)
		if strings.HasPrefix(name, ".") || name == ArchiveDir {
			continue
		}
		linked(path)
		if fi, err := os.Stat(path); err != nil || !fi.IsDir() {
			continue
		}
		paths = append(paths, path)
		inside, _ := os.ReadDir(path)
		for _, p := range inside {
			sub := filepath.Join(path, p.Name())
			linked(sub)
			if fi, err := os.Stat(sub); err == nil && fi.IsDir() && !strings.HasPrefix(p.Name(), ".") {
				paths = append(paths, sub) // a book inside a tab
				pages, _ := os.ReadDir(sub)
				for _, q := range pages {
					linked(filepath.Join(sub, q.Name()))
				}
			}
		}
	}
	return paths
}

// Watch signals on the returned channel after the folder changed. Events are
// not interpreted per file, because editors and agents often save by writing
// a temporary file and renaming it; the receiver simply rescans. The timer
// starts at the first event and is not restarted by later ones, so a steady
// stream of writes still produces a signal every Debounce. The channel is
// closed when ctx ends.
//
// Books and linked files are watched too. What there is to watch is looked
// up again after every change, so a folder made later is followed as well.
func (s *Store) Watch(ctx context.Context) (<-chan struct{}, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := w.Add(s.Dir); err != nil {
		w.Close()
		return nil, err
	}
	follow := func() {
		for _, p := range s.watched() {
			_ = w.Add(p) // adding a path twice is harmless
		}
	}
	follow()
	ch := make(chan struct{}, 1)
	go func() {
		defer close(ch)
		defer w.Close()
		var fire <-chan time.Time
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-w.Events:
				if !ok {
					return
				}
				if fire == nil {
					fire = time.After(Debounce)
				}
			case _, ok := <-w.Errors:
				if !ok {
					return
				}
			case <-fire:
				fire = nil
				follow()
				select {
				case ch <- struct{}{}:
				default: // a signal is already waiting
				}
			}
		}
	}()
	return ch, nil
}
