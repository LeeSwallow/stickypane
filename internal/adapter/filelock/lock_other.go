//go:build !(darwin || linux || freebsd || openbsd || netbsd || dragonfly || windows)

package filelock

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// try makes the folder's lock file exclusively, without waiting. A lock
// file older than Stale was left by a process that died, and is removed.
func try(dir string) (unlock func(), busy bool, err error) {
	path := filepath.Join(dir, Name)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err == nil {
		f.Close()
		return func() { os.Remove(path) }, false, nil
	}
	if !errors.Is(err, fs.ErrExist) {
		return nil, false, err
	}
	if fi, serr := os.Stat(path); serr == nil && time.Since(fi.ModTime()) > Stale {
		os.Remove(path)
	}
	return nil, true, nil
}
