package main

import (
	"encoding/json"
	"os"
	osexec "os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/adapter/git"
)

// cliOnly are the commands with no <tool> + action of the same words over
// MCP, and why. A new command must be paired or listed here.
var cliOnly = map[string]string{
	"init": "sets up a project", "setup": "installs the plugin", "config": "a setting of the user's",
	"keys": "the board's keys", "env": "the machine", "version": "the program", "help": "the command line's own help",
	"mcp": "the server itself", "bridge": "the desktop app",
	"kinds":   "the guide tool with a shape as topic",
	"guide":   "the guide tool",
	"index":   "note action list",
	"answers": "read_answers", "wait": "read_answers with wait_seconds", "watch": "wait_event",
	"api": "the api tool",
}

// renamed are the actions whose CLI verb is another word.
var renamed = map[string]string{"note read": "note cat"}

// Every <noun> <verb> of the command table is a <tool> + action over MCP,
// and every <tool> + action a command, so one surface cannot gain a
// command the other lacks.
func TestEveryCommandIsAToolAction(t *testing.T) {
	t.Chdir(t.TempDir())
	code, out, errOut := execIn(t, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`+"\n", "mcp")
	if code != 0 {
		t.Fatal(errOut)
	}
	var list struct {
		Result struct {
			Tools []struct {
				Name        string
				InputSchema struct {
					Properties struct {
						Action struct{ Enum []string }
					}
				}
			}
		}
	}
	if err := json.Unmarshal([]byte(out), &list); err != nil {
		t.Fatal(err)
	}
	mcp := map[string]bool{}
	for _, tool := range list.Result.Tools {
		for _, a := range tool.InputSchema.Properties.Action.Enum {
			pair := tool.Name + " " + a
			if r, ok := renamed[pair]; ok {
				pair = r
			}
			mcp[pair] = true
		}
	}
	verbs := regexp.MustCompile(`^stickypane \S+ ([a-z]+(\|[a-z]+)*)( |$)`)
	cli := map[string]bool{}
	for _, c := range commands {
		name := c.names[0]
		if strings.Contains(name, " ") {
			cli[name] = true
			continue
		}
		m := verbs.FindStringSubmatch(c.help.syntax)
		if m == nil {
			if cliOnly[name] == "" {
				t.Errorf("%s has no verb and no tool: pair it, or say why in cliOnly", name)
			}
			continue
		}
		for _, v := range strings.Split(m[1], "|") {
			cli[name+" "+v] = true
		}
	}
	for pair := range cli {
		if !mcp[pair] {
			t.Errorf("stickypane %s has no MCP tool and action", pair)
		}
	}
	for pair := range mcp {
		if !cli[pair] {
			t.Errorf("the MCP %s has no command", pair)
		}
	}
}

// page, worktree and diff over MCP answer what the command line prints,
// the lists as its --json.
func TestPageWorktreeAndDiffOverMCPAreTheCommandLine(t *testing.T) {
	if _, err := osexec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	for _, v := range git.Local {
		t.Setenv(v, "")
		os.Unsetenv(v)
	}
	root, _ := filepath.EvalSymlinks(t.TempDir())
	main := filepath.Join(root, "main")
	run := func(dir string, args ...string) {
		t.Helper()
		cmd := osexec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	os.MkdirAll(main, 0o755)
	run(main, "init", "-q", "-b", "main")
	os.WriteFile(filepath.Join(main, "a.txt"), []byte("a\n"), 0o644)
	run(main, "add", ".")
	run(main, "commit", "-q", "-m", "first")
	run(main, "worktree", "add", "-q", "-b", "feat", filepath.Join(root, "feat"))
	os.WriteFile(filepath.Join(root, "feat", "a.txt"), []byte("a\nb\n"), 0o644)
	t.Chdir(main)
	if code, _, errOut := exec(t, "init", "--no-agent-docs"); code != 0 {
		t.Fatal(errOut)
	}
	os.MkdirAll(filepath.Join(main, ".sticky", "deploy", "docs"), 0o755)
	os.WriteFile(filepath.Join(main, ".sticky", "deploy", "docs", "01-intro.md"), []byte("x\n"), 0o644)

	texts, failed := mcpTools(t,
		"page", `{"action":"add","name":"deploy/docs","text":"Release notes"}`,
		"page", `{"action":"list","name":"deploy/docs"}`,
		"worktree", `{"action":"list"}`,
		"worktree", `{"action":"show","name":"feat"}`,
		"diff", `{"action":"list"}`,
		"diff", `{"action":"list","name":"feat"}`,
		"diff", `{"action":"show","name":"feat","file":"a.txt"}`,
		"diff", `{"action":"show","name":"feat","file":"nope.txt"}`,
	)
	if slices.Contains(failed[:7], true) || !failed[7] {
		t.Fatalf("failed = %v: %q", failed, texts)
	}
	if texts[0] != "added deploy/docs/release-notes.md" {
		t.Errorf("page add = %q", texts[0])
	}
	cli := [][]string{
		{"page", "list", "deploy/docs", "--json"},
		{"worktree", "list", "--json"},
		{"worktree", "show", "feat", "--json"},
		{"diff", "list", "--json"},
		{"diff", "list", "feat", "--json"},
		{"diff", "show", "feat", "a.txt"},
	}
	for i, args := range cli {
		if _, out, _ := exec(t, args...); out != texts[i+1] {
			t.Errorf("stickypane %v prints %q, MCP answers %q", args, out, texts[i+1])
		}
	}
	if !strings.Contains(texts[6], "+b") {
		t.Errorf("diff show = %q", texts[6])
	}
}
