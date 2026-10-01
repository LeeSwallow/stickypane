package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// View is how a note is arranged on the screen: what the user did with the
// keys that open, size, color and pin a note. A field that is nil, empty or
// zero says nothing, and the note's own front matter or its kind decides.
//
// Views are kept in one file next to the notes instead of in each note, for
// three reasons: a log, a script or a linked README has no front matter to
// keep them in; the board then never rewrites a note just because the user
// rearranged the screen; and the file can be committed to share a layout.
type View struct {
	Open  *bool  `json:"open,omitempty"`
	Pin   *bool  `json:"pin,omitempty"`
	Size  string `json:"size,omitempty"`
	Rows  int    `json:"rows,omitempty"`
	Color string `json:"color,omitempty"`
}

func (v View) empty() bool {
	return v.Open == nil && v.Pin == nil && v.Size == "" && v.Rows == 0 && v.Color == ""
}

// Views maps a note's name to its view.
type Views map[string]View

// viewKey and orderKey are the keys this version reads. Other keys are kept
// as they are, so a later version can add to the file.
const (
	viewKey  = "notes"
	orderKey = "order"
)

// readViews returns the file's top-level keys and the views among them.
func (s *Store) readViews() (map[string]json.RawMessage, Views, error) {
	top := map[string]json.RawMessage{}
	views := Views{}
	b, err := os.ReadFile(filepath.Join(s.Dir, ViewFile))
	if errors.Is(err, fs.ErrNotExist) {
		return top, views, nil
	}
	if err != nil {
		return top, views, err
	}
	if err := json.Unmarshal(b, &top); err != nil {
		return top, views, fmt.Errorf("%s is not valid JSON: %w", ViewFile, err)
	}
	if raw, ok := top[viewKey]; ok {
		if err := json.Unmarshal(raw, &views); err != nil {
			return top, Views{}, fmt.Errorf("%s: %q is not a map of notes: %w", ViewFile, viewKey, err)
		}
	}
	return top, views, nil
}

// Views returns how the notes are arranged. Without the file there is
// nothing to return; a file that cannot be read is an error, and the notes
// are then shown as they arrange themselves.
func (s *Store) Views() (Views, error) {
	_, views, err := s.readViews()
	return views, err
}

// Order returns the notes the user put in an order of their own, first to
// last. Notes that are not in it follow in the order of their names.
func (s *Store) Order() []string {
	top, _, err := s.readViews()
	if err != nil {
		return nil
	}
	var order []string
	_ = json.Unmarshal(top[orderKey], &order)
	return order
}

// SetOrder writes the order of the notes. Like SetView it leaves a file it
// cannot read alone.
func (s *Store) SetOrder(names []string) error {
	top, _, err := s.readViews()
	if err != nil {
		return err
	}
	raw, err := json.Marshal(names)
	if err != nil {
		return err
	}
	top[orderKey] = raw
	return s.writeViews(top)
}

// SetView changes one note's view and writes the file in one step. Entries
// that say nothing, and entries of notes that no longer exist, are dropped.
// A file that cannot be read is left alone: overwriting it would lose
// whatever the user was in the middle of fixing.
func (s *Store) SetView(name string, change func(*View)) error {
	top, views, err := s.readViews()
	if err != nil {
		return err
	}
	v := views[name]
	change(&v)
	views[name] = v
	for n, v := range views {
		if _, err := os.Stat(filepath.Join(s.Dir, filepath.FromSlash(n))); v.empty() || errors.Is(err, fs.ErrNotExist) {
			delete(views, n)
		}
	}
	raw, err := json.Marshal(views)
	if err != nil {
		return err
	}
	top[viewKey] = raw
	return s.writeViews(top)
}

func (s *Store) writeViews(top map[string]json.RawMessage) error {
	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(top); err != nil {
		return err
	}
	return writeAtomic(filepath.Join(s.Dir, ViewFile), out.Bytes())
}
