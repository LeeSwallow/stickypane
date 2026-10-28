// Package boardfs looks at project folders on disk for the index: the
// ports.Boards adapter. It only asks whether a .sticky/ is there and when
// it changed; what is in it is the bridge's to read.
package boardfs

import (
	"os"
	"path/filepath"
	"time"

	"github.com/LeeSwallow/stickypane/desktop/internal/index/ports"
)

// Disk is the file system.
type Disk struct{}

var _ ports.Boards = Disk{}

// Exists says whether path and its .sticky/ are there.
func (Disk) Exists(path string) bool {
	fi, err := os.Stat(filepath.Join(path, ".sticky"))
	return err == nil && fi.IsDir()
}

// Changed is when path/.sticky/ last changed.
func (Disk) Changed(path string) (time.Time, bool) {
	fi, err := os.Stat(filepath.Join(path, ".sticky"))
	if err != nil {
		return time.Time{}, false
	}
	return fi.ModTime(), true
}
