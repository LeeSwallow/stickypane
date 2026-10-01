package doc

import (
	"errors"
	"reflect"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	cases := []string{
		"",
		"just a line\n",
		"no trailing newline",
		"---\ntype: board\ntitle: Auth\n---\n## To do\n- a\n",
		"---\ntype: log\n---",
		"---\r\ntype: board\r\n---\r\nbody\r\n",
		"---\nunclosed: true\nbody\n",
		"---\n",
		"--- not a fence\ntext\n",
	}
	for _, src := range cases {
		if got := string(Parse([]byte(src)).Bytes()); got != src {
			t.Errorf("round trip changed %q into %q", src, got)
		}
	}
}

func TestParseSplitsFrontAndBody(t *testing.T) {
	d := Parse([]byte("---\ntype: board\ntitle: Auth\n---\n## To do\n"))
	if !d.HasFront {
		t.Fatal("HasFront = false, want true")
	}
	if want := []string{"type: board", "title: Auth"}; !reflect.DeepEqual(d.Front, want) {
		t.Errorf("Front = %q, want %q", d.Front, want)
	}
	if d.Body != "## To do\n" {
		t.Errorf("Body = %q", d.Body)
	}
}

func TestUnclosedFrontIsBody(t *testing.T) {
	src := "---\ntitle: x\nstill body\n"
	d := Parse([]byte(src))
	if d.HasFront || d.Body != src {
		t.Errorf("got HasFront=%v Body=%q, want the whole file as body", d.HasFront, d.Body)
	}
}

func TestGet(t *testing.T) {
	d := Parse([]byte("---\ntype: Board\ntitle:   spaced  \nq1: \"a: b\"\nq2: 'it''s'\n nested: no\nlist:\n  - a\npin: true\r\n---\n"))
	cases := []struct {
		key, want string
		ok        bool
	}{
		{"type", "Board", true},
		{"title", "spaced", true},
		{"q1", "a: b", true},
		{"q2", "it's", true},
		{"nested", "", false},
		{"list", "", true},
		{"pin", "true", true},
		{"missing", "", false},
	}
	for _, c := range cases {
		got, ok := d.Get(c.key)
		if got != c.want || ok != c.ok {
			t.Errorf("Get(%q) = %q, %v; want %q, %v", c.key, got, ok, c.want, c.ok)
		}
	}
	if d.Type() != "board" {
		t.Errorf("Type() = %q, want lower-cased %q", d.Type(), "board")
	}
	if !d.Pinned() {
		t.Error("Pinned() = false, want true")
	}
}

func TestSet(t *testing.T) {
	src := "---\ntype: board\ntitle: Old\nowner: me\n---\nbody\n"
	cases := []struct {
		name, key, value, want string
	}{
		{"replace keeps order", "title", "New", "---\ntype: board\ntitle: New\nowner: me\n---\nbody\n"},
		{"append new key", "pin", "true", "---\ntype: board\ntitle: Old\nowner: me\npin: true\n---\nbody\n"},
		{"quote when needed", "title", "Auth: phase 2", "---\ntype: board\ntitle: \"Auth: phase 2\"\nowner: me\n---\nbody\n"},
	}
	for _, c := range cases {
		d := Parse([]byte(src))
		if got := string(d.Set(c.key, c.value).Bytes()); got != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
		if string(d.Bytes()) != src {
			t.Errorf("%s: Set changed the original document", c.name)
		}
	}
}

func TestSetReadsBackQuotedValue(t *testing.T) {
	d := Parse([]byte("hello\n")).Set("title", "Auth: phase 2")
	if got, _ := Parse(d.Bytes()).Get("title"); got != "Auth: phase 2" {
		t.Errorf("Get after Set = %q", got)
	}
}

func TestSetCreatesFront(t *testing.T) {
	got := string(Parse([]byte("hello\n")).Set("pin", "true").Bytes())
	if want := "---\npin: true\n---\nhello\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSetKeepsCRLF(t *testing.T) {
	d := Parse([]byte("---\r\ntitle: Old\r\n---\r\nbody\r\n"))
	got := string(d.Set("title", "New").Set("pin", "true").Bytes())
	if want := "---\r\ntitle: New\r\npin: true\r\n---\r\nbody\r\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSetKeyOp(t *testing.T) {
	var op Op = SetKey{Key: "color", Value: "blue"}
	d, err := op.Apply(Parse([]byte("note\n")))
	if err != nil {
		t.Fatal(err)
	}
	if got := string(d.Bytes()); got != "---\ncolor: blue\n---\nnote\n" {
		t.Errorf("got %q", got)
	}
}

func TestLinesJoinEOL(t *testing.T) {
	body := "a\r\nb\r\n"
	if got := Join(Lines(body)); got != body {
		t.Errorf("Join(Lines()) = %q", got)
	}
	if EOL(body) != "\r" || EOL("a\nb\n") != "" {
		t.Error("EOL should be \"\\r\" for CRLF bodies and empty otherwise")
	}
	if !errors.Is(ErrConflict, ErrConflict) {
		t.Error("ErrConflict must be comparable with errors.Is")
	}
}
