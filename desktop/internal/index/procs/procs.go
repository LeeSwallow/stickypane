// Package procs answers whether a recorded process still lives: the
// ports.ProcessChecker adapter, through ps (macOS and Linux, ADR-0014).
package procs

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/LeeSwallow/stickypane/desktop/internal/index/ports"
)

// PS asks ps.
type PS struct{}

var _ ports.ProcessChecker = PS{}

// Alive says whether the process exists and was started when the run file
// says (within a few seconds, since the file is written just after the
// process begins). A pid reused by another program started later does not
// match.
func (PS) Alive(pid int, started time.Time) bool {
	cmd := exec.Command("ps", "-o", "lstart=", "-p", strconv.Itoa(pid))
	cmd.Env = append(os.Environ(), "LC_ALL=C", "LANG=C")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	s := strings.Join(strings.Fields(string(out)), " ")
	if s == "" {
		return false
	}
	t, err := time.ParseInLocation("Mon Jan 2 15:04:05 2006", s, time.Local)
	if err != nil {
		return false
	}
	d := started.Sub(t)
	return d > -10*time.Second && d < 10*time.Second
}
