package doc

import "testing"

func TestIsDetail(t *testing.T) {
	for line, want := range map[string]bool{"  more": true, "\tmore": true, "   ": false, "- item": false, "": false} {
		if got := IsDetail(line); got != want {
			t.Errorf("IsDetail(%q) = %v", line, got)
		}
	}
}

func TestOneLine(t *testing.T) {
	if got := OneLine("  two\r\nlines\n and  more "); got != "two lines and more" {
		t.Errorf("OneLine = %q", got)
	}
}

func TestAppendLineKeepsTheBodysLineEndings(t *testing.T) {
	for _, c := range []struct{ body, want string }{
		{"", "new\n"},
		{"a\n", "a\nnew\n"},
		{"a", "a\nnew\n"},
		{"a\r\n", "a\r\nnew\r\n"},
		{"a\r\nb", "a\r\nb\r\nnew\r\n"},
	} {
		if got := AppendLine(c.body, "new"); got != c.want {
			t.Errorf("AppendLine(%q) = %q, want %q", c.body, got, c.want)
		}
	}
}

func TestSetLineKeepsItsEnding(t *testing.T) {
	lines := []string{"a\r", "b"}
	SetLine(lines, 0, "x")
	SetLine(lines, 1, "y")
	if lines[0] != "x\r" || lines[1] != "y" {
		t.Errorf("lines = %q", lines)
	}
}
