package arrange

import (
	"testing"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

func yes() *bool { b := true; return &b }
func no() *bool  { b := false; return &b }

func front(src string) doc.Document { return doc.Parse([]byte(src)) }

func TestTheUsersChoiceWinsOverTheNoteAndTheKind(t *testing.T) {
	agent := front("---\nopen: true\nsize: card\nrows: 5\npin: true\ncolor: pink\ntitle: Plan\n---\nx\n")
	user := store.View{Open: no(), Size: "page", Rows: 9, Pin: no(), Color: "blue", Title: "Mine"}
	kind := widget.Kind{Rows: 3, Size: func(doc.Document) string { return widget.SizeHalf }}

	if open, said := Open(user, agent); open || !said {
		t.Errorf("Open = %v, %v; sticky.json closes it", open, said)
	}
	if got := Size(user, agent, kind, agent); got != "page" {
		t.Errorf("Size = %q", got)
	}
	if got := Rows(user, agent, kind); got != 9 {
		t.Errorf("Rows = %d", got)
	}
	if Pinned(user, agent) {
		t.Error("Pinned: sticky.json unpins it")
	}
	if got := Color(user, agent); got != "blue" {
		t.Errorf("Color = %q", got)
	}
	if got := Title(user, agent, "plan.md"); got != "Mine" {
		t.Errorf("Title = %q", got)
	}
}

func TestTheNoteProposesWhenTheUserSaidNothing(t *testing.T) {
	agent := front("---\nopen: TRUE\nsize: Card\nrows: 5\npin: true\ncolor: pink\ntitle: Plan\n---\nx\n")
	kind := widget.Kind{Rows: 3, Size: func(doc.Document) string { return widget.SizeHalf }}
	var none store.View

	if open, said := Open(none, agent); !open || !said {
		t.Errorf("Open = %v, %v", open, said)
	}
	if got := Size(none, agent, kind, agent); got != "card" {
		t.Errorf("Size = %q", got)
	}
	if got := Rows(none, agent, kind); got != 5 {
		t.Errorf("Rows = %d", got)
	}
	if !Pinned(none, agent) || Color(none, agent) != "pink" || Title(none, agent, "plan.md") != "Plan" {
		t.Error("pin, color and title come from the front matter")
	}
}

func TestTheKindDecidesLast(t *testing.T) {
	var none store.View
	plain := front("just text\n")
	kind := widget.Kind{Rows: 10, Size: func(d doc.Document) string { return widget.SizeCard }}

	if open, said := Open(none, plain); open || said {
		t.Errorf("Open = %v, %v; nothing says", open, said)
	}
	if got := Size(none, front("---\nsize: huge\n---\n"), kind, plain); got != "card" {
		t.Errorf("an invalid size falls through to the kind: %q", got)
	}
	if got := Size(none, plain, widget.Kind{}, plain); got != widget.SizePage {
		t.Errorf("a kind without a size takes the page: %q", got)
	}
	if got := Rows(none, front("---\nrows: lots\n---\n"), kind); got != 10 {
		t.Errorf("Rows = %d", got)
	}
	if Pinned(none, plain) || Color(none, plain) != "" {
		t.Error("nothing pins or colors a plain note")
	}
	if got := Title(none, plain, "docs/02-usage.md"); got != "02-usage" {
		t.Errorf("Title = %q; the file name without folder and extension", got)
	}
}

func TestChangeTurnsAKeyIntoAChangeOfTheView(t *testing.T) {
	var v store.View
	for _, kv := range [][2]string{{"open", "true"}, {"pin", "False"}, {"size", "half"}, {"rows", "12"}, {"color", "Blue"}} {
		change, ok, err := Change(kv[0], kv[1])
		if !ok || err != nil {
			t.Fatalf("Change(%q, %q) = %v, %v", kv[0], kv[1], ok, err)
		}
		change(&v)
	}
	if v.Open == nil || !*v.Open || v.Pin == nil || *v.Pin || v.Size != "half" || v.Rows != 12 || v.Color != "blue" {
		t.Errorf("view = %+v", v)
	}
	change, _, _ := Change("open", "")
	change(&v)
	if v.Open != nil {
		t.Error("an empty value takes the key back out")
	}
	if _, ok, _ := Change("title", "x"); ok {
		t.Error("title is not about the arrangement")
	}
	for _, kv := range [][2]string{{"open", "maybe"}, {"size", "huge"}, {"rows", "0"}, {"color", "teal"}} {
		if _, ok, err := Change(kv[0], kv[1]); !ok || err == nil {
			t.Errorf("Change(%q, %q) should be refused", kv[0], kv[1])
		}
	}
}
