package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LeeSwallow/stickypane/internal/adapter/store"
	"github.com/LeeSwallow/stickypane/internal/core/kinds"
	"github.com/LeeSwallow/stickypane/internal/core/widget/note"
	"github.com/LeeSwallow/stickypane/internal/surface/mcp"
	"github.com/LeeSwallow/stickypane/internal/usecase/api"
)

// toolPairs are the tools before <noun> + action (#162) and the calls that
// replace them, in an order where each step has what it needs.
var toolPairs = []struct{ old, oldArgs, new, newArgs string }{
	{"write_note", `{"name":"plan","content":"- [ ] a\n","type":"checklist","title":"Plan"}`, "note", `{"action":"write","name":"plan","content":"- [ ] a\n","type":"checklist","title":"Plan"}`},
	{"read_note", `{"name":"plan"}`, "note", `{"action":"read","name":"plan"}`},
	{"set_keys", `{"name":"plan","keys":{"size":"1/2","open":true}}`, "note", `{"action":"set","name":"plan","keys":{"size":"1/2","open":true}}`},
	{"log", `{"name":"worklog","line":"x"}`, "log", `{"action":"add","name":"worklog","line":"x"}`},
	{"say", `{"name":"chat","text":"hi"}`, "chat", `{"action":"add","name":"chat","text":"hi"}`},
	{"say", `{"name":"chat","text":"ok","as":"ci"}`, "chat", `{"action":"add","name":"chat","text":"ok","as":"ci"}`},
	{"index", `{}`, "note", `{"action":"list"}`},
	{"arrange_note", `{"name":"plan","action":"hide"}`, "note", `{"action":"hide","name":"plan"}`},
	{"arrange_note", `{"name":"plan","action":"show"}`, "note", `{"action":"show","name":"plan"}`},
	{"arrange_note", `{"name":"plan","action":"move","to":"roadmap"}`, "note", `{"action":"mv","name":"plan","to":"roadmap"}`},
	{"arrange_note", `{"name":"worklog","action":"remove"}`, "note", `{"action":"rm","name":"worklog"}`},
	{"arrange_note", `{"name":"worklog","action":"restore"}`, "note", `{"action":"restore","name":"worklog"}`},
	{"arrange_note", `{"name":"roadmap","action":"list"}`, "note", `{"action":"nope","name":"roadmap"}`},
}

// Every old tool and its new tool and action give the same answer and
// leave the same files, step by step.
func TestOldAndNewToolsDoTheSame(t *testing.T) {
	at := time.Date(2026, 10, 3, 14, 2, 5, 0, time.Local)
	old, new := t.TempDir(), t.TempDir()
	call := func(root, tool, args string) toolResult {
		t.Helper()
		dir := filepath.Join(root, store.DirName)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		s := &mcp.Server{API: api.NewWithClock(store.Open(dir), kinds.Default(note.Plain), func() time.Time { return at }), Version: "test"}
		var out bytes.Buffer
		req := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":%q,"arguments":%s}}`+"\n", tool, args)
		if err := s.Serve(context.Background(), strings.NewReader(req), &out); err != nil {
			t.Fatal(err)
		}
		var r response
		if err := json.Unmarshal(out.Bytes(), &r); err != nil || r.Error != nil {
			t.Fatalf("%s %s: %s", tool, args, out.String())
		}
		var res toolResult
		if err := json.Unmarshal(r.Result, &res); err != nil {
			t.Fatal(err)
		}
		return res
	}
	for _, p := range toolPairs {
		a, b := call(old, p.old, p.oldArgs), call(new, p.new, p.newArgs)
		wantErr := strings.Contains(p.newArgs, "nope")
		if a.IsError != wantErr || b.IsError != wantErr {
			t.Errorf("%s %s: failed %v, %s %s: failed %v; want %v", p.old, p.oldArgs, a.IsError, p.new, p.newArgs, b.IsError, wantErr)
		}
		if !wantErr && a.Content[0].Text != b.Content[0].Text {
			t.Errorf("%s says %q, %s %s says %q", p.old, a.Content[0].Text, p.new, p.newArgs, b.Content[0].Text)
		}
		if a, b := boardFiles(t, old), boardFiles(t, new); a != b {
			t.Fatalf("after %s and %s %s the boards differ:\n%s\n---\n%s", p.old, p.new, p.newArgs, a, b)
		}
	}
}

// boardFiles is every file of the board under root, with its contents.
func boardFiles(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	dir := filepath.Join(root, store.DirName)
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		body, err := os.ReadFile(path)
		b.WriteString("== " + rel + "\n" + string(body) + "\n")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return b.String()
}
