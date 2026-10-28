package logview

import (
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/core/doc"
	"github.com/LeeSwallow/stickypane/internal/core/widget"
)

func TestComposeWritesTheFirstEntries(t *testing.T) {
	t.Parallel()
	b, err := Kind.Make("Worklog", map[string]string{"entries": "started\nread the spec"})
	if err != nil {
		t.Fatal(err)
	}
	d := doc.Parse(b)
	if d.Type() != "log" || strings.Join(parse(d, widget.DefaultEnv()).lines, "|") != "started|read the spec" {
		t.Errorf("type %q, lines %q", d.Type(), parse(d, widget.DefaultEnv()).lines)
	}
}
