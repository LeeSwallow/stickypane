package editor

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// feed types a script: plain characters are keys, and <esc>, <enter>, <bs>,
// <del>, <tab>, <space>, <left>, <right>, <up>, <down>, <c-x> and <pgdn>,
// <pgup> are the named ones.
func feed(e *Editor, script string) (last Action) {
	names := map[string]string{
		"esc": "esc", "enter": "enter", "bs": "backspace", "del": "delete", "tab": "tab", "space": "space",
		"left": "left", "right": "right", "up": "up", "down": "down", "pgdn": "pgdown", "pgup": "pgup",
	}
	for i := 0; i < len(script); {
		if script[i] == '<' {
			if end := strings.IndexByte(script[i:], '>'); end > 0 {
				name := script[i+1 : i+end]
				if key, ok := names[name]; ok {
					text := ""
					if key == "space" {
						text = " "
					}
					last = e.Key(key, text)
					i += end + 1
					continue
				}
				if strings.HasPrefix(name, "c-") && len(name) == 3 {
					last = e.Key("ctrl+"+name[2:], "")
					i += end + 1
					continue
				}
			}
		}
		r := []rune(script[i:])[0]
		s := string(r)
		key := s
		if s == " " {
			key = "space"
		}
		last = e.Key(key, s)
		i += len(s)
	}
	return last
}

func TestEdits(t *testing.T) {
	cases := []struct{ name, text, script, want string }{
		{"insert at the cursor", "world\n", "ihello <esc>", "hello world\n"},
		{"append after the cursor", "ab\n", "aX<esc>", "aXb\n"},
		{"append at the end of the line", "ab\n", "A!<esc>", "ab!\n"},
		{"insert at the first non-blank", "  ab\n", "$I-<esc>", "  -ab\n"},
		{"open a line below, keeping the indent", "  - a\nb\n", "o- c<esc>", "  - a\n  - c\nb\n"},
		{"open a line above", "a\nb\n", "jOx<esc>", "a\nx\nb\n"},
		{"enter splits the line", "ab\n", "li<enter><esc>", "a\nb\n"},
		{"backspace joins lines", "a\nb\n", "ji<bs><esc>", "ab\n"},
		{"backspace deletes a wide character whole", "한글\n", "A<bs><esc>", "한\n"},
		{"delete a character", "abc\n", "lx", "ac\n"},
		{"delete a line", "a\nb\nc\n", "jdd", "a\nc\n"},
		{"delete the only line", "a\n", "dd", "\n"},
		{"delete to the end of the line", "hello world\n", "wD", "hello \n"},
		{"delete a word", "one two three\n", "wdw", "one three\n"},
		{"change a word", "one two three\n", "wcwTWO<esc>", "one TWO three\n"},
		{"change the line", "  one\ntwo\n", "ccx<esc>", "x\ntwo\n"},
		{"change to the end", "one two\n", "wCx<esc>", "one x\n"},
		{"replace a character", "abc\n", "lrX", "aXc\n"},
		{"join lines", "a\n  b\n", "J", "a b\n"},
		{"yank and paste a line", "a\nb\n", "yyjp", "a\nb\na\n"},
		{"paste a line above", "a\nb\n", "jyykP", "b\na\nb\n"},
		{"deleted text is pasted back", "abc\n", "x$p", "bca\n"},
		{"undo", "a\nb\n", "ddu", "a\nb\n"},
		{"undo takes back one whole insert", "a\n", "Abc<esc>Ade<esc>u", "abc\n"},
		{"redo", "a\nb\n", "ddu<c-r>", "b\n"},
		{"word motions cross lines", "one\ntwo\n", "wix<esc>", "one\nxtwo\n"},
		{"back by a word", "one two\n", "$bix<esc>", "one xtwo\n"},
		{"end of word", "one two\n", "eax<esc>", "onex two\n"},
		{"line ends and starts", "abc\n", "$ix<esc>0iy<esc>", "yabxc\n"},
		{"first and last line", "a\nb\nc\n", "Gix<esc>ggiy<esc>", "ya\nb\nxc\n"},
		{"go to a line by number", "a\nb\nc\n", ":2<enter>ix<esc>", "a\nxb\nc\n"},
		{"search forward and again", "a foo\nb foo\n", "/foo<enter>nix<esc>", "a foo\nb xfoo\n"},
		{"search backward", "a foo\nb foo\n", "G/foo<enter>Nix<esc>", "a xfoo\nb foo\n"},
		{"arrows move in insert mode", "ab\n", "i<right>x<left><left>y<esc>", "yaxb\n"},
		{"tab inserts two spaces", "a\n", "i<tab><esc>", "  a\n"},
		{"a file without a final newline keeps none", "a", "Ab<esc>", "ab"},
		{"an empty file", "", "ihi<esc>", "hi"},
		{"crlf line ends are kept", "a\r\nb\r\n", "Ax<esc>", "ax\r\nb\r\n"},
		{"esc cancels a pending operator", "a\nb\n", "d<esc>jx", "a\n\n"},
		{"keys that mean nothing do nothing", "a\n", "QzZ~", "a\n"},
	}
	for _, c := range cases {
		e := New(c.text)
		feed(e, c.script)
		if got := e.Text(); got != c.want {
			t.Errorf("%s: %q after %q = %q, want %q", c.name, c.text, c.script, got, c.want)
		}
	}
}

