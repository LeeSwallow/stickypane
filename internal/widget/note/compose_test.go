package note

import (
	"testing"

	"github.com/LeeSwallow/stickypane/internal/doc"
)

func TestComposeKeepsTheMarkdown(t *testing.T) {
	t.Parallel()
	md := "# Plan\n\n- one\n  - two\n"
	b, err := NewKind(Plain).Make("Plan", map[string]string{"text": md})
	if err != nil {
		t.Fatal(err)
	}
	d := doc.Parse(b)
	if title, _ := d.Get("title"); d.Body != md || title != "Plan" || d.Type() != "" {
		t.Errorf("body %q, title %q, type %q", d.Body, title, d.Type())
	}
	b, _ = NewKind(Plain).Make("", map[string]string{"text": "no newline"})
	if doc.Parse(b).Body != "no newline\n" {
		t.Errorf("a note ends with a newline: %q", b)
	}
}
