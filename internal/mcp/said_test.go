package mcp_test

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// TestEveryChangeSaysItsLine pins the text each tool that changes the
// board answers, in order on one board. An agent reads these words, so a
// change to one is a change of behaviour.
func TestEveryChangeSaysItsLine(t *testing.T) {
	readme := filepath.Join(t.TempDir(), "README.md")
	if err := os.WriteFile(readme, []byte("# readme\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	quoted := strconv.Quote(readme)
	steps := []struct {
		tool, args, text string
	}{
		{"write_note", `{"name":"memo","content":"first\n"}`, "wrote memo.md"},
		{"todo", `{"name":"plan","action":"add","item":"write tests"}`, "plan.md: 0/1"},
		{"todo", `{"name":"plan","action":"add","item":"ship it"}`, "plan.md: 0/2"},
		{"todo", `{"name":"plan","action":"check","item":"tests"}`, "plan.md: 1/2"},
		{"todo", `{"name":"plan","action":"uncheck","item":"#1"}`, "plan.md: 0/2"},
		{"card", `{"name":"work","action":"add","card":"login API"}`, "work.md: 1 card"},
		{"card", `{"name":"work","action":"add","card":"schema","to":"Done"}`, "work.md: 2 cards"},
		{"card", `{"name":"work","action":"move","card":"login","to":"Done"}`, "work.md: 2 cards"},
		{"chart", `{"name":"tests","action":"set","label":"app","value":67}`, "tests.md: app = 67"},
		{"chart", `{"name":"tests","action":"add","label":"app","value":"3"}`, "tests.md: app = 70"},
		{"chart", `{"name":"tests","action":"add","label":"store","value":-2}`, "tests.md: store = -2"},
		{"log", `{"name":"worklog","line":"tests passed"}`, "worklog.md: 1 line"},
		{"log", `{"name":"worklog","line":"review done"}`, "worklog.md: 2 lines"},
		{"log", `{"name":"memo","line":"a plain note says no summary"}`, "memo.md"},
		{"say", `{"name":"chat","text":"41/41 pass"}`, "chat.md: 1 message"},
		{"say", `{"name":"chat","as":"me","text":"next?"}`, "chat.md: 2 messages"},
		{"set_keys", `{"name":"plan","keys":{"open":false,"size":"card","rows":8}}`, "plan.md: set open, rows, size"},
		{"set_keys", `{"name":"plan","keys":{"title":"The plan"}}`, "plan.md: set title"},
		{"set_keys", `{"name":"plan","keys":{"title":"","pin":true}}`, "plan.md: set pin; removed title"},
		{"arrange_note", `{"name":"plan","action":"hide"}`, "hidden plan.md"},
		{"arrange_note", `{"name":"plan","action":"show"}`, "showing plan.md"},
		{"arrange_note", `{"name":` + quoted + `,"action":"show"}`, "showing README.md (linked to ../README.md)"},
		{"arrange_note", `{"name":` + quoted + `,"action":"show"}`, "showing README.md"},
		{"arrange_note", `{"name":"memo","action":"move","to":"notes/"}`, "moved memo.md to notes/memo.md"},
		{"arrange_note", `{"name":"notes/memo","action":"move","to":"memo"}`, "moved notes/memo.md to memo.md"},
		{"arrange_note", `{"name":"memo","action":"remove"}`, "moved memo.md to .trash/memo.md; `stickypane restore memo` brings it back"},
		{"arrange_note", `{"name":"memo","action":"restore"}`, "restored memo.md"},
	}
	var requests []string
	for i, s := range steps {
		requests = append(requests, `{"jsonrpc":"2.0","id":`+strconv.Itoa(i+1)+`,"method":"tools/call","params":{"name":"`+s.tool+`","arguments":`+s.args+`}}`)
	}
	rs, _ := serveIn(t, filepath.Dir(readme), requests...)
	for i, s := range steps {
		if got := call(t, byID(t, rs, i+1)); got.IsError || got.Content[0].Text != s.text {
			t.Errorf("%s %s = %+v, want %q", s.tool, s.args, got, s.text)
		}
	}
}
