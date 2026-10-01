// Package app is the terminal UI: it shows the notes of a store as a board
// and connects keys to widgets and to the store.
package app

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/layout"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

type mode int

const (
	modeBoard   mode = iota // the board with every note
	modeModal               // one note opened
	modeInput               // a one-line prompt over the previous screen
	modeCatalog             // choosing a shape for a new note
	modeConfirm             // a yes/no question over the previous screen
	modeHelp                // the key reference
)

// Screens register themselves in these tables from their own files, so
// adding a screen or a board key never means editing this file.
var (
	// handlers receive every message that is not handled globally.
	handlers = map[mode]func(*Model, tea.Msg) tea.Cmd{}
	// bodies draw everything above the bottom line, in at most the given
	// number of lines. A mode without a body shows the one it came from.
	bodies = map[mode]func(*Model, int) []string{}
	// footers draw the bottom line.
	footers = map[mode]func(*Model) string{}
	// boardKeys are the keys of the board screen.
	boardKeys = map[string]func(*Model) tea.Cmd{}
)

// item is a note on the board with the widget that draws it.
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

	width, height int
	rects         []layout.Rect
	canvas        []string
	scroll        int

	mode   mode
	back   mode   // where input, confirm and help return to
	status string // shown on the bottom line until the next key

	modalName string // modal.go: file name of the open note

	input      textinput.Model // input.go
	inputLabel string
	onSubmit   func(string)

	catalogIdx int // create.go

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
// folder cannot be watched; the board then refreshes on "r" and after edits.
func New(st *store.Store, reg widget.Registry, watch <-chan struct{}) *Model {
	m := &Model{store: st, reg: reg, watch: watch, now: time.Now}
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
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		m.status = ""
		cmd = m.dispatch(msg)
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

// reload rescans the folder. Widgets of notes whose body did not change are
// kept, and changed ones are synced, so cursors and scroll positions survive.
func (m *Model) reload() {
	notes, err := m.store.Scan()
	if err != nil {
		m.status = "Cannot read notes: " + err.Error()
		return
	}
	old := make(map[string]item, len(m.items))
	for _, it := range m.items {
		old[it.note.Name] = it
	}
	prev := m.index(m.focus)

	items := make([]item, 0, len(notes))
	for _, n := range notes {
		if n.Err != nil {
			n.Doc = doc.Document{Body: "Cannot show this note: " + n.Err.Error()}
		}
		it := item{note: n, kind: m.reg.Lookup(n.Doc.Type())}
		switch o, ok := old[n.Name]; {
		case !ok || o.kind.Name != it.kind.Name:
			it.w = it.kind.Parse(n.Doc)
		case o.note.Doc.Body == n.Doc.Body:
			it.w = o.w
		default:
			it.w = o.w.Sync(n.Doc)
		}
		items = append(items, it)
	}
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
	}
	m.markSeen(m.focus)
	if m.mode == modeModal && m.index(m.modalName) < 0 {
		m.mode = modeBoard
		m.status = "The note was removed."
	}
}

// apply writes an intent to a note and refreshes the board. On a conflict
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
	m.markSeen(name)
}

// label names a note in prompts: its title, or its file name without ".md".
func label(it item) string {
	if t, _ := it.note.Doc.Get("title"); t != "" {
		return widget.Clean(t)
	}
	return strings.TrimSuffix(it.note.Name, filepath.Ext(it.note.Name))
}

// heading is the text in a note's top border. A plain note without a title
// has none, like a sticky note; other shapes fall back to the file name.
func (m *Model) heading(it item) string {
	t, _ := it.note.Doc.Get("title")
	t = widget.Clean(t)
	if t == "" && (it.kind.Name != m.reg[0].Name || it.note.Err != nil) {
		t = label(it)
	}
	if m.changed(it) {
		t = strings.TrimSpace("● " + t)
	}
	if it.note.Doc.Pinned() {
		t = strings.TrimSpace("📌 " + t)
	}
	return t
}

// relayout redraws every note and places it. It runs after every update, so
// the positions used for moving the focus always match the screen.
func (m *Model) relayout() {
	if m.width <= 0 {
		m.rects, m.canvas = nil, nil
		return
	}
	n, cw := layout.Columns(m.width)
	boxes := make([]string, len(m.items))
	flow := make([]layout.Item, len(m.items))
	for i, it := range m.items {
		w := cw
		if it.kind.FullRow {
			w = n * cw
		}
		boxes[i] = frame(m.heading(it), it.w.Preview(max(w-4, 1)), w, noteColor(it), it.note.Name == m.focus)
		flow[i] = layout.Item{Height: strings.Count(boxes[i], "\n") + 1, FullRow: it.kind.FullRow}
	}
	m.rects = layout.Flow(m.width, flow)
	m.canvas = layout.Compose(m.rects, boxes)

	h := m.height - 1
	if i := m.index(m.focus); i >= 0 && h > 0 {
		r := m.rects[i]
		if r.Y+r.H > m.scroll+h {
			m.scroll = r.Y + r.H - h
		}
		if r.Y < m.scroll {
			m.scroll = r.Y // also wins when the note is taller than the screen
		}
	}
	m.scroll = widget.ClampOffset(m.scroll, len(m.canvas), h)
}
