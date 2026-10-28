//go:build darwin

package store

import (
	"io/fs"
	"syscall"
	"time"
)

// birthTime is when the file was made, as macOS keeps it.
func birthTime(fi fs.FileInfo) time.Time {
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		return time.Unix(st.Birthtimespec.Sec, st.Birthtimespec.Nsec)
	}
	return time.Time{}
}
