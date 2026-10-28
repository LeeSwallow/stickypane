package runner

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/LeeSwallow/stickypane/internal/store"
)

// The head of a script's log names the program, the script and the second
// it started, in one fixed form.
func TestTheLogHeadIsOneFixedForm(t *testing.T) {
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
	at := time.Date(2026, 10, 3, 14, 2, 5, 0, time.Local)
	if _, err := Prepare(store.Open(dir), "job.sh", func() time.Time { return at }); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, LogOf("job.sh")))
	if want := "$ sh job.sh   (2026-10-03 14:02:05)\n"; string(b) != want {
		t.Errorf("head = %q, want %q", b, want)
	}
}
