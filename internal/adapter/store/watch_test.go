package store

import (
	"context"
	"testing"
	"time"
)

func TestWatchSignalsChanges(t *testing.T) {
	s := newStore(t)
	ch, err := s.Watch(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	write(t, s, "a.md", "hello\n")
	select {
	case _, ok := <-ch:
		if !ok {
			t.Fatal("channel closed early")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no signal after a file was written")
	}
}

func TestWatchSignalsDuringContinuousWrites(t *testing.T) {
	s := newStore(t)
	ch, err := s.Watch(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
				write(t, s, "log.md", time.Now().String())
				time.Sleep(20 * time.Millisecond)
			}
		}
	}()
	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		t.Error("a steady stream of writes must not starve the signal")
	}
	close(stop)
	<-done
}

func TestWatchClosesOnCancel(t *testing.T) {
	s := newStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := s.Watch(ctx)
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("got a signal instead of a closed channel")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("channel was not closed after cancel")
	}
}
