package widget

import (
	"errors"
	"strings"
	"testing"
)

func composeKind() Kind {
	return Kind{
		Name:     "thing",
		Template: func(title string) []byte { return NewFile("thing", title, "") },
		Options: []Option{
			{Name: "view", Type: OptChoice, Label: "View", Choices: []string{"bar", "spark"}, Default: "bar"},
			{Name: "count", Type: OptNumber, Label: "Count", Default: "3"},
			{Name: "on", Type: OptBool, Label: "On", Default: "false"},
			{Name: "name", Type: OptText, Label: "Name"},
			{Name: "rows", Type: OptLines, Label: "Rows", Default: "a\nb"},
		},
		Compose: func(title string, opts map[string]string) ([]byte, error) {
			var b strings.Builder
			for _, k := range []string{"view", "count", "on", "name", "rows"} {
				b.WriteString(k + "=" + opts[k] + ";")
			}
			return []byte(b.String()), nil
		},
	}
}

func TestMakeFillsDefaultsAndChecksValues(t *testing.T) {
	t.Parallel()
	k := composeKind()
	got, err := k.Make("T", nil)
	if err != nil || string(got) != "view=bar;count=3;on=false;name=;rows=a\nb;" {
		t.Fatalf("defaults: %q, %v", got, err)
	}
	got, err = k.Make("T", map[string]string{"view": "SPARK", "count": "2.5", "on": "1", "name": " x ", "rows": "c\r\nd"})
	if err != nil || string(got) != "view=spark;count=2.5;on=true;name=x;rows=c\nd;" {
		t.Fatalf("given: %q, %v", got, err)
	}
	for _, bad := range []map[string]string{
		{"view": "pie"},
		{"count": "many"},
		{"on": "maybe"},
		{"name": "two\nlines"},
		{"colour": "red"},
	} {
		_, err := k.Make("T", bad)
		if !errors.Is(err, ErrBadOption) {
			t.Errorf("%v: err = %v, want ErrBadOption", bad, err)
			continue
		}
		for name := range bad {
			if !strings.Contains(err.Error(), name) {
				t.Errorf("%v: the error should name the option: %v", bad, err)
			}
		}
	}
	if _, err := k.Make("T", map[string]string{"colour": "red"}); !strings.Contains(err.Error(), "view, count, on, name, rows") {
		t.Errorf("an unknown option should list the known ones: %v", err)
	}
}

func TestMakeWithoutComposeUsesTheTemplate(t *testing.T) {
	t.Parallel()
	k := composeKind()
	k.Compose, k.Options = nil, nil
	if got, err := k.Make("T", nil); err != nil || string(got) != string(k.Template("T")) {
		t.Errorf("Make = %q, %v", got, err)
	}
	if _, err := k.Make("T", map[string]string{"x": "1"}); !errors.Is(err, ErrBadOption) {
		t.Errorf("a kind without options refuses them: %v", err)
	}
}

func TestLinesDropsBlanks(t *testing.T) {
	t.Parallel()
	if got := Lines(" a \n\n b\n"); strings.Join(got, "|") != "a|b" {
		t.Errorf("Lines = %q", got)
	}
}
