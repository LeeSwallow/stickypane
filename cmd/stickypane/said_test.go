package main

import (
	"os"
	"path/filepath"
	"testing"
)

// saidLines are the commands that change the board, each with the line it
// prints, in order on one project. The words are the command line's: an
// agent reads them, so a change to one is a change of behaviour.
var saidLines = []struct {
	args []string
	out  string
}{
	{[]string{"todo", "plan", "add", "write", "tests"}, "plan.md: 0/1\n"},
	{[]string{"todo", "plan", "add", "ship it"}, "plan.md: 0/2\n"},
	{[]string{"todo", "plan", "check", "tests"}, "plan.md: 1/2\n"},
	{[]string{"todo", "plan", "uncheck", "#1"}, "plan.md: 0/2\n"},
	{[]string{"card", "work", "add", "login API"}, "work.md: 1 card\n"},
	{[]string{"card", "work", "add", "schema", "--to", "Done"}, "work.md: 2 cards\n"},
	{[]string{"card", "work", "move", "login", "--to", "Done"}, "work.md: 2 cards\n"},
	{[]string{"chart", "tokens", "set", "input", "1,200"}, "tokens.md: input = 1,200\n"},
	{[]string{"chart", "tokens", "add", "input", "800"}, "tokens.md: input = 2,000\n"},
	{[]string{"chart", "tokens", "add", "output", "-3"}, "tokens.md: output = -3\n"},
	{[]string{"log", "worklog", "tests passed"}, "worklog.md: 1 line\n"},
	{[]string{"log", "worklog", "review done"}, "worklog.md: 2 lines\n"},
	{[]string{"log", "memo", "a plain note says no summary"}, "memo.md\n"},
	{[]string{"say", "chat", "41/41 pass"}, "chat.md: 1 message\n"},
	{[]string{"say", "chat", "--as", "me", "next?"}, "chat.md: 2 messages\n"},
	{[]string{"set", "plan", "size=card", "title=The plan"}, "plan.md: set size, title\n"},
	{[]string{"set", "plan", "title="}, "plan.md: removed title\n"},
	{[]string{"set", "plan", "rows=8", "title=", "pin=true"}, "plan.md: set rows, pin; removed title\n"},
	{[]string{"hide", "plan"}, "hidden plan.md\n"},
	{[]string{"show", "plan"}, "showing plan.md\n"},
	{[]string{"show", "README.md"}, "showing README.md (linked to ../README.md)\n"},
	{[]string{"show", "README.md"}, "showing README.md\n"},
	{[]string{"link", "docs/guide.md"}, "linked guide.md to ../docs/guide.md\n"},
	{[]string{"link", "docs", "handbook"}, "linked handbook to ../docs\n"},
	{[]string{"mv", "memo", "notes/"}, "moved memo.md to notes/memo.md\n"},
	{[]string{"mv", "notes/memo", "memo"}, "moved notes/memo.md to memo.md\n"},
	{[]string{"rm", "memo"}, "moved memo.md to .trash/memo.md; `stickypane restore memo` brings it back\n"},
	{[]string{"restore", "memo"}, "restored memo.md\n"},
	{[]string{"archive", "memo"}, "moved memo.md to archive/memo.md\n"},
}

// TestEveryChangeSaysItsLine pins what each command that changes the board
// prints. The command line speaks English in every language: the lines are
// read by agents, and the summaries in them are the board's English words.
func TestEveryChangeSaysItsLine(t *testing.T) {
	for _, lang := range []string{"en", "ko"} {
		t.Run(lang, func(t *testing.T) {
			t.Setenv("STICKYPANE_LANG", lang)
			root := project(t)
			if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# readme\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "docs", "guide.md"), []byte("guide\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if code, out, errOut := execIn(t, "first\n", "write", "memo"); code != 0 || out != "wrote memo.md\n" {
				t.Fatalf("write: code = %d, out = %q, stderr = %q", code, out, errOut)
			}
			for _, s := range saidLines {
				if code, out, errOut := exec(t, s.args...); code != 0 || out != s.out {
					t.Errorf("%v: code = %d, out = %q, want %q; stderr = %q", s.args, code, out, s.out, errOut)
				}
			}
		})
	}
}
