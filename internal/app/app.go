// Package app is the terminal UI: it lists every note in a title bar, draws
// the open ones in full, and connects keys to widgets and to the store.
package app

import (
	"errors"
	"image/color"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/editor"
	"github.com/LeeSwallow/stickypane/internal/layout"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/theme"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

type mode int

const (
	modeBoard   mode = iota // the main screen: the title bar and the open notes
	modeZoom                // one note fills the screen
	modeInput               // a one-line prompt over the previous screen
	modeCatalog             // choosing a shape for a new note
	modeConfirm             // a yes/no question over the previous screen
	modeHelp                // the key reference
	modeEdit                // edit.go: a note in the built-in editor
	modeMove                // move.go: choosing where to move a note
)

// reveal says what the next layout should scroll into view.
type reveal int

const (
	revealNothing reveal = iota // leave the scroll position alone
	revealNote                  // the focused note, from its top
	revealCursor                // the selection inside the focused note
)

// cardWidth is the narrowest a note is drawn when the pane allows it. A pane
// at least two cards wide shows half-size notes side by side.
const cardWidth = 36

// Screens register themselves in these tables from their own files, so
// adding a screen or a key never means editing this file.
var (
	// handlers receive every message that is not handled globally.
	handlers = map[mode]func(*Model, tea.Msg) tea.Cmd{}
	// bodies draw everything above the bottom line, in at most the given
	// number of lines. A mode without a body shows the one it came from.
	bodies = map[mode]func(*Model, int) []string{}
	// footers draw the bottom line.
	footers = map[mode]func(*Model) string{}
	// boardKeys are the keys of the main screen. A focused open note takes
	// the keys its kind lists first; these get the rest.
	boardKeys = map[string]func(*Model) tea.Cmd{}
)

// item is a note with the widget that draws it. A book, which is a folder,
// has pages: its widget is the book, drawn as one scroll of all of them, and
// its kind is that of the page the view is on.
type item struct {
	note  store.Note
	kind  widget.Kind
	w     widget.Widget
	pages []page // a book's pages; nil for a file
	page  int    // the page the view is on
}

// page is one file of a book.
type page struct {
	note store.Note
	kind widget.Kind
	w    widget.Widget
}

// file is the note whose file the item shows and changes: its own, or the
// one of the page that is shown.
func (it item) file() store.Note {
	if len(it.pages) > 0 {
		return it.pages[it.page].note
	}
	return it.note
}

// Model is the Bubble Tea model.
type Model struct {
	store *store.Store
	reg   widget.Registry
	watch <-chan struct{}
	now   func() time.Time

	theme *theme.Holder // the theme in use, shared with the Markdown renderer
	dark  bool          // what the terminal said about its background

	items   []item
	focus   string               // file name of the focused note
	seen    map[string]time.Time // modification time last looked at, by file name
	known   map[string]bool      // notes that existed at the previous scan
	peek    map[string]bool      // notes shown open for this run without an "open" key
	views   store.Views          // how the user arranged the notes, from sticky.json
	anchors map[string]anchor    // where the view is in each book, by the book's name

	width, height int
	bar           []string       // the title bar, one or more lines
	tabs          []hit          // mouse.go: where each note's title is in the bar
	panes         []pane         // the open notes, each in its place
	screen        int            // which screen of panes is shown
	screens       int            // how many screens the open notes take
	offsets       map[string]int // how far the user scrolled inside a note, by file name
	lastClick     lastClick      // mouse.go
	canvas        []string       // the panes of the current screen, drawn
	reveal        reveal

	mode   mode
	back   mode   // where input, confirm and help return to
	status string // shown on the bottom line until the next key

	zoomName   string // zoom.go: file name of the zoomed note
	zoomLines  []string
	zoomScroll int

	helpScroll int // help.go

	input      textinput.Model // input.go
	inputLabel string
	inputEmpty bool // an empty line is submitted too
	onSubmit   func(string)

	catalogIdx int // create.go

	edit     *editor.Editor // edit.go
	editName string         // file name of the note being edited
	editDisk string         // the file as it was when loaded or last saved
	editBack mode           // where the editor returns to

	confirmMsg string // manage.go
	onConfirm  func()

	undo *undo // move.go: how to take back the last delete or archive

	running map[string]bool // run.go: the scripts that are running, by file name
	pending tea.Cmd         // run.go: what the current update started

	moveFile    string   // move.go: the file being moved
	moveTargets []string // where it can go; "" is the top level
	moveIdx     int
}

// changedMsg arrives when the watched folder changed.
type changedMsg struct{}

// reloadMsg asks for a rescan and, when err is set, reports it first.
type reloadMsg struct {
	what string
	err  error
}

// New returns a model showing the notes in st. watch may be nil when the
// folder cannot be watched; the screen then refreshes on "r" and after
// edits. th holds the theme in use; the Markdown renderer shares it.
func New(st *store.Store, reg widget.Registry, watch <-chan struct{}, th *theme.Holder) *Model {
	m := &Model{store: st, reg: reg, watch: watch, now: time.Now, peek: map[string]bool{}, offsets: map[string]int{}, anchors: map[string]anchor{}, reveal: revealNote, theme: th, dark: true}
	m.useTheme(theme.Pick(st.Theme(), true))
	m.reload()
	return m
}

// useTheme makes the whole screen draw with t from now on. Every widget is
// rebuilt, so nothing drawn with the old colors is kept.
func (m *Model) useTheme(t theme.Theme) {
	m.theme.Set(t)
	widget.Apply(t.Styles())
	m.items = nil
}

// chooseTheme makes the chosen theme the one in use, remembers the choice
// in sticky.json and says which theme that is.
func (m *Model) chooseTheme(name string) {
	if err := m.store.SetTheme(name); err != nil {
		m.status = "The theme was not saved: " + err.Error()
	}
	m.useTheme(theme.Pick(name, m.dark))
	m.reload()
	m.status = "Theme: " + m.theme.Get().Name + ". T tries the next one."
}

// SetStatus shows a message on the bottom line until the next key.
func (m *Model) SetStatus(s string) { m.status = s }

// Init implements tea.Model. Asking for the background color here, instead
// of before the program starts, keeps startup instant even in a terminal
// that never answers.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(waitForChange(m.watch), tea.RequestBackgroundColor)
}

