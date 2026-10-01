package store

import (
	"context"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Debounce is how long changes are collected before one signal is sent.
const Debounce = 100 * time.Millisecond

// Watch signals on the returned channel after the folder changed. Events are
// not interpreted per file, because editors and agents often save by writing
// a temporary file and renaming it; the receiver simply rescans. The timer
// starts at the first event and is not restarted by later ones, so a steady
// stream of writes still produces a signal every Debounce. The channel is
// closed when ctx ends.
func (s *Store) Watch(ctx context.Context) (<-chan struct{}, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := w.Add(s.Dir); err != nil {
		w.Close()
		return nil, err
	}
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
				select {
				case ch <- struct{}{}:
				default: // a signal is already waiting
				}
			}
		}
	}()
	return ch, nil
}
