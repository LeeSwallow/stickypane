package bridge

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

var fakeBin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "fakebridge")
	if err != nil {
		panic(err)
	}
	fakeBin = filepath.Join(dir, "stickypane")
	if runtime.GOOS == "windows" {
		fakeBin += ".exe"
	}
	if out, err := exec.Command("go", "build", "-o", fakeBin, "./testdata/fakebridge").CombinedOutput(); err != nil {
		os.Stderr.Write(out)
		os.RemoveAll(dir)
		os.Exit(1)
	}
	code := m.Run()
	os.RemoveAll(dir)
	if realBin != "" {
		os.RemoveAll(filepath.Dir(realBin))
	}
	os.Exit(code)
}

func ctx(t *testing.T) context.Context {
	t.Helper()
	c, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	return c
}

func TestClientHelloChecksTheProtocol(t *testing.T) {
	project := t.TempDir()
	c, err := Start(ctx(t), fakeBin, project)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if c.Hello().Protocol != 1 || c.Hello().Version != "fake" {
		t.Fatalf("hello = %+v", c.Hello())
	}
	// the bridge ran in the project folder
	want, _ := filepath.EvalSymlinks(project)
	got, _ := filepath.EvalSymlinks(c.Hello().Project)
	if got != want {
		t.Fatalf("project = %q, want %q", got, want)
	}

	t.Setenv("FAKE_PROTOCOL", "2")
	if _, err := Start(ctx(t), fakeBin, project); !errors.Is(err, ErrProtocol) {
		t.Fatalf("err = %v, want ErrProtocol", err)
	}
}

func TestClientCallsAndReceivesNotifications(t *testing.T) {
	c, err := Start(ctx(t), fakeBin, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	var got struct{ A, B int }
	if err := c.Call(ctx(t), "echo", map[string]int{"A": 1, "B": 2}, &got); err != nil {
		t.Fatal(err)
	}
	if got.A != 1 || got.B != 2 {
		t.Fatalf("got %+v", got)
	}
	// no params, no result wanted
	if err := c.Call(ctx(t), "echo", nil, nil); err != nil {
		t.Fatal(err)
	}

	err = c.Call(ctx(t), "fail", nil, nil)
	var be *Error
	if !errors.As(err, &be) || be.Code != CodeStale {
		t.Fatalf("err = %v, want a stale *Error", err)
	}

	if err := c.Call(ctx(t), "emit", nil, nil); err != nil {
		t.Fatal(err)
	}
	select {
	case n := <-c.Notifications():
		if n.Method != "board.changed" || len(n.Params) == 0 {
			t.Fatalf("notification = %+v", n)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no notification")
	}

	// a call whose context ended gives up without breaking the client
	done, cancel := context.WithCancel(ctx(t))
	cancel()
	if err := c.Call(done, "echo", nil, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want canceled", err)
	}
	if err := c.Call(ctx(t), "echo", nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestClientClosedRefusesCalls(t *testing.T) {
	c, err := Start(ctx(t), fakeBin, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if err := c.Call(ctx(t), "echo", nil, nil); !errors.Is(err, ErrClosed) {
		t.Fatalf("err = %v, want ErrClosed", err)
	}
	if _, ok := <-c.Notifications(); ok {
		t.Fatal("notifications still open")
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestClientRestartsADeadBridge(t *testing.T) {
	t.Setenv("FAKE_STARTS", filepath.Join(t.TempDir(), "starts"))
	t.Setenv("FAKE_DIE_UNTIL", "1") // the first process dies at its first echo
	c, err := Start(ctx(t), fakeBin, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	if err := c.Call(ctx(t), "echo", nil, nil); !errors.Is(err, ErrConnectionLost) {
		t.Fatalf("err = %v, want ErrConnectionLost", err)
	}
	// the same client, a new process
	if err := c.Call(ctx(t), "echo", nil, nil); err != nil {
		t.Fatal(err)
	}
	select {
	case n := <-c.Notifications():
		if n.Method != MethodRestarted {
			t.Fatalf("notification = %+v", n)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no bridge.restarted")
	}
}

func TestClientGivesUpAfterThreeRestarts(t *testing.T) {
	t.Setenv("FAKE_STARTS", filepath.Join(t.TempDir(), "starts"))
	t.Setenv("FAKE_DIE_UNTIL", "100") // every process dies
	c, err := Start(ctx(t), fakeBin, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	lost := 0
	for i := 0; i < 10; i++ {
		err := c.Call(ctx(t), "echo", nil, nil)
		if errors.Is(err, ErrBridgeDown) {
			if lost != 4 { // the first process and three restarts each died on a call
				t.Fatalf("down after %d lost calls, want 4", lost)
			}
			// and it stays down
			if err := c.Call(ctx(t), "echo", nil, nil); !errors.Is(err, ErrBridgeDown) {
				t.Fatalf("err = %v, want ErrBridgeDown", err)
			}
			return
		}
		if !errors.Is(err, ErrConnectionLost) {
			t.Fatalf("call %d: err = %v", i, err)
		}
		lost++
	}
	t.Fatal("never down")
}

func TestFindLooksBeyondTheGUIPath(t *testing.T) {
	empty, first, second := t.TempDir(), t.TempDir(), t.TempDir()
	log := filepath.Join(t.TempDir(), "runs.log")
	put := func(dir string) string {
		p := fakeStickypane(t, dir, "0.2.0", true, log)
		resetProbes()
		return p
	}
	isolate(t, empty, filepath.Join(empty, "nope"), first, second)

	if _, err := Find(); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	pSecond := put(second)
	if got, err := Find(); err != nil || got != pSecond {
		t.Fatalf("Find = %q, %v; want %q", got, err, pSecond)
	}
	pFirst := put(first)
	if got, _ := Find(); got != pFirst {
		t.Fatalf("Find = %q, want the earlier directory %q", got, pFirst)
	}
	pPath := put(empty)
	if got, _ := Find(); got != pPath {
		t.Fatalf("Find = %q, want PATH first %q", got, pPath)
	}
	t.Setenv("STICKYPANE_BIN", pSecond)
	if got, _ := Find(); got != pSecond {
		t.Fatalf("Find = %q, want STICKYPANE_BIN %q", got, pSecond)
	}
	t.Setenv("STICKYPANE_BIN", filepath.Join(empty, "missing"))
	if _, err := Find(); !errors.Is(err, ErrNotFound) {
		t.Fatalf("a bad STICKYPANE_BIN: err = %v, want ErrNotFound", err)
	}
}