func waitForChange(ch <-chan struct{}) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		if _, ok := <-ch; !ok {
			return nil
		}
		return changedMsg{}
	}
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case changedMsg:
		m.reload()
		cmd = waitForChange(m.watch)
	case reloadMsg:
		if msg.err != nil {
			m.status = msg.what + ": " + msg.err.Error()
		}
		m.reload()
	case tea.BackgroundColorMsg:
		// The terminal has said whether it is dark: an automatic theme
		// can be picked now, and every widget is redrawn with it.
		m.dark = msg.IsDark()
		m.useTheme(theme.Pick(m.store.Theme(), m.dark))
		m.reload()
	case tea.KeyPressMsg:
		// In the editor ctrl+c leaves insert mode, as it does in vi:
		// quitting there would drop what was typed.
		if msg.String() == "ctrl+c" && m.mode != modeEdit {
			return m, tea.Quit
		}
		m.status = ""
		cmd = m.dispatch(msg)
	case scriptDoneMsg:
		m.finished(msg)
	case tea.MouseClickMsg:
		m.click(msg.Mouse())
	case tea.MouseWheelMsg:
		m.wheel(msg.Mouse())
	default:
		cmd = m.dispatch(msg)
	}
	// Something the update started, such as a script.
	if m.pending != nil {
		cmd, m.pending = tea.Batch(cmd, m.pending), nil
	}
	m.relayout()
	return m, cmd
}

func (m *Model) dispatch(msg tea.Msg) tea.Cmd {
	if h, ok := handlers[m.mode]; ok {
		return h(m, msg)
	}
	return nil
}

// index returns the position of the note with the given file name, or -1.
func (m *Model) index(name string) int {
	for i, it := range m.items {
		if it.note.Name == name {
			return i
		}
	}
	return -1
}

func (m *Model) setFocus(name string) {
	m.focus = name
	m.reveal = revealNote
	m.markSeen(name)
}

