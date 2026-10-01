package checklist

import (
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/doc"
)

func applyTo(t *testing.T, src string, op doc.Op) string {
	t.Helper()
	d, err := op.Apply(doc.Parse([]byte(src)))
	if err != nil {
		t.Fatalf("%+v: %v", op, err)
	}
	return string(d.Bytes())
}

func TestCheckFindsTheItemByNameOrPosition(t *testing.T) {
	src := "intro\n- [ ] write tests\n  detail\n- [ ] ship it\n"
	if got := applyTo(t, src, Check{Item: "tests", Checked: true}); got != "intro\n- [x] write tests\n  detail\n- [ ] ship it\n" {
		t.Errorf("by a part of the text: %q", got)
	}
	if got := applyTo(t, src, Check{Item: "#2", Checked: true}); got != "intro\n- [ ] write tests\n  detail\n- [x] ship it\n" {
		t.Errorf("by position: %q", got)
	}
	done := "- [x] write tests\n"
	if got := applyTo(t, done, Check{Item: "write tests", Checked: true}); got != done {
		t.Errorf("checking a checked item changes nothing: %q", got)
	}
	if got := applyTo(t, done, Check{Item: "write tests", Checked: false}); got != "- [ ] write tests\n" {
		t.Errorf("uncheck: %q", got)
	}
	if _, err := (Check{Item: "deploy", Checked: true}).Apply(doc.Parse([]byte(src))); err == nil || !strings.Contains(err.Error(), "ship it") {
		t.Errorf("an item that is not there should list the items: %v", err)
	}
}
