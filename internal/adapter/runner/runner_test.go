package runner

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/LeeSwallow/stickypane/internal/adapter/store"
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
	// Nothing holds the log once Wait is over: on Windows a handle left
	// open would keep it from being removed.
	if err := os.Remove(filepath.Join(dir, LogOf("job.sh"))); err != nil {
		t.Errorf("the log is still held after Wait: %v", err)
	}
}

// Prepare writes the head of the log and starts nothing; Wait starts the
// script.
func TestPrepareStartsNothingUntilWait(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("needs sh")
	}
	dir := filepath.Join(t.TempDir(), store.DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "job.sh"), []byte("echo hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := Prepare(store.Open(dir), "job.sh", time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if r.cmd != nil {
		t.Fatal("Prepare started the script")
	}
	b, _ := os.ReadFile(filepath.Join(dir, LogOf("job.sh")))
	if log := string(b); !strings.HasPrefix(log, "$ ") || strings.Contains(log, "hi") {
		t.Errorf("after Prepare the log is its head alone:\n%s", log)
	}
	if code, _, err := r.Wait(); err != nil || code != 0 {
		t.Fatalf("Wait = %d, %v; want 0, nil", code, err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, LogOf("job.sh"))); !strings.Contains(string(b), "hi\n") {
		t.Errorf("Wait runs the script:\n%s", b)
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
