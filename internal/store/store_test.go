package store

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/LeeSwallow/stickypane/internal/doc"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	dir := filepath.Join(t.TempDir(), DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return Open(dir)
}

func write(t *testing.T, s *Store, name, content string) {
	t.Helper()
	path := filepath.Join(s.Dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, s *Store, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(s.Dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

type failOp struct{}

func (failOp) Apply(d doc.Document) (doc.Document, error) { return d, doc.ErrConflict }

func TestResolve(t *testing.T) {
	root := t.TempDir()
	board := filepath.Join(root, "proj", DirName)
	deep := filepath.Join(root, "proj", "sub", "deep")
	for _, dir := range []string{board, deep} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, arg := range []string{deep, filepath.Join(root, "proj"), board} {
		got, err := Resolve(arg)
		if err != nil || got != board {
			t.Errorf("Resolve(%q) = %q, %v; want %q", arg, got, err, board)
		}
	}
	if _, err := Resolve(t.TempDir()); !errors.Is(err, ErrNotFound) {
		t.Errorf("Resolve without a board: err = %v, want ErrNotFound", err)
	}
}

func TestScanListsNotesAndSkipsTheRest(t *testing.T) {
	s := newStore(t)
	write(t, s, "b.md", "---\ntype: board\n---\n## A\n")
	write(t, s, "a.md", "hello\n")
	write(t, s, "UPPER.MD", "x\n")
	write(t, s, ".hidden.md", "x\n")
	write(t, s, "picture.png", "x\n")
	write(t, s, "archive/old.md", "x\n")
	notes, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if got := names(notes); got != "UPPER.MD,a.md,b.md" {
		t.Fatalf("names = %s", got)
	}
	if notes[2].Doc.Type() != "board" || notes[1].Doc.Body != "hello\n" {
		t.Errorf("documents were not parsed: %+v", notes)
	}
	if notes[1].ModTime.IsZero() || notes[1].Path != filepath.Join(s.Dir, "a.md") {
		t.Errorf("note metadata is missing: %+v", notes[1])
	}
}

func TestScanReplacesInvalidUTF8ForDisplayOnly(t *testing.T) {
	s := newStore(t)
	raw := "ok \xff\xfe bad\n"
	write(t, s, "a.md", raw)
	notes, _ := s.Scan()
	if !utf8.ValidString(notes[0].Doc.Body) {
		t.Errorf("Body is not valid UTF-8: %q", notes[0].Doc.Body)
	}
	if err := s.Apply("a.md", doc.SetKey{Key: "pin", Value: "true"}); err != nil {
		t.Fatal(err)
	}
	if got := read(t, s, "a.md"); !strings.HasSuffix(got, raw) {
		t.Errorf("Apply must keep the original bytes, got %q", got)
	}
}

func TestScanLargeFiles(t *testing.T) {
	s := newStore(t)
	var entries bytes.Buffer
	for i := 0; entries.Len() <= MaxSize; i++ {
		fmt.Fprintf(&entries, "entry %06d\n", i)
	}
	last := entries.String()[entries.Len()-len("entry 000000\n"):]
	write(t, s, "big-log.md", "---\ntype: log\n---\n"+entries.String())
	write(t, s, "big-note.md", entries.String())
	notes, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}
	log, plain := notes[0], notes[1]
	if log.Err != nil {
		t.Fatalf("a large log must still load: %v", log.Err)
	}
	if n := len(log.Doc.Body); n == 0 || n > 64<<10 {
		t.Errorf("log body is %d bytes, want the last 64 KB at most", n)
	}
	if !strings.HasPrefix(log.Doc.Body, "entry ") || !strings.HasSuffix(log.Doc.Body, last) {
		t.Errorf("log tail should start at a line boundary and end with the last entry")
	}
	if !errors.Is(plain.Err, ErrTooLarge) {
		t.Errorf("a large plain note: Err = %v, want ErrTooLarge", plain.Err)
	}
}

func TestScanReportsUnreadableFiles(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read everything")
	}
	s := newStore(t)
	write(t, s, "secret.md", "x\n")
	if err := os.Chmod(filepath.Join(s.Dir, "secret.md"), 0); err != nil {
		t.Fatal(err)
	}
	notes, err := s.Scan()
	if err != nil || len(notes) != 1 || notes[0].Err == nil {
		t.Errorf("want one note carrying a read error, got %+v, %v", notes, err)
	}
}

func TestApplyWritesAndKeepsMode(t *testing.T) {
	s := newStore(t)
	write(t, s, "a.md", "hello\n")
	path := filepath.Join(s.Dir, "a.md")
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := s.Apply("a.md", doc.SetKey{Key: "pin", Value: "true"}); err != nil {
		t.Fatal(err)
	}
	if got := read(t, s, "a.md"); got != "---\npin: true\n---\nhello\n" {
		t.Errorf("content = %q", got)
	}
	if fi, _ := os.Stat(path); fi.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v, want 0600", fi.Mode().Perm())
	}
}

