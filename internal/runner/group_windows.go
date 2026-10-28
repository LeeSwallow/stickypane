//go:build windows

package runner

import (
	"os"
	"os/exec"
)

// Windows keeps today's behaviour: only the script's own process is
// killed, not what it started. The board runs there; the desktop overlay
// (the only caller of Kill) does not.
func inOwnGroup(*exec.Cmd) {}

func killAll(p *os.Process) { _ = p.Kill() }