func (m *Model) markSeen(name string) {
	if i := m.index(name); i >= 0 {
		m.seen[name] = m.items[i].note.ModTime
	}
}

// changed reports whether the note changed since it was last focused.
func (m *Model) changed(it item) bool {
	seen, ok := m.seen[it.note.Name]
	return !ok || !seen.Equal(it.note.ModTime)
}

// How a note is arranged comes from three places, in this order: what the
// user did on the board, kept in sticky.json; the note's own front matter,
// which is how an agent proposes an arrangement; and the note's kind.

// isOpen reports whether a note is drawn on the main screen. A note that
// neither sticky.json nor its front matter opens is closed, with two
// exceptions that last for this run and are written nowhere: a note that
// appeared while stickypane was running, so that what an agent just wrote
// is seen at once, and a note that was open when a rewrite dropped its key,
// so that an agent rewriting a note does not make it vanish from the screen.
// A note that cannot be read is always open, to say why.
func (m *Model) isOpen(it item) bool {
	if it.note.Err != nil {
		return true
	}
	if v := m.views[it.note.Name]; v.Open != nil {
		return *v.Open
	}
	if v, ok := it.note.Doc.Get("open"); ok {
		return strings.EqualFold(v, "true")
	}
	return m.peek[it.note.Name]
}

func validSize(v string) bool {
	return v == widget.SizePage || v == widget.SizeHalf || v == widget.SizeCard
}

// sizeOf returns the note's size.
func (m *Model) sizeOf(it item) string {
	if v := strings.ToLower(m.views[it.note.Name].Size); validSize(v) {
		return v
	}
	if v, _ := it.note.Doc.Get("size"); validSize(strings.ToLower(v)) {
		return strings.ToLower(v)
	}
	if it.kind.Size != nil {
		return it.kind.Size(it.file().Doc)
	}
	return widget.SizePage
}

// rowsOf returns the height in lines the note asks for, or 0 when it asks
// for as much as its content takes.
func (m *Model) rowsOf(it item) int {
	if n := m.views[it.note.Name].Rows; n > 0 {
		return n
	}
	if v, ok := it.note.Doc.Get("rows"); ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return it.kind.Rows
}

// pinned reports whether the note is kept first.
func (m *Model) pinned(it item) bool {
	if v := m.views[it.note.Name]; v.Pin != nil {
		return *v.Pin
	}
	return it.note.Doc.Pinned()
}

// color returns the note's color: the one chosen for it, or one derived
// from its name so that it keeps its color between runs.
func (m *Model) color(it item) color.Color {
	key := m.views[it.note.Name].Color
	if key == "" {
		key, _ = it.note.Doc.Get("color")
	}
	return m.noteColor(colorIndex(it.note.Name, key))
}

// setView changes how a note is arranged and shows the result.
func (m *Model) setView(name string, change func(*store.View)) {
	if err := m.store.SetView(name, change); err != nil {
		m.status = "The arrangement was not saved: " + err.Error()
	}
	m.reload()
}

// widthOf turns a size into cells for the current screen.
func (m *Model) widthOf(size string) int {
	switch size {
	case widget.SizeCard:
		return m.width / max(m.width/cardWidth, 1)
	case widget.SizeHalf:
		if m.width >= 2*cardWidth {
			return m.width / 2
		}
	}
	return m.width
}

// widgetFor returns the kind and the widget for a note. The widget of a
// note that did not change is kept, and one that changed is synced, so
// cursors survive.
func (m *Model) widgetFor(n store.Note, old page, had bool) (widget.Kind, widget.Widget) {
	kind := m.reg.For(n.Name, n.Doc)
	switch {
	case !had || old.kind.Name != kind.Name:
		return kind, kind.Parse(n.Doc)
	case string(old.note.Doc.Bytes()) == string(n.Doc.Bytes()):
		// Front matter counts too: a form shows whether it was
		// submitted and a chart which view it has.
		return kind, old.w
	}
	return kind, old.w.Sync(n.Doc)
}

