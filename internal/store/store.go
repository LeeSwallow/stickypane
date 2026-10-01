// Package store reads, writes and watches the notes folder.
package store

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/LeeSwallow/stickypane/internal/doc"
)

const (
	// DirName is the notes folder inside a project.
	DirName = ".stickypane"
	// ArchiveDir is where detached notes go, inside DirName.
	ArchiveDir = "archive"
	// MaxSize is the largest file read in full.
	MaxSize = 1 << 20

	logTail   = 64 << 10 // how much of an oversized log is shown
	slugRunes = 40
	tmpPrefix = ".stickypane-tmp-"
)

var (
	// ErrNotFound means no notes folder exists at or above the start path.
	ErrNotFound = errors.New("no " + DirName + " directory found")
	// ErrTooLarge marks a note that is too big to show.
	ErrTooLarge = errors.New("file is larger than 1 MB")
)

// Note is one file in the notes folder.
type Note struct {
	Name    string // file name, such as "10-plan.md"
	Path    string
	Doc     doc.Document
	ModTime time.Time
	Err     error // why the note cannot be shown; Doc is empty when set
}

// Store is a notes folder.
type Store struct{ Dir string }

// Open returns the store for a notes folder.
func Open(dir string) *Store { return &Store{Dir: dir} }

// Find walks up from start until it finds a notes folder.
func Find(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, DirName)
		if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNotFound
		}
		dir = parent
	}
}

// Resolve accepts either a notes folder itself or any path inside a project.
func Resolve(arg string) (string, error) {
	abs, err := filepath.Abs(arg)
	if err != nil {
		return "", err
	}
	if filepath.Base(abs) == DirName {
		if fi, err := os.Stat(abs); err == nil && fi.IsDir() {
			return abs, nil
		}
	}
	return Find(abs)
}

// Scan reads every note, sorted by file name. A file that cannot be read
// still appears, carrying its error.
func (s *Store) Scan() ([]Note, error) {
	entries, err := os.ReadDir(s.Dir) // sorted by name
	if err != nil {
		return nil, err
	}
	var notes []Note
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || strings.HasPrefix(name, ".") || !strings.EqualFold(filepath.Ext(name), ".md") {
			continue
		}
		notes = append(notes, s.read(name))
	}
	return notes, nil
}

func (s *Store) read(name string) Note {
	n := Note{Name: name, Path: filepath.Join(s.Dir, name)}
	fi, err := os.Stat(n.Path)
	if err != nil {
		n.Err = err
		return n
	}
	n.ModTime = fi.ModTime()
	if fi.Size() <= MaxSize {
		b, err := os.ReadFile(n.Path)
		if err != nil {
			n.Err = err
			return n
		}
		n.Doc = doc.Parse(displayable(b))
		return n
	}
	head, tail, err := readEnds(n.Path, fi.Size())
	if err != nil {
		n.Err = err
		return n
	}
	d := doc.Parse(displayable(head))
	if d.Type() != "log" {
		n.Err = ErrTooLarge
		return n
	}
	if i := bytes.IndexByte(tail, '\n'); i >= 0 {
		tail = tail[i+1:] // start at a line boundary
	}
	d.Body = string(displayable(tail))
	n.Doc = d
	return n
}

// displayable replaces invalid UTF-8 so the text is safe to draw. Only what
// the screen shows is changed; Apply always works on the file's real bytes.
func displayable(b []byte) []byte { return bytes.ToValidUTF8(b, []byte("\uFFFD")) }

// readEnds returns the first and last logTail bytes of a large file.
func readEnds(path string, size int64) (head, tail []byte, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	head = make([]byte, logTail)
	n, err := io.ReadFull(f, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, nil, err
	}
	head = head[:n]
	tail = make([]byte, logTail)
	n, err = f.ReadAt(tail, size-logTail)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, nil, err
	}
	return head, tail[:n], nil
}

// Apply reads the file as it is now, applies op and replaces the file in one
// step. The edit is never based on what the screen last saw, so changes made
// by someone else in the meantime survive. A missing file is a conflict.
func (s *Store) Apply(name string, op doc.Op) error {
	path := filepath.Join(s.Dir, name)
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return doc.ErrConflict
	}
	if err != nil {
		return err
	}
	out, err := op.Apply(doc.Parse(b))
	if err != nil {
		return err
	}
	return writeAtomic(path, out.Bytes())
}

func writeAtomic(path string, data []byte) error {
	mode := fs.FileMode(0o644)
	if fi, err := os.Stat(path); err == nil {
		mode = fi.Mode().Perm()
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), tmpPrefix+"*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // a no-op once the rename succeeded
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// Create writes a new note named after text and returns the file name. It
// never overwrites: a taken name gets "-2", "-3" and so on.
func (s *Store) Create(text string, content []byte, now time.Time) (string, error) {
	base := Slug(text, now)
	for i := 1; ; i++ {
		name := base + ".md"
		if i > 1 {
			name = fmt.Sprintf("%s-%d.md", base, i)
		}
		f, err := os.OpenFile(filepath.Join(s.Dir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		_, werr := f.Write(content)
		if cerr := f.Close(); werr == nil {
			werr = cerr
		}
		return name, werr
	}
}

// Slug turns text into a file name stem: letters and digits are kept (Latin
// letters lower-cased), runs of spaces, "-" and "_" become one "-", and
// everything else is dropped. An empty result becomes "note-<timestamp>".
func Slug(text string, now time.Time) string {
	var b strings.Builder
	n, dash := 0, true // dash starts true so the slug never begins with "-"
	for _, r := range text {
		if n >= slugRunes {
			break
		}
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
			n, dash = n+1, false
		case (unicode.IsSpace(r) || r == '-' || r == '_') && !dash:
			b.WriteRune('-')
			n, dash = n+1, true
		}
	}
	if slug := strings.TrimRight(b.String(), "-"); slug != "" {
		return slug
	}
	return "note-" + now.Format("20060102-150405")
}

// Archive moves a note into the archive folder without overwriting.
func (s *Store) Archive(name string) error {
	dir := filepath.Join(s.Dir, ArchiveDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for i := 1; ; i++ {
		target := name
		if i > 1 {
			target = fmt.Sprintf("%s-%d%s", base, i, ext)
		}
		dst := filepath.Join(dir, target)
		if _, err := os.Lstat(dst); err == nil {
			continue
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return os.Rename(filepath.Join(s.Dir, name), dst)
	}
}

// Delete removes a note for good.
func (s *Store) Delete(name string) error {
	return os.Remove(filepath.Join(s.Dir, name))
}
