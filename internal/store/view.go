package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// View is how a note is arranged on the screen: what the user did with the
// keys that open, size, color and pin a note. A field that is nil, empty or
// zero says nothing, and the note's own front matter or its kind decides.
//
// Views are kept in a sticky.json next to the notes instead of in each note,
// for three reasons: a log, a script or a linked README has no front matter
// to keep them in; the board then never rewrites a note just because the
// user rearranged the screen; and the file can be committed to share a
// layout. Every tab (a folder of the notes folder) has a file of its own
// for its notes, the way Bruno keeps a folder's settings in the folder; the
// root file also holds what is global: the theme, the language, the ignore
// list and the active tab.
type View struct {
	// Title is the name shown for a note that cannot carry one itself: a
	// book, a log, a script. Renaming it does not rename the file.
	Title string `json:"title,omitempty"`
	Open  *bool  `json:"open,omitempty"`
	Pin   *bool  `json:"pin,omitempty"`
	Size  string `json:"size,omitempty"`
	Rows  int    `json:"rows,omitempty"`
	Color string `json:"color,omitempty"`
}

func (v View) empty() bool {
	return v.Title == "" && v.Open == nil && v.Pin == nil && v.Size == "" && v.Rows == 0 && v.Color == ""
}

// Views maps a note's full name ("deploy/build.log") to its view.
type Views map[string]View

// The keys this version reads. Other keys are kept as they are, so a later
// version can add to the file.
const (
	viewKey    = "notes"
	orderKey   = "order"
	versionKey = "version"
	ignoreKey  = "ignore"
	themeKey   = "theme"
	langKey    = "language"
	tabKey     = "tab"
	titleKey   = "title"
)

// readFile reads a settings file. Tests count the reads through it.
var readFile = os.ReadFile

// split returns the tab a note's name is in ("" for the root) and the
// name inside that tab, when the tab exists as a folder.
func (s *Store) split(name string) (tab, inside string) {
	if i := strings.Index(name, "/"); i > 0 {
		if fi, err := os.Stat(filepath.Join(s.Dir, name[:i])); err == nil && fi.IsDir() {
			return name[:i], name[i+1:]
		}
	}
	return "", name
}

// TabOf returns the tab a note's name is in: "" for the root.
func (s *Store) TabOf(name string) string {
	tab, _ := s.split(name)
	return tab
}

// viewPath is the settings file of a tab.
func (s *Store) viewPath(tab string) string {
	return filepath.Join(s.Dir, filepath.FromSlash(tab), ViewFile)
}

// readViews returns a tab's file: its top-level keys and the views among
// them, keyed by the name inside the tab.
func (s *Store) readViews(tab string) (map[string]json.RawMessage, Views, error) {
	top := map[string]json.RawMessage{}
	views := Views{}
	b, err := readFile(s.viewPath(tab))
	if errors.Is(err, fs.ErrNotExist) {
		return top, views, nil
	}
	if err != nil {
		return top, views, err
	}
	where := path.Join(tab, ViewFile)
	if err := json.Unmarshal(b, &top); err != nil {
		return top, views, fmt.Errorf("%s is not valid JSON: %w", where, err)
	}
	if raw, ok := top[viewKey]; ok {
		if err := json.Unmarshal(raw, &views); err != nil {
			return top, Views{}, fmt.Errorf("%s: %q is not a map of notes: %w", where, viewKey, err)
		}
	}
	return top, views, nil
}

func (s *Store) writeViews(tab string, top map[string]json.RawMessage) error {
	if _, ok := top[versionKey]; !ok {
		top[versionKey] = json.RawMessage("1")
	}
	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(top); err != nil {
		return err
	}
	return writeAtomic(s.viewPath(tab), out.Bytes())
}

// tabs returns the names of the folders that are tabs.
func (s *Store) tabs(ignored func(string) bool) []string {
	entries, _ := os.ReadDir(s.Dir)
	var out []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || name == ArchiveDir || ignored(name) {
			continue
		}
		if isDir(e, filepath.Join(s.Dir, name)) {
			out = append(out, name)
		}
	}
	return out
}

// Settings is what the sticky.json files of a board say, read once: the
// reader's side of the settings. Asking it anything reads nothing more.
type Settings struct {
	files map[string]tabFile // by tab; "" is the root file
	tabs  []string           // the folders that are tabs, in name order
	err   error              // the first file that could not be read
}

type tabFile struct {
	top   map[string]json.RawMessage
	views Views
}

// Settings reads the root file and the file of every tab.
func (s *Store) Settings() Settings {
	set := Settings{files: map[string]tabFile{}}
	read := func(tab string) {
		top, views, err := s.readViews(tab)
		if err != nil && set.err == nil {
			set.err = err
		}
		set.files[tab] = tabFile{top, views}
	}
	read("")
	set.tabs = s.tabs(set.ignored())
	for _, tab := range set.tabs {
		read(tab)
	}
	return set
}

// Err returns the first settings file that could not be read. Its tab
// arranges itself; the other files are still read.
func (set Settings) Err() error { return set.err }

func (set Settings) str(tab, key string) string {
	var v string
	_ = json.Unmarshal(set.files[tab].top[key], &v)
	return v
}

// Theme returns the name of the theme the user chose, or "" for none.
func (set Settings) Theme() string { return set.str("", themeKey) }

