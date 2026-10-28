// Package atomicfile writes a file so that a reader sees either the old
// content or the new, never half of it: the one way stickypane replaces a
// file, for the notes folder, the registry and the files init sets up.
package atomicfile

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

// tmpInfix marks a file being written. Its name begins with a dot, so the
// board, which shows no hidden file, never shows one, and ends in random
// digits, so nothing that looks for ".json" or ".md" takes it.
const tmpInfix = ".tmp-"

// WriteFile replaces the file at path with data in one step: data goes to
// a temporary file in the same folder, is synced to the disk, and is then
// renamed over path, and the folder is synced where the system allows it.
// As with os.WriteFile, perm is the mode of a new file; a file that is
// there keeps its own. A path that is a symbolic link is written where it
// points, so the link stays a link.
func WriteFile(path string, data []byte, perm fs.FileMode) error {
	if target, err := filepath.EvalSymlinks(path); err == nil {
		path = target
	}
	if fi, err := os.Stat(path); err == nil {
		perm = fi.Mode().Perm()
	}
	tmp, err := temp(path, data, perm)
	if err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	syncDir(filepath.Dir(path))
	return nil
}

// Create makes the file at path with data and fails with an error that is
// fs.ErrExist when anything is at path already: it never writes over a file
// made a moment before. As with WriteFile, the name never shows half the
// data.
func Create(path string, data []byte, perm fs.FileMode) error {
	tmp, err := temp(path, data, perm)
	if err != nil {
		return err
	}
	defer os.Remove(tmp)
	// A hard link fails when the name is taken, and gives the name the
	// whole file at once.
	err = os.Link(tmp, path)
	if err == nil {
		syncDir(filepath.Dir(path))
		return nil
	}
	if errors.Is(err, fs.ErrExist) {
		return err
	}
	if _, serr := os.Lstat(path); serr == nil {
		return &fs.PathError{Op: "create", Path: path, Err: fs.ErrExist}
	}
	// No hard links here (some file systems): create exclusively and
	// write. Only here may a reader see the file before it is whole.
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	_, werr := f.Write(data)
	if werr == nil {
		werr = f.Sync()
	}
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		os.Remove(path)
		return werr
	}
	syncDir(filepath.Dir(path))
	return nil
}

// temp writes data to a new temporary file next to path, with mode perm,
// synced, and returns its name.
func temp(path string, data []byte, perm fs.FileMode) (string, error) {
	f, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+tmpInfix+"*")
	if err != nil {
		return "", err
	}
	_, werr := f.Write(data)
	if werr == nil {
		werr = f.Sync()
	}
	if werr == nil {
		werr = f.Chmod(perm)
	}
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		os.Remove(f.Name())
		return "", werr
	}
	return f.Name(), nil
}

// syncDir makes a rename in dir last through a crash. Windows cannot open
// a folder to sync it, and some file systems refuse; the rename is done
// either way.
func syncDir(dir string) {
	if runtime.GOOS == "windows" {
		return
	}
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		d.Close()
	}
}
