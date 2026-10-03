package store

import (
	"fmt"
	"testing"
)

func benchStore(b *testing.B) *Store {
	b.Helper()
	s := Open(b.TempDir())
	put := func(name, body string) {
		if err := s.Write(name, []byte(body)); err != nil {
			b.Fatal(err)
		}
	}
	for i := range 24 {
		put(fmt.Sprintf("note%02d.md", i), "---\nopen: true\n---\nbody\n")
	}
	put("deploy/run.md", "deploy\n")
	return s
}

func BenchmarkSettings(b *testing.B) {
	s := benchStore(b)
	b.ReportAllocs()
	for b.Loop() {
		s.Settings()
	}
}

func BenchmarkLoad(b *testing.B) {
	s := benchStore(b)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := s.Load(); err != nil {
			b.Fatal(err)
		}
	}
}
