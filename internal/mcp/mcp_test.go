package mcp_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/api"
	"github.com/LeeSwallow/stickypane/internal/kinds"
	"github.com/LeeSwallow/stickypane/internal/mcp"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/widget/note"
)

type response struct {
	ID     json.RawMessage `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type toolResult struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	IsError bool `json:"isError"`
}

// serve runs the server over the given request lines and returns one parsed
// response per line of output.
func serve(t *testing.T, requests ...string) ([]response, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), store.DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("---\ntitle: First\n---\nhello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := &mcp.Server{API: api.New(store.Open(dir), kinds.Default(note.Plain)), Guide: "GUIDE TEXT", Version: "test"}
	var out bytes.Buffer
	if err := s.Serve(strings.NewReader(strings.Join(requests, "\n")+"\n"), &out); err != nil {
		t.Fatal(err)
	}
	var responses []response
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if line == "" {
			continue
		}
		var r response
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			t.Fatalf("output line is not JSON: %q", line)
		}
		responses = append(responses, r)
	}
	return responses, dir
}

// byID finds the response to a request. A call that waits for the user is
// answered on the side, so responses need not come back in request order.
func byID(t *testing.T, rs []response, id int) response {
	t.Helper()
	for _, r := range rs {
		if string(r.ID) == strconv.Itoa(id) {
			return r
		}
	}
	t.Fatalf("no response with id %d in %+v", id, rs)
	return response{}
}

func call(t *testing.T, r response) toolResult {
	t.Helper()
	if r.Error != nil {
		t.Fatalf("unexpected protocol error: %+v", r.Error)
	}
	var res toolResult
	if err := json.Unmarshal(r.Result, &res); err != nil || len(res.Content) != 1 || res.Content[0].Type != "text" {
		t.Fatalf("tool result should be one text block: %s", r.Result)
	}
	return res
}

func TestInitializeAndListTools(t *testing.T) {
	rs, _ := serve(t,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"ping"}`,
	)
	if len(rs) != 3 {
		t.Fatalf("got %d responses, want 3: a notification gets no answer", len(rs))
	}
	var init struct {
		ProtocolVersion string `json:"protocolVersion"`
		Capabilities    struct {
			Tools *struct{} `json:"tools"`
		} `json:"capabilities"`
		ServerInfo struct{ Name, Version string } `json:"serverInfo"`
	}
	if err := json.Unmarshal(rs[0].Result, &init); err != nil {
		t.Fatal(err)
	}
	if init.ProtocolVersion != "2025-06-18" || init.Capabilities.Tools == nil || init.ServerInfo.Name != "stickypane" {
		t.Errorf("initialize result = %s", rs[0].Result)
	}
	var list struct {
		Tools []struct {
			Name        string          `json:"name"`
			Description string          `json:"description"`
			InputSchema json.RawMessage `json:"inputSchema"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(rs[1].Result, &list); err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range list.Tools {
		names = append(names, tool.Name)
		if tool.Description == "" || !strings.Contains(string(tool.InputSchema), `"type":"object"`) {
			t.Errorf("tool %s needs a description and an object schema: %s", tool.Name, tool.InputSchema)
		}
	}
	if got := strings.Join(names, ","); got != "list_notes,read_note,write_note,todo,card,chart,log,set_keys,show_note,hide_note,move_note,remove_note,restore_note,read_answers,guide" {
		t.Errorf("tools = %s", got)
	}
	if string(rs[2].ID) != "3" || string(rs[2].Result) != "{}" {
		t.Errorf("ping = %s %s", rs[2].ID, rs[2].Result)
	}
}

func TestToolsReadAndWriteNotes(t *testing.T) {
	rs, dir := serve(t,
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_notes","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"read_note","arguments":{"name":"a"}}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"write_note","arguments":{"name":"plan","type":"checklist","title":"Plan","open":true,"size":"half","content":"- [ ] 한글 항목\n"}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"guide","arguments":{}}}`,
	)
	if got := call(t, rs[0]).Content[0].Text; !strings.Contains(got, `"name":"a.md"`) || !strings.Contains(got, `"title":"First"`) {
		t.Errorf("list_notes = %s", got)
	}
	if got := call(t, rs[1]).Content[0].Text; got != "---\ntitle: First\n---\nhello\n" {
		t.Errorf("read_note = %q", got)
	}
	if got := call(t, rs[2]); got.IsError || !strings.Contains(got.Content[0].Text, "plan.md") {
		t.Errorf("write_note = %+v", got)
	}
	b, err := os.ReadFile(filepath.Join(dir, "plan.md"))
	if err != nil || string(b) != "---\ntype: checklist\ntitle: Plan\nopen: true\nsize: half\n---\n- [ ] 한글 항목\n" {
		t.Errorf("file = %q, %v", b, err)
	}
	if got := call(t, rs[3]).Content[0].Text; got != "GUIDE TEXT" {
		t.Errorf("guide = %q", got)
	}
}

