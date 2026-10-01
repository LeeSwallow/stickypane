package widget

import (
	"strings"
	"testing"
)

func TestPick(t *testing.T) {
	names := []string{"Write tests", "write docs", "Ship", "ship it"}
	cases := []struct {
		want string
		idx  int
	}{
		{"Ship", 2},        // the exact text wins over a longer match
		{"write tests", 0}, // case does not matter
		{"docs", 1},        // a part of the text is enough when only one has it
		{"#3", 2},          // by position, counting from one
		{"4", 3},
	}
	for _, c := range cases {
		if got, err := Pick(names, c.want, "item"); err != nil || got != c.idx {
			t.Errorf("Pick(%q) = %d, %v, want %d", c.want, got, err, c.idx)
		}
	}
	if _, err := Pick(names, "write", "item"); err == nil || !strings.Contains(err.Error(), "Write tests") || !strings.Contains(err.Error(), "write docs") {
		t.Errorf("an unclear name should list what it could mean: %v", err)
	}
	if _, err := Pick(names, "deploy", "item"); err == nil || !strings.Contains(err.Error(), "Ship") {
		t.Errorf("a name that matches nothing should list what there is: %v", err)
	}
	if _, err := Pick(names, "#9", "item"); err == nil {
		t.Error("a position past the end is an error")
	}
	if _, err := Pick(nil, "x", "item"); err == nil || !strings.Contains(err.Error(), "no item") {
		t.Errorf("nothing to pick from: %v", err)
	}
}
