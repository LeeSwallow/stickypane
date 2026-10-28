package procs

import (
	"os/exec"
	"testing"
	"time"
)

func TestAliveComparesStartTimes(t *testing.T) {
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Skip(err)
	}
	t.Cleanup(func() { cmd.Process.Kill(); cmd.Wait() })
	now := time.Now()
	if !(PS{}).Alive(cmd.Process.Pid, now) {
		t.Fatal("running child not alive")
	}
	if (PS{}).Alive(cmd.Process.Pid, now.Add(-time.Hour)) {
		t.Fatal("pid reused by a later process counted as alive")
	}
	if (PS{}).Alive(1<<22+12345, now) {
		t.Fatal("missing pid alive")
	}
}
