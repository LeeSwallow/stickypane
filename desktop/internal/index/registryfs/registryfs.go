// Package registryfs reads the records stickypane keeps in its home folder
// (~/.stickypane: events/*.jsonl and run/*.json) for the index: the
// ports.Records adapter. It never writes or deletes there; stickypane
// alone does (internal/adapter/registry, which prunes its own old events and dead
// run files). The records' form is the main module's, pinned by
// internal/adapter/registry/testdata/home, which this package's tests read.
package registryfs

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/LeeSwallow/stickypane/desktop/internal/index/ports"
)

// Home is a stickypane home folder.
type Home string

var _ ports.Records = Home("")

// EventFiles lists the names of events/*.jsonl, in name (so date) order.
func (h Home) EventFiles() ([]string, error) {
	ents, err := os.ReadDir(filepath.Join(string(h), "events"))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range ents {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".jsonl") {
			out = append(out, e.Name())
		}
	}
	return out, nil
}

// ReadFrom returns events/name from offset on, and its size.
func (h Home) ReadFrom(name string, offset int64) ([]byte, int64, error) {
	f, err := os.Open(filepath.Join(string(h), "events", name))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, 0, ports.ErrGone
	}
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, 0, err
	}
	if fi.Size() < offset {
		return nil, fi.Size(), nil
	}
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return nil, 0, err
	}
	data, err := io.ReadAll(f)
	return data, fi.Size(), err
}

// Runs returns the regular files of run/ with what they hold; one that
// cannot be read (removed meanwhile) is left out.
func (h Home) Runs() ([]ports.RunFile, error) {
	dir := filepath.Join(string(h), "run")
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []ports.RunFile
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		out = append(out, ports.RunFile{Name: e.Name(), Data: b})
	}
	return out, nil
}
