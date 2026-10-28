package api

import (
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestTodoMakesAndUpdatesAChecklist(t *testing.T) {
	a, dir := newAPI(t, nil)
	if got, err := a.Todo("plan", "add", "write tests"); err != nil || got != (Edited{Note: "plan.md", Action: "add", Item: "write tests", Summary: "0/1"}) {
		t.Fatalf("Todo add = %+v, %v", got, err)
	}
	if got := read(t, dir, "plan.md"); got != "---\ntype: checklist\ntitle: plan\nopen: true\n---\n- [ ] write tests\n" {
		t.Fatalf("a note that is not there yet is made, open: %q", got)
	}
	if _, err := a.Todo("plan", "add", "ship it"); err != nil {
		t.Fatal(err)
	}
	if got, err := a.Todo("plan", "check", "tests"); err != nil || got != (Edited{Note: "plan.md", Action: "check", Item: "tests", Summary: "1/2"}) {
		t.Errorf("Todo check = %+v, %v", got, err)
	}
	if got, err := a.Todo("plan", "uncheck", "#1"); err != nil || got != (Edited{Note: "plan.md", Action: "uncheck", Item: "#1", Summary: "0/2"}) {
		t.Errorf("Todo uncheck = %+v, %v", got, err)
	}
	if _, err := a.Todo("plan", "check", "deploy"); err == nil || !strings.Contains(err.Error(), "ship it") {
		t.Errorf("an unknown item should list the items: %v", err)
	}
	if _, err := a.Todo("plan", "finish", "x"); err == nil || !strings.Contains(err.Error(), "add, check or uncheck") {
		t.Errorf("an unknown action should say which there are: %v", err)
	}
	if _, err := a.Todo("plan", "add", "  "); err == nil {
		t.Error("an empty item is an error")
	}
}

func TestEditsRefuseANoteOfAnotherShape(t *testing.T) {
	a, dir := newAPI(t, map[string]string{"b.md": "---\ntype: board\n---\n## To do\n- x\n", "n.md": "plain\n"})
	if _, err := a.Todo("b", "add", "y"); err == nil || !strings.Contains(err.Error(), "board") {
		t.Errorf("a board is not a checklist: %v", err)
	}
	if _, err := a.Chart("n", "set", "a", "1"); err == nil || !strings.Contains(err.Error(), "note") {
		t.Errorf("a plain note is not a chart: %v", err)
	}
	if got := read(t, dir, "b.md"); got != "---\ntype: board\n---\n## To do\n- x\n" {
		t.Errorf("a refused edit must not touch the file: %q", got)
	}
	if _, err := a.Todo("../x", "add", "y"); err == nil {
		t.Error("a name outside the folder is refused")
	}
}

func TestCardAddsAndMoves(t *testing.T) {
	a, dir := newAPI(t, nil)
	if got, err := a.Card("work", "add", "login API", "Doing"); err != nil || got != (Edited{Note: "work.md", Action: "add", Item: "login API", To: "Doing", Summary: "1 card"}) {
		t.Fatalf("Card add = %+v, %v", got, err)
	}
	if _, err := a.Card("work", "add", "payments", "To do"); err != nil {
		t.Fatal(err)
	}
	if got, err := a.Card("work", "move", "login", "to do"); err != nil || got != (Edited{Note: "work.md", Action: "move", Item: "login", To: "to do", Summary: "2 cards"}) {
		t.Errorf("Card move = %+v, %v", got, err)
	}
	// A new board starts with the usual three columns.
	want := "---\ntype: board\ntitle: work\nopen: true\n---\n## To do\n- payments\n- login API\n\n## Doing\n\n## Done\n"
	if got := read(t, dir, "work.md"); got != want {
		t.Errorf("file = %q", got)
	}
	if _, err := a.Card("work", "move", "login", ""); err == nil {
		t.Error("a move needs a column")
	}
}

func TestChartSetsAndCounts(t *testing.T) {
	a, dir := newAPI(t, nil)
	if got, err := a.Chart("tests", "set", "app", "67"); err != nil || got != (Edited{Note: "tests.md", Action: "set", Item: "app", Value: "67", Summary: "1 value"}) {
		t.Fatalf("Chart set = %+v, %v", got, err)
	}
	if got, err := a.Chart("tests", "add", "app", "3"); err != nil || got != (Edited{Note: "tests.md", Action: "add", Item: "app", Value: "70", Summary: "1 value"}) {
		t.Errorf("Chart add = %+v, %v", got, err)
	}
	if got, err := a.Chart("tests", "add", "store", "-2"); err != nil || got != (Edited{Note: "tests.md", Action: "add", Item: "store", Value: "-2", Summary: "2 values"}) {
		t.Errorf("Chart add to a new label = %+v, %v", got, err)
	}
	if got := read(t, dir, "tests.md"); got != "---\ntype: chart\ntitle: tests\nopen: true\n---\napp: 70\nstore: -2\n" {
		t.Errorf("file = %q", got)
	}
	if _, err := a.Chart("tests", "add", "app", "lots"); err == nil {
		t.Error("an amount that is not a number is an error")
	}
}

func TestLogAppends(t *testing.T) {
	a, dir := newAPI(t, map[string]string{"plain.md": "first\n"})
	if got, err := a.Log("worklog", "tests passed"); err != nil || got != (Edited{Note: "worklog.md", Action: "log", Item: "tests passed", Summary: "1 line"}) {
		t.Fatalf("Log = %+v, %v", got, err)
	}
	if got, err := a.Log("worklog", "review done"); err != nil || got != (Edited{Note: "worklog.md", Action: "log", Item: "review done", Summary: "2 lines"}) {
		t.Errorf("Log = %+v, %v", got, err)
	}
	if got := read(t, dir, "worklog.md"); got != "---\ntype: log\ntitle: worklog\nopen: true\n---\ntests passed\nreview done\n" {
		t.Errorf("file = %q", got)
	}
	// A line can be added to a plain note too: it is the same as appending.
	// A plain note has no summary.
	if got, err := a.Log("plain", "second"); err != nil || got.Summary != "" || read(t, dir, "plain.md") != "first\nsecond\n" {
		t.Errorf("Log on a plain note: %v, %q", err, read(t, dir, "plain.md"))
	}
}

func TestSetChangesKeysAndNothingElse(t *testing.T) {
	a, dir := newAPI(t, map[string]string{"n.md": "---\ntitle: Old\nview: spark\n---\nbody stays\n"})
	if got, err := a.Set("n", []string{"title=New title", "type=chart", "view="}); err != nil || !reflect.DeepEqual(got, Keyed{Note: "n.md", Set: []string{"title", "type"}, Removed: []string{"view"}}) {
		t.Fatalf("Set = %+v, %v", got, err)
	}
	if got := read(t, dir, "n.md"); got != "---\ntitle: New title\ntype: chart\n---\nbody stays\n" {
		t.Errorf("file = %q", got)
	}
	for _, bad := range [][]string{{"no equals"}, {"Bad Key=1"}, {"=1"}, nil} {
		if _, err := a.Set("n", bad); err == nil {
			t.Errorf("Set(%q) should fail", bad)
		}
	}
	if _, err := a.Set("missing", []string{"open=true"}); err == nil {
		t.Error("Set does not make a note")
	}
}

func TestSayMakesAChatAndAddsMessages(t *testing.T) {
	a, dir := newAPI(t, nil)
	if got, err := a.Say("chat", "claude", "41/41 pass"); err != nil || got != (Edited{Note: "chat.md", Action: "say", Item: "41/41 pass", Who: "claude", Summary: "1 message"}) {
		t.Fatalf("Say = %+v, %v", got, err)
	}
	if got, err := a.Say("chat", "", "next?"); err != nil || got != (Edited{Note: "chat.md", Action: "say", Item: "next?", Who: "agent", Summary: "2 messages"}) {
		t.Errorf("Say = %+v, %v", got, err)
	}
	got := read(t, dir, "chat.md")
	if !strings.Contains(got, "type: chat") || !regexp.MustCompile(`(?m)^## \d{4}-\d{2}-\d{2}\n@claude \d\d:\d\d 41/41 pass\n@agent \d\d:\d\d next\?\n$`).MatchString(got) {
		t.Errorf("file = %q", got)
	}
	if _, err := a.Say("chat", "me", " \n "); err == nil {
		t.Error("an empty message is refused")
	}
}

func TestANewNoteIsTitledWithoutItsOrderNumber(t *testing.T) {
	a, dir := newAPI(t, nil)
	if _, err := a.Say("04-chat", "claude", "hello"); err != nil {
		t.Fatal(err)
	}
	if got := read(t, dir, "04-chat.md"); !strings.Contains(got, "title: chat\n") {
		t.Errorf("file = %q", got)
	}
}

func TestANameWithoutItsExtensionFindsTheNoteThatHasIt(t *testing.T) {
	a, _ := newAPI(t, map[string]string{"live.http": "### Ping\nGET http://x\n", "build.log": "ok\n", "plan.md": "x\n"})
	for name, want := range map[string]Shown{"live": {Note: "live.http", Open: true}, "build": {Note: "build.log", Open: true}, "plan": {Note: "plan.md", Open: true}} {
		if got, err := a.Show(name); err != nil || got != want {
			t.Errorf("Show(%q) = %+v, %v; want %+v", name, got, err, want)
		}
	}
	if got, err := a.Hide("live"); err != nil || got != (Shown{Note: "live.http"}) {
		t.Errorf("Hide = %+v, %v", got, err)
	}
	if got, err := a.Set("live", []string{"size=page"}); err != nil || got.Note != "live.http" {
		t.Errorf("Set = %+v, %v", got, err)
	}
	if b, err := a.Cat("live"); err != nil || !strings.Contains(string(b), "### Ping") {
		t.Errorf("Cat = %q, %v", b, err)
	}
	if _, err := a.Show("nothing"); err == nil || !strings.Contains(err.Error(), "nothing.md") {
		t.Errorf("a missing note still names the .md it looked for: %v", err)
	}
}
