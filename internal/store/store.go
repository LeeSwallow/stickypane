// Package store reads, writes and watches the notes folder.
package store

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/LeeSwallow/stickypane/internal/doc"
)

const (
	// DirName is the notes folder inside a project.
	DirName = ".sticky"
	// LegacyDirName is what the folder was called before. A project that
	// has it keeps working.
	LegacyDirName = ".stickypane"
	// ArchiveDir is where detached notes go, inside DirName.
	ArchiveDir = "archive"
	// TrashDir is where deleted notes go, inside DirName, until restored.
	TrashDir = ".trash"
	// MaxSize is the largest file read in full.
	MaxSize = 1 << 20

	logTail = 64 << 10 // how much of an oversized log is shown
	// ViewFile holds how the notes are arranged on the screen.
	ViewFile  = "sticky.json"
	slugRunes = 40
	tmpPrefix = ".stickypane-tmp-"
)

var (
	// ErrNotFound means no notes folder exists at or above the start path.
	ErrNotFound = errors.New("no " + DirName + " directory found")
	// ErrTooLarge marks a note that is too big to show.
	ErrTooLarge = errors.New("file is larger than 1 MB")
)

// extensions are the files that are notes. Only Markdown has front matter;
// the others are shown as they are.
var extensions = map[string]bool{".md": true, ".log": true, ".txt": true, ".out": true, ".sh": true}

// Linked reports whether the note is a symbolic link to a file elsewhere.
// Such a file belongs to the project, not to the board: what the board
// arranges about it is kept in sticky.json, never written into it.
func (s *Store) Linked(name string) bool {
	fi, err := os.Lstat(filepath.Join(s.Dir, filepath.FromSlash(name)))
	return err == nil && fi.Mode()&os.ModeSymlink != 0
}

// IsNote reports whether a file name is one the board shows.
func IsNote(name string) bool {
	base := filepath.Base(name)
	return !strings.HasPrefix(base, ".") && extensions[strings.ToLower(filepath.Ext(base))]
}

// isMarkdown reports whether a file may start with front matter.
func isMarkdown(name string) bool { return strings.EqualFold(filepath.Ext(name), ".md") }

// Note is one file in the notes folder, or one folder: a book, whose pages
// are the files in it.
type Note struct {
	// Name is the path inside the notes folder: "10-plan.md", "docs" for a
	// book, "docs/intro.md" for one of its pages.
	Name    string
	Path    string
	Doc     doc.Document
	ModTime time.Time
	Err     error  // why the note cannot be shown; Doc is empty when set
	Pages   []Note // a book's pages, by file name; nil for a file
}

// Book reports whether the note is a folder of pages.
func (n Note) Book() bool { return n.Pages != nil }

// Store is a notes folder.
type Store struct {
	Dir string

	mu    sync.Mutex
	known map[string]readNote // notes read by the last Load, by path
	fresh map[string]readNote // what this Load has read so far
}

// readNote is a note as read, with what its file looked like then. A file
// whose size and time are the same is not read again.
type readNote struct {
	size int64
	mod  time.Time
	note Note
}

