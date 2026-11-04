package bridgeclient

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// A bridge started from an argv: here sh standing in for ssh, running the
// fake bridge in a folder as a remote shell would.
func TestStartArgvRunsTheCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh")
	}
	project := t.TempDir()
	c, err := StartArgv(ctx(t), []string{"sh", "-c", "cd '" + project + "' && exec '" + fakeBin + "' bridge"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	want, _ := filepath.EvalSymlinks(project)
	got, _ := filepath.EvalSymlinks(c.Hello().Project)
	if got != want {
		t.Fatalf("project = %q, want %q", got, want)
	}
	var out map[string]any
	if err := c.Call(ctx(t), "echo", map[string]any{"a": 1}, &out); err != nil {
		t.Fatal(err)
	}
}

// ssh that cannot connect says why on stderr and ends; the error says it.
func TestStartArgvKeepsStderr(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh")
	}
	_, err := StartArgv(ctx(t), []string{"sh", "-c", "echo 'ssh: connect to host dev port 22: Connection refused' >&2; exit 255"})
	if err == nil || !strings.Contains(err.Error(), "Connection refused") {
		t.Fatalf("err = %v, want ssh's words", err)
	}
	if _, err := StartArgv(ctx(t), nil); err == nil {
		t.Error("no command is an error")
	}
}

// A start that ends before hello is a *StartError: ssh's words apart from
// the reason, which is still ErrConnectionLost.
func TestAStartThatEndsEarlyIsAStartError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh")
	}
	_, err := StartArgv(ctx(t), []string{"sh", "-c", "echo 'ssh: Could not resolve hostname dev.lan' >&2; exit 255"})
	var se *StartError
	if !errors.As(err, &se) {
		t.Fatalf("err = %T %v, want a *StartError", err, err)
	}
	if !errors.Is(err, ErrConnectionLost) {
		t.Errorf("err = %v, want ErrConnectionLost under it", err)
	}
	if se.Stderr != "ssh: Could not resolve hostname dev.lan" {
		t.Errorf("Stderr = %q", se.Stderr)
	}
	if want := "bridge: hello: bridge: connection lost: ssh: Could not resolve hostname dev.lan"; err.Error() != want {
		t.Errorf("Error() = %q, want %q", err, want)
	}
}

func TestRunArgv(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh")
	}
	out, err := RunArgv(ctx(t), []string{"sh", "-c", "echo stickypane 1.2.3"})
	if err != nil || strings.TrimSpace(out) != "stickypane 1.2.3" {
		t.Fatalf("out %q, err %v", out, err)
	}
	_, err = RunArgv(ctx(t), []string{"sh", "-c", "echo 'Permission denied (publickey).' >&2; exit 255"})
	if err == nil || err.Error() != "Permission denied (publickey)." {
		t.Fatalf("err = %v", err)
	}
	// something left behind with stderr open does not hold the answer
	start := time.Now()
	out, err = RunArgv(ctx(t), []string{"sh", "-c", "(sleep 5 >/dev/null) & echo ok"})
	if err != nil || strings.TrimSpace(out) != "ok" || time.Since(start) > 3*time.Second {
		t.Fatalf("out %q, err %v, after %v", out, err, time.Since(start))
	}
	c, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, err := RunArgv(c, []string{"sleep", "5"}); err == nil || !strings.Contains(err.Error(), "no answer") {
		t.Fatalf("err = %v", err)
	}
}