// Language returns the language the user chose ("ko"), or "" for none.
func (set Settings) Language() string { return set.str("", langKey) }

// Tab returns the tab the board was on, "" for the root.
func (set Settings) Tab() string { return set.str("", tabKey) }

// TabTitle returns the title a tab's file gives it, or "".
func (set Settings) TabTitle(tab string) string { return set.str(tab, titleKey) }

// Views returns how every note is arranged, keyed by full name.
func (set Settings) Views() Views {
	all := Views{}
	for tab, f := range set.files {
		for k, v := range f.views {
			all[path.Join(tab, k)] = v
		}
	}
	return all
}

// Order returns the notes of a tab that the user put in an order of their
// own, by full name, first to last. Notes not in it follow by name.
func (set Settings) Order(tab string) []string {
	var order []string
	_ = json.Unmarshal(set.files[tab].top[orderKey], &order)
	for i, n := range order {
		order[i] = path.Join(tab, n)
	}
	return order
}

// ignored returns a test for the names the root file's "ignore" list
// leaves out. An entry is a name or a pattern such as "*.tmp.md", matched
// against the note's full name ("docs/draft.md") and against its last part
// ("draft.md"). The list is written by hand.
func (set Settings) ignored() func(name string) bool {
	var patterns []string
	_ = json.Unmarshal(set.files[""].top[ignoreKey], &patterns)
	return func(name string) bool {
		for _, p := range patterns {
			if ok, _ := path.Match(p, name); ok {
				return true
			}
			if ok, _ := path.Match(p, path.Base(name)); ok {
				return true
			}
		}
		return false
	}
}

// Views returns how every note is arranged, keyed by full name. A tab's
// file that cannot be read is reported and that tab's notes arrange
// themselves; the other tabs are still read.
func (s *Store) Views() (Views, error) {
	set := s.Settings()
	return set.Views(), set.Err()
}

// update is the writer's side of the settings: it reads a tab's file as it
// is now, lets change edit it, and replaces the file in one step. A file
// that cannot be read is left alone: overwriting it would lose whatever the
// user was in the middle of fixing.
func (s *Store) update(tab string, change func(top map[string]json.RawMessage, views Views) error) error {
	top, views, err := s.readViews(tab)
	if err != nil {
		return err
	}
	if err := change(top, views); err != nil {
		return err
	}
	return s.writeViews(tab, top)
}

// put sets one key of a settings file to value, as JSON.
func put(top map[string]json.RawMessage, key string, value any) error {
	raw, err := json.Marshal(value)
	if err == nil {
		top[key] = raw
	}
	return err
}

// SetView changes one note's view and writes its tab's file in one step.
// Entries that say nothing, and entries of notes that no longer exist, are
// dropped.
func (s *Store) SetView(name string, change func(*View)) error {
	tab, inside := s.split(name)
	return s.update(tab, func(top map[string]json.RawMessage, views Views) error {
		v := views[inside]
		change(&v)
		views[inside] = v
		for n, v := range views {
			if _, err := os.Lstat(filepath.Join(s.Dir, filepath.FromSlash(path.Join(tab, n)))); v.empty() || errors.Is(err, fs.ErrNotExist) {
				delete(views, n)
			}
		}
		return put(top, viewKey, views)
	})
}

// Order returns the notes of a tab that the user put in an order of their
// own, by full name. It reads only that tab's file.
func (s *Store) Order(tab string) []string { return s.one(tab).Order(tab) }

// SetOrder writes the order of a tab's notes, given by full name.
func (s *Store) SetOrder(tab string, names []string) error {
	inside := make([]string, 0, len(names))
	for _, n := range names {
		inside = append(inside, strings.TrimPrefix(strings.TrimPrefix(n, tab), "/"))
	}
	return s.update(tab, func(top map[string]json.RawMessage, _ Views) error {
		return put(top, orderKey, inside)
	})
}

// one reads a single tab's file, for a caller that needs one key of it.
func (s *Store) one(tab string) Settings {
	top, views, err := s.readViews(tab)
	return Settings{files: map[string]tabFile{tab: {top, views}}, err: err}
}

func (s *Store) setString(tab, key, value string) error {
	return s.update(tab, func(top map[string]json.RawMessage, _ Views) error {
		return put(top, key, value)
	})
}

// Theme returns the name of the theme the user chose, or "" for none.
func (s *Store) Theme() string { return s.one("").Theme() }

// SetTheme writes the name of the chosen theme.
func (s *Store) SetTheme(name string) error { return s.setString("", themeKey, name) }

// Language returns the language the user chose ("ko"), or "" for none.
func (s *Store) Language() string { return s.one("").Language() }

// SetLanguage writes the chosen language.
func (s *Store) SetLanguage(code string) error { return s.setString("", langKey, code) }

// Tab returns the tab the board was on, "" for the root.
func (s *Store) Tab() string { return s.one("").Tab() }

// SetTab writes which tab the board is on.
func (s *Store) SetTab(name string) error { return s.setString("", tabKey, name) }

// TabTitle returns the title a tab's file gives it, or "".
func (s *Store) TabTitle(tab string) string { return s.one(tab).TabTitle(tab) }

// SetTabTitle names a tab in its own file.
func (s *Store) SetTabTitle(tab, title string) error { return s.setString(tab, titleKey, title) }

// ignored returns the test of the root file's ignore list.
func (s *Store) ignored() func(name string) bool { return s.one("").ignored() }
