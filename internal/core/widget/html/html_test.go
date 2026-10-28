package html

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/core/doc"
	"github.com/LeeSwallow/stickypane/internal/core/widget"
)

func draw(src string, width int) string {
	out, _ := Kind.Parse(doc.Document{Body: src}, widget.DefaultEnv()).Draw(width, false)
	return ansi.Strip(out)
}

func TestHTMLViewShowsTextWithoutTags(t *testing.T) {
	t.Parallel()
	out := draw("<h1>Bench</h1><script>x()</script><style>p{color:red}</style><p>a &amp; b</p>", 40)
	for _, want := range []string{"Bench", "a & b", "open it in the app to see it drawn"} {
		if !strings.Contains(out, want) {
			t.Errorf("view lacks %q:\n%s", want, out)
		}
	}
	for _, bad := range []string{"x()", "color", "<", "&amp;"} {
		if strings.Contains(out, bad) {
			t.Errorf("view shows %q:\n%s", bad, out)
		}
	}
	lines := strings.Split(out, "\n")
	if last := lines[len(lines)-1]; !strings.Contains(last, "the app") {
		t.Errorf("last line = %q", last)
	}
}

func TestHTMLViewKeepsBlocksApart(t *testing.T) {
	t.Parallel()
	out := draw("<p>one</p><p>two</p><!-- hidden --><SCRIPT type=x>no</SCRIPT>", 40)
	if !strings.Contains(out, "one\ntwo") || strings.Contains(out, "hidden") || strings.Contains(out, "no\n") {
		t.Errorf("view = %q", out)
	}
}

func TestHTMLViewFitsTheWidth(t *testing.T) {
	t.Parallel()
	for _, w := range []int{1, 7, 20} {
		for _, l := range strings.Split(draw("<p>a long line of words that must wrap</p>", w), "\n") {
			if len(l) > w && ansi.StringWidth(l) > w {
				t.Errorf("width %d: %q too wide", w, l)
			}
		}
	}
}

func TestHTMLDataIsTheSource(t *testing.T) {
	t.Parallel()
	src := "<h1>Bench</h1>\n"
	b, err := json.Marshal(Kind.Data(doc.Document{Body: src}, widget.DefaultEnv()))
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"html":"\u003ch1\u003eBench\u003c/h1\u003e\n"}`; string(b) != want {
		t.Errorf("Data = %s, want %s", b, want)
	}
}

func TestHTMLIsViewOnly(t *testing.T) {
	t.Parallel()
	if Kind.Act != nil || Kind.Events != nil {
		t.Error("an html note has no Act and no Events")
	}
}

func TestHTMLTextSurvivesOddSource(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct{ src, want string }{
		"unclosed script":     {"<script>İİİ", ""},
		"unclosed script 2":   {"hi<SCRIPT>İ alert(1)", "hi"},
		"upper script, style": {"<SCRIPT>a()</SCRIPT>x<Style>b{}</STYLE>y", "xy"},
		"non-ascii before":    {"İİİ<script>a()</script>ok", "İİİok"},
		"comment":             {"a<!-- İ hidden -->b", "ab"},
		"unclosed comment":    {"a<!-- İ", "a"},
	} {
		t.Run(name, func(t *testing.T) {
			if got := text(tc.src); got != tc.want {
				t.Errorf("text(%q) = %q, want %q", tc.src, got, tc.want)
			}
		})
	}
}

// A '>' inside a quoted attribute ends the tag early, so the rest of the
// attribute shows as text. A text preview may leak it; the overlay draws the
// page properly.
func TestHTMLQuotedGreaterThanMayLeak(t *testing.T) {
	t.Parallel()
	if got := text(`<a title="x>y">z</a>`); got != `y">z` {
		t.Errorf("text = %q", got)
	}
}
