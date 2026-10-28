//go:build darwin || linux || freebsd || openbsd || netbsd || dragonfly

package filelock

import (
	"errors"
	"os"
	"syscall"
)

// try takes flock(2) on the folder itself without waiting. Each call opens
// the folder anew, so two goroutines of one process exclude each other as
// two processes do.
func try(dir string) (unlock func(), busy bool, err error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, false, err
	}
	for {
		err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if !errors.Is(err, syscall.EINTR) {
			break
		}
	}
	switch {
	case err == nil:
		return func() {
			_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
			f.Close()
		}, false, nil
	case errors.Is(err, syscall.EWOULDBLOCK):
		f.Close()
		return nil, true, nil
	}
	// The file system does not lock: go on without.
	f.Close()
	return nothing, false, nil
}
