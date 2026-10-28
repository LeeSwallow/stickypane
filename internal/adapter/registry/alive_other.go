//go:build !unix

package registry

import "os"

// processAlive asks the system for the process. On Windows, finding it
// opens it, which fails once it has gone.
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	p.Release()
	return true
}
