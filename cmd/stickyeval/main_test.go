package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

const evalDir = "../../plugin/evals"

// fixture is a recorded run in testdata/<case>: events.jsonl in the common
// format, the board after it (sticky/) and the files it had before (before).
func fixture(t *testing.T, name string) Run {
	t.Helper()
	dir := filepath.Join("testdata", name)
	f, err := os.Open(filepath.Join(dir, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	events, err := readEvents(f)
	if err != nil {
		t.Fatal(err)
	}
	return Run{Case: name, Events: events, Sticky: filepath.Join(dir, "sticky"), Before: readBefore(filepath.Join(dir, "before"))}
}

func TestListPrintsEveryCaseAndRunsNothing(t *testing.T) {
	var out bytes.Buffer
	if code := run([]string{"-evals", evalDir}, &out, nil); code != 0 {
		t.Fatalf("exit %d: %s", code, out.String())
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 12 { // a heading and 11 cases
		t.Fatalf("want 11 cases:\n%s", out.String())
	}
	for i, l := range lines[1:] {
		if !strings.HasPrefix(l, []string{"01-", "02-", "03-", "04-", "05-", "06-", "07-", "08-", "09-", "10-", "11-"}[i]) {
			t.Errorf("line %d: %q", i+1, l)
		}
	}
}

func TestPlanNamesWhatARunWouldCostAndRunsNothing(t *testing.T) {
	var out bytes.Buffer
	if code := run([]string{"-evals", evalDir, "-plan", "-codex-model", "gpt-x"}, &out, nil); code != 0 {
		t.Fatalf("exit %d: %s", code, out.String())
	}
	for _, want := range []string{
		"11 cases × 2 arms (with, without) × k=3 × 2 agents = 132 runs",
		"claude plugin eval plugin --runs 3 --ablation with-without",
		"codex exec --json --ephemeral",
		"--disable plugins",
		"model: gpt-x",
		"nothing was run",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("plan lacks %q:\n%s", want, out.String())
		}
	}
}

// Every case has a grader, and the hand-made run of an agent doing it right
// passes.
func TestEveryCaseHasAGraderAndAPassingRun(t *testing.T) {
	cases, err := loadCases(evalDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 11 {
		t.Fatalf("%d cases", len(cases))
	}
	for _, c := range cases {
		if checks[c.Name] == nil {
			t.Errorf("%s has no grader", c.Name)
			continue
		}
		if r := grade(fixture(t, c.Name)); !r.Pass {
			t.Errorf("%s: the right run fails: %s", c.Name, r.Reason)
		}
	}
}

func at(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

// The wrong move of each case fails, with its reason.
func TestEachCaseFailsTheWrongMove(t *testing.T) {
	cmd := func(text string, exit int) Event {
		return Event{At: at("2026-11-25T10:00:20+09:00"), Kind: "command", Text: text, Exit: exit}
	}
	tests := []struct {
		name, reason string
		change       func(r *Run)
	}{
		{"01-worktree", "outside worktrees/", func(r *Run) { r.Before = map[string]bool{}; r.Sticky = "testdata/02-steps/sticky" }},
		{"02-steps", "ticked before", func(r *Run) { r.Events[5], r.Events[4] = r.Events[4], r.Events[5] }},
		{"02-steps", "no tracking-the-work", func(r *Run) { r.Events = r.Events[1:] }},
		{"03-decision", "asked in the chat", func(r *Run) {
			r.Events = append(r.Events, Event{Kind: "message", Text: "Which license do you want, MIT or Apache?"})
		}},
		{"03-decision", "without --timeout", func(r *Run) { r.Events[2].Text = "stickypane wait license" }},
		{"04-chat", "wrote chat.md", func(r *Run) { r.Events = append(r.Events, Event{Kind: "write", Text: ".sticky/chat.md"}) }},
		{"04-chat", "no watch", func(r *Run) { r.Events = r.Events[:3] }},
		{"05-from-chart", "wrote numbers", func(r *Run) { r.Events = append(r.Events, cmd("stickypane chart set progress api 90", 0)) }},
		{"05-from-chart", "no from: chart", func(r *Run) { r.Sticky = "testdata/07-shapes/sticky" }},
		{"06-no-board", "ran stickypane", func(r *Run) { r.Events = append(r.Events, cmd("stickypane log add calc 391", 0)) }},
		{"07-shapes", "not a command", func(r *Run) { r.Events = append(r.Events, cmd("stickypane graph add tokens 1", 1)) }},
		{"07-shapes", "no script", func(r *Run) { r.Before = map[string]bool{"clear-cache.sh": true} }},
		{"08-approval", "before the form", func(r *Run) { r.Events[3].At = at("2026-11-25T10:00:10+09:00") }},
		{"08-approval", "no risky", func(r *Run) { r.Events = r.Events[:3] }},
		{"09-timeout", "risky action", func(r *Run) { r.Events = append(r.Events, cmd("rm -rf data", 0)) }},
		{"09-timeout", "not logged", func(r *Run) { r.Events = r.Events[:3]; r.Sticky = "testdata/08-approval/sticky" }},
		{"10-old-answer", "old answer", func(r *Run) {
			r.Events = []Event{cmd("stickypane wait delete-data --timeout 1m", 0), cmd("rm -rf data/", 0)}
		}},
		{"11-language", "not in Korean", func(r *Run) { r.Sticky = "testdata/02-steps/sticky"; r.Before = map[string]bool{} }},
	}
	for _, tt := range tests {
		r := fixture(t, tt.name)
		tt.change(&r)
		got := grade(r)
		if got.Pass || !strings.Contains(got.Reason, tt.reason) {
			t.Errorf("%s: want a FAIL saying %q, got %+v", tt.name, tt.reason, got)
		}
	}
}

func TestCodexEventsBecomeTheCommonFormat(t *testing.T) {
	jsonl := `{"type":"thread.started","thread_id":"t1"}
{"type":"turn.started"}
{"type":"item.started","item":{"id":"i1","type":"command_execution","command":"bash -lc 'cat ~/.codex/plugins/board/skills/tracking-the-work/SKILL.md'","status":"in_progress"}}
{"type":"item.completed","item":{"id":"i1","type":"command_execution","command":"bash -lc 'cat ~/.codex/plugins/board/skills/tracking-the-work/SKILL.md'","aggregated_output":"...","exit_code":0,"status":"completed"}}
{"type":"item.started","item":{"id":"i2","type":"command_execution","command":"bash -lc 'stickypane wait go --timeout 1m'","status":"in_progress"}}
{"type":"item.completed","item":{"id":"i2","type":"command_execution","command":"bash -lc 'stickypane wait go --timeout 1m'","exit_code":3,"status":"completed"}}
{"type":"item.completed","item":{"id":"i3","type":"file_change","changes":[{"path":"/r/.sticky/chat.md","kind":"update"}],"status":"completed"}}
{"type":"item.completed","item":{"id":"i4","type":"agent_message","text":"done"}}
{"type":"turn.completed","usage":{"input_tokens":1200,"cached_input_tokens":200,"output_tokens":300}}
`
	ticks := 0
	now := func() time.Time {
		ticks++
		return at("2026-11-25T10:00:00+09:00").Add(time.Duration(ticks) * time.Second)
	}
	events, use, err := codexEvents(strings.NewReader(jsonl), now)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range events {
		got = append(got, e.Kind+":"+e.Text)
	}
	want := []string{
		"command:bash -lc 'cat ~/.codex/plugins/board/skills/tracking-the-work/SKILL.md'",
		"skill:tracking-the-work",
		"command:bash -lc 'stickypane wait go --timeout 1m'",
		"write:/r/.sticky/chat.md",
		"message:done",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("got %q", got)
	}
	// A command is placed when it started, and keeps its exit code.
	if events[2].Exit != 3 || !events[2].At.Before(events[3].At) || events[2].At != events[0].At.Add(2*time.Second) {
		t.Errorf("wait: %+v", events[2])
	}
	if use != (Usage{Input: 1200, Cached: 200, Output: 300}) {
		t.Errorf("usage %+v", use)
	}
}

// The board hears of each run with log add and chart add, never chart set:
// set would wipe the earlier runs.
func TestReportAddsToTheBoardAndNeverSets(t *testing.T) {
	var calls []string
	x := func(dir string, args ...string) error {
		calls = append(calls, strings.Join(args, " "))
		return nil
	}
	results := []Result{
		{Case: "02-steps", Agent: "codex", Arm: "with", Pass: true, Cmds: 9, Use: Usage{Input: 1200, Output: 300}},
		{Case: "08-approval", Agent: "codex", Arm: "without", Reason: "risky action before the form was submitted", Cmds: 2},
	}
	if err := report("/board", results, x); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"log add eval --time case 02-steps: PASS codex with, 9 cmds",
		"chart add eval-pass-codex-with 02-steps 1",
		"chart add eval-tokens-codex-with input 1200",
		"chart add eval-tokens-codex-with output 300",
		"log add eval --time case 08-approval: FAIL codex without, 2 cmds: risky action before the form was submitted",
		"chart add eval-pass-codex-without 08-approval 0",
	}
	if !slices.Equal(calls, want) {
		t.Fatalf("got\n%s", strings.Join(calls, "\n"))
	}
}

func TestTableShowsPassHatK(t *testing.T) {
	var out bytes.Buffer
	table(&out, []Result{
		{Case: "02-steps", Agent: "codex", Arm: "with", Pass: true},
		{Case: "02-steps", Agent: "codex", Arm: "with", Pass: true},
		{Case: "02-steps", Agent: "codex", Arm: "without", Pass: true},
		{Case: "02-steps", Agent: "codex", Arm: "without", Reason: "x"},
	})
	for _, want := range []string{"02-steps  codex  with     2/2  pass^2 yes", "02-steps  codex  without  1/2  pass^2 no"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("want %q in\n%s", want, out.String())
		}
	}
}

// With no user at the board, a stand-in presses the first button of a form
// the agent made, and leaves the forms that were there before alone.
func TestPressAnswersOnlyANewForm(t *testing.T) {
	dir := t.TempDir()
	f := "---\ntype: form\ntitle: Go?\n---\n## Target\n- ( ) data/\n\n[ Delete ] [ Cancel ]\n"
	for _, n := range []string{"old.md", "new.md"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte(f), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	press(ctx, dir, map[string]bool{"old.md": true}, func() time.Time { return at("2026-11-25T10:00:30+09:00") })
	b, _ := os.ReadFile(filepath.Join(dir, "new.md"))
	if !strings.Contains(string(b), "submitted: Delete\nsubmitted_at: 2026-11-25T10:00:30+09:00") {
		t.Errorf("new.md:\n%s", b)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "old.md")); strings.Contains(string(b), "submitted") {
		t.Errorf("old.md was answered:\n%s", b)
	}
}
