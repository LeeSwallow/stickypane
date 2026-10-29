package bridgeclient

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	shortTimeout = 300 * time.Millisecond
	slack        = time.Second // a call whose context ended returns within it
)

func short() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), shortTimeout)
}

// gaveUp fails unless err is the short context's end and came promptly.
func gaveUp(t *testing.T, what string, err error, start time.Time) {
	t.Helper()
	if d := time.Since(start); !errors.Is(err, context.DeadlineExceeded) || d > shortTimeout+slack {
		t.Fatalf("%s: err = %v after %v, want the deadline within %v", what, err, d, shortTimeout+slack)
	}
}

func TestClientNeverWaitsForTheConsumer(t *testing.T) {
	c, err := Start(ctx(t), fakeBin, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	// nobody reads Notifications while the bridge floods
	if err := c.Call(ctx(t), "flood", map[string]int{"n": 20000}, nil); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 1000; i++ {
		if err := c.Call(ctx(t), "echo", nil, nil); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	// what is kept is bounded, and the newest is there
	got := 0
	for {
		select {
		case n := <-c.Notifications():
			got++
			if string(n.Params) == `{"i":19999}` {
				if got > queueMax+16+1 {
					t.Fatalf("kept %d notifications", got)
				}
				return
			}
		case <-time.After(2 * time.Second):
			t.Fatal("the newest notification was lost")
		}
	}
}

func TestClientRestartedIsNeverDropped(t *testing.T) {
	c, err := Start(ctx(t), fakeBin, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.notify(Notification{Method: MethodRestarted})
	for i := 0; i < 3*queueMax; i++ {
		c.notify(Notification{Method: "board.changed"})
	}
	for {
		select {
		case n := <-c.Notifications():
			if n.Method == MethodRestarted {
				return
			}
		case <-time.After(2 * time.Second):
			t.Fatal("bridge.restarted was dropped")
		}
	}
}

func TestClientCancelledCallReturnsWhileTheWriteBlocks(t *testing.T) {
	c, err := Start(ctx(t), fakeBin, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	x, cancel := short()
	defer cancel()
	c.Call(x, "stall", nil, nil) // the bridge reads no more
	big := string(make([]byte, 4<<20))
	start := time.Now()
	y, cancel2 := short()
	defer cancel2()
	gaveUp(t, "the call whose write blocks", c.Call(y, "echo", big, nil), start) // fills the pipe
	// a call that waits for the pipe gives up with its context too
	z, cancel3 := short()
	defer cancel3()
	start = time.Now()
	gaveUp(t, "the call that waits for the pipe", c.Call(z, "echo", nil, nil), start)
}

func TestClientAbandonedWriteLeavesTheStreamWhole(t *testing.T) {
	c, err := Start(ctx(t), fakeBin, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err := c.Call(ctx(t), "pause", map[string]int{"ms": 1500}, nil); err != nil { // reads nothing for a while
		t.Fatal(err)
	}
	big := strings.Repeat("a", 4<<20) // quick to encode: the line is begun, and blocks
	y, cancel := short()
	defer cancel()
	start := time.Now()
	gaveUp(t, "the call whose write blocks", c.Call(y, "sink", big, nil), start)
	// a line given up mid-way is still finished whole before the next; one
	// whose context ended before it was begun is not written at all
	var got struct{ Lines, Bad int }
	if err := c.Call(ctx(t), "lines", nil, &got); err != nil {
		t.Fatal(err)
	}
	if got.Bad != 0 || got.Lines < 3 || got.Lines > 4 { // hello, pause, sink perhaps, lines
		t.Fatalf("the bridge read %+v", got)
	}
	select {
	case n := <-c.Notifications():
		t.Fatalf("got %v: the bridge was started again", n.Method)
	default:
	}
}

func TestClientCloseIsNotHeldByAGrandchild(t *testing.T) {
	t.Setenv("FAKE_GRANDCHILD", "1")
	c, err := Start(ctx(t), fakeBin, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	c.Close()
	if d := time.Since(start); d > 8*time.Second {
		t.Fatalf("Close took %v", d)
	}
}

func TestClientErrorsCarryTheBridgesStderr(t *testing.T) {
	t.Setenv("FAKE_STARTS", filepath.Join(t.TempDir(), "starts"))
	t.Setenv("FAKE_DIE_UNTIL", "100")
	c, err := Start(ctx(t), fakeBin, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	var err2 error
	for i := 0; i < 10 && !errors.Is(err2, ErrBridgeDown); i++ {
		err2 = c.Call(ctx(t), "echo", nil, nil)
	}
	if !errors.Is(err2, ErrBridgeDown) || !strings.Contains(err2.Error(), "boom: the fake died") {
		t.Fatalf("err = %v", err2)
	}
}

func TestClientProtocolMismatchOnRestartSurfaces(t *testing.T) {
	t.Setenv("FAKE_STARTS", filepath.Join(t.TempDir(), "starts"))
	t.Setenv("FAKE_DIE_UNTIL", "1")
	t.Setenv("FAKE_PROTO_FROM", "2")
	c, err := Start(ctx(t), fakeBin, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err := c.Call(ctx(t), "echo", nil, nil); !errors.Is(err, ErrConnectionLost) {
		t.Fatalf("err = %v", err)
	}
	if err := c.Call(ctx(t), "echo", nil, nil); !errors.Is(err, ErrProtocol) {
		t.Fatalf("err = %v, want ErrProtocol", err)
	}
}