func TestToolFailuresAreToolErrorsNotCrashes(t *testing.T) {
	rs, _ := serve(t,
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"read_note","arguments":{"name":"missing"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"write_note","arguments":{"name":"../escape","content":"x"}}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"no_such_tool","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"no/such/method"}`,
		`this is not json`,
		`{"jsonrpc":"2.0","id":5,"method":"ping"}`,
	)
	if len(rs) != 6 {
		t.Fatalf("got %d responses, want 6", len(rs))
	}
	wipe, dir := serve(t, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"write_note","arguments":{"name":"a"}}}`)
	if res := call(t, wipe[0]); !res.IsError {
		t.Errorf("write_note without content must fail, got %+v", res)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "a.md")); string(b) != "---\ntitle: First\n---\nhello\n" {
		t.Errorf("write_note without content emptied the note: %q", b)
	}
	for i := 0; i < 2; i++ {
		if res := call(t, rs[i]); !res.IsError || res.Content[0].Text == "" {
			t.Errorf("response %d should be a tool error with a message: %+v", i+1, res)
		}
	}
	if rs[2].Error == nil || rs[2].Error.Code != -32602 {
		t.Errorf("unknown tool: %+v, want error -32602", rs[2].Error)
	}
	if rs[3].Error == nil || rs[3].Error.Code != -32601 {
		t.Errorf("unknown method: %+v, want error -32601", rs[3].Error)
	}
	if rs[4].Error == nil || rs[4].Error.Code != -32700 {
		t.Errorf("bad JSON: %+v, want error -32700", rs[4].Error)
	}
	if string(rs[5].Result) != "{}" {
		t.Errorf("the server should keep serving after errors: %s", rs[5].Result)
	}
}

func TestReadAnswers(t *testing.T) {
	form := `---\ntype: form\nsubmitted: Go\n---\n## Where?\n- (x) staging\n- ( ) production\n\n[ Go ]\n`
	rs, _ := serve(t,
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"write_note","arguments":{"name":"deploy","content":"`+form+`"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"read_answers","arguments":{"name":"deploy"}}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"read_answers","arguments":{"name":"a"}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"write_note","arguments":{"name":"ask","type":"form","content":"- ( ) yes\n"}}}`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"read_answers","arguments":{"name":"ask","wait_seconds":0.05}}}`,
	)
	if got := call(t, byID(t, rs, 2)); got.IsError || got.Content[0].Text != "submitted: Go\nWhere?: staging\n" {
		t.Errorf("read_answers = %+v", got)
	}
	if got := call(t, byID(t, rs, 3)); !got.IsError || !strings.Contains(got.Content[0].Text, "form") {
		t.Errorf("read_answers of a plain note should fail: %+v", got)
	}
	if got := call(t, byID(t, rs, 5)); got.IsError || !strings.HasPrefix(got.Content[0].Text, "submitted: no\n") {
		t.Errorf("a wait that runs out returns the answers so far: %+v", got)
	}
}

