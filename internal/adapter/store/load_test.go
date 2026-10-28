package store

import (
	"os"
	"path/filepath"
	"testing"
)

// Load reads the board and what every sticky.json says in one pass: each
// settings file once, however many things are asked of it afterwards.
func TestLoadReadsEachSettingsFileOnce(t *testing.T) {
	s := newStore(t)
	write(t, s, "a.md", "a\n")
	write(t, s, "deploy/run.md", "r\n")
	write(t, s, ViewFile, `{"notes":{"a.md":{"size":"half"}},"order":["a.md"],"theme":"nord","language":"ko","tab":"deploy","title":"Home","ignore":["*.tmp.md"]}`)
	write(t, s, "deploy/"+ViewFile, `{"title":"Deploy","notes":{"run.md":{"open":true}},"order":["run.md"]}`)
	write(t, s, "skip.tmp.md", "x\n")

	reads := map[string]int{}
	orig := readFile
	readFile = func(name string) ([]byte, error) {
		if filepath.Base(name) == ViewFile {
			rel, _ := filepath.Rel(s.Dir, name)
			reads[rel]++
		}
		return orig(name)
	}
	t.Cleanup(func() { readFile = orig })

	b, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Tabs) != 2 || b.Tabs[0].Title != "Home" || b.Tabs[1].Title != "Deploy" || len(b.Tabs[0].Notes) != 1 {
		t.Errorf("tabs = %+v", b.Tabs)
	}
	set := b.Settings
	if set.Theme() != "nord" || set.Language() != "ko" || set.Tab() != "deploy" {
		t.Errorf("root keys: %q %q %q", set.Theme(), set.Language(), set.Tab())
	}
	if v := set.Views(); v["a.md"].Size != "half" || v["deploy/run.md"].Open == nil {
		t.Errorf("views = %+v", v)
	}
	if o := set.Order("deploy"); len(o) != 1 || o[0] != "deploy/run.md" {
		t.Errorf("order = %v", o)
	}
	for file, n := range reads {
		if n != 1 {
			t.Errorf("%s was read %d times, want once", file, n)
		}
	}
	if len(reads) != 2 {
		t.Errorf("read %v, want the two settings files", reads)
	}
}

// A settings file that cannot be read is reported once; the rest of the
// board is still read and that tab arranges itself.
func TestLoadReportsABrokenSettingsFile(t *testing.T) {
	s := newStore(t)
	write(t, s, "deploy/run.md", "r\n")
	write(t, s, "deploy/"+ViewFile, `{not json`)
	b, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if b.Settings.Err() == nil || len(b.Tabs) != 2 {
		t.Errorf("err = %v, tabs = %d", b.Settings.Err(), len(b.Tabs))
	}
	if _, err := os.Stat(filepath.Join(s.Dir, "deploy", ViewFile)); err != nil {
		t.Error("the broken file is left alone")
	}
}

// A reload after one note changed reads that note only: the others did not
// change size or time, so what was read before is used again.
func TestLoadReadsOnlyTheNotesThatChanged(t *testing.T) {
	s := newStore(t)
	write(t, s, "a.md", "a\n")
	write(t, s, "b.md", "b\n")
	write(t, s, "deploy/c.md", "c\n")

	reads := map[string]int{}
	orig := readFile
	readFile = func(name string) ([]byte, error) {
		if filepath.Base(name) != ViewFile {
			rel, _ := filepath.Rel(s.Dir, name)
			reads[filepath.ToSlash(rel)]++
		}
		return orig(name)
	}
	t.Cleanup(func() { readFile = orig })

	if _, err := s.Load(); err != nil {
		t.Fatal(err)
	}
	if len(reads) != 3 {
		t.Fatalf("the first load reads every note: %v", reads)
	}
	clear(reads)
	write(t, s, "b.md", "b changed\n")
	b, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(reads) != 1 || reads["b.md"] != 1 {
		t.Errorf("the second load should read only b.md: %v", reads)
	}
	for _, n := range b.Tabs[0].Notes {
		if n.Name == "b.md" && n.Doc.Body != "b changed\n" {
			t.Errorf("b.md = %q", n.Doc.Body)
		}
	}
	if err := os.Remove(filepath.Join(s.Dir, "a.md")); err != nil {
		t.Fatal(err)
	}
	if b, _ := s.Load(); len(b.Tabs[0].Notes) != 1 {
		t.Errorf("a removed note is gone: %+v", b.Tabs[0].Notes)
	}
}
