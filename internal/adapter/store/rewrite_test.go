package store

import (
	"errors"
	"testing"
)

func TestRewriteHandsOverTheFileAsItIsNow(t *testing.T) {
	s := newStore(t)
	err := s.Rewrite("new/a.md", func(old []byte, ok bool) ([]byte, error) {
		if ok || old != nil {
			t.Errorf("a missing note was handed over as %q, %v", old, ok)
		}
		return []byte("one\n"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	err = s.Rewrite("new/a.md", func(old []byte, ok bool) ([]byte, error) {
		if !ok || string(old) != "one\n" {
			t.Errorf("handed over %q, %v; want the file", old, ok)
		}
		return append(old, "two\n"...), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := read(t, s, "new/a.md"); got != "one\ntwo\n" {
		t.Fatalf("got %q", got)
	}
}

func TestRewriteThatFailsWritesNothing(t *testing.T) {
	s := newStore(t)
	write(t, s, "a.md", "kept\n")
	no := errors.New("no")
	if err := s.Rewrite("a.md", func([]byte, bool) ([]byte, error) { return []byte("lost\n"), no }); !errors.Is(err, no) {
		t.Fatalf("err = %v", err)
	}
	if got := read(t, s, "a.md"); got != "kept\n" {
		t.Fatalf("got %q", got)
	}
}
