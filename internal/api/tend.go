package api

import (
	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/store"
)

// TendAll writes into the notes of every tab what their kinds fill in by
// themselves (Kind.Tend), such as the time an item was ticked by hand in an
// editor, at the time the file changed. before is what each file said when
// the caller last read it, by name; a note it does not hold is new to the
// caller. It reports whether it wrote anything. The screen calls it on
// every reload; Write, Create and every edit tend the note they write.
func (a *API) TendAll(tabs []store.Tab, before map[string]doc.Document) bool {
	wrote := false
	visit := func(n store.Note) {
		if n.Err != nil || n.Book() {
			return
		}
		if k := a.reg.For(n.Name, n.Doc); k.Tend != nil {
			if op := k.Tend(before[n.Name], n.Doc, n.ModTime); op != nil && a.st.Apply(n.Name, op) == nil {
				wrote = true
			}
		}
	}
	for _, t := range tabs {
		for _, n := range t.Notes {
			visit(n)
			for _, p := range n.Pages {
				visit(p)
			}
		}
	}
	return wrote
}
