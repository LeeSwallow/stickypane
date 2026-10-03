package httpfile

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LeeSwallow/stickypane/internal/proto/prototest"
	"github.com/LeeSwallow/stickypane/internal/rpc/rpctest"
	"github.com/LeeSwallow/stickypane/internal/wsock"
)

const protocolsFile = `@base = http://localhost

### Chat
# @websocket timeout=2s idle-timeout=300ms subprotocols=chat.v2,json
# @ws send {"type":"hello","user":"{{user}}"}
# @ws send-json {"type":"join"}
# @ws wait 50ms
# @ws ping beat
# @ws close 1000 bye
# @assert response.received >= 2
# @assert response.json("[0].type") == "welcome"
GET {{ws}}/chat
Authorization: Bearer {{token}}

### Get user
# @grpc users.v1.Users/Get
# @grpc-plaintext true
# @grpc-descriptor users.protoset
# @grpc-metadata x-trace-id: {{trace}}
# @timeout 3s
# @assert response.grpc.status == "OK"
# @assert response.json("name") == "min"
GRPC {{grpc}}

{"id": 7}
`

func TestTheFileSaysWhichProtocolEachRequestSpeaks(t *testing.T) {
	f := Parse(protocolsFile)
	if len(f.Requests) != 2 {
		t.Fatalf("requests = %d", len(f.Requests))
	}
	chat, get := f.Requests[0], f.Requests[1]
	if chat.Protocol() != "websocket" || get.Protocol() != "grpc" {
		t.Errorf("protocols %s, %s", chat.Protocol(), get.Protocol())
	}
	ws := chat.WS
	if ws == nil || ws.Timeout != 2*time.Second || ws.Idle != 300*time.Millisecond || strings.Join(ws.Subprotocols, ",") != "chat.v2,json" || len(ws.Steps) != 5 {
		t.Fatalf("ws = %+v", ws)
	}
	if s := ws.Steps[0]; s.Op != "send" || s.Arg != `{"type":"hello","user":"{{user}}"}` {
		t.Errorf("first step = %+v", s)
	}
	g := get.GRPC
	if g == nil || g.Method != "users.v1.Users/Get" || !g.Plaintext || !g.Reflection || g.Descriptor != "users.protoset" || len(g.Metadata) != 1 || get.Timeout != 3*time.Second || get.URL != "{{grpc}}" {
		t.Errorf("grpc = %+v, timeout %v, url %q", g, get.Timeout, get.URL)
	}
	if (Request{Method: "GET", URL: "wss://x"}).Protocol() != "websocket" || (Request{Method: "GET", URL: "https://x"}).Protocol() != "http" {
		t.Error("a ws:// URL is a WebSocket even without @websocket")
	}
}

// chatServer greets, echoes text, answers a close, and wants the token.
func chatServer(t *testing.T) string {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer s3cret" {
			http.Error(w, "who are you", http.StatusUnauthorized)
			return
		}
		c, err := wsock.Accept(w, r, "chat.v2")
		if err != nil {
			return
		}
		defer c.End()
		c.Send(wsock.Text, []byte(`{"type":"welcome"}`))
		for {
			m, err := c.Read(3 * time.Second)
			if err != nil {
				return
			}
			switch m.Kind {
			case wsock.Text:
				c.Send(wsock.Text, append([]byte(`{"type":"echo","got":`), append(m.Data, '}')...))
			case wsock.Close:
				c.Close(1000, "bye")
				return
			}
		}
	}))
	t.Cleanup(srv.Close)
	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

func runner(t *testing.T, vars map[string]string) (*Runner, string) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "users.protoset"), prototest.UsersSet(), 0o644); err != nil {
		t.Fatal(err)
	}
	return &Runner{Root: dir, Here: dir, Saved: vars}, dir
}

func TestAWebSocketSessionIsAScriptWithATranscript(t *testing.T) {
	rn, _ := runner(t, map[string]string{"ws": chatServer(t), "token": "s3cret", "user": "min"})
	res := rn.Send(context.Background(), Parse(protocolsFile), 0)
	if res.Err != nil {
		t.Fatalf("err: %v\n%s", res.Err, Log(res))
	}
	if res.Status != 101 || res.Protocol != "websocket" {
		t.Errorf("status %d, protocol %s", res.Status, res.Protocol)
	}
	var kinds []string
	for _, f := range res.Frames {
		dir := "←"
		if f.Out {
			dir = "→"
		}
		kinds = append(kinds, dir+f.Kind)
	}
	got := strings.Join(kinds, " ")
	for _, want := range []string{"←text", "→text", "→wait", "→ping", "←pong", "→close", "←close"} {
		if !strings.Contains(got, want) {
			t.Errorf("transcript %s is missing %s", got, want)
		}
	}
	if res.Passed() != 2 {
		t.Errorf("checks: %+v", res.Checks)
	}
	log := Log(res)
	t.Log("\n" + log)
	for _, want := range []string{"$ websocat", "Bearer ••••", `→ `, `{"type":"hello","user":"min"}`, "← ", `"type":"welcome"`, "101 Switching Protocols", "Chat", "↑", "↓", "2/2 ✔"} {
		if !strings.Contains(log, want) {
			t.Errorf("the log should have %q:\n%s", want, log)
		}
	}
	if strings.Contains(log, "s3cret") {
		t.Errorf("the token must not reach the log:\n%s", log)
	}
}

