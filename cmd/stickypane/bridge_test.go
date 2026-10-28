package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// The desktop app starts `stickypane bridge` and says hello first; the
// bridge ends when its input does, and it is a session in the registry.
func TestBridgeCommandSaysHello(t *testing.T) {
	root := project(t)
	home := appHome(t)
	code, out, errOut := execIn(t, `{"id":1,"method":"hello"}`+"\n", "bridge")
	if code != 0 {
		t.Fatalf("bridge failed: %s", errOut)
	}
	var m struct {
		ID     int `json:"id"`
		Result struct {
			Protocol int    `json:"protocol"`
			Version  string `json:"version"`
			Project  string `json:"project"`
		} `json:"result"`
	}
	line, _, _ := strings.Cut(out, "\n")
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		t.Fatalf("%v in %q", err, out)
	}
	if m.ID != 1 || m.Result.Protocol != 1 || m.Result.Version == "" || !same(m.Result.Project, root) {
		t.Errorf("hello = %+v", m)
	}
	got := recorded(t, home)
	if len(got) != 2 || got[0].Type != "session.start" || got[0].Kind != "bridge" || got[1].Type != "session.end" {
		t.Errorf("want session.start and session.end of bridge: %+v", got)
	}
	if left, _ := filepath.Glob(filepath.Join(home, "run", "*.json")); len(left) != 0 {
		t.Errorf("the run file should be gone: %v", left)
	}
}
