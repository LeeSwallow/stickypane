package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/doc"
)

func TestApplyFollowsASymlinkedNote(t *testing.T) {
	s := newStore(t)
	project := filepath.Dir(s.Dir)
	real := filepath.Join(project, "TODO.md")
	if err := os.WriteFile(real, []byte("real content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(s.Dir, "todo.md")
	if err := os.Symlink(real, link); err != nil {
		t.Skip("symlinks are not available:", err)
	}
	if err := s.Apply("todo.md", doc.SetKey{Key: "pin", Value: "true"}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(real)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(b); got != "---\npin: true\n---\nreal content\n" {
		t.Errorf("the real file was not updated: %q", got)
	}
	if fi, err := os.Lstat(link); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("the note must stay a symlink, got mode %v, err %v", fi.Mode(), err)
	}
	for _, dir := range []string{s.Dir, project} {
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			if name := e.Name(); name != "todo.md" && name != "TODO.md" && name != DirName {
				t.Errorf("unexpected file left in %s: %s", dir, name)
			}
		}
	}
}
