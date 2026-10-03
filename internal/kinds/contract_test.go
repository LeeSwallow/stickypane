package kinds

import (
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/doc"
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
			w := k.Parse(d)
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
			if empty := k.Parse(doc.Document{}); empty == nil {
				t.Error("an empty file must still give a widget")
			} else {
				empty.Draw(20, true)
			}
		})
	}
}