// isDir reports whether a folder entry is a folder, following a symbolic
// link only when the entry is one: the entry already says what it is.
func isDir(e os.DirEntry, path string) bool {
	if e.Type()&fs.ModeSymlink == 0 {
		return e.IsDir()
	}
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

// Open returns the store for a notes folder.
func Open(dir string) *Store { return &Store{Dir: dir} }

// Find walks up from start until it finds a notes folder.
func Find(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		for _, name := range []string{DirName, LegacyDirName} {
			candidate := filepath.Join(dir, name)
			if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
				return candidate, nil
			}
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
	if base := filepath.Base(abs); base == DirName || base == LegacyDirName {
		if fi, err := os.Stat(abs); err == nil && fi.IsDir() {
			return abs, nil
		}
	}
	return Find(abs)
}

// Tab is one screen of the board: the notes folder itself (the root tab,
// named after the project) or one folder in it. Its notes are the files in
// the folder, and a folder in it is a book whose pages are its files.
type Tab struct {
	Name  string // "" for the root, else the folder's name
	Title string // the folder's name, or what its sticky.json calls it
	Notes []Note
}

// Board is the board as read at one moment: its tabs and what the settings
// files say about them.
type Board struct {
	Tabs     []Tab
	Settings Settings
}

// Load reads the board in one pass: every settings file once, then the
// notes of every tab. It is what a screen reads on each change.
func (s *Store) Load() (Board, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	set := s.Settings()
	s.fresh = map[string]readNote{}
	tabs, err := s.scan(set)
	// Only what this load saw is kept, so a removed note is forgotten.
	s.known, s.fresh = s.fresh, nil
	return Board{Tabs: tabs, Settings: set}, err
}

// Tabs reads the board: the root tab, then one tab per folder, in name
// order. Folders are read two levels deep: a folder in a tab is a book.
// The archive, hidden folders and ignored names are not read; a folder
// with nothing to show is not a tab. A file that cannot be read still
// appears, carrying its error.
func (s *Store) Tabs() ([]Tab, error) {
	b, err := s.Load()
	return b.Tabs, err
}

func (s *Store) scan(set Settings) ([]Tab, error) {
	ignored := set.ignored()
	root := Tab{Title: filepath.Base(filepath.Dir(s.Dir))}
	if title := set.TabTitle(""); title != "" {
		root.Title = title
	}
	tabs := []Tab{root}
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || name == ArchiveDir || ignored(name) {
			continue
		}
		if isDir(e, filepath.Join(s.Dir, name)) {
			tab := Tab{Name: name, Title: name}
			if title := set.TabTitle(name); title != "" {
				tab.Title = title
			}
			tab.Notes = s.notesIn(name, ignored, true)
			if len(tab.Notes) > 0 {
				tabs = append(tabs, tab)
			}
			continue
		}
		if IsNote(name) {
			tabs[0].Notes = append(tabs[0].Notes, s.read(name))
		}
	}
	return tabs, nil
}

// notesIn reads the notes of a folder: its files, and when books is set,
// its folders as books.
func (s *Store) notesIn(dir string, ignored func(string) bool, books bool) []Note {
	entries, err := os.ReadDir(filepath.Join(s.Dir, filepath.FromSlash(dir)))
	if err != nil {
		return nil
	}
	var notes []Note
	for _, e := range entries {
		name := e.Name()
		full := dir + "/" + name
		if strings.HasPrefix(name, ".") || ignored(full) {
			continue
		}
		if isDir(e, filepath.Join(s.Dir, filepath.FromSlash(full))) {
			if books {
				if book, ok := s.book(full, ignored); ok {
					notes = append(notes, book)
				}
			}
			continue
		}
		if IsNote(name) {
			notes = append(notes, s.read(full))
		}
	}
	return notes
}

// Scan reads every note of every tab as one list, sorted by name, for
// callers that do not show screens: the command line and the MCP server.
func (s *Store) Scan() ([]Note, error) {
	tabs, err := s.Tabs()
	if err != nil {
		return nil, err
	}
	var notes []Note
	for _, t := range tabs {
		notes = append(notes, t.Notes...)
	}
	return notes, nil
}

// book reads a folder as one note. A folder without notes is not a book.
func (s *Store) book(name string, ignored func(string) bool) (Note, bool) {
	b := Note{Name: name, Path: filepath.Join(s.Dir, filepath.FromSlash(name))}
	b.Pages = s.notesIn(name, ignored, false)
	for _, p := range b.Pages {
		if p.ModTime.After(b.ModTime) {
			b.ModTime = p.ModTime
		}
	}
	return b, len(b.Pages) > 0
}

// remember keeps a note read during a Load for the next one.
func (s *Store) remember(r readNote) {
	if s.fresh != nil {
		s.fresh[r.note.Path] = r
	}
}

