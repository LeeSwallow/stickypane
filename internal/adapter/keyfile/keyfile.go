// Package keyfile reads and writes keys.json, the board's keys a user
// changed, in the stickypane home (~/.stickypane or $STICKYPANE_HOME). The
// file names actions and the keys each takes in place of its own:
//
//	{ "note-archive": ["X"], "jot": [] }
//
// What a name or a key means is the terminal's to say; this package only
// keeps the file. Having the file does not turn on the records the
// registry keeps in the same folder.
package keyfile

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/adapter/atomicfile"
	"github.com/LeeSwallow/stickypane/internal/adapter/registry"
)

// Name is the file's name in the home folder.
const Name = "keys.json"

// Path is where the file is: keys.json in registry.Home(), or "" when
// there is no home.
func Path() string {
	home := registry.Home()
	if home == "" {
		return ""
	}
	return filepath.Join(home, Name)
}

// Shown is path as a message names it: under the user's home folder it
// starts with ~.
func Shown(path string) string {
	h, err := os.UserHomeDir()
	if err != nil || h == "" {
		return path
	}
	if rest, ok := strings.CutPrefix(path, h+string(filepath.Separator)); ok {
		return filepath.Join("~", rest)
	}
	return path
}

// Read is what the file at path asks for, action by action. A missing file
// (or no path) asks for nothing and is not an error; a file that is not an
// object of lists of strings is.
func Read(path string) (map[string][]string, error) {
	if path == "" {
		return nil, nil
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var keys map[string][]string
	dec := json.NewDecoder(bytes.NewReader(b))
	if err := dec.Decode(&keys); err != nil {
		return nil, fmt.Errorf("not an object of key lists: %w", err)
	}
	if dec.More() {
		return nil, errors.New("not an object of key lists: more after it")
	}
	return keys, nil
}

// Write replaces the file at path with keys, sorted and indented, in one
// step. Its folder is made when it is missing. An action with no keys is
// written as [], which unbinds it.
func Write(path string, keys map[string][]string) error {
	if path == "" {
		return errors.New("no home folder for " + Name)
	}
	out := make(map[string][]string, len(keys))
	for name, k := range keys {
		if k == nil {
			k = []string{}
		}
		out[name] = k
	}
	b, err := json.MarshalIndent(out, "", "  ") // a map's keys come sorted
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), atomicfile.DirMode); err != nil {
		return err
	}
	return atomicfile.WriteFile(path, append(b, '\n'), atomicfile.FileMode)
}

// Remove deletes the file at path; a file that is not there is no error.
func Remove(path string) error {
	if path == "" {
		return nil
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
