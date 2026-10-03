package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/LeeSwallow/stickypane/internal/api"
	"github.com/LeeSwallow/stickypane/internal/kinds"
	"github.com/LeeSwallow/stickypane/internal/mcp"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/widget/note"
)

// wait_event waits for the first event on the board that the call names,
// so an agent can react to what the user does without being told.
func TestWaitEventReturnsWhatHappened(t *testing.T) {
	dir := filepath.Join(t.TempDir(), store.DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	plan := filepath.Join(dir, "plan.md")
	if err := os.WriteFile(plan, []byte("---\ntype: checklist\n---\n- [ ] tests\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := &mcp.Server{API: api.New(store.Open(dir), kinds.Default(note.Plain)), Version: "test"}
	in, feed := io.Pipe()
	var out lockedBuffer
	done := make(chan error, 1)
	go func() { done <- s.Serve(context.Background(), in, &out) }()
	io.WriteString(feed, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"wait_event","arguments":{"notes":"plan","types":"item","wait_seconds":5}}}`+"\n")
	time.Sleep(300 * time.Millisecond)
	if err := os.WriteFile(plan, []byte("---\ntype: checklist\n---\n- [x] tests\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for !strings.Contains(out.String(), "item.ticked") && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	feed.Close()
	<-done
	var r struct {
		Result struct {
			Content []struct{ Text string } `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal(bytes.TrimSpace([]byte(out.String())), &r); err != nil || len(r.Result.Content) == 0 {
		t.Fatalf("response %q: %v", out.String(), err)
	}
	var e map[string]any
	if err := json.Unmarshal([]byte(r.Result.Content[0].Text), &e); err != nil || e["type"] != "item.ticked" || e["item"] != "tests" || e["note"] != "plan.md" {
		t.Errorf("event = %q", r.Result.Content[0].Text)
	}
}

// lockedBuffer is a buffer the server writes to while the test reads it.
type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}
