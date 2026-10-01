package logview

import (
	"testing"

	"github.com/LeeSwallow/stickypane/internal/doc"
)

func TestAppendAddsALineAtTheEnd(t *testing.T) {
	for src, want := range map[string]string{
		"":                             "built\n",
		"first\n":                      "first\nbuilt\n",
		"first":                        "first\nbuilt\n",
		"---\ntype: log\n---\n":        "---\ntype: log\n---\nbuilt\n",
		"first\r\n":                    "first\r\nbuilt\r\n",
		"---\ntype: log\n---\nfirst\n": "---\ntype: log\n---\nfirst\nbuilt\n",
	} {
		d, err := (Append{Line: "built"}).Apply(doc.Parse([]byte(src)))
		if err != nil || string(d.Bytes()) != want {
			t.Errorf("Append to %q = %q, %v, want %q", src, d.Bytes(), err, want)
		}
	}
	d, _ := (Append{Line: "two\nlines"}).Apply(doc.Parse([]byte("")))
	if string(d.Bytes()) != "two lines\n" {
		t.Errorf("an entry is one line: %q", d.Bytes())
	}
}
