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
	if b, _ := os.ReadFile(filepath.Join(root, ".sticky", "release.md")); string(b) != want {
		t.Errorf("file = %q", b)
	}

	code, out, _ = exec(t, "cat", "release")
	if code != 0 || out != want {
		t.Errorf("cat: code = %d, out = %q", code, out)
	}
	if code, out, _ := exec(t, "hide", "release"); code != 0 || out != "hidden release.md\n" {
		t.Errorf("hide: code = %d, out = %q", code, out)
	}
	if code, out, _ := exec(t, "show", "release"); code != 0 || out != "showing release.md\n" {
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
	if code, _, errOut := exec(t, "cat", "missing"); code != 1 || errOut == "" {
		t.Errorf("cat missing: code = %d, stderr = %q", code, errOut)
	}
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
	if _, show, _ := exec(t, "cat", "hello"); show != "---\nopen: true\n---\nfrom mcp\n" {
		t.Errorf("note written through MCP = %q", show)
	}
}

const formNote = "---\ntype: form\n---\n## Where?\n- (x) staging\n- ( ) production\n\n## Note\n> after lunch\n\n[ Go ]\n"

func TestAnswersAndWait(t *testing.T) {
	root := project(t)
	file := filepath.Join(root, ".sticky", "deploy.md")
	if err := os.WriteFile(file, []byte(formNote), 0o644); err != nil {
		t.Fatal(err)
	}

	code, out, _ := exec(t, "answers", "deploy")
	if code != 0 || out != "submitted: no\nWhere?: staging\nNote: after lunch\n" {
		t.Errorf("answers: code = %d, out = %q", code, out)
	}

	code, out, errOut := exec(t, "wait", "deploy", "--timeout", "30ms")
	if code != 3 || out != "" || !strings.Contains(errOut, "deploy.md") {
		t.Errorf("wait should give up with code 3: code = %d, out = %q, stderr = %q", code, out, errOut)
	}

	pressed := strings.Replace(formNote, "type: form\n", "type: form\nsubmitted: Go\nsubmitted_at: 2026-10-02T14:03:05Z\n", 1)
	if err := os.WriteFile(file, []byte(pressed), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ = exec(t, "wait", "deploy", "--timeout", "5s")
	if code != 0 || out != "submitted: Go\nat: 2026-10-02T14:03:05Z\nWhere?: staging\nNote: after lunch\n" {
		t.Errorf("wait: code = %d, out = %q", code, out)
	}

	code, out, _ = exec(t, "wait", "deploy", "--json")
	var got struct {
		Submitted bool
		Button    string
		Answers   []struct {
			Question string
			Values   []string
		}
	}
	if err := json.Unmarshal([]byte(out), &got); code != 0 || err != nil || !got.Submitted || got.Button != "Go" || len(got.Answers) != 2 || got.Answers[1].Values[0] != "after lunch" {
		t.Errorf("wait --json: code = %d, err = %v, out = %q", code, err, out)
	}

	if code, _, errOut := exec(t, "answers", "welcome"); code != 1 || !strings.Contains(errOut, "form") {
		t.Errorf("answers of a plain note: code = %d, stderr = %q", code, errOut)
	}
	if code, _, _ := exec(t, "wait"); code != 2 {
		t.Errorf("wait without a name: code = %d", code)
	}
}

func TestWaitRejectsANegativeTimeout(t *testing.T) {
	root := project(t)
	if err := os.WriteFile(filepath.Join(root, ".sticky", "deploy.md"), []byte(formNote), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, errOut := exec(t, "wait", "deploy", "--timeout", "-1s"); code != 2 || errOut == "" {
		t.Errorf("code = %d, stderr = %q", code, errOut)
	}
}

func TestSmallEditsFromTheCommandLine(t *testing.T) {
	root := project(t)
	steps := []struct {
		args []string
		out  string
	}{
		{[]string{"todo", "plan", "add", "write", "tests"}, "plan.md: 0/1\n"},
		{[]string{"todo", "plan", "check", "tests"}, "plan.md: 1/1\n"},
		{[]string{"card", "work", "add", "login API", "--to", "Doing"}, "work.md: 1 card\n"},
		{[]string{"card", "work", "add", "--to", "Done", "schema"}, "work.md: 2 cards\n"},
		{[]string{"card", "work", "move", "login", "--to", "Done"}, "work.md: 2 cards\n"},
		{[]string{"chart", "tokens", "set", "input", "tokens", "1,200"}, "tokens.md: input tokens = 1,200\n"},
		{[]string{"chart", "tokens", "add", "input tokens", "800"}, "tokens.md: input tokens = 2,000\n"},
		{[]string{"log", "worklog", "tests", "passed"}, "worklog.md: 1 line\n"},
		{[]string{"set", "plan", "size=card", "title=The plan"}, "plan.md: set size, title\n"},
	}
	for _, s := range steps {
		if code, out, errOut := exec(t, s.args...); code != 0 || out != s.out {
			t.Errorf("%v: code = %d, out = %q, stderr = %q", s.args, code, out, errOut)
		}
	}
	if b, _ := os.ReadFile(filepath.Join(root, ".sticky", "work.md")); !strings.HasSuffix(string(b), "## Doing\n\n## Done\n- schema\n- login API\n") {
		t.Errorf("work.md = %q", b)
	}
	if b, _ := os.ReadFile(filepath.Join(root, ".sticky", "plan.md")); !strings.Contains(string(b), "title: The plan\n") || !strings.Contains(string(b), "- [x] write tests\n") {
		t.Errorf("plan.md = %q", b)
	}
	for _, bad := range [][]string{{"todo"}, {"todo", "plan"}, {"todo", "plan", "add"}, {"card", "work", "move", "login"}, {"chart", "tokens", "set", "input"}, {"log", "worklog"}, {"set", "plan"}} {
		if code, _, errOut := exec(t, bad...); code != 2 || errOut == "" {
			t.Errorf("%v: code = %d, stderr = %q, want a usage error", bad, code, errOut)
		}
	}
	if code, _, errOut := exec(t, "todo", "plan", "check", "nothing like this"); code != 1 || !strings.Contains(errOut, "write tests") {
		t.Errorf("an item that is not there: code = %d, stderr = %q", code, errOut)
	}
}

func TestLogCanStampTheTime(t *testing.T) {
	root := project(t)
	if code, _, errOut := exec(t, "log", "worklog", "--time", "deployed"); code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, errOut)
	}
	b, _ := os.ReadFile(filepath.Join(root, ".sticky", "worklog.md"))
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	last := lines[len(lines)-1]
	if len(last) < 6 || last[2] != ':' || !strings.HasSuffix(last, " deployed") {
		t.Errorf("--time should put the time before the entry: %q", last)
	}
}

