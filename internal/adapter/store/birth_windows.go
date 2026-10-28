//go:build windows

package store

import (
	"io/fs"
	"syscall"
	"time"
)

// birthTime is when the file was made, as Windows keeps it.
func birthTime(fi fs.FileInfo) time.Time {
	if d, ok := fi.Sys().(*syscall.Win32FileAttributeData); ok {
		return time.Unix(0, d.CreationTime.Nanoseconds())
	}
	return time.Time{}
}
