package store

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func names(notes []Note) string {
	var out []string
	for _, n := range notes {
		name := n.Name
		if n.Book() {
			var pages []string
			for _, p := range n.Pages {
				pages = append(pages, p.Name)
			}
			name += "[" + strings.Join(pages, " ") + "]"
		}
		out = append(out, name)
	}
	return strings.Join(out, ",")
}

func TestScanReadsLogsScriptsAndFolders(t *testing.T) {
	s := newStore(t)
	write(t, s, "a.md", "note\n")
	write(t, s, "build.log", "---\nnot front matter\n---\nline\n")
	write(t, s, "deploy.sh", "#!/bin/sh\necho hi\n")
	write(t, s, "notes.txt", "plain\n")
	write(t, s, "data.json", "{}")
	write(t, s, "sticky.json", "{}")
	write(t, s, "docs/02-usage.md", "usage\n")
	write(t, s, "docs/01-intro.md", "intro\n")
	write(t, s, "docs/image.png", "x")
	write(t, s, "docs/deeper/skip.md", "too deep\n")
	write(t, s, "empty/readme.pdf", "x")
	write(t, s, "archive/old.md", "old\n")
	write(t, s, ".hidden/x.md", "x\n")

	notes, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}
	// docs is a tab, whose files are notes and whose folder deeper is a book.
	if got := names(notes); got != "a.md,build.log,deploy.sh,notes.txt,docs/01-intro.md,docs/02-usage.md,docs/deeper[docs/deeper/skip.md]" {
		t.Fatalf("Scan = %s", got)
	}
	if log := notes[1]; log.Doc.HasFront || log.Doc.Body != "---\nnot front matter\n---\nline\n" {
		t.Errorf("only Markdown has front matter; a log is shown as it is: %+v", log.Doc)
	}
	book := notes[6]
	if !book.Book() || book.Pages[0].Doc.Body != "too deep\n" || book.Path != filepath.Join(s.Dir, "docs", "deeper") || book.ModTime.IsZero() {
		t.Errorf("a folder in a tab is one note whose pages are its files: %+v", book)
	}
	if notes[4].Doc.Body != "intro\n" {
		t.Errorf("a file in a tab is a note: %+v", notes[4])
	}
	if notes[0].Book() {
		t.Error("a file is not a book")
	}
}

func TestALargeLogFileShowsItsEnd(t *testing.T) {
	s := newStore(t)
	var sb strings.Builder
	for sb.Len() <= MaxSize {
		sb.WriteString("an old line that will not be shown\n")
	}
	sb.WriteString("the last line\n")
	write(t, s, "big.log", sb.String())
	notes, _ := s.Scan()
	if len(notes) != 1 || notes[0].Err != nil {
		t.Fatalf("a large log is still shown: %+v", notes)
	}
	body := notes[0].Doc.Body
	if !strings.HasSuffix(body, "the last line\n") || len(body) > logTail || !strings.HasPrefix(body, "an old line") {
		t.Errorf("a large log shows whole lines from its end: %d bytes, starts %q", len(body), body[:20])
	}
}

func TestApplyAndReadWorkOnAPage(t *testing.T) {
	s := newStore(t)
	write(t, s, "docs/plan.md", "- [ ] a\n")
	if b, err := s.Read("docs/plan.md"); err != nil || string(b) != "- [ ] a\n" {
		t.Errorf("Read = %q, %v", b, err)
	}
	if err := s.Write("docs/plan.md", []byte("changed\n")); err != nil || read(t, s, "docs/plan.md") != "changed\n" {
		t.Errorf("Write: %v", err)
	}
	if err := s.Write("fresh/page.md", []byte("new\n")); err != nil || read(t, s, "fresh/page.md") != "new\n" {
		t.Errorf("Write makes the folder of a new page: %v", err)
	}
}

func TestWatchSeesChangesInsideAFolderAndBehindALink(t *testing.T) {
	s := newStore(t)
	write(t, s, "docs/a.md", "a\n")
	outside := filepath.Join(filepath.Dir(s.Dir), "README.md")
	if err := os.WriteFile(outside, []byte("readme\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(s.Dir, "readme.md")); err != nil {
		t.Skip("symlinks are not available:", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := s.Watch(ctx)
	if err != nil {
		t.Fatal(err)
	}
	signalled := func() bool {
		select {
		case <-ch:
			return true
		case <-time.After(2 * time.Second):
			return false
		}
	}
	write(t, s, "docs/a.md", "changed\n")
	if !signalled() {
		t.Error("a change to a page should be signalled")
	}
	if err := os.WriteFile(outside, []byte("readme changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !signalled() {
		t.Error("a change to a linked file should be signalled")
	}
	write(t, s, "later/b.log", "first\n")
	if !signalled() {
		t.Fatal("a new folder should be signalled")
	}
	time.Sleep(3 * Debounce) // let the watcher notice the new folder
	for len(ch) > 0 {
		<-ch
	}
	write(t, s, "later/b.log", "first\nsecond\n")
	if !signalled() {
		t.Error("a change inside a folder made while watching should be signalled")
	}
}
