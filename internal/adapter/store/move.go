package store

// Where a note goes: archived, trashed, restored, moved or deleted. None
// of it writes over another note.

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/adapter/atomicfile"
)

// put moves the note called name to dst, a path inside the notes folder.
// It never writes over what is there: a taken name gets "-2", "-3" and so
// on. It returns where the note went. A folder that is left without
// anything in it is removed.
func (s *Store) put(name, dst string) (to string, err error) {
	err = s.locked(func() error {
		to, err = s.putLocked(name, dst)
		return err
	})
	return to, err
}

func (s *Store) putLocked(name, dst string) (string, error) {
	src := s.Path(name)
	if _, err := os.Lstat(src); err != nil {
		return "", err
	}
	ext := path.Ext(dst)
	base := strings.TrimSuffix(dst, ext)
	for i := 1; ; i++ {
		target := dst
		if i > 1 {
			target = fmt.Sprintf("%s-%d%s", base, i, ext)
		}
		full := s.Path(target)
		if _, err := os.Lstat(full); err == nil {
			continue
		} else if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
		if err := os.MkdirAll(filepath.Dir(full), atomicfile.DirMode); err != nil {
			return "", err
		}
		if err := os.Rename(src, full); err != nil {
			return "", err
		}
		s.tidy(name)
		return target, nil
	}
}

// tidy removes the folder a note was in when the folder is now empty.
func (s *Store) tidy(name string) {
	if dir := path.Dir(name); dir != "." && dir != "/" {
		_ = os.Remove(s.Path(dir)) // fails, rightly, unless empty
	}
}

// Archive moves a note out of sight into the archive folder and returns
// where it went. The archive is one flat folder: a page leaves its book.
func (s *Store) Archive(name string) (string, error) {
	return s.put(name, ArchiveDir+"/"+path.Base(name))
}

// Trash moves a note, a page or a whole book to the trash folder and
// returns where it went. Nothing is removed from the disk: Restore brings it
// back. The trash keeps the note's place, so a page goes back to its book.
func (s *Store) Trash(name string) (string, error) {
	return s.put(name, TrashDir+"/"+name)
}

// Restore moves what Trash or Archive put at from back to name. It does
// not write over a note that is there.
func (s *Store) Restore(from, name string) error {
	clean := path.Clean(from)
	if clean != from || !(strings.HasPrefix(from, TrashDir+"/") || strings.HasPrefix(from, ArchiveDir+"/")) {
		return fmt.Errorf("%s is not in the trash or the archive", from)
	}
	return s.Move(from, name)
}

// Trashed returns where in the trash the note called name is. When several
// notes of that name were deleted it is the one deleted last.
func (s *Store) Trashed(name string) (string, bool) {
	ext := path.Ext(name)
	stem := path.Base(strings.TrimSuffix(name, ext))
	dir := path.Dir(TrashDir + "/" + name)
	entries, err := os.ReadDir(s.Path(dir))
	if err != nil {
		return "", false
	}
	best, found := 0, ""
	for _, e := range entries {
		rest, ok := strings.CutPrefix(strings.TrimSuffix(e.Name(), ext), stem)
		if !ok || !strings.HasSuffix(e.Name(), ext) {
			continue
		}
		n := 1
		if rest != "" {
			if n, err = strconv.Atoi(strings.TrimPrefix(rest, "-")); err != nil || !strings.HasPrefix(rest, "-") {
				continue
			}
		}
		if n > best {
			best, found = n, dir+"/"+e.Name()
		}
	}
	return found, best > 0
}

// Move renames a note: under a new name, into a book, or out of one. It
// does not write over a note that is there.
func (s *Store) Move(name, to string) error {
	return s.locked(func() error { return s.move(name, to) })
}

func (s *Store) move(name, to string) error {
	src := s.Path(name)
	dst := s.Path(to)
	if _, err := os.Lstat(src); err != nil {
		return err
	}
	if _, err := os.Lstat(dst); err == nil {
		return fmt.Errorf("%s: %w", to, fs.ErrExist)
	}
	if err := os.MkdirAll(filepath.Dir(dst), atomicfile.DirMode); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err != nil {
		return err
	}
	s.tidy(name)
	return nil
}

// MoveArranged moves a note as Move does, and gives the new name what
// sticky.json arranged for the old one. It is all or nothing: when the
// arrangement cannot be written, the note is moved back and the error
// returned. A note with nothing arranged, or whose tab's file cannot be
// read, is only moved.
func (s *Store) MoveArranged(name, to string) error {
	return s.locked(func() error { return s.moveArranged(name, to) })
}

func (s *Store) moveArranged(name, to string) error {
	tab, inside := s.split(name)
	_, views, err := s.readViews(tab)
	v, had := views[inside]
	if err := s.move(name, to); err != nil {
		return err
	}
	if err != nil || !had {
		return nil
	}
	if err := s.setView(to, func(n *View) { *n = v }); err != nil {
		if back := s.move(to, name); back != nil {
			return fmt.Errorf("%w; and %s could not be moved back from %s: %v", err, name, to, back)
		}
		return err
	}
	return nil
}

// Delete removes a note for good.
func (s *Store) Delete(name string) error {
	return os.Remove(s.Path(name))
}