func TestARefusedWebSocketSaysWhy(t *testing.T) {
	rn, _ := runner(t, map[string]string{"ws": chatServer(t), "token": "wrong", "user": "min"})
	res := rn.Send(context.Background(), Parse(protocolsFile), 0)
	if res.Err == nil || !strings.Contains(res.Err.Error(), "401") || !res.Failed() {
		t.Errorf("err = %v", res.Err)
	}
}

func TestAGRPCCallEndsLikeARequest(t *testing.T) {
	rn, _ := runner(t, map[string]string{"grpc": rpctest.Server(t, false), "trace": "t-1"})
	res := rn.Send(context.Background(), Parse(protocolsFile), 1)
	if res.Err != nil {
		t.Fatalf("err: %v\n%s", res.Err, Log(res))
	}
	if res.Status != 200 || res.Text != "200 OK" || res.GRPC == nil || res.GRPC.Schema != "descriptor" || res.Header.Get("X-Got-Trace") != "t-1" {
		t.Errorf("status %d %q, grpc %+v, header %v", res.Status, res.Text, res.GRPC, res.Header)
	}
	if res.Passed() != 2 {
		t.Errorf("checks: %+v", res.Checks)
	}
	log := Log(res)
	t.Log("\n" + log)
	for _, want := range []string{"$ grpcurl -plaintext", "-H 'x-trace-id: t-1'", "users.v1.Users/Get", `"name": "min"`, "[200 OK", "Get user", "2/2 ✔"} {
		if !strings.Contains(log, want) {
			t.Errorf("the log should have %q:\n%s", want, log)
		}
	}
}

func TestAGRPCErrorIsAFailedRequestWithItsMessage(t *testing.T) {
	file := strings.Replace(protocolsFile, `{"id": 7}`, `{"id": "404"}`, 1)
	rn, _ := runner(t, map[string]string{"grpc": rpctest.Server(t, false), "trace": "t-1"})
	res := rn.Send(context.Background(), Parse(file), 1)
	if res.Status != 404 || res.Text != "404 NOT_FOUND" || !res.Failed() {
		t.Errorf("status %d %q", res.Status, res.Text)
	}
	if end := End(res); !strings.Contains(end, "user 404 not found") || !strings.HasPrefix(end, "[404 NOT_FOUND") {
		t.Errorf("end = %s", end)
	}
	if m := EndLine.FindStringSubmatch(End(res)); m == nil || m[1] != "404" {
		t.Errorf("the board reads the end line: %v", m)
	}
}

func TestReflectionAndRawCallsNeedNoDescriptorFile(t *testing.T) {
	file := strings.Replace(protocolsFile, "# @grpc-descriptor users.protoset\n", "", 1)
	rn, _ := runner(t, map[string]string{"grpc": rpctest.Server(t, true), "trace": "t"})
	if res := rn.Send(context.Background(), Parse(file), 1); res.Err != nil || res.GRPC.Schema != "reflection" || res.Passed() != 2 {
		t.Errorf("reflection: %v, %+v\n%s", res.Err, res.GRPC, Log(res))
	}
	raw := strings.Replace(strings.Replace(file, `{"id": 7}`, `{"1": 7}`, 1), `response.json("name")`, `response.json("2")`, 1)
	rn, _ = runner(t, map[string]string{"grpc": rpctest.Server(t, false), "trace": "t"})
	if res := rn.Send(context.Background(), Parse(raw), 1); res.Err != nil || !strings.HasPrefix(res.GRPC.Schema, "raw") || res.Passed() != 2 {
		t.Errorf("raw: %v, %+v\n%s", res.Err, res.GRPC, Log(res))
	}
}

// A .http file's log keeps the last response of each request under its
// title, so sending one request does not wipe out what another got.
func TestALogKeepsASectionPerRequest(t *testing.T) {
	order := []string{"Chat", "Get user"}
	log := PutSection("", "Get user", "$ grpcurl x\n[200 OK · 1ms · Get user]\n", order)
	log = PutSection(log, "Chat", "$ websocat y\n[101 Switching Protocols · 3ms · Chat]\n", order)
	if !strings.HasPrefix(log, "### Chat\n$ websocat y\n") || !strings.Contains(log, "\n### Get user\n$ grpcurl x\n") {
		t.Fatalf("sections in the file's order:\n%s", log)
	}
	log = PutSection(log, "Get user", "$ grpcurl z\n[404 NOT_FOUND · 2ms · Get user]\n", order)
	if s, ok := Section(log, "Get user"); !ok || !strings.Contains(s, "grpcurl z") || strings.Contains(s, "grpcurl x") {
		t.Errorf("a resend replaces its section: %q", s)
	}
	if s, ok := Section(log, "Chat"); !ok || !strings.Contains(s, "websocat y") {
		t.Errorf("the other section stays: %q", s)
	}
	if _, ok := Section(log, "Nope"); ok {
		t.Error("a request never sent has no section")
	}
	// A log written before sections is replaced, not mixed in.
	if got := PutSection("$ curl old\n[200 OK]\n", "Chat", "new\n", order); strings.Contains(got, "old") {
		t.Errorf("an old log goes: %q", got)
	}
}
