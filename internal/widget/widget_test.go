package widget

import (
	"reflect"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/doc"
)

func TestCleanDropsControlCharacters(t *testing.T) {
	got := Clean("a\tb \x1b[31mred\x1b[0m\x07 end\r\nnext")
	if want := "a    b red end\nnext"; got != want {
		t.Errorf("Clean = %q, want %q", got, want)
	}
}

func TestTruncateAndPadCountCells(t *testing.T) {
	if got := Truncate("한글 abcdef", 6); got != "한글 …" {
		t.Errorf("Truncate = %q", got)
	}
	if got := Truncate("anything", 0); got != "" {
		t.Errorf("Truncate to 0 = %q, want empty", got)
	}
	for _, s := range []string{"", "abc", "한글입니다", "📌 pinned note title"} {
		if w := Width(Pad(s, 8)); w != 8 {
			t.Errorf("Width(Pad(%q, 8)) = %d, want 8", s, w)
		}
	}
}

func TestWindow(t *testing.T) {
	lines := []string{"1", "2", "3", "4", "5"}
	cases := []struct {
		offset, height int
		want           []string
	}{
		{0, 2, []string{"1", "2"}},
		{3, 2, []string{"4", "5"}},
		{99, 2, []string{"4", "5"}},
		{-5, 2, []string{"1", "2"}},
		{0, 9, lines},
		{0, 0, nil},
	}
	for _, c := range cases {
		if got := Window(lines, c.offset, c.height); !reflect.DeepEqual(got, c.want) {
			t.Errorf("Window(offset=%d, height=%d) = %q, want %q", c.offset, c.height, got, c.want)
		}
	}
}

func TestScrollKey(t *testing.T) {
	if off, ok := ScrollKey(3, "j"); off != 4 || !ok {
		t.Errorf("j: got %d, %v", off, ok)
	}
	if off, ok := ScrollKey(0, "k"); off != 0 || !ok {
		t.Errorf("k at top: got %d, %v", off, ok)
	}
	if off, _ := ScrollKey(7, "g"); off != 0 {
		t.Errorf("g: got %d", off)
	}
	if _, ok := ScrollKey(0, "n"); ok {
		t.Error("n is not a scroll key")
	}
}

func TestRegistryLookupFallsBackToFirst(t *testing.T) {
	r := Registry{{Name: "note"}, {Name: "board"}}
	if r.Lookup("board").Name != "board" {
		t.Error("Lookup(board) should find board")
	}
	for _, name := range []string{"", "mystery"} {
		if got := r.Lookup(name).Name; got != "note" {
			t.Errorf("Lookup(%q) = %q, want the first kind", name, got)
		}
	}
}

func TestNewFile(t *testing.T) {
	cases := []struct{ kind, title, body, want string }{
		{"note", "", "hello\n", "hello\n"},
		{"note", "Plan", "", "---\ntitle: Plan\n---\n"},
		{"board", "Auth", "## To do\n", "---\ntype: board\ntitle: Auth\n---\n## To do\n"},
	}
	for _, c := range cases {
		if got := string(NewFile(c.kind, c.title, c.body)); got != c.want {
			t.Errorf("NewFile(%q, %q) = %q, want %q", c.kind, c.title, got, c.want)
		}
	}
	if doc.Parse(NewFile("log", "L", "")).Type() != "log" {
		t.Error("NewFile should write the type key")
	}
}