func TestRemoveRestoreArchiveAndMoveFromTheCommandLine(t *testing.T) {
	root := project(t)
	dir := filepath.Join(root, ".sticky")
	if err := os.WriteFile(filepath.Join(dir, "plan.md"), []byte("the plan\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	has := func(name string) bool {
		_, err := os.Stat(filepath.Join(dir, filepath.FromSlash(name)))
		return err == nil
	}
	if code, out, errOut := exec(t, "rm", "plan"); code != 0 || !strings.Contains(out, ".trash/plan.md") || has("plan.md") {
		t.Fatalf("rm: code = %d, out = %q, stderr = %q", code, out, errOut)
	}
	if code, out, _ := exec(t, "restore", "plan"); code != 0 || out != "restored plan.md\n" || !has("plan.md") {
		t.Fatalf("restore: code = %d, out = %q", code, out)
	}
	if code, out, _ := exec(t, "mv", "plan", "docs/"); code != 0 || out != "moved plan.md to docs/plan.md\n" || !has("docs/plan.md") {
		t.Fatalf("mv into a folder: code = %d, out = %q", code, out)
	}
	if code, out, _ := exec(t, "mv", "docs/plan", "roadmap"); code != 0 || out != "moved docs/plan.md to roadmap.md\n" || !has("roadmap.md") {
		t.Fatalf("mv to a new name: code = %d, out = %q", code, out)
	}
	if code, out, _ := exec(t, "archive", "roadmap"); code != 0 || out != "moved roadmap.md to archive/roadmap.md\n" || !has("archive/roadmap.md") {
		t.Fatalf("archive: code = %d, out = %q", code, out)
	}
	for _, bad := range [][]string{{"rm"}, {"mv", "x"}, {"restore"}, {"archive"}, {"rm", "a", "b"}} {
		if code, _, errOut := exec(t, bad...); code != 2 || errOut == "" {
			t.Errorf("%v: code = %d, want a usage error", bad, code)
		}
	}
	if code, _, errOut := exec(t, "rm", "nothing"); code != 1 || !strings.Contains(errOut, "nothing.md") {
		t.Errorf("rm of a missing note: code = %d, stderr = %q", code, errOut)
	}
}

func TestThemeCommand(t *testing.T) {
	root := project(t)
	code, out, _ := exec(t, "theme")
	if code != 0 || !strings.Contains(out, "stickypane-dark") || !strings.Contains(out, "auto") {
		t.Fatalf("theme should list the themes: code = %d, out = %q", code, out)
	}
	if code, out, _ := exec(t, "theme", "Nord"); code != 0 || out != "theme: nord\n" {
		t.Fatalf("theme nord: code = %d, out = %q", code, out)
	}
	if b, _ := os.ReadFile(filepath.Join(root, ".sticky", "sticky.json")); !strings.Contains(string(b), `"theme": "nord"`) {
		t.Errorf("sticky.json = %s", b)
	}
	if code, out, _ := exec(t, "theme"); code != 0 || !strings.Contains(out, "* nord") {
		t.Errorf("the chosen theme should be marked: %q", out)
	}
	if code, _, errOut := exec(t, "theme", "no-such-theme"); code != 1 || !strings.Contains(errOut, "nord") {
		t.Errorf("an unknown theme should be refused and the themes listed: code = %d, stderr = %q", code, errOut)
	}
	if code, out, _ := exec(t, "theme", "auto"); code != 0 || out != "theme: auto\n" {
		t.Errorf("auto: code = %d, out = %q", code, out)
	}
}