func TestCommandsAskTheAppToAct(t *testing.T) {
	e := New("a\n")
	if got := feed(e, ":w<enter>"); got != Save {
		t.Errorf(":w = %v", got)
	}
	if got := feed(e, ":q<enter>"); got != Quit {
		t.Errorf(":q on a clean buffer = %v", got)
	}
	feed(e, "x")
	if !e.Dirty() {
		t.Error("an edit should make the buffer dirty")
	}
	if got := feed(e, ":q<enter>"); got != None || !strings.Contains(e.Message(), "!") {
		t.Errorf(":q with changes should refuse and say how to force it: %v, %q", got, e.Message())
	}
	for script, want := range map[string]Action{
		":q!<enter>": ForceQuit, ":wq<enter>": SaveQuit, ":x<enter>": SaveQuit, "ZZ": SaveQuit,
		":w!<enter>": ForceSave, ":e!<enter>": Reload,
	} {
		if got := feed(e, script); got != want {
			t.Errorf("%s = %v, want %v", script, got, want)
		}
	}
	if got := feed(e, ":nonsense<enter>"); got != None || !strings.Contains(e.Message(), "nonsense") {
		t.Errorf("an unknown command should be named: %v, %q", got, e.Message())
	}
	if got := feed(e, ":w<esc>"); got != None || e.Mode() != Normal {
		t.Errorf("esc should leave the command line: %v, %v", got, e.Mode())
	}
	e.Saved()
	if e.Dirty() {
		t.Error("Saved should make the buffer clean")
	}
	feed(e, "ix<esc>u")
	if e.Dirty() {
		t.Error("undoing back to the saved text is clean again")
	}
}

func TestModesAndStatus(t *testing.T) {
	e := New("hello\nworld\n")
	if e.Mode() != Normal {
		t.Fatalf("mode = %v", e.Mode())
	}
	feed(e, "i")
	if left, _ := e.Status(); e.Mode() != Insert || !strings.Contains(left, "INSERT") {
		t.Errorf("status in insert mode = %q", left)
	}
	feed(e, "<esc>jl:")
	if left, right := e.Status(); e.Mode() != Command || !strings.HasPrefix(left, ":") || !strings.Contains(right, "2:2") {
		t.Errorf("status = %q | %q", left, right)
	}
}

func view(e *Editor, w, h int) []string {
	lines := e.View(w, h)
	for i, l := range lines {
		lines[i] = ansi.Strip(l)
	}
	return lines
}

func TestViewShowsNumbersAndWrapsLongLines(t *testing.T) {
	e := New("short\n" + strings.Repeat("가", 12) + "\nlast\n")
	lines := e.View(12, 6)
	if len(lines) != 6 {
		t.Fatalf("View should fill the height: %d lines", len(lines))
	}
	for _, l := range lines {
		if w := ansi.StringWidth(l); w > 12 {
			t.Errorf("line %q is %d cells wide, want at most 12", ansi.Strip(l), w)
		}
	}
	plain := view(e, 12, 6)
	want := []string{"1 short", "2 가가가가가", "  가가가가가", "  가가", "3 last", "~"}
	for i, w := range want {
		if strings.TrimRight(plain[i], " ") != w {
			t.Errorf("line %d = %q, want %q", i, plain[i], w)
		}
	}
}

func TestViewFollowsTheCursorAndPagesMove(t *testing.T) {
	var sb strings.Builder
	for i := 1; i <= 50; i++ {
		sb.WriteString("line\n")
	}
	e := New(sb.String())
	e.View(20, 10)
	feed(e, "G")
	if plain := view(e, 20, 10); !strings.HasPrefix(plain[9], "50 line") {
		t.Errorf("the last line should be visible after G: %q", plain)
	}
	feed(e, "gg<c-f>")
	if _, right := e.Status(); !strings.Contains(right, "11:1") || !strings.Contains(right, "2/5") {
		t.Errorf("ctrl+f should go one page down: %q", right)
	}
	feed(e, "<pgdn><pgup><c-b>")
	if _, right := e.Status(); !strings.Contains(right, "1:1") || !strings.Contains(right, "1/5") {
		t.Errorf("paging back should return to the top: %q", right)
	}
	feed(e, "<c-d>")
	if _, right := e.Status(); !strings.Contains(right, "6:1") {
		t.Errorf("ctrl+d should go half a page down: %q", right)
	}
}

func TestTinyViewsDoNotBreak(t *testing.T) {
	e := New("한글 line\n\nmore\n")
	for _, size := range [][2]int{{0, 0}, {1, 1}, {2, 3}, {3, 1}, {80, 1}} {
		feed(e, "jl")
		for _, l := range e.View(size[0], size[1]) {
			if w := ansi.StringWidth(l); w > size[0] {
				t.Errorf("%v: line %q is %d cells wide", size, ansi.Strip(l), w)
			}
		}
	}
}
