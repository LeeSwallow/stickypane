//go:build windows

package filelock

import (
	"errors"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// try takes LockFileEx on the folder's lock file without waiting. Each call
// opens the file anew, and Windows locks a range per handle, so two
// goroutines of one process exclude each other as two processes do.
func try(dir string) (unlock func(), busy bool, err error) {
	f, err := os.OpenFile(filepath.Join(dir, Name), os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, false, err
	}
	h := windows.Handle(f.Fd())
	ol := new(windows.Overlapped)
	err = windows.LockFileEx(h, windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, ol)
	switch {
	case err == nil:
		return func() {
			_ = windows.UnlockFileEx(h, 0, 1, 0, ol)
			f.Close()
		}, false, nil
	case errors.Is(err, windows.ERROR_LOCK_VIOLATION), errors.Is(err, windows.ERROR_IO_PENDING):
		f.Close()
		return nil, true, nil
	}
	// The file system does not lock: go on without.
	f.Close()
	return nothing, false, nil
}
