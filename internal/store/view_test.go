package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func yes() *bool { b := true; return &b }

func TestViewsAreKeptInStickyJSON(t *testing.T) {
	s := newStore(t)
	write(t, s, "a.md", "a\n")
	write(t, s, "build.log", "x\n")
	if v, err := s.Views(); err != nil || len(v) != 0 {
		t.Fatalf("without the file there are no views: %v, %v", v, err)
	}
	if err := s.SetView("a.md", func(v *View) { v.Open, v.Size = yes(), "half" }); err != nil {
		t.Fatal(err)
	}
	if err := s.SetView("build.log", func(v *View) { v.Rows, v.Color, v.Pin = 12, "blue", yes() }); err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"notes\": {\n    \"a.md\": {\n      \"open\": true,\n      \"size\": \"half\"\n    },\n    \"build.log\": {\n      \"pin\": true,\n      \"rows\": 12,\n      \"color\": \"blue\"\n    }\n  }\n}\n"
	if got := read(t, s, ViewFile); got != want {
		t.Errorf("sticky.json =\n%s\nwant\n%s", got, want)
	}
	views, err := s.Views()
	if err != nil || views["a.md"].Size != "half" || views["a.md"].Open == nil || !*views["a.md"].Open || views["build.log"].Rows != 12 {
		t.Errorf("Views = %+v, %v", views, err)
	}
}

func TestSetViewDropsWhatIsEmptyOrGone(t *testing.T) {
	s := newStore(t)
	write(t, s, "a.md", "a\n")
	write(t, s, "docs/p.md", "p\n")
	write(t, s, ViewFile, `{"notes":{"a.md":{"size":"half"},"gone.md":{"open":true},"docs":{"open":true}},"theme":"keep me"}`)
	if err := s.SetView("a.md", func(v *View) { v.Size = "" }); err != nil {
		t.Fatal(err)
	}
	got := read(t, s, ViewFile)
	if strings.Contains(got, "a.md") || strings.Contains(got, "gone.md") {
		t.Errorf("an entry that says nothing, or whose note is gone, is dropped:\n%s", got)
	}
	if !strings.Contains(got, `"docs"`) || !strings.Contains(got, `"theme": "keep me"`) {
		t.Errorf("a book's entry and keys this version does not know must stay:\n%s", got)
	}
}

func TestABrokenStickyJSONIsReportedAndLeftAlone(t *testing.T) {
	s := newStore(t)
	write(t, s, "a.md", "a\n")
	write(t, s, ViewFile, "{ this is not json")
	if v, err := s.Views(); err == nil || len(v) != 0 {
		t.Errorf("Views = %v, %v", v, err)
	}
	if err := s.SetView("a.md", func(v *View) { v.Size = "half" }); err == nil {
		t.Error("a file that cannot be read must not be overwritten")
	}
	if got := read(t, s, ViewFile); got != "{ this is not json" {
		t.Errorf("the broken file was changed: %q", got)
	}
}

func TestStickyJSONIsNotANote(t *testing.T) {
	s := newStore(t)
	write(t, s, ViewFile, "{}")
	if notes, _ := s.Scan(); len(notes) != 0 {
		t.Errorf("Scan = %+v", notes)
	}
	if _, err := os.Stat(filepath.Join(s.Dir, ViewFile)); err != nil {
		t.Fatal(err)
	}
}

func TestOrderIsKeptNextToTheViews(t *testing.T) {
	s := newStore(t)
	write(t, s, "a.md", "a\n")
	if got := s.Order(); len(got) != 0 {
		t.Fatalf("Order = %v", got)
	}
	if err := s.SetView("a.md", func(v *View) { v.Size = "half" }); err != nil {
		t.Fatal(err)
	}
	if err := s.SetOrder([]string{"b.md", "a.md"}); err != nil {
		t.Fatal(err)
	}
	if got := s.Order(); strings.Join(got, ",") != "b.md,a.md" {
		t.Errorf("Order = %v", got)
	}
	if v, _ := s.Views(); v["a.md"].Size != "half" {
		t.Errorf("writing the order must keep the views: %+v", v)
	}
	write(t, s, ViewFile, "{ broken")
	if err := s.SetOrder([]string{"a.md"}); err == nil || s.Order() != nil {
		t.Error("a broken file is neither read nor overwritten")
	}
}