func TestAWaitDoesNotBlockOtherRequests(t *testing.T) {
	rs, _ := serve(t,
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"write_note","arguments":{"name":"ask","type":"form","content":"- ( ) yes\n"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"read_answers","arguments":{"name":"ask","wait_seconds":30}}}`,
		`{"jsonrpc":"2.0","id":3,"method":"ping"}`,
	)
	// The input ends right after the ping: the ping must have been answered
	// while the wait was still running, and the wait must end with the input.
	if len(rs) != 3 || string(rs[1].ID) != "3" || string(rs[2].ID) != "2" {
		t.Fatalf("responses = %+v", rs)
	}
	if got := call(t, rs[2]); got.IsError || !strings.HasPrefix(got.Content[0].Text, "submitted: no\n") {
		t.Errorf("a wait cut short returns the answers so far: %+v", got)
	}
}

func TestAWaitCanBeCancelled(t *testing.T) {
	rs, _ := serve(t,
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"write_note","arguments":{"name":"ask","type":"form","content":"- ( ) yes\n"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"read_answers","arguments":{"name":"ask","wait_seconds":30}}}`,
		`{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":2}}`,
		`{"jsonrpc":"2.0","id":3,"method":"ping"}`,
	)
	if len(rs) != 3 {
		t.Fatalf("responses = %+v", rs)
	}
}

func TestSmallEdits(t *testing.T) {
	rs, dir := serve(t,
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"todo","arguments":{"name":"plan","action":"add","item":"write tests"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"todo","arguments":{"name":"plan","action":"check","item":"tests"}}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"card","arguments":{"name":"work","action":"add","card":"login API","to":"Doing"}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"chart","arguments":{"name":"tests","action":"set","label":"app","value":67}}}`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"chart","arguments":{"name":"tests","action":"add","label":"app","value":"3"}}}`,
		`{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"log","arguments":{"name":"worklog","line":"tests passed"}}}`,
		`{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"set_keys","arguments":{"name":"plan","keys":{"open":false,"size":"card","rows":8}}}}`,
		`{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"todo","arguments":{"name":"plan","action":"check","item":"nothing"}}}`,
	)
	want := map[int]string{1: "plan.md: 0/1", 2: "plan.md: 1/1", 3: "work.md: 1 card", 4: "tests.md: app = 67", 5: "tests.md: app = 70", 6: "worklog.md: 1 line", 7: "plan.md: set open, rows, size"}
	for id, text := range want {
		if got := call(t, byID(t, rs, id)); got.IsError || got.Content[0].Text != text {
			t.Errorf("request %d = %+v, want %q", id, got, text)
		}
	}
	if got := call(t, byID(t, rs, 8)); !got.IsError || !strings.Contains(got.Content[0].Text, "write tests") {
		t.Errorf("a failed edit should say what there is: %+v", got)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "plan.md")); !strings.Contains(string(b), "- [x] write tests\n") {
		t.Errorf("plan.md = %q", b)
	}
	views, err := store.Open(dir).Views()
	if v := views["plan.md"]; err != nil || v.Open == nil || *v.Open || v.Size != "card" || v.Rows != 8 {
		t.Errorf("the arrangement goes to sticky.json: %+v, %v", v, err)
	}
}

func TestRemoveMoveAndRestore(t *testing.T) {
	rs, dir := serve(t,
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"move_note","arguments":{"name":"a","to":"docs/"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"remove_note","arguments":{"name":"docs/a"}}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"restore_note","arguments":{"name":"docs/a"}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"remove_note","arguments":{"name":"missing"}}}`,
	)
	if got := call(t, byID(t, rs, 1)); got.IsError || got.Content[0].Text != "moved a.md to docs/a.md" {
		t.Errorf("move_note = %+v", got)
	}
	if got := call(t, byID(t, rs, 2)); got.IsError || !strings.Contains(got.Content[0].Text, ".trash/docs/a.md") {
		t.Errorf("remove_note = %+v", got)
	}
	if got := call(t, byID(t, rs, 3)); got.IsError || got.Content[0].Text != "restored docs/a.md" {
		t.Errorf("restore_note = %+v", got)
	}
	if got := call(t, byID(t, rs, 4)); !got.IsError {
		t.Errorf("removing a missing note should fail: %+v", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "docs", "a.md")); err != nil {
		t.Errorf("the note should be back in its book: %v", err)
	}
}
