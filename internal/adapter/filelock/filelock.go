// Package filelock is an advisory lock on a folder, held by one process,
// or one goroutine, at a time: what the board takes around reading a file,
// changing it and writing it back, so that two writers in different
// processes do not lose each other's change.
//
// It is advisory: only code that takes it waits for it. On macOS, Linux
// and the BSDs it is flock(2) on the folder itself, so no file is made,
// and it is let go when the process ends, however it ends. On Windows it
// is LockFileEx on the file Name in the folder. Elsewhere it is that file,
// made exclusively and removed on unlock; one older than Stale is taken to
// be left by a process that died.
package filelock

import (
	"errors"
	"fmt"
	"time"
)

// Name is the lock file in the folder, where the system needs one.
const Name = ".stickypane.lock"

// Stale is how old a lock file must be, where the lock is a file, before
// it is taken to be left behind: far longer than any read, change and
// write.
const Stale = 30 * time.Second

// ErrTimeout is what Lock answers when another holder did not let go in
// time.
var ErrTimeout = errors.New("the board is busy: another writer holds its lock")

// Lock takes the lock of the folder dir, waiting up to wait for whoever
// holds it, and returns the function that lets it go. A file system that
// cannot lock (some network ones) is written without the lock: the board
// still works there, as it did before it had one.
func Lock(dir string, wait time.Duration) (unlock func(), err error) {
	deadline := time.Now().Add(wait)
	pause := time.Millisecond
	for {
		unlock, busy, err := try(dir)
		if err != nil {
			return nil, err
		}
		if !busy {
			return unlock, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("%s: %w", dir, ErrTimeout)
		}
		time.Sleep(pause)
		pause = min(2*pause, 20*time.Millisecond)
	}
}

func nothing() {}
