// Package app is the terminal UI: it lists every note in a title bar, draws
// the open ones in full, and connects keys to widgets and to the store.
package app

import (
	"errors"
	"path/filepath"
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

// item is a note with the widget that draws it.
type item struct {
	note store.Note
	kind widget.Kind
	w    widget.Widget
}

// Model is the Bubble Tea model.
type Model struct {
	store *store.Store
	reg   widget.Registry
	watch <-chan struct{}
	now   func() time.Time

	// OnBackground, when set, is told whether the terminal background is
	// dark once the terminal reports it. Every note is redrawn afterwards.
	OnBackground func(dark bool)

	items []item
	focus string               // file name of the focused note
	seen  map[string]time.Time // modification time last looked at, by file name
	known map[string]bool      // notes that existed at the previous scan
	peek  map[string]bool      // notes shown open for this run without an "open" key

	width, height int
	bar           []string      // the title bar, one or more lines
	tabs          []tab         // mouse.go: where each note's title is in the bar
	rects         []layout.Rect // of the open notes
	placed        []placed      // mouse.go: which note is in each rect
	lastClick     lastClick     // mouse.go
	canvas        []string      // the open notes, laid out
	scroll        int
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
}

// changedMsg arrives when the watched folder changed.
type changedMsg struct{}

// reloadMsg asks for a rescan and, when err is set, reports it first.
type reloadMsg struct {
	what string
	err  error
}

// New returns a model showing the notes in st. watch may be nil when the
// folder cannot be watched; the screen then refreshes on "r" and after edits.
func New(st *store.Store, reg widget.Registry, watch <-chan struct{}) *Model {
	m := &Model{store: st, reg: reg, watch: watch, now: time.Now, peek: map[string]bool{}, reveal: revealNote}
	m.reload()
	return m
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
		if m.OnBackground != nil {
			m.OnBackground(msg.IsDark())
		}
		m.items = nil // rebuild every widget so it redraws with the new colors
		m.reload()
	case tea.KeyPressMsg:
		// In the editor ctrl+c leaves insert mode, as it does in vi:
		// quitting there would drop what was typed.
		if msg.String() == "ctrl+c" && m.mode != modeEdit {
			return m, tea.Quit
		}
		m.status = ""
		cmd = m.dispatch(msg)
	case tea.MouseClickMsg:
		m.click(msg.Mouse())
	case tea.MouseWheelMsg:
		m.wheel(msg.Mouse())
	default:
		cmd = m.dispatch(msg)
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

// isOpen reports whether a note is drawn on the main screen. The note's
// "open" key decides. A note without the key is closed, with two exceptions
// that last for this run and never touch the file: a note that appeared
// while stickypane was running, so that what an agent just wrote is seen at
// once, and a note that was open when a rewrite dropped its key, so that an
// agent rewriting a note does not make it vanish from the screen.
// A note that cannot be read is always open, to say why.
func (m *Model) isOpen(it item) bool {
	if it.note.Err != nil {
		return true
	}
	if v, ok := it.note.Doc.Get("open"); ok {
		return strings.EqualFold(v, "true")
	}
	return m.peek[it.note.Name]
}

// sizeOf returns the note's size: its "size" key, or the default of its kind.
func (m *Model) sizeOf(it item) string {
	v, _ := it.note.Doc.Get("size")
	switch v = strings.ToLower(v); v {
	case widget.SizePage, widget.SizeHalf, widget.SizeCard:
		return v
	}
	if it.kind.Size != nil {
		return it.kind.Size(it.note.Doc)
	}
	return widget.SizePage
}

// rowsOf returns the note's fixed height in lines, or 0 when it is as tall
// as its content: its "rows" key, or the default of its kind.
func (m *Model) rowsOf(it item) int {
	if v, ok := it.note.Doc.Get("rows"); ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return it.kind.Rows
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

// reload rescans the folder. Widgets of notes whose body did not change are
// kept, and changed ones are synced, so cursors survive.
func (m *Model) reload() {
	notes, err := m.store.Scan()
	if err != nil {
		m.status = "Cannot read notes: " + err.Error()
		return
	}
	old := make(map[string]item, len(m.items))
	wasOpen := make(map[string]bool, len(m.items))
	for _, it := range m.items {
		old[it.note.Name] = it
		wasOpen[it.note.Name] = m.isOpen(it)
	}
	prev := m.index(m.focus)
	first := m.known == nil

	known := make(map[string]bool, len(notes))
	items := make([]item, 0, len(notes))
	for _, n := range notes {
		if n.Err != nil {
			n.Doc = doc.Document{Body: "Cannot show this note: " + n.Err.Error()}
		}
		it := item{note: n, kind: m.reg.Lookup(n.Doc.Type())}
		switch o, ok := old[n.Name]; {
		case !ok || o.kind.Name != it.kind.Name:
			it.w = it.kind.Parse(n.Doc)
		case string(o.note.Doc.Bytes()) == string(n.Doc.Bytes()):
			// Front matter counts too: a form shows whether it was
			// submitted and a chart which view it has.
			it.w = o.w
		default:
			it.w = o.w.Sync(n.Doc)
		}
		items = append(items, it)
		known[n.Name] = true
		if _, says := n.Doc.Get("open"); !says && ((!first && !m.known[n.Name]) || wasOpen[n.Name]) {
			m.peek[n.Name] = true
		}
	}
	for name := range m.peek {
		if !known[name] {
			delete(m.peek, name)
		}
	}
	m.known = known
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].note.Doc.Pinned() && !items[j].note.Doc.Pinned()
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
	if m.zoomed() && m.index(m.zoomName) < 0 {
		m.mode = modeBoard
		m.status = "The note was removed."
	}
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

// apply writes an intent to a note and refreshes the screen. On a conflict
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
	if i := m.index(name); err != nil && i >= 0 {
		// The widget may already show the change that did not happen.
		// Rebuild it from the file so the screen never claims otherwise.
		m.items[i].w = m.items[i].w.Sync(m.items[i].note.Doc)
	}
	m.markSeen(name)
}

// toWidget sends a key to a note's widget and carries out what it asks for.
func (m *Model) toWidget(i int, key string) {
	name := m.items[i].note.Name
	w, res := m.items[i].w.Update(key)
	m.items[i].w = w
	m.reveal = revealCursor
	m.act(name, res)
}

// act carries out what a widget asked for.
func (m *Model) act(name string, res widget.Result) {
	if res.Op != nil {
		m.apply(name, res.Op)
	}
	if p := res.Prompt; p != nil {
		m.ask(p.Label, p.Initial, func(text string) {
			m.apply(name, p.Submit(text))
			m.reveal = revealCursor
		})
		m.inputEmpty = p.Empty
	}
}

// label names a note in the title bar and in prompts: its title, or its file
// name without ".md".
func label(it item) string {
	if t, _ := it.note.Doc.Get("title"); t != "" {
		return widget.Clean(t)
	}
	return widget.Clean(strings.TrimSuffix(it.note.Name, filepath.Ext(it.note.Name)))
}

// heading is the text in a note's top border. A plain note without a title
// has none, like a sticky note; other shapes fall back to the file name.
func (m *Model) heading(it item) string {
	t, _ := it.note.Doc.Get("title")
	if t == "" && (it.kind.Name != m.reg[0].Name || it.note.Err != nil) {
		return label(it)
	}
	return widget.Clean(t)
}

// lines draws a note for the main screen at width and returns its lines and
// the selection within them. A note with a fixed height shows a window of
// its content: around the selection if it has one, the end for a log, the
// start otherwise.
func (m *Model) lines(it item, width int, active bool) (shown []string, at widget.Span, start int) {
	out, at := it.w.Draw(width, active)
	lines := strings.Split(out, "\n")
	rows := m.rowsOf(it)
	if rows <= 0 || len(lines) <= rows {
		return lines, at, 0
	}
	switch {
	case at.Ok():
		start = show(at, at.Start-rows/2, rows)
	case it.kind.Tail:
		start = len(lines) - rows
	}
	start = widget.ClampOffset(start, len(lines), rows)
	if at.Ok() {
		at = widget.Span{Start: at.Start - start, End: min(at.End-start, rows)}
	}
	return lines[start : start+rows], at, start
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

// relayout redraws the title bar and every open note and places them. It
// runs after every update, then scrolls to whatever the update asked to see.
func (m *Model) relayout() {
	m.bar, m.rects, m.canvas, m.zoomLines, m.placed = nil, nil, nil, nil, nil
	if m.width <= 0 {
		return
	}
	m.bar = m.titleBar()

	var boxes []string
	var sizes []layout.Item
	focusRect, at := -1, widget.NoSpan
	for _, it := range m.items {
		if !m.isOpen(it) {
			continue
		}
		focused := it.note.Name == m.focus
		w := m.widthOf(m.sizeOf(it))
		lines, sel, start := m.lines(it, max(w-4, 1), focused)
		if focused {
			focusRect, at = len(boxes), sel
		}
		m.placed = append(m.placed, placed{name: it.note.Name, start: start})
		boxes = append(boxes, frame(box{
			title: m.heading(it), icon: it.kind.Icon, summary: it.w.Summary(),
			body: strings.Join(lines, "\n"), width: w, color: noteColor(it), focused: focused,
		}))
		sizes = append(sizes, layout.Item{W: max(w, 8), H: len(lines) + 2})
	}
	m.rects = layout.Shelf(m.width, sizes)
	m.canvas = layout.Compose(m.rects, boxes)

	h := m.bodyHeight()
	if focusRect >= 0 && h > 0 && m.reveal != revealNothing {
		r := m.rects[focusRect]
		target := widget.Span{Start: r.Y, End: r.Y + r.H}
		if m.reveal == revealCursor && at.Ok() {
			target = at.Shift(r.Y + 1) // the frame's top border is one line
		}
		m.scroll = show(target, m.scroll, h)
	}
	m.scroll = widget.ClampOffset(m.scroll, len(m.canvas), h)

	if m.zoomed() {
		m.layoutZoom()
	}
	m.reveal = revealNothing
}
