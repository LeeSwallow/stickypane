package form

import (
	"errors"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// readBack reads a composed form with the form's own scan: its questions,
// its options and whether they are one choice, its fields and its buttons.
func readBack(t *testing.T, b []byte) (questions, options []string, radio bool, fields int, buttons []string) {
	t.Helper()
	d := doc.Parse(b)
	if d.Type() != "form" {
		t.Errorf("type = %q", d.Type())
	}
	lines, questions := scan(d.Body)
	for _, l := range lines {
		switch l.kind {
		case option:
			options, radio = append(options, l.text), l.radio
		case field:
			fields++
		}
	}
	return questions, options, radio, fields, buttonLabels(lines)
}

func TestComposeWritesOneChoice(t *testing.T) {
	t.Parallel()
	b, err := NewKind(nil).Make("Deploy", map[string]string{"question": "Where to?", "answer": "one", "choices": "staging\nproduction", "buttons": "Deploy\nCancel"})
	if err != nil {
		t.Fatal(err)
	}
	q, opts, radio, fields, buttons := readBack(t, b)
	if strings.Join(q, "|") != "Where to?" || strings.Join(opts, "|") != "staging|production" || !radio || fields != 0 || strings.Join(buttons, "|") != "Deploy|Cancel" {
		t.Errorf("read back %q %q %v %d %q from %q", q, opts, radio, fields, buttons, b)
	}
}

func TestComposeWritesManyChoicesAndText(t *testing.T) {
	t.Parallel()
	b, err := NewKind(nil).Make("Pick", map[string]string{"answer": "many", "choices": "a\nb\nc"})
	if err != nil {
		t.Fatal(err)
	}
	_, opts, radio, _, buttons := readBack(t, b)
	if strings.Join(opts, "|") != "a|b|c" || radio || strings.Join(buttons, "|") != defaultButton {
		t.Errorf("many: %q %v %q from %q", opts, radio, buttons, b)
	}
	b, err = NewKind(nil).Make("Why", map[string]string{"question": "Why?", "answer": "text"})
	if err != nil {
		t.Fatal(err)
	}
	q, opts, _, fields, _ := readBack(t, b)
	if strings.Join(q, "|") != "Why?" || len(opts) != 0 || fields != 1 {
		t.Errorf("text: %q %q %d from %q", q, opts, fields, b)
	}
}

func TestComposeRefusesWhatAFormCannotHold(t *testing.T) {
	t.Parallel()
	for _, opts := range []map[string]string{
		{"answer": "one", "choices": ""},
		{"buttons": "[ x ]"},
		{"answer": "maybe"},
	} {
		if _, err := NewKind(nil).Make("x", opts); !errors.Is(err, widget.ErrBadOption) {
			t.Errorf("%v: err = %v", opts, err)
		}
	}
}
