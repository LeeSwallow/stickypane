package shell

import "os"

// mine is true: Windows has no shared connections (controlPath).
func mine(os.FileInfo) bool { return true }
