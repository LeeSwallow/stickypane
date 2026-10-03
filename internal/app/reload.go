package app

import (
	"path"
	"sort"
	"strings"
	"time"

	"github.com/LeeSwallow/stickypane/internal/api"
	"github.com/LeeSwallow/stickypane/internal/arrange"
	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/kinds"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// widgetFor returns the kind and the widget for a note. The widget of a
// note that did not change is kept, and one that changed is synced, so
// cursors survive. kept reports that the old widget is shown as it was,
// so what it drew last is still good.
func (m *Model) widgetFor(n store.Note, old page, had bool) (kind widget.Kind, w widget.Widget, kept bool) {
	kind = m.reg.For(n.Name, n.Doc)
	switch {
	case !had || old.kind.Name != kind.Name:
		return kind, kind.Parse(n.Doc), false
	case old.note.Err == nil && n.Err == nil && old.note.ModTime.Equal(n.ModTime) && old.note.Path == n.Path:
		// The file was not touched: the store handed back what it read.
		return kind, old.w, true
	case string(old.note.Doc.Bytes()) == string(n.Doc.Bytes()):
		// Front matter counts too: a form shows whether it was
		// submitted and a chart which view it has.
		return kind, old.w, true
	}
	return kind, old.w.Sync(n.Doc), false
}

// reload rescans the folder and reads the arrangement again. The notes
// shown are those of the active tab; a tab chosen from outside (an agent's
// stickypane show) is followed.
func (m *Model) reload() {
	// A language chosen from outside (stickypane language ko) is followed;
	// every widget is rebuilt so nothing keeps the old words.
	board, err := m.store.Load()
	if err != nil {
		m.status = say(tr.CannotReadNotes, map[string]any{"Err": err.Error()})
		return
	}
	// What a note's kind fills in by itself, the board writes, so that an
	// agent editing the file never has to; then it reads the board again.
	// Charts computed from other notes (from:) are computed too.
	derived, _ := api.New(m.store, m.reg).Derive()
	if m.tend(board.Tabs, m.shownDocs()) || derived > 0 {
		if board, err = m.store.Load(); err != nil {
			m.status = say(tr.CannotReadNotes, map[string]any{"Err": err.Error()})
			return
		}
	}
	set := board.Settings
	m.applyPrefs(set)
	if l := set.Language(); l != m.language {
		m.language = l
		UseLanguage(l)
		m.items = nil
		m.drawn = nil
	}
	tabs := board.Tabs
	m.tabList = tabs
	m.tabName = set.Tab()
	m.tab = 0
	for i, t := range tabs {
		if t.Name == m.tabName {
			m.tab = i
		}
	}
	m.tabName = tabs[m.tab].Name
	notes := tabs[m.tab].Notes
	// What each file showed before, by file name: a note or a book's page.
	old := map[string]page{}
	wasOpen := make(map[string]bool, len(m.items))
	for _, it := range m.items {
		wasOpen[it.note.Name] = m.isOpen(it)
		if len(it.pages) == 0 {
			old[it.note.Name] = page{it.note, it.kind, it.w}
		}
		for _, p := range it.pages {
			old[p.note.Name] = p
		}
	}
	prev := m.index(m.focus)
	first := m.known == nil

	views := set.Views()
	if err := set.Err(); err != nil {
		m.status = say(tr.ViewsBroken, map[string]any{"Err": err.Error()})
	}
	m.views = views

	unreadable := func(n store.Note) store.Note {
		if n.Err != nil {
			n.Doc = doc.Document{Body: say(tr.CannotShowNote, map[string]any{"Err": n.Err.Error()})}
		}
		return n
	}
	known := make(map[string]bool, len(notes))
	items := make([]item, 0, len(notes))
	changedAny := false
	m.setEmbeds(board)
	// A script's log is shown in the script's pane, not as a note of its own.
	logs := logsOfScripts(notes)
	shownBy := make(map[string]bool, len(logs))
	for _, l := range logs {
		shownBy[l.Name] = true
	}
	for _, n := range notes {
		if shownBy[n.Name] {
			continue
		}
		it := item{note: unreadable(n)}
		if n.Book() {
			for i, pn := range n.Pages {
				pn = unreadable(pn)
				o, had := old[pn.Name]
				kind, w, _ := m.widgetFor(pn, o, had)
				it.pages = append(it.pages, page{pn, kind, w})
				if pn.Name == m.anchors[n.Name].page {
					it.page = i
				}
			}
			it.kind, it.w = it.pages[it.page].kind, &book{pages: it.pages, at: it.page}
		} else {
			o, had := old[n.Name]
			var oldLog store.Note
			if wl, ok := o.w.(*withLog); ok {
				o.w, oldLog = wl.script, wl.log
			}
			var kept bool
			it.kind, it.w, kept = m.widgetFor(it.note, o, had)
			changedAny = changedAny || !kept
			if l, ok := logs[n.Name]; ok {
				l = unreadable(l)
				kept = kept && l.Path == oldLog.Path && l.ModTime.Equal(oldLog.ModTime)
				it.w = &withLog{owner: path.Ext(n.Name), script: it.w, log: l, now: m.now}
			}
			if !kept {
				m.forget(n.Name)
			}
		}
		items = append(items, it)
		known[n.Name] = true
		if _, says := arrange.Open(views[n.Name], n.Doc); !says && ((!first && !m.known[n.Name]) || wasOpen[n.Name]) {
			m.peek[n.Name] = true
		}
	}
	for name := range m.peek {
		if !known[name] {
			delete(m.peek, name)
		}
	}
	for name := range m.anchors {
		if !known[name] {
			delete(m.anchors, name)
		}
	}
	m.known = known
	// A note that shows another (![[name]]) is drawn again when anything
	// changed, since what it shows may have.
	if changedAny {
		for i := range items {
			if it := &items[i]; len(it.pages) == 0 && strings.Contains(it.note.Doc.Body, "![[") {
				it.w = it.kind.Parse(it.note.Doc)
				m.forget(it.note.Name)
			}
		}
	}
	for name := range m.drawn {
		if !known[name] {
			delete(m.drawn, name)
		}
	}
	// Notes keep the order of their names, except that the ones the user
	// moved come first in the order they were given, and pinned ones before
	// everything.
	place := map[string]int{}
	for i, name := range set.Order(m.tabName) {
		place[name] = i + 1
	}
	rank := func(it item) int {
		if p, ok := place[it.note.Name]; ok {
			return p
		}
		return len(place) + 1
	}
	sort.SliceStable(items, func(i, j int) bool {
		if pi, pj := m.pinned(items[i]), m.pinned(items[j]); pi != pj {
			return pi
		}
		return rank(items[i]) < rank(items[j])
	})
	m.items = items

	if m.seen == nil { // first load: nothing counts as changed yet
		m.seen = make(map[string]time.Time, len(items))
		for _, it := range items {
			m.seen[it.note.Name] = it.note.ModTime
		}
	}
	if m.index(m.focus) < 0 {
		m.focus = m.tabFocus[m.tabName]
		if m.index(m.focus) < 0 {
			m.focus = ""
			if len(items) > 0 {
				m.focus = items[max(min(prev, len(items)-1), 0)].note.Name
			}
		}
		m.reveal = revealNote
	}
	m.tabFocus[m.tabName] = m.focus
	m.markSeen(m.focus)
	m.followEdit()
	if m.zoomed() && m.index(m.zoomName) < 0 {
		m.mode = modeBoard
		m.status = tr.NoteRemoved
	}
}

// anchor is where the view is in a book: a page, and how far into it. It
// is kept by page rather than by line so that a page added or removed
// before it does not move the view.
type anchor struct {
	page   string
	within int
	start  int // the line the page started at when the anchor was taken
}

// turn scrolls a book to the start of the page before or after the one the
// view is on.
func (m *Model) turn(i, delta int) {
	it := &m.items[i]
	b, ok := it.w.(*book)
	if !ok || len(it.pages) == 0 {
		return
	}
	to := max(min(b.at+delta, len(it.pages)-1), 0)
	b.at, it.page, it.kind = to, to, it.pages[to].kind
	if to < len(b.starts) {
		m.anchors[it.note.Name] = anchor{page: it.pages[to].note.Name, start: b.starts[to]}
		m.offsets[it.note.Name] = b.starts[to]
		m.zoomScroll = b.starts[to]
	}
}

// setWidget replaces the widget an item shows. A book's widget stays the
// book: its pages are replaced inside it.
func (m *Model) setWidget(i int, w widget.Widget) {
	m.forget(m.items[i].note.Name)
	if _, ok := m.items[i].w.(*book); ok {
		return
	}
	m.items[i].w = w
}

// showing returns the position of the item that shows the file called
// name, or -1.
func (m *Model) showing(name string) int {
	for i, it := range m.items {
		if it.file().Name == name {
			return i
		}
	}
	return -1
}

// tend writes into the notes of every tab what their kinds fill in by
// themselves (Kind.Tend), such as the time an item was ticked by hand, at
// the time the file changed. It reports whether it wrote anything.
func (m *Model) tend(tabs []store.Tab, before map[string]doc.Document) bool {
	wrote := false
	visit := func(n store.Note) {
		if n.Err != nil || n.Book() {
			return
		}
		if k := m.reg.For(n.Name, n.Doc); k.Tend != nil {
			if op := k.Tend(before[n.Name], n.Doc, n.ModTime); op != nil && m.store.Apply(n.Name, op) == nil {
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

// shownDocs is what each file of the screen said when it was last read, so
// that a kind can tell what changed: a card moved to another column.
func (m *Model) shownDocs() map[string]doc.Document {
	docs := make(map[string]doc.Document, len(m.items))
	for _, it := range m.items {
		docs[it.note.Name] = it.note.Doc
		for _, p := range it.pages {
			docs[p.note.Name] = p.note.Doc
		}
	}
	return docs
}

// setEmbeds tells the Markdown renderer how to find the note a ![[name]]
// line shows: among every note of every tab, drawn by its own kind.
func (m *Model) setEmbeds(b store.Board) {
	files := map[string]store.Note{}
	var names []string
	for _, t := range b.Tabs {
		for _, n := range t.Notes {
			for _, f := range append([]store.Note{n}, n.Pages...) {
				if !f.Book() && f.Err == nil {
					files[f.Name] = f
					names = append(names, f.Name)
				}
			}
		}
	}
	reg := m.reg
	kinds.Embeds = func(name string) (string, widget.Widget, bool) {
		f, ok := files[api.Resolve(name, names)]
		if !ok {
			return "", nil, false
		}
		return arrange.Title(store.View{}, f.Doc, f.Name), reg.For(f.Name, f.Doc).Parse(f.Doc), true
	}
}
