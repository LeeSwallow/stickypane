package lineedit

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func keys(l *Line, ks ...string) {
	for _, k := range ks {
		if len([]rune(k)) == 1 {
			l.Type(k)
			continue
		}
		l.Key(k)
	}
}

func TestEditing(t *testing.T) {
	l := New("hello", 20)
	keys(l, "left", "left", "X")
	if l.Value() != "helXlo" {
		t.Errorf("insert at the cursor: %q", l.Value())
	}
	keys(l, "backspace", "delete")
	if l.Value() != "helo" {
		t.Errorf("backspace and delete: %q", l.Value())
	}
	keys(l, "home", ">", "end", "<")
	if l.Value() != ">helo<" {
		t.Errorf("home and end: %q", l.Value())
	}
	l = New("one two three", 20)
	keys(l, "ctrl+w")
	if l.Value() != "one two " {
		t.Errorf("ctrl+w takes the word before: %q", l.Value())
	}
	keys(l, "alt+left", "ctrl+k")
	if l.Value() != "one " {
		t.Errorf("alt+left, then ctrl+k takes the rest: %q", l.Value())
	}
	keys(l, "ctrl+a", "ctrl+e", "ctrl+u")
	if l.Value() != "" {
		t.Errorf("ctrl+u takes everything before: %q", l.Value())
	}
	l.Type("두 줄\n붙여넣기")
	if l.Value() != "두 줄 붙여넣기" {
		t.Errorf("pasted lines become one: %q", l.Value())
	}
	if l.Key("f5") {
		t.Error("a key it does not know is not handled")
	}
}

// The view fits its width and keeps the cursor in sight, wide characters
// included.
func TestTheViewKeepsTheCursorInSight(t *testing.T) {
	l := New(strings.Repeat("가", 30), 10)
	v := ansi.Strip(l.View())
	if w := ansi.StringWidth(v); w > 10 {
		t.Errorf("view is %d cells wide: %q", w, v)
	}
	keys(l, "home")
	if v := ansi.Strip(l.View()); !strings.HasPrefix(v, "가") {
		t.Errorf("at the start the view shows the start: %q", v)
	}
	if v := New("", 10).View(); ansi.StringWidth(ansi.Strip(v)) != 1 {
		t.Errorf("an empty line shows the cursor: %q", v)
	}
}
