//go:build !darwin && !windows

package store

import (
	"io/fs"
	"time"
)

// birthTime is unknown on systems whose file status has no portable birth
// time; a note stickypane made still says when, in its front matter.
func birthTime(fs.FileInfo) time.Time { return time.Time{} }
