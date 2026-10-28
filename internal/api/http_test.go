package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/kinds"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/widget/note"
)

// SendPart is how every front end sends one request: it answers how the
// request ended and writes its log, and Request finds the part by title.
func TestSendPartSendsOneRequestAndKeepsItsLog(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bad" {
			w.WriteHeader(http.StatusTeapot)
		}
	}))
	defer srv.Close()
	dir := filepath.Join(t.TempDir(), store.DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "### ping\nGET " + srv.URL + "/ping\n\n### bad\nGET " + srv.URL + "/bad\n"
	if err := os.WriteFile(filepath.Join(dir, "live.http"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	a := New(store.Open(dir), kinds.Default(note.Plain))
	part, err := a.Request("live.http", "bad")
	if err != nil || part != 1 {
		t.Fatalf("Request = %d, %v", part, err)
	}
	if _, err := a.Request("live.http", "nothing like it"); err == nil {
		t.Error("a request that is not there is an error")
	}
	got, err := a.SendPart(context.Background(), "live.http", part, "")
	if err != nil || got.Status != http.StatusTeapot || !got.Failed || got.Log != "live.log" || !strings.Contains(got.End, "418") {
		t.Fatalf("SendPart = %+v, %v", got, err)
	}
	if got, _ := a.SendPart(context.Background(), "live.http", 0, ""); got.Failed || got.Status != http.StatusOK {
		t.Errorf("ping = %+v", got)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "live.log")); !strings.Contains(string(b), "ping") {
		t.Errorf("log:\n%s", b)
	}
}
