//go:build !windows

package runner

import (
	"os"
	"os/exec"
	"syscall"
)

// inOwnGroup starts the script as the leader of its own process group, so
// that Kill can stop what the script started too (a server, a sleep).
func inOwnGroup(c *exec.Cmd) { c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }

// killAll kills the script's whole group.
func killAll(p *os.Process) {
	if syscall.Kill(-p.Pid, syscall.SIGKILL) != nil {
		_ = p.Kill()
	}
}
