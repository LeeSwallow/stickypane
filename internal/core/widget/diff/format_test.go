package diff

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite testdata/*.want from the parser")

// dump writes what Parse read, one fact a line, for the .want beside each
// fixture.
func dump(files []File) string {
	var b strings.Builder
	for _, f := range files {
		add, del := f.Counts()
		fmt.Fprintf(&b, "file %s old=%q new=%q", f.Status, f.Old, f.New)
		if f.OldMode != "" || f.NewMode != "" {
			fmt.Fprintf(&b, " mode=%s→%s", f.OldMode, f.NewMode)
		}
		if f.Similarity != "" {
			fmt.Fprintf(&b, " similarity=%s", f.Similarity)
		}
		if f.Binary {
			b.WriteString(" binary")
		}
		fmt.Fprintf(&b, " +%d -%d\n", add, del)
		for _, h := range f.Hunks {
			fmt.Fprintf(&b, "  %s\n", h.Header)
			for _, l := range h.Lines {
				fmt.Fprintf(&b, "    %c|%s\n", l.Op, l.Text)
			}
		}
	}
	return b.String()
}

// The fixtures are real `git diff` output (git 2.x, -M, no color, no
// external diff): each .diff beside the .want of what it reads as.
func TestParseReadsRealGitDiffs(t *testing.T) {
	t.Parallel()
	diffs, _ := filepath.Glob("testdata/*.diff")
	if len(diffs) < 9 {
		t.Fatalf("fixtures missing: %v", diffs)
	}
	for _, p := range diffs {
		t.Run(filepath.Base(p), func(t *testing.T) {
			src, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			got := dump(Parse(string(src)))
			want := strings.TrimSuffix(p, ".diff") + ".want"
			if *update {
				os.WriteFile(want, []byte(got), 0o644)
			}
			w, err := os.ReadFile(want)
			if err != nil {
				t.Fatal(err)
			}
			if got != string(w) {
				t.Errorf("Parse(%s):\n%s\nwant:\n%s", p, got, w)
			}
		})
	}
}

// What the fixtures stand for, said once more in code, so a .want
// rewritten by -update cannot quietly change them.
func TestParseFacts(t *testing.T) {
	t.Parallel()
	read := func(name string) File {
		b, err := os.ReadFile(filepath.Join("testdata", name+".diff"))
		if err != nil {
			t.Fatal(err)
		}
		files := Parse(string(b))
		if len(files) != 1 {
			t.Fatalf("%s: %d files", name, len(files))
		}
		return files[0]
	}
	if f := read("rename"); f.Status != "R" || f.Old != "moved.txt" || f.New != "moved-to.txt" || f.Similarity != "100%" || len(f.Hunks) != 0 {
		t.Errorf("rename = %+v", f)
	}
	if f := read("rename-edit"); f.Status != "R" || f.Old != "renamed-edit.txt" || f.New != "renamed-done.txt" || f.Similarity != "86%" || len(f.Hunks) != 1 {
		t.Errorf("rename with an edit = %+v", f)
	}
	if f := read("binary"); !f.Binary || len(f.Hunks) != 0 || f.Path() != "image.bin" {
		t.Errorf("binary = %+v", f)
	}
	f := read("no-newline")
	if add, del := f.Counts(); add != 1 || del != 0 {
		t.Errorf("no newline counts +%d -%d, want +1 -0", add, del)
	}
	if l := f.Hunks[0].Lines; l[len(l)-1].Op != '\\' || l[len(l)-2].Text != "c" {
		t.Errorf("no newline lines = %+v", l)
	}
	if f := read("mode"); f.OldMode != "100644" || f.NewMode != "100755" || len(f.Hunks) != 0 || f.Status != "M" {
		t.Errorf("mode = %+v", f)
	}
	if f := read("mode-edit"); f.OldMode != "100644" || f.NewMode != "100755" || len(f.Hunks) != 1 {
		t.Errorf("mode with an edit = %+v", f)
	}
	if f := read("added"); f.Status != "A" || f.Old != "" || f.New != "added.txt" {
		t.Errorf("added = %+v", f)
	}
	if f := read("deleted"); f.Status != "D" || f.Old != "gone.txt" || f.New != "" || f.Path() != "gone.txt" {
		t.Errorf("deleted = %+v", f)
	}
	if files := Parse(""); len(files) != 0 {
		t.Errorf("empty = %+v", files)
	}
}

// A removed line that reads "-- x" is a line of its hunk, not the ---
// line of a next file: the hunk's header says how many lines it holds.
func TestParseCountsHunkLines(t *testing.T) {
	t.Parallel()
	src := "diff --git a/x b/x\n--- a/x\n+++ b/x\n@@ -1,2 +1 @@\n--- a/y\n-++ b/y\ndiff --git a/z b/z\n"
	files := Parse(src)
	if len(files) != 2 || files[0].Old != "x" || len(files[0].Hunks[0].Lines) != 2 || files[0].Hunks[0].Lines[0].Text != "-- a/y" {
		t.Fatalf("Parse = %s", dump(files))
	}
}

// A path git quotes, and one with a space, read back as they are.
func TestParseQuotedAndSpacedPaths(t *testing.T) {
	t.Parallel()
	files := Parse("diff --git \"a/tab\\there\" \"b/tab\\there\"\nold mode 100644\nnew mode 100755\ndiff --git a/a b/c b/a b/c\nold mode 100644\nnew mode 100755\n")
	if len(files) != 2 || files[0].Path() != "tab\there" || files[1].Path() != "a b/c" {
		t.Fatalf("Parse = %s", dump(files))
	}
}

// A diff of thousands of lines reads in one pass and keeps every line.
func TestParseALargeDiff(t *testing.T) {
	t.Parallel()
	var b strings.Builder
	b.WriteString("diff --git a/big b/big\n--- a/big\n+++ b/big\n@@ -1,5000 +1,5000 @@\n")
	for i := range 5000 {
		fmt.Fprintf(&b, "-old %d\n+new %d\n", i, i)
	}
	f := Parse(b.String())
	if add, del := f[0].Counts(); add != 5000 || del != 5000 {
		t.Fatalf("counts +%d -%d", add, del)
	}
}
