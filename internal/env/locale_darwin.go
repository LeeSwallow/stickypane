//go:build darwin

package env

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// systemLocale asks macOS for the user's locale ("ko_KR"), for a terminal
// that sets no LANG. It gives up quickly rather than hold the board.
func systemLocale() string {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	out, err := exec.CommandContext(ctx, "defaults", "read", "-g", "AppleLocale").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
