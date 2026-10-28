//go:build windows

package runner

import (
	"os"
	"os/exec"
)

// On Windows a script started by StartGroup is no different: only its own
// process is killed, not what it started.
func inOwnGroup(*exec.Cmd) {}

func killAll(p *os.Process) { _ = p.Kill() }
