package registry

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/LeeSwallow/stickypane/internal/core/when"
)

func lines(t *testing.T, path string) []string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	sc := bufio.NewScanner(strings.NewReader(string(b)))
	for sc.Scan() {
		out = append(out, sc.Text())
	}
	return out
}

func today() string { return time.Now().Format(when.Date) }

func TestNothingIsWrittenWithoutTheHomeFolder(t *testing.T) {
	home := filepath.Join(t.TempDir(), "absent")
	t.Setenv("STICKYPANE_HOME", home)
	if Enabled() {
		t.Fatal("Enabled is true without the home folder")
	}
	Append(Event{Type: "project.init", Project: "/p"})
	Live("/p", "tui")()
	Touch("/p", "card")
	if _, err := os.Stat(home); !os.IsNotExist(err) {
		t.Fatalf("the home folder appeared: %v", err)
	}
}

func TestAppendWritesADatedLine(t *testing.T) {
	home := t.TempDir()
	t.Setenv("STICKYPANE_HOME", home)
	Append(Event{Type: "project.init", Project: "/p"})
	got := lines(t, filepath.Join(home, "events", today()+".jsonl"))
	if len(got) != 1 {
		t.Fatalf("lines = %v", got)
	}
	var e Event
	if err := json.Unmarshal([]byte(got[0]), &e); err != nil {
		t.Fatal(err)
	}
	if e.V != 1 || e.Type != "project.init" || e.Project != "/p" || e.At == "" {
		t.Fatalf("event = %+v", e)
	}
	if _, err := time.Parse(time.RFC3339, e.At); err != nil {
		t.Fatalf("at = %q: %v", e.At, err)
	}
	if strings.Contains(got[0], `"pid"`) || strings.Contains(got[0], `"cmd"`) {
		t.Fatalf("empty fields not omitted: %s", got[0])
	}
}

func TestAppendDropsALineOverThePipeBuffer(t *testing.T) {
	home := t.TempDir()
	t.Setenv("STICKYPANE_HOME", home)
	Append(Event{Type: "project.touch", Project: strings.Repeat("x", 5000)})
	if _, err := os.Stat(filepath.Join(home, "events", today()+".jsonl")); err == nil {
		b, _ := os.ReadFile(filepath.Join(home, "events", today()+".jsonl"))
		if len(b) > 0 {
			t.Fatalf("a long line was written: %d bytes", len(b))
		}
	}
}

// TestHelperAppend is the child of the many-processes test, not a test.
func TestHelperAppend(t *testing.T) {
	if os.Getenv("STICKYPANE_HELPER") != "1" {
		t.Skip("helper process")
	}
	n := os.Getpid()
	for i := 0; i < 200; i++ {
		Append(Event{Type: "project.touch", Project: "/p" + strconv.Itoa(n), Cmd: strings.Repeat("c", 100)})
	}
}

func TestAppendFromManyProcessesKeepsWholeLines(t *testing.T) {
	home := t.TempDir()
	var cmds []*exec.Cmd
	for i := 0; i < 20; i++ {
		c := exec.Command(os.Args[0], "-test.run=^TestHelperAppend$")
		c.Env = append(os.Environ(), "STICKYPANE_HELPER=1", "STICKYPANE_HOME="+home)
		if err := c.Start(); err != nil {
			t.Fatal(err)
		}
		cmds = append(cmds, c)
	}
	for _, c := range cmds {
		if err := c.Wait(); err != nil {
			t.Fatal(err)
		}
	}
	got := lines(t, filepath.Join(home, "events", today()+".jsonl"))
	if len(got) != 4000 {
		t.Fatalf("lines = %d, want 4000", len(got))
	}
	for i, l := range got {
		var e Event
		if err := json.Unmarshal([]byte(l), &e); err != nil {
			t.Fatalf("line %d is not JSON: %v: %q", i, err, l)
		}
	}
}

func TestAppendIgnoresAnUnwritableHome(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("permissions are not enforced here")
	}
	home := t.TempDir()
	if err := os.Chmod(home, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(home, 0o700) })
	t.Setenv("STICKYPANE_HOME", home)
	Append(Event{Type: "project.init", Project: "/p"})
	Live("/p", "tui")()
	Touch("/p", "card")
}

func TestLiveCreatesAndRemovesTheRunFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("STICKYPANE_HOME", home)
	end := Live("/p", "tui")
	file := filepath.Join(home, "run", fmt.Sprintf("%d.json", os.Getpid()))
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	m := map[string]any{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if pid, _ := m["pid"].(float64); int(pid) != os.Getpid() || m["project"] != "/p" || m["kind"] != "tui" || m["started"] == "" {
		t.Fatalf("run file = %s", b)
	}
	end()
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatalf("run file remains: %v", err)
	}
	got := lines(t, filepath.Join(home, "events", today()+".jsonl"))
	if len(got) != 2 || !strings.Contains(got[0], "session.start") || !strings.Contains(got[1], "session.end") {
		t.Fatalf("events = %v", got)
	}
}

func TestTouchIsThrottled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("STICKYPANE_HOME", home)
	log := filepath.Join(home, "events", today()+".jsonl")
	Touch("/p", "card")
	Touch("/p", "card")
	if n := len(lines(t, log)); n != 1 {
		t.Fatalf("lines = %d, want 1", n)
	}
	Touch("/p", "todo")
	if n := len(lines(t, log)); n != 2 {
		t.Fatalf("another command: lines = %d, want 2", n)
	}
	Touch("/q", "card")
	if n := len(lines(t, log)); n != 3 {
		t.Fatalf("another project: lines = %d, want 3", n)
	}
	// A marker older than a minute no longer throttles.
	ents, _ := os.ReadDir(filepath.Join(home, "run"))
	old := time.Now().Add(-2 * time.Minute)
	for _, e := range ents {
		os.Chtimes(filepath.Join(home, "run", e.Name()), old, old)
	}
	Touch("/p", "card")
	if n := len(lines(t, log)); n != 4 {
		t.Fatalf("after a minute: lines = %d, want 4", n)
	}
}