func (s *Store) read(name string) Note {
	n := Note{Name: name, Path: filepath.Join(s.Dir, filepath.FromSlash(name))}
	fi, err := os.Stat(n.Path)
	if err != nil {
		n.Err = err
		return n
	}
	n.ModTime = fi.ModTime()
	if old, ok := s.known[n.Path]; ok && old.size == fi.Size() && old.mod.Equal(n.ModTime) && old.note.Err == nil {
		s.remember(old)
		return old.note
	}
	defer func() { s.remember(readNote{fi.Size(), n.ModTime, n}) }()
	parse := func(b []byte) doc.Document {
		if isMarkdown(name) {
			return doc.Parse(displayable(b))
		}
		return doc.Document{Body: string(displayable(b))}
	}
	if fi.Size() <= MaxSize {
		b, err := readFile(n.Path)
		if err != nil {
			n.Err = err
			return n
		}
		n.Doc = parse(b)
		return n
	}
	// Too large to read whole. A log is still worth showing: its end is
	// what matters.
	head, tail, err := readEnds(n.Path, fi.Size())
	if err != nil {
		n.Err = err
		return n
	}
	d := parse(head)
	if isMarkdown(name) && d.Type() != "log" {
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
//
// A note that is a symlink is edited where it really lives: replacing the
// link itself would silently detach the note from its file.
func (s *Store) Apply(name string, op doc.Op) error {
	path := filepath.Join(s.Dir, filepath.FromSlash(name))
	if target, err := filepath.EvalSymlinks(path); err == nil {
		path = target
	}
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

// Read returns a note's bytes exactly as they are on disk.
func (s *Store) Read(name string) ([]byte, error) {
	return os.ReadFile(filepath.Join(s.Dir, filepath.FromSlash(name)))
}

// Write replaces a note with content in one step, or creates it. Unlike
// Apply it does not look at what is there: it is for callers that hand over
// a whole note, such as the command line and the MCP server.
func (s *Store) Write(name string, content []byte) error {
	path := filepath.Join(s.Dir, filepath.FromSlash(name))
	if target, err := filepath.EvalSymlinks(path); err == nil {
		path = target
	} else if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err // a new page of a new book needs its folder
	}
	return writeAtomic(path, content)
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

// Create writes a new note named after text, with the extension ext, and
// returns the file name. It never overwrites: a taken name gets "-2", "-3"
// and so on.
func (s *Store) Create(text, ext string, content []byte, now time.Time) (string, error) {
	base := Slug(text, now)
	for i := 1; ; i++ {
		name := base + ext
		if i > 1 {
			name = fmt.Sprintf("%s-%d%s", base, i, ext)
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

// put moves the note called name to dst, a path inside the notes folder.
// It never writes over what is there: a taken name gets "-2", "-3" and so
// on. It returns where the note went. A folder that is left without
// anything in it is removed.
func (s *Store) put(name, dst string) (string, error) {
	src := filepath.Join(s.Dir, filepath.FromSlash(name))
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
		full := filepath.Join(s.Dir, filepath.FromSlash(target))
		if _, err := os.Lstat(full); err == nil {
			continue
		} else if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
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
		_ = os.Remove(filepath.Join(s.Dir, filepath.FromSlash(dir))) // fails, rightly, unless empty
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
	entries, err := os.ReadDir(filepath.Join(s.Dir, filepath.FromSlash(dir)))
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
	src := filepath.Join(s.Dir, filepath.FromSlash(name))
	dst := filepath.Join(s.Dir, filepath.FromSlash(to))
	if _, err := os.Lstat(src); err != nil {
		return err
	}
	if _, err := os.Lstat(dst); err == nil {
		return fmt.Errorf("%s: %w", to, fs.ErrExist)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err != nil {
		return err
	}
	s.tidy(name)
	return nil
}

// Delete removes a note for good.
func (s *Store) Delete(name string) error {
	return os.Remove(filepath.Join(s.Dir, filepath.FromSlash(name)))
}
