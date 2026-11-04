package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// A save that cannot land says so, and leaves no temporary file beside
// the state file.
func TestAFailedSaveSaysSoAndLeavesNothing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, StateFile)
	if err := os.Mkdir(path, 0o755); err != nil { // a folder where the file goes: the rename fails
		t.Fatal(err)
	}
	err := State{Hotkey: DefaultHotkey}.Save(path)
	if err == nil || !strings.HasPrefix(err.Error(), "save the app's state: ") {
		t.Fatalf("err = %v", err)
	}
	left, _ := filepath.Glob(filepath.Join(dir, ".overlay-*.json"))
	if len(left) != 0 {
		t.Errorf("left behind %v", left)
	}
}

// A damaged state file names itself.
func TestADamagedStateFileNamesItself(t *testing.T) {
	path := filepath.Join(t.TempDir(), StateFile)
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := LoadState(path)
	if err == nil || !strings.HasPrefix(err.Error(), path+": ") {
		t.Fatalf("err = %v", err)
	}
	if st.Hotkey != DefaultHotkey {
		t.Errorf("a damaged file is the defaults: %+v", st)
	}
}

// A save no caller waits for is logged when it fails, with where.
func TestKeepLogsAFailedSave(t *testing.T) {
	var mu sync.Mutex
	var logged []string
	r := newRig(t, func(c *Config) {
		c.Logf = func(format string, args ...any) {
			mu.Lock()
			logged = append(logged, fmt.Sprintf(format, args...))
			mu.Unlock()
		}
	})
	if err := os.Mkdir(r.s.statePath, 0o755); err != nil {
		t.Fatal(err)
	}
	r.s.keep("closing")
	mu.Lock()
	defer mu.Unlock()
	if len(logged) != 1 || !strings.HasPrefix(logged[0], "closing: save the app's state: ") {
		t.Errorf("logged %q", logged)
	}
}
