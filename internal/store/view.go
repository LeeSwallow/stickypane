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
	b, err := os.ReadFile(s.viewPath(tab))
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
func (s *Store) tabs() []string {
	entries, _ := os.ReadDir(s.Dir)
	var out []string
	ignored := s.ignored()
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || name == ArchiveDir || ignored(name) {
			continue
		}
		if fi, err := os.Stat(filepath.Join(s.Dir, name)); err == nil && fi.IsDir() {
			out = append(out, name)
		}
	}
	return out
}

// Views returns how every note is arranged, keyed by full name. A tab's
// file that cannot be read is reported and that tab's notes arrange
// themselves; the other tabs are still read.
func (s *Store) Views() (Views, error) {
	all := Views{}
	var failed error
	for _, tab := range append([]string{""}, s.tabs()...) {
		_, views, err := s.readViews(tab)
		if err != nil && failed == nil {
			failed = err
		}
		for k, v := range views {
			all[path.Join(tab, k)] = v
		}
	}
	return all, failed
}

// SetView changes one note's view and writes its tab's file in one step.
// Entries that say nothing, and entries of notes that no longer exist, are
// dropped. A file that cannot be read is left alone: overwriting it would
// lose whatever the user was in the middle of fixing.
func (s *Store) SetView(name string, change func(*View)) error {
	tab, inside := s.split(name)
	top, views, err := s.readViews(tab)
	if err != nil {
		return err
	}
	v := views[inside]
	change(&v)
	views[inside] = v
	for n, v := range views {
		if _, err := os.Lstat(filepath.Join(s.Dir, filepath.FromSlash(path.Join(tab, n)))); v.empty() || errors.Is(err, fs.ErrNotExist) {
			delete(views, n)
		}
	}
	raw, err := json.Marshal(views)
	if err != nil {
		return err
	}
	top[viewKey] = raw
	return s.writeViews(tab, top)
}

// Order returns the notes of a tab that the user put in an order of their
// own, by full name, first to last. Notes not in it follow by name.
func (s *Store) Order(tab string) []string {
	top, _, err := s.readViews(tab)
	if err != nil {
		return nil
	}
	var order []string
	_ = json.Unmarshal(top[orderKey], &order)
	for i, n := range order {
		order[i] = path.Join(tab, n)
	}
	return order
}

// SetOrder writes the order of a tab's notes, given by full name.
func (s *Store) SetOrder(tab string, names []string) error {
	top, _, err := s.readViews(tab)
	if err != nil {
		return err
	}
	inside := make([]string, 0, len(names))
	for _, n := range names {
		inside = append(inside, strings.TrimPrefix(strings.TrimPrefix(n, tab), "/"))
	}
	raw, err := json.Marshal(inside)
	if err != nil {
		return err
	}
	top[orderKey] = raw
	return s.writeViews(tab, top)
}

// rootString reads one string key of the root file.
func (s *Store) rootString(key string) string {
	top, _, err := s.readViews("")
	if err != nil {
		return ""
	}
	var v string
	_ = json.Unmarshal(top[key], &v)
	return v
}

// setRootString writes one string key of the root file.
func (s *Store) setRootString(key, value string) error {
	top, _, err := s.readViews("")
	if err != nil {
		return err
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	top[key] = raw
	return s.writeViews("", top)
}

// Theme returns the name of the theme the user chose, or "" for none.
func (s *Store) Theme() string { return s.rootString(themeKey) }

// SetTheme writes the name of the chosen theme.
func (s *Store) SetTheme(name string) error { return s.setRootString(themeKey, name) }

// Language returns the language the user chose ("ko"), or "" for none.
func (s *Store) Language() string { return s.rootString(langKey) }

// SetLanguage writes the chosen language.
func (s *Store) SetLanguage(code string) error { return s.setRootString(langKey, code) }

// Tab returns the tab the board was on, "" for the root.
func (s *Store) Tab() string { return s.rootString(tabKey) }

// SetTab writes which tab the board is on.
func (s *Store) SetTab(name string) error { return s.setRootString(tabKey, name) }

// TabTitle returns the title a tab's file gives it, or "".
func (s *Store) TabTitle(tab string) string {
	top, _, err := s.readViews(tab)
	if err != nil {
		return ""
	}
	var v string
	_ = json.Unmarshal(top[titleKey], &v)
	return v
}

// SetTabTitle names a tab in its own file.
func (s *Store) SetTabTitle(tab, title string) error {
	top, _, err := s.readViews(tab)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(title)
	if err != nil {
		return err
	}
	top[titleKey] = raw
	return s.writeViews(tab, top)
}

// ignored returns a test for the names the root file's "ignore" list
// leaves out. An entry is a name or a pattern such as "*.tmp.md", matched
// against the note's full name ("docs/draft.md") and against its last part
// ("draft.md"). The list is written by hand.
func (s *Store) ignored() func(name string) bool {
	top, _, err := s.readViews("")
	var patterns []string
	if err == nil {
		_ = json.Unmarshal(top[ignoreKey], &patterns)
	}
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