func TestApplyUsesTheFileAsItIsNow(t *testing.T) {
	s := newStore(t)
	write(t, s, "a.md", "first\n")
	if _, err := s.Scan(); err != nil {
		t.Fatal(err)
	}
	write(t, s, "a.md", "first\nadded by the agent\n")
	if err := s.Apply("a.md", doc.SetKey{Key: "color", Value: "blue"}); err != nil {
		t.Fatal(err)
	}
	if got := read(t, s, "a.md"); got != "---\ncolor: blue\n---\nfirst\nadded by the agent\n" {
		t.Errorf("the agent's edit was lost: %q", got)
	}
}

func TestApplyConflictLeavesFileAlone(t *testing.T) {
	s := newStore(t)
	write(t, s, "a.md", "hello\n")
	if err := s.Apply("a.md", failOp{}); !errors.Is(err, doc.ErrConflict) {
		t.Errorf("err = %v, want ErrConflict", err)
	}
	if got := read(t, s, "a.md"); got != "hello\n" {
		t.Errorf("content = %q", got)
	}
	entries, _ := os.ReadDir(s.Dir)
	if len(entries) != 1 {
		t.Errorf("temporary files were left behind: %v", entries)
	}
}

func TestApplyOnMissingFileIsConflict(t *testing.T) {
	if err := newStore(t).Apply("gone.md", doc.SetKey{Key: "pin", Value: "true"}); !errors.Is(err, doc.ErrConflict) {
		t.Errorf("err = %v, want ErrConflict", err)
	}
}

func TestCreatePicksUniqueNames(t *testing.T) {
	s := newStore(t)
	now := time.Date(2026, 10, 1, 14, 2, 0, 0, time.UTC)
	var got []string
	for i := 0; i < 3; i++ {
		name, err := s.Create("Check env", []byte("Check env\n"), now)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, name)
	}
	if want := "check-env.md,check-env-2.md,check-env-3.md"; strings.Join(got, ",") != want {
		t.Errorf("names = %v, want %s", got, want)
	}
	if read(t, s, "check-env-2.md") != "Check env\n" {
		t.Error("content was not written")
	}
}

func TestSlug(t *testing.T) {
	now := time.Date(2026, 10, 1, 14, 2, 5, 0, time.UTC)
	cases := []struct{ text, want string }{
		{"Check env", "check-env"},
		{"배포 전에 env 확인!", "배포-전에-env-확인"},
		{"  Hello,   World_2  ", "hello-world-2"},
		{"a/b\\c:d", "abcd"},
		{"!!!", "note-20261001-140205"},
		{"", "note-20261001-140205"},
		{strings.Repeat("가", 50), strings.Repeat("가", 40)},
		{strings.Repeat("a", 39) + " tail", strings.Repeat("a", 39)},
	}
	for _, c := range cases {
		if got := Slug(c.text, now); got != c.want {
			t.Errorf("Slug(%q) = %q, want %q", c.text, got, c.want)
		}
	}
}

func TestArchiveMovesFile(t *testing.T) {
	s := newStore(t)
	write(t, s, "a.md", "first\n")
	if err := s.Archive("a.md"); err != nil {
		t.Fatal(err)
	}
	write(t, s, "a.md", "second\n")
	if err := s.Archive("a.md"); err != nil {
		t.Fatal(err)
	}
	if notes, _ := s.Scan(); len(notes) != 0 {
		t.Errorf("archived notes are still on the board: %s", names(notes))
	}
	if read(t, s, "archive/a.md") != "first\n" || read(t, s, "archive/a-2.md") != "second\n" {
		t.Error("archive should keep both files")
	}
}

func TestDelete(t *testing.T) {
	s := newStore(t)
	write(t, s, "a.md", "x\n")
	if err := s.Delete("a.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(s.Dir, "a.md")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("file still exists: %v", err)
	}
}
