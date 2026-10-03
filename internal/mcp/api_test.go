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

	"github.com/LeeSwallow/stickypane/internal/api"
	"github.com/LeeSwallow/stickypane/internal/kinds"
	"github.com/LeeSwallow/stickypane/internal/mcp"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/widget/note"
)

// The experimental api tool lists the requests of a .http note and sends
// one or all, so an agent can call the REST API the user keeps on the board.
func TestTheAPIToolListsAndSends(t *testing.T) {
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
