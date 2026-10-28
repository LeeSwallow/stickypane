//go:build !windows

package bridge

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/LeeSwallow/stickypane/internal/core/kinds"
	"github.com/LeeSwallow/stickypane/internal/core/widget/note"
)

func pidAlive(pid int) bool {
	p, err := os.FindProcess(pid)
	return err == nil && p.Signal(syscall.Signal(0)) == nil
}

// Ending the session stops what a script started too, not only its shell.
func TestServeStopsTheProgramsAScriptStarted(t *testing.T) {
	dir := board(t, map[string]string{"long.sh": "sleep 30 &\necho $! > bg.pid\nsleep 30 &\necho $! >> bg.pid\nwait\n"})
	pidFile := filepath.Join(filepath.Dir(dir), "bg.pid")
	// The script runs in the project folder, the notes folder's parent.
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	go io.Copy(io.Discard, outR)
	done := make(chan error, 1)
	go func() {
		done <- Serve(context.Background(), newAPI(dir), kinds.Default(note.Plain), inR, outW)
		outW.Close()
	}()
	io.WriteString(inW, `{"id":1,"method":"note.run","params":{"note":"long.sh"}}`+"\n")
	var pids []int
	deadline := time.Now().Add(5 * time.Second)
	for len(pids) < 2 {
		if time.Now().After(deadline) {
			t.Fatal("the script did not start its programs")
		}
		pids = pids[:0]
		b, _ := os.ReadFile(pidFile)
		for _, f := range strings.Fields(string(b)) {
			var n int
			fmt.Sscan(f, &n)
			pids = append(pids, n)
		}
		time.Sleep(20 * time.Millisecond)
	}
	for _, pid := range pids {
		if !pidAlive(pid) {
			t.Fatalf("pid %d is not running before the stop", pid)
		}
	}
	inW.Close()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Serve did not end")
	}
	for _, pid := range pids {
		gone := false
		for i := 0; i < 100 && !gone; i++ {
			gone = !pidAlive(pid)
			time.Sleep(20 * time.Millisecond)
		}
		if !gone {
			syscall.Kill(pid, syscall.SIGKILL)
			t.Errorf("pid %d outlived the session", pid)
		}
	}
}
