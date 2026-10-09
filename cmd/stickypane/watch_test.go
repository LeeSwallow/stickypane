package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// watch prints what happens on the board, filtered, one event a line; with
// --once it waits for the first and ends, the way an agent waits for the
// user to do something.
func TestWatchWaitsForTheFirstEvent(t *testing.T) {
	root := project(t)
	if code, _, _ := execIn(t, "## To do\n- login\n## Done\n", "write", "work", "--type", "board"); code != 0 {
		t.Fatal("write")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var out, errOut bytes.Buffer
	done := make(chan int, 1)
	go func() {
		done <- runCtx(ctx, []string{"watch", "--once", "--json", "--note", "work", "--type", "card"}, strings.NewReader(""), &out, &errOut)
	}()
	time.Sleep(300 * time.Millisecond) // let it start watching
	if code, _, _ := exec(t, "card", "work", "move", "login", "--to", "Done"); code != 0 {
		t.Fatal("move")
	}
	select {
	case code := <-done:
		var e map[string]any
		if code != 0 || json.Unmarshal(out.Bytes(), &e) != nil || e["type"] != "card.moved" || e["item"] != "login" || e["to"] != "Done" || e["note"] != "work.md" {
			t.Errorf("code %d, out %q, err %q", code, out.String(), errOut.String())
		}
	case <-ctx.Done():
		t.Fatalf("no event: out %q err %q", out.String(), errOut.String())
	}
	_ = root
}

// --exec runs a command for each event, with what happened in STICKY_*
// variables: how one note reacts to another.
func TestWatchRunsACommandForEachEvent(t *testing.T) {
	if runtime.GOOS == "windows" {
		// --exec runs the line in the user's shell, which there is PowerShell
		// or cmd; this line is sh. That watch sees the event is
		// TestWatchWaitsForTheFirstEvent.
		t.Skip("the command is written for sh")
	}
	root := project(t)
	if code, _, _ := execIn(t, "- [ ] tests\n", "write", "plan", "--type", "checklist"); code != 0 {
		t.Fatal("write")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	marker := filepath.Join(root, "reacted.txt")
	var out, errOut bytes.Buffer
	done := make(chan int, 1)
	go func() {
		done <- runCtx(ctx, []string{"watch", "--once", "--type", "item.ticked", "--exec", `printf '%s %s' "$STICKY_TYPE" "$STICKY_ITEM" > ` + marker}, strings.NewReader(""), &out, &errOut)
	}()
	time.Sleep(300 * time.Millisecond)
	if code, _, _ := exec(t, "todo", "plan", "check", "tests"); code != 0 {
		t.Fatal("check")
	}
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("no event")
	}
	if b, _ := os.ReadFile(marker); string(b) != "item.ticked tests" {
		t.Errorf("the command saw %q; out %q err %q", b, out.String(), errOut.String())
	}
}
