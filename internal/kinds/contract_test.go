package kinds

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/i18n"
	"github.com/LeeSwallow/stickypane/internal/widget"
	"github.com/LeeSwallow/stickypane/internal/widget/note"
)

// Every registered shape keeps the widget contract, so a new shape only has
// to pass this to work on the board, in the catalog and in `kinds`.
func TestEveryKindKeepsTheContract(t *testing.T) {
	for _, k := range Default(note.Plain) {
		t.Run(k.Name, func(t *testing.T) {
			for field, v := range map[string]string{"Icon": k.Icon, "Label": k.Label, "Blurb": k.Blurb, "Example": k.Example, "Command": k.Command, "Usage": k.Usage} {
				if strings.TrimSpace(v) == "" {
					t.Errorf("%s is empty: the catalog and `stickypane kinds` show it", field)
				}
			}
			if k.Parse == nil || k.Template == nil {
				t.Fatal("a kind needs Parse and Template")
			}
			d := doc.Parse(widget.NewFile(k.Name, "", k.Example))
			if k.New != "" {
				d = doc.Document{Body: k.Example}
			}
			w := k.Parse(d, widget.DefaultEnv())
			for _, width := range []int{1, 7, 20, 80} {
				for _, active := range []bool{false, true} {
					out, at := w.Draw(width, active)
					for _, line := range strings.Split(out, "\n") {
						if widget.Width(line) > width {
							t.Errorf("Draw(%d, %v): line %q is wider than the pane", width, active, line)
						}
					}
					if n := len(strings.Split(out, "\n")); at.Ok() && (at.Start > at.End || at.End > n) {
						t.Errorf("Draw(%d, %v): span %+v is outside the %d lines drawn", width, active, at, n)
					}
				}
			}
			_ = w.Summary()
			for _, key := range strings.Fields(k.Keys) {
				w, _ = w.Update(key)
				if w == nil {
					t.Fatalf("Update(%q) returned no widget", key)
				}
			}
			if w.Sync(d) == nil {
				t.Error("Sync returned no widget")
			}
			if empty := k.Parse(doc.Document{}, widget.DefaultEnv()); empty == nil {
				t.Error("an empty file must still give a widget")
			} else {
				empty.Draw(20, true)
			}
		})
	}
}

// Every shape says what it holds, for a screen that is not a terminal, and
// a shape that takes actions refuses the ones it does not know.
func TestEveryKindHasDataAndRefusesUnknownActions(t *testing.T) {
	for _, k := range Default(note.Plain) {
		t.Run(k.Name, func(t *testing.T) {
			if k.Data == nil {
				t.Fatal("a kind needs Data")
			}
			d := doc.Parse(widget.NewFile(k.Name, "", k.Example))
			if k.New != "" {
				d = doc.Document{Body: k.Example}
			}
			b, err := json.Marshal(k.Data(d, widget.DefaultEnv()))
			if err != nil || string(b) == "null" {
				t.Errorf("Data of the example = %s, %v", b, err)
			}
			if k.Act != nil {
				if _, err := k.Act(d, widget.Action{Name: "no-such"}, widget.DefaultEnv()); !errors.Is(err, widget.ErrUnknownAction) {
					t.Errorf("Act(no-such) = %v, want ErrUnknownAction", err)
				}
			}
		})
	}
}

// Every shape can be made from a screen's options: what it asks is well
// formed and translated, and what it makes with no option given is a note
// of the same shape.
func TestEveryKindComposesFromItsOptions(t *testing.T) {
	ko, err := i18n.Load("ko")
	if err != nil {
		t.Fatal(err)
	}
	reg := Default(note.Plain)
	for _, k := range reg {
		t.Run(k.Name, func(t *testing.T) {
			seen := map[string]bool{}
			for _, o := range k.Options {
				if o.Name == "" || seen[o.Name] || o.Label == "" {
					t.Errorf("option %+v: needs a name of its own and a label", o)
				}
				seen[o.Name] = true
				if o.Type == widget.OptChoice && len(o.Choices) == 0 {
					t.Errorf("option %s: a choice needs choices", o.Name)
				}
				if _, err := o.Check(o.Default); err != nil {
					t.Errorf("option %s: the default %q is not a value: %v", o.Name, o.Default, err)
				}
				if ko.Labels[o.Label] == "" {
					t.Errorf("option %s: label %q has no Korean in labels", o.Name, o.Label)
				}
			}
			if len(k.Options) > 0 && k.Compose == nil {
				t.Error("a kind with options needs Compose")
			}
			b, err := k.Make("Made here", nil)
			if err != nil {
				t.Fatalf("Make with the defaults: %v", err)
			}
			ext := k.New
			if ext == "" {
				ext = ".md"
			}
			if got := reg.For("made-here"+ext, doc.Parse(b)); got.Name != k.Name {
				t.Errorf("made %q, which reads as a %s", b, got.Name)
			}
		})
	}
}