// reload rescans the folder and reads the arrangement again.
func (m *Model) reload() {
	notes, err := m.store.Scan()
	if err != nil {
		m.status = "Cannot read notes: " + err.Error()
		return
	}
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

	views, err := m.store.Views()
	if err != nil {
		m.status = err.Error() + ". Notes are shown as they arrange themselves."
	}
	m.views = views

	unreadable := func(n store.Note) store.Note {
		if n.Err != nil {
			n.Doc = doc.Document{Body: "Cannot show this note: " + n.Err.Error()}
		}
		return n
	}
	known := make(map[string]bool, len(notes))
	items := make([]item, 0, len(notes))
	for _, n := range notes {
		it := item{note: unreadable(n)}
		if n.Book() {
			for i, pn := range n.Pages {
				pn = unreadable(pn)
				o, had := old[pn.Name]
				kind, w := m.widgetFor(pn, o, had)
				it.pages = append(it.pages, page{pn, kind, w})
				if pn.Name == m.anchors[n.Name].page {
					it.page = i
				}
			}
			it.kind, it.w = it.pages[it.page].kind, &book{pages: it.pages, at: it.page}
		} else {
			o, had := old[n.Name]
			it.kind, it.w = m.widgetFor(it.note, o, had)
		}
		items = append(items, it)
		known[n.Name] = true
		_, says := n.Doc.Get("open")
		if says = says || views[n.Name].Open != nil; !says && ((!first && !m.known[n.Name]) || wasOpen[n.Name]) {
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
	// Notes keep the order of their names, except that the ones the user
	// moved come first in the order they were given, and pinned ones before
	// everything.
	place := map[string]int{}
	for i, name := range m.store.Order() {
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
		m.focus = ""
		if len(items) > 0 {
			m.focus = items[max(min(prev, len(items)-1), 0)].note.Name
		}
		m.reveal = revealNote
	}
	m.markSeen(m.focus)
	m.followEdit()
	if m.zoomed() && m.index(m.zoomName) < 0 {
		m.mode = modeBoard
		m.status = "The note was removed."
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

// zoomed reports whether a note is zoomed, either in front or underneath a
// prompt, a confirmation or the help screen.
func (m *Model) zoomed() bool {
	switch m.mode {
	case modeZoom:
		return true
	case modeInput, modeConfirm, modeHelp:
		return m.back == modeZoom
	}
	return false
}

// apply writes an intent to the file called name and refreshes the screen. On a conflict
// nothing is written and the screen simply catches up with the file.
func (m *Model) apply(name string, op doc.Op) {
	err := m.store.Apply(name, op)
	switch {
	case errors.Is(err, doc.ErrConflict):
		m.status = "The file changed on disk, so the change was not applied."
	case err != nil:
		m.status = "Write failed: " + err.Error()
	}
	m.reload()
	i := m.showing(name)
	if i < 0 {
		return
	}
	if err != nil {
		// The widget may already show the change that did not happen.
		// Rebuild it from the file so the screen never claims otherwise.
		m.setWidget(i, m.items[i].w.Sync(m.items[i].file().Doc))
	}
	m.markSeen(m.items[i].note.Name)
}

// toWidget sends a key to a note's widget and carries out what it asks for.
func (m *Model) toWidget(i int, key string) {
	name := m.items[i].file().Name
	w, res := m.items[i].w.Update(key)
	m.setWidget(i, w)
	m.reveal = revealCursor
	m.act(name, res)
}

// act carries out what a widget asked for.
func (m *Model) act(name string, res widget.Result) {
	if res.Op != nil {
		m.apply(name, res.Op)
	}
	if res.Run {
		m.askRun(name)
	}
	if p := res.Prompt; p != nil {
		m.ask(p.Label, p.Initial, func(text string) {
			m.apply(name, p.Submit(text))
			m.reveal = revealCursor
		})
		m.inputEmpty = p.Empty
	}
}

// nameOf names a file: its title, or its file name without the folder and
// the extension.
func nameOf(n store.Note) string {
	if t, _ := n.Doc.Get("title"); t != "" {
		return widget.Clean(t)
	}
	base := path.Base(n.Name)
	return widget.Clean(strings.TrimSuffix(base, path.Ext(base)))
}

// label names a note in the title bar and in prompts: the name sticky.json
// gives it, else its title, else its name without the extension. A book
// goes by its folder's name unless it was given one.
func (m *Model) label(it item) string {
	if t := m.views[it.note.Name].Title; t != "" {
		return widget.Clean(t)
	}
	return nameOf(it.note)
}

// heading is the text in a note's top border. A plain note without a title
// has none, like a sticky note; other shapes fall back to the file name. A
// book shows its name and the page it is on.
func (m *Model) heading(it item) string {
	if len(it.pages) > 0 {
		return m.label(it) + " · " + nameOf(it.file())
	}
	t, _ := it.note.Doc.Get("title")
	if named := m.views[it.note.Name].Title; named != "" || (t == "" && (it.kind.Name != m.reg[0].Name || it.note.Err != nil)) {
		return m.label(it)
	}
	return widget.Clean(t)
}

// summary is the text at the right end of a note's top border: what its
// widget counts and, for a book, which page it is on. A script that is
// running says so instead.
func (m *Model) summary(it item) string {
	if m.running[it.file().Name] {
		return "running…"
	}
	return it.w.Summary()
}

// show returns the scroll offset that brings a span into a window of the
// given height, moving as little as possible from offset. A span taller than
// the window is shown from its start.
func show(at widget.Span, offset, height int) int {
	if at.End > offset+height {
		offset = at.End - height
	}
	if at.Start < offset {
		offset = at.Start
	}
	return offset
}

// bodyHeight is how many lines the open notes get on the main screen.
func (m *Model) bodyHeight() int {
	return max(m.height-1-min(len(m.bar), max(m.height-1, 0)), 0)
}

// minPane is the least a note gets on the screen, borders included, before
// the notes after it go to the next screen.
const minPane = 10

// pane is an open note in its place: a fixed rectangle on one of the
// screens, showing a window of the note's lines.
type pane struct {
	name   string // the note
	file   string // the file shown: the note's, or a page's. Scrolling is kept by file
	screen int
	rect   layout.Rect
	total  int // lines the note has
	offset int // the first line shown
}

// rows is how many lines of the note fit in the pane.
func (p pane) rows() int { return max(p.rect.H-2, 0) }

// relayout redraws the title bar and the open notes. The notes tile the
// screen: each has a fixed place and shows as much of itself as fits there,
// and the rest is reached by scrolling inside the note. Notes that do not
// fit on one screen go to the next, and the screen with the focused note is
// the one shown when the update asked to see it.
func (m *Model) relayout() {
	m.bar, m.panes, m.canvas, m.zoomLines = nil, nil, nil, nil
	if m.width <= 0 {
		return
	}
	if len(m.items) > 0 {
		m.bar = []string{""} // one line, drawn once the screens are known
	}
	h := m.bodyHeight()

	var open []item
	var widths []int
	for _, it := range m.items {
		if m.isOpen(it) {
			open = append(open, it)
			widths = append(widths, m.widthOf(m.sizeOf(it)))
		}
	}
	cells := layout.Rows(m.width, widths)

	// Draw every note at the width its place gives it. A row asks for the
	// height of its tallest note, or for the height a note fixes.
	lines := make([][]string, len(open))
	spans := make([]widget.Span, len(open))
	var need []int
	for i, it := range open {
		out, at := it.w.Draw(max(cells[i].W-4, 1), it.note.Name == m.focus)
		lines[i], spans[i] = strings.Split(out, "\n"), at
		want := len(lines[i]) + 2
		if rows := m.rowsOf(it); rows > 0 {
			want = rows + 2
		}
		if cells[i].Row == len(need) {
			need = append(need, 0)
		}
		need[cells[i].Row] = max(need[cells[i].Row], want)
	}
	slots := layout.Stack(h, need, minPane)

	m.screens = 1
	for _, s := range slots {
		m.screens = max(m.screens, s.Screen+1)
	}
	for i, it := range open {
		if it.note.Name == m.focus && m.reveal != revealNothing {
			m.screen = slots[cells[i].Row].Screen
		}
	}
	m.screen = max(min(m.screen, m.screens-1), 0)

	var rects []layout.Rect
	var boxes []string
	for i, it := range open {
		slot := slots[cells[i].Row]
		p := pane{
			name: it.note.Name, file: it.note.Name, screen: slot.Screen, total: len(lines[i]),
			rect: layout.Rect{X: cells[i].X, Y: slot.Y, W: cells[i].W, H: slot.H},
		}
		// Where the user left it; a log that was never scrolled shows its end.
		offset, scrolled := m.offsets[p.file]
		if !scrolled && it.kind.Tail && len(it.pages) == 0 {
			offset = p.total
		}
		if b, ok := it.w.(*book); ok {
			// A book keeps its place by page: when the page the view is
			// on moved, because pages before it came or went, the view
			// moves with it.
			if a, ok := m.anchors[p.name]; ok && b.at < len(b.starts) && a.page == it.pages[b.at].note.Name && a.start != b.starts[b.at] {
				offset, scrolled = b.starts[b.at]+a.within, true
				m.offsets[p.file] = offset
			}
		}
		focused := p.name == m.focus
		if focused && m.reveal == revealCursor && spans[i].Ok() {
			offset = show(spans[i], offset, p.rows())
			m.offsets[p.file] = offset
		}
		p.offset = widget.ClampOffset(offset, p.total, p.rows())
		if scrolled && it.kind.Tail && len(it.pages) == 0 && p.offset >= p.total-p.rows() {
			delete(m.offsets, p.file) // back at the end: follow the log again
		}
		m.settle(i, p.offset, p.rows())
		m.panes = append(m.panes, p)
		if p.screen != m.screen || p.rect.H < 2 {
			continue
		}
		body := append([]string(nil), widget.Window(lines[i], p.offset, p.rows())...)
		for len(body) < p.rows() {
			body = append(body, "")
		}
		rects = append(rects, p.rect)
		boxes = append(boxes, frame(box{
			title: m.heading(it), icon: it.kind.Icon, summary: m.summary(it),
			body: strings.Join(body, "\n"), width: p.rect.W, color: m.color(it), focused: focused,
			offset: p.offset, total: p.total,
		}))
	}
	m.canvas = layout.Compose(rects, boxes)
	m.bar = m.titleBar()

	if m.zoomed() {
		m.layoutZoom()
	}
	m.reveal = revealNothing
}

// settle notes which page of a book the view is on after a scroll, so the
// next keys go to that page and the place survives a reload. i is the
// index into the open notes of the current layout.
func (m *Model) settle(i, offset, rows int) {
	for k := range m.items {
		it := &m.items[k]
		b, ok := it.w.(*book)
		if !ok || it.note.Name != m.openName(i) {
			continue
		}
		b.settle(offset, rows)
		it.page, it.kind = b.at, it.pages[b.at].kind
		if b.at < len(b.starts) {
			m.anchors[it.note.Name] = anchor{page: it.pages[b.at].note.Name, within: offset - b.starts[b.at], start: b.starts[b.at]}
		}
	}
}

// openName returns the name of the i-th open note.
func (m *Model) openName(i int) string {
	n := 0
	for _, it := range m.items {
		if m.isOpen(it) {
			if n == i {
				return it.note.Name
			}
			n++
		}
	}
	return ""
}

// paneOf returns the pane of an open note.
func (m *Model) paneOf(name string) (pane, bool) {
	for _, p := range m.panes {
		if p.name == name {
			return p, true
		}
	}
	return pane{}, false
}

// scrollPane moves the window of an open note to offset. Out of range
// values are fixed when the screen is drawn.
func (m *Model) scrollPane(p pane, offset int) {
	m.offsets[p.file] = max(offset, 0)
}

// flip shows the next or previous screen of notes and moves the focus to
// its first note.
func (m *Model) flip(delta int) {
	to := max(min(m.screen+delta, m.screens-1), 0)
	if to == m.screen {
		return
	}
	m.screen = to
	for _, p := range m.panes {
		if p.screen == to {
			m.focus = p.name
			m.markSeen(p.name)
			break
		}
	}
}
