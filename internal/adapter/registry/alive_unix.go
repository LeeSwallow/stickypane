//go:build unix

package registry

import (
	"errors"
	"syscall"
)

// processAlive sends pid the signal 0, which checks without signalling. A
// process of another user answers EPERM, and is alive too.
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}
