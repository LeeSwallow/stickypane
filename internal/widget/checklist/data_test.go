package checklist

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

const listSrc = "Release plan\n- [ ] build\n- [x] test ✅ 2026-10-03 14:02\n- [ ] ship\n"

func dataOf(t *testing.T, d doc.Document) string {
	t.Helper()
	b, err := json.Marshal(Kind.Data(d))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestDataOfAList(t *testing.T) {
	old := time.Local
	time.Local = time.FixedZone("KST", 9*3600)
	defer func() { time.Local = old }()
	got := dataOf(t, doc.Parse([]byte(listSrc)))
	want := `{"items":[{"text":"build","checked":false},{"text":"test","checked":true,"done":"2026-10-03T14:02:00+09:00"},{"text":"ship","checked":false}],"prose":["Release plan"],"done":1,"total":3}`
	if got != want {
		t.Errorf("Data = %s\nwant   %s", got, want)
	}
}

func TestDataOfAnEmptyList(t *testing.T) {
	got := dataOf(t, doc.Parse(Kind.Template("Plan")))
	want := `{"items":[],"prose":[],"done":0,"total":0}`
	if got != want {
		t.Errorf("Data = %s, want %s", got, want)
	}
}

func doAct(t *testing.T, src, name string, args map[string]string) (doc.Document, error) {
	t.Helper()
	d := doc.Parse([]byte(src))
	op, err := Kind.Act(d, widget.Action{Name: name, Args: args})
	if err != nil {
		if op != nil {
			t.Errorf("Op = %v with an error", op)
		}
		return d, err
	}
	out, err := op.Apply(d)
	if err != nil {
		t.Fatal(err)
	}
	return out, nil
}

func TestActToggleTicksTheItem(t *testing.T) {
	d, err := doAct(t, listSrc, "toggle", map[string]string{"i": "1"})
	if err != nil {
		t.Fatal(err)
	}
	// Item 1 is "test", already ticked: toggling unticks it.
	if got := read(d).entries[2]; got.checked {
		t.Errorf("item 1 still ticked: %+v", got)
	}
	d, err = doAct(t, listSrc, "toggle", map[string]string{"i": "0"})
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Items []struct{ Checked bool } `json:"items"`
	}
	if err := json.Unmarshal([]byte(dataOf(t, d)), &out); err != nil {
		t.Fatal(err)
	}
	if !out.Items[0].Checked {
		t.Errorf("item 0 not ticked: %s", dataOf(t, d))
	}
}

func TestActToggleWritesNoTime(t *testing.T) {
	d, err := doAct(t, listSrc, "toggle", map[string]string{"i": "0"})
	if err != nil {
		t.Fatal(err)
	}
	if !read(d).entries[1].done.IsZero() {
		t.Errorf("Act wrote a time: %q", d.Body)
	}
}

func TestActToggleSameTextCountsPosition(t *testing.T) {
	src := "- [ ] a\n- [ ] a\n"
	d, err := doAct(t, src, "toggle", map[string]string{"i": "1"})
	if err != nil {
		t.Fatal(err)
	}
	if e := read(d).entries; e[0].checked || !e[1].checked {
		t.Errorf("body = %q", d.Body)
	}
}

func TestActToggleOutOfRange(t *testing.T) {
	for _, i := range []string{"99", "-1", "3", "x", ""} {
		if _, err := doAct(t, listSrc, "toggle", map[string]string{"i": i}); err == nil {
			t.Errorf("toggle i=%q: no error", i)
		}
	}
}

func TestActAddAppends(t *testing.T) {
	d, err := doAct(t, listSrc, "add", map[string]string{"text": "announce"})
	if err != nil {
		t.Fatal(err)
	}
	want := listSrc + "- [ ] announce\n"
	if d.Body != want {
		t.Errorf("body = %q, want %q", d.Body, want)
	}
}

func TestActAddNeedsText(t *testing.T) {
	for _, s := range []string{"", "  ", "a\nb"} {
		if _, err := doAct(t, listSrc, "add", map[string]string{"text": s}); err == nil {
			t.Errorf("add text=%q: no error", s)
		}
	}
}

func TestActUnknown(t *testing.T) {
	_, err := doAct(t, listSrc, "delete", nil)
	if !errors.Is(err, widget.ErrUnknownAction) {
		t.Errorf("err = %v, want ErrUnknownAction", err)
	}
}
