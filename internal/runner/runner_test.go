package runner

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/LeeSwallow/stickypane/internal/store"
)

func TestStartWritesTheLogAndExitCode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("needs sh")
	}
	dir := filepath.Join(t.TempDir(), store.DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "job.sh"), []byte("echo hi\nexit 3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := Start(store.Open(dir), "job.sh", time.Now)
	if err != nil {
		t.Fatal(err)
	}
	code, _, err := r.Wait()
	if err != nil || code != 3 {
		t.Fatalf("Wait = %d, %v; want 3, nil", code, err)
	}
	b, err := os.ReadFile(filepath.Join(dir, LogOf("job.sh")))
	if err != nil {
		t.Fatal(err)
	}
	if log := string(b); !strings.HasPrefix(log, "$ ") || !strings.Contains(log, "hi\n") || !strings.Contains(log, "[exit 3") {
		t.Errorf("the log should have a head, the output and the exit line:\n%s", log)
	}
}

func TestKillEndsAScriptThatNeverEnds(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("needs sh")
	}
	dir := filepath.Join(t.TempDir(), store.DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "long.sh"), []byte("exec sleep 60\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := Start(store.Open(dir), "long.sh", time.Now)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan int, 1)
	go func() {
		code, _, _ := r.Wait()
		done <- code
	}()
	r.Kill()
	r.Kill() // killing twice is harmless
	select {
	case code := <-done:
		if code == 0 {
			t.Error("a killed script did not exit with 0")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Wait did not return after Kill")
	}
}
