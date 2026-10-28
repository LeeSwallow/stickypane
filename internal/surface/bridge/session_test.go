package bridge

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/LeeSwallow/stickypane/internal/core/kinds"
	"github.com/LeeSwallow/stickypane/internal/core/widget/note"
)

// A script that never ends does not keep the bridge alive once the app is
// gone: the session ends, and the script is stopped.
func TestServeStopsRunningScriptsWhenTheSessionEnds(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh")
	}
	dir := board(t, map[string]string{"long.sh": "exec sleep 60\n"})
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	go io.Copy(io.Discard, outR)
	done := make(chan error, 1)
	go func() {
		done <- Serve(context.Background(), newAPI(dir), kinds.Default(note.Plain), inR, outW)
		outW.Close()
	}()
	io.WriteString(inW, `{"id":1,"method":"note.run","params":{"note":"long.sh"}}`+"\n")
	deadline := time.Now().Add(5 * time.Second)
	for !strings.Contains(readOrEmpty(dir, "long.log"), "$ ") {
		if time.Now().After(deadline) {
			t.Fatal("the script never started")
		}
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(100 * time.Millisecond)
	start := time.Now()
	inW.Close() // the app went away
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Serve: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Serve kept the session alive for a script that never ends")
	}
	if took := time.Since(start); took > 8*time.Second {
		t.Errorf("took %v to end", took)
	}
}

func readOrEmpty(dir, name string) string {
	b, _ := os.ReadFile(filepath.Join(dir, name))
	return string(b)
}

func goroutinesIn(substr string) int {
	buf := make([]byte, 1<<20)
	buf = buf[:runtime.Stack(buf, true)]
	n := 0
	for _, g := range strings.Split(string(buf), "\n\n") {
		if strings.Contains(g, substr) {
			n++
		}
	}
	return n
}

// Serve called in-process and cancelled leaves nothing behind, not even
// the goroutine that reads the input.
func TestServeLeavesNoReaderBehindWhenCancelled(t *testing.T) {
	dir := board(t, nil)
	inR, inW := io.Pipe()
	defer inW.Close()
	outR, outW := io.Pipe()
	go io.Copy(io.Discard, outR)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Serve(ctx, newAPI(dir), kinds.Default(note.Plain), inR, outW) }()
	io.WriteString(inW, `{"id":1,"method":"hello"}`+"\n")
	time.Sleep(100 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return on cancel")
	}
	for i := 0; ; i++ {
		if goroutinesIn("bridge.Serve") == 0 {
			return
		}
		if i > 100 {
			t.Fatalf("goroutines of Serve are left:\n%d", goroutinesIn("bridge.Serve"))
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// The watcher runs before the board is first read, so that a change made
// in between is seen by one or the other.
func TestServeWatchesBeforeItReadsTheBoard(t *testing.T) {
	dir := board(t, nil)
	var mu sync.Mutex
	var order []string
	trace = func(what string) { mu.Lock(); order = append(order, what); mu.Unlock() }
	defer func() { trace = func(string) {} }()
	s := serve(t, dir)
	s.ok("hello", nil, nil)
	mu.Lock()
	defer mu.Unlock()
	if got := fmt.Sprint(order); got != "[watching read]" {
		t.Errorf("order = %s, want the watcher first", got)
	}
}

// Many callers choosing the same title never write over each other.
func TestCreateNeverOverwritesAFileMadeInBetween(t *testing.T) {
	dir := board(t, nil)
	srv := &server{api: newAPI(dir), reg: kinds.Default(note.Plain)}
	const n = 12
	var wg sync.WaitGroup
	names := make(chan string, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := srv.noteCreate([]byte(fmt.Sprintf(`{"title":"Same","body":"body %d"}`, i)))
			if err != nil {
				t.Error(err)
				return
			}
			names <- res.(map[string]string)["note"]
		}()
	}
	wg.Wait()
	close(names)
	seen := map[string]bool{}
	for name := range names {
		seen[name] = true
	}
	if len(seen) != n {
		t.Errorf("%d distinct notes of %d creates: %v", len(seen), n, seen)
	}
	for i := 0; i < n; i++ {
		found := false
		for name := range seen {
			found = found || strings.Contains(readOrEmpty(dir, name), fmt.Sprintf("body %d", i))
		}
		if !found {
			t.Errorf("body %d was written over", i)
		}
	}
}

// A book whose folder has a dot in its name is restored as a book.
func TestBridgeRestoresABookWithADotInItsName(t *testing.T) {
	dir := board(t, map[string]string{"t/v1.2/a.md": "a\n", "t/v1.2/b.md": "b\n", "t/keep.md": "k\n"})
	s := serve(t, dir)
	s.ok("note.trash", map[string]any{"note": "t/v1.2"}, nil)
	if _, err := os.Stat(filepath.Join(dir, "t", "v1.2")); err == nil {
		t.Fatal("the book is still there")
	}
	s.ok("note.restore", map[string]any{"note": "t/v1.2"}, nil)
	if read(t, dir, "t/v1.2/a.md") != "a\n" || read(t, dir, "t/v1.2/b.md") != "b\n" {
		t.Error("the pages did not come back")
	}
}
