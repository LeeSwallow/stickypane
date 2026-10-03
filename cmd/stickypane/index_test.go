package main

import (
	"strings"
	"testing"
)

func TestIndexPrintsTheBoardInALinePerNote(t *testing.T) {
	project(t)
	if code, _, _ := execIn(t, "- [ ] write tests\n", "write", "plan", "--type", "checklist", "--title", "Plan"); code != 0 {
		t.Fatal("write")
	}
	code, out, _ := exec(t, "index")
	if code != 0 || !strings.Contains(out, "**Plan** `plan.md` · 0/1 — write tests") {
		t.Errorf("index: %d\n%s", code, out)
	}
	if code, out, _ := exec(t, "index", "--json"); code != 0 || !strings.Contains(out, `"gist":"write tests"`) {
		t.Errorf("index --json: %d %s", code, out)
	}
}
