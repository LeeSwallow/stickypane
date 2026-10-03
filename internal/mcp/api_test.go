package mcp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LeeSwallow/stickypane/internal/api"
	"github.com/LeeSwallow/stickypane/internal/kinds"
	"github.com/LeeSwallow/stickypane/internal/mcp"
	"github.com/LeeSwallow/stickypane/internal/proto/prototest"
	"github.com/LeeSwallow/stickypane/internal/rpc/rpctest"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/widget/note"
	"github.com/LeeSwallow/stickypane/internal/wsock"
)

// The experimental api tool lists the requests of a .http note and sends
// one or all, so an agent can call the REST API the user keeps on the board.
func TestTheAPIToolListsAndSends(t *testing.T) {
	testAPITool(t)
}

// An agent sends a REST request, a WebSocket session and a gRPC call from
// one note in one call, and reads each one's log.
func TestTheAPIToolSpeaksEveryProtocol(t *testing.T) {
	rest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer rest.Close()
	ws := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := wsock.Accept(w, r)
		if err != nil {
			return
		}
		defer c.End()
		for {
			m, err := c.Read(2 * time.Second)
			if err != nil || m.Kind == wsock.Close {
				c.Close(1000, "")
				return
			}
			c.Send(wsock.Text, append([]byte("got "), m.Data...))
		}
	}))
	defer ws.Close()
	grpc := rpctest.Server(t, false)
	dir := filepath.Join(t.TempDir(), store.DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "users.protoset"), prototest.UsersSet(), 0o644)
	src := "### Health\nGET " + rest.URL + "/health\n\n" +
		"### Echo\n# @ws send hello\n# @ws close\n# @assert response.json(\"[0]\") == \"got hello\"\nGET ws" + strings.TrimPrefix(ws.URL, "http") + "\n\n" +
		"### Get user\n# @grpc users.v1.Users/Get\n# @grpc-plaintext true\n# @grpc-descriptor users.protoset\n# @assert response.json(\"name\") == \"min\"\nGRPC " + grpc + "\n\n{\"id\": 7}\n"
	if err := os.WriteFile(filepath.Join(dir, "svc.http"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	s := &mcp.Server{API: api.New(store.Open(dir), kinds.Default(note.Plain)), Version: "test"}
	got := toolText(t, s, `{"name":"svc"}`)
	for _, want := range []string{"#1  GET    Health", "#2  WS     Echo", "#3  GRPC   Get user  users.v1.Users/Get"} {
		if !strings.Contains(got, want) {
			t.Errorf("listing should have %q:\n%s", want, got)
		}
	}
	got = toolText(t, s, `{"name":"svc","all":true}`)
	for _, want := range []string{"[200 OK", "Health", "→ ", "← ", "got hello", "101 Switching Protocols", "1/1 ✔", `"name": "min"`, "Get user"} {
		if !strings.Contains(got, want) {
			t.Errorf("the logs should have %q:\n%s", want, got)
		}
	}
}

func toolText(t *testing.T, s *mcp.Server, args string) string {
	t.Helper()
	var out strings.Builder
	req := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"api","arguments":` + args + `}}` + "\n"
	if err := s.Serve(context.Background(), strings.NewReader(req), &out); err != nil {
		t.Fatal(err)
	}
	var r struct {
		Result struct {
			Content []struct{ Text string } `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out.String())), &r); err != nil || len(r.Result.Content) == 0 {
		t.Fatalf("response %q: %v", out.String(), err)
	}
	return r.Result.Content[0].Text
}

func testAPITool(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()
	dir := filepath.Join(t.TempDir(), store.DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := "### Health\nGET " + srv.URL + "/health\n\n### Who\nGET " + srv.URL + "/me\n"
	if err := os.WriteFile(filepath.Join(dir, "svc.http"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	s := &mcp.Server{API: api.New(store.Open(dir), kinds.Default(note.Plain)), Version: "test"}
	call := func(args string) string {
		var out strings.Builder
		req := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"api","arguments":` + args + `}}` + "\n"
		if err := s.Serve(context.Background(), strings.NewReader(req), &out); err != nil {
			t.Fatal(err)
		}
		var r struct {
			Result struct {
				Content []struct{ Text string } `json:"content"`
				IsError bool                    `json:"isError"`
			} `json:"result"`
		}
		if err := json.Unmarshal([]byte(strings.TrimSpace(out.String())), &r); err != nil || len(r.Result.Content) == 0 {
			t.Fatalf("response %q: %v", out.String(), err)
		}
		return r.Result.Content[0].Text
	}
	if got := call(`{"name":"svc"}`); !strings.Contains(got, "#1") || !strings.Contains(got, "Health") || !strings.Contains(got, "Who") {
		t.Errorf("listing = %q", got)
	}
	if got := call(`{"name":"svc","request":"health"}`); !strings.Contains(got, "200") || !strings.Contains(got, `"ok": true`) {
		t.Errorf("sending = %q", got)
	}
}
