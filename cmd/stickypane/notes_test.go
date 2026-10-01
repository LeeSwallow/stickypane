package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// project makes a temporary project with a notes folder and moves into it.
func project(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Chdir(root)
	if code, _, errOut := exec(t, "init", "--no-agent-docs"); code != 0 {
		t.Fatalf("init failed: %s", errOut)
	}
	return root
}

func TestWriteListAndShow(t *testing.T) {
	root := project(t)
	code, out, errOut := execIn(t, "- [ ] build\n- [ ] ship\n", "write", "release", "--type", "checklist", "--title", "Release", "--open", "--size", "half")
	if code != 0 || !strings.Contains(out, "release.md") {
		t.Fatalf("write: code = %d, out = %q, stderr = %q", code, out, errOut)
	}
	want := "---\ntype: checklist\ntitle: Release\nopen: true\nsize: half\n---\n- [ ] build\n- [ ] ship\n"
	if b, _ := os.ReadFile(filepath.Join(root, ".stickypane", "release.md")); string(b) != want {
		t.Errorf("file = %q", b)
	}

	code, out, _ = exec(t, "show", "release")
	if code != 0 || out != want {
		t.Errorf("show: code = %d, out = %q", code, out)
	}

	code, out, _ = exec(t, "list", "--json")
	var notes []map[string]any
	if err := json.Unmarshal([]byte(out), &notes); code != 0 || err != nil || len(notes) != 2 {
		t.Fatalf("list --json: code = %d, err = %v, out = %q", code, err, out)
	}
	if notes[0]["name"] != "release.md" || notes[0]["type"] != "checklist" || notes[0]["open"] != true || notes[0]["size"] != "half" {
		t.Errorf("first note = %v", notes[0])
	}

	code, out, _ = exec(t, "list")
	if code != 0 || !strings.Contains(out, "release.md") || !strings.Contains(out, "welcome.md") || !strings.Contains(out, "checklist") {
		t.Errorf("list: code = %d, out = %q", code, out)
	}
}

func TestNoteCommandsReportProblems(t *testing.T) {
	project(t)
	if code, _, errOut := exec(t, "show", "missing"); code != 1 || errOut == "" {
		t.Errorf("show missing: code = %d, stderr = %q", code, errOut)
	}
	if code, _, errOut := execIn(t, "x\n", "write", "../escape"); code != 1 || errOut == "" {
		t.Errorf("write ../escape: code = %d, stderr = %q", code, errOut)
	}
	for _, args := range [][]string{{"show"}, {"write"}, {"write", "a", "--size"}, {"list", "extra"}} {
		if code, _, errOut := exec(t, args...); code != 2 || errOut == "" {
			t.Errorf("%v: code = %d, stderr = %q; want 2 and a message", args, code, errOut)
		}
	}
}

func TestNoteCommandsNeedANotesFolder(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, args := range [][]string{{"list"}, {"show", "a"}, {"write", "a"}, {"mcp"}} {
		if code, _, errOut := exec(t, args...); code != 1 || !strings.Contains(errOut, "stickypane init") {
			t.Errorf("%v: code = %d, stderr = %q", args, code, errOut)
		}
	}
}

func TestMCPServesOverStandardIO(t *testing.T) {
	project(t)
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"write_note","arguments":{"name":"hello","content":"from mcp\n","open":true}}}` + "\n"
	code, out, errOut := execIn(t, request, "mcp")
	if code != 0 || !strings.Contains(out, "hello.md") {
		t.Fatalf("mcp: code = %d, out = %q, stderr = %q", code, out, errOut)
	}
	if _, show, _ := exec(t, "show", "hello"); show != "---\nopen: true\n---\nfrom mcp\n" {
		t.Errorf("note written through MCP = %q", show)
	}
}
