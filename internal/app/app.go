// Package app is the terminal UI: it lists every note in a title bar, draws
// the open ones in full, and connects keys to widgets and to the store.
package app

import (
	"reflect"
	"time"

	"github.com/LeeSwallow/stickypane/internal/env"
	"github.com/LeeSwallow/stickypane/internal/lineedit"
	"github.com/LeeSwallow/stickypane/internal/when"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/editor"
	"github.com/LeeSwallow/stickypane/internal/i18n"
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

// tr is what the screen says, in the language in use. One board runs per
// process, so the strings are the package's.
var tr = i18n.English()

// Message returns a message of the screen's language by its field name,
// for callers outside the package. An unknown name gives "".
func Message(field string) string {
	v := reflect.ValueOf(tr).FieldByName(field)
	if v.IsValid() && v.Kind() == reflect.String {
		return v.String()
	}
	return ""
}

// say fills a message's placeholders.
func say(s string, values map[string]any) string { return i18n.Fill(s, values) }

// UseLanguage makes the screen speak the language of a choice ("ko",
// "auto") from now on. Widgets are told too.
func UseLanguage(choice string) {
	tr = i18n.Pick(choice)
	widget.Translate = tr.L
}

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

	tabList  []store.Tab       // the tabs of the board: the root and each folder
	tab      int               // the active tab
	tabName  string            // its folder name, "" for the root
	tabFocus map[string]string // the focused note of each tab, by tab name
	tabHits  []hit             // where each tab is on the first line of the screen

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

	input      *lineedit.Line // input.go
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

	language string // the language choice in sticky.json the screen speaks

	drawn map[string]drawn // panes.go: what each note drew last, by name

	wakeAt time.Time // wake.go: when the screen next redraws by itself

	confirmYes, confirmNo [2]int // manage.go: the cells of a question's buttons
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
	m := &Model{store: st, reg: reg, watch: watch, now: time.Now, peek: map[string]bool{}, offsets: map[string]int{}, anchors: map[string]anchor{}, tabFocus: map[string]string{}, reveal: revealNote, theme: th, dark: true}
	widget.Now = func() time.Time { return m.now() }
	m.language = st.Language()
	UseLanguage(m.language)
	// Times are written the way the user's country does, whatever language
	// the screen speaks.
	when.UseLocale(env.Detect().Locale)
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
	m.drawn = nil
}

// chooseTheme makes the chosen theme the one in use, remembers the choice
// in sticky.json and says which theme that is.
func (m *Model) chooseTheme(name string) {
	if err := m.store.SetTheme(name); err != nil {
		m.status = say(tr.ThemeNotSaved, map[string]any{"Err": err.Error()})
	}
	m.useTheme(theme.Pick(name, m.dark))
	m.reload()
	m.status = say(tr.ThemeChosen, map[string]any{"Name": m.theme.Get().Name})
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
	case sentMsg:
		m.sent(msg)
	case wakeMsg:
		// Time passed to a moment a note changes by itself: draw again.
		m.drawn, m.wakeAt = nil, time.Time{}
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
	return m, tea.Batch(cmd, m.wake())
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
	m.tabFocus[m.tabName] = name
	m.reveal = revealNote
	m.markSeen(name)
}

// switchTab shows another tab. The choice is written to the root
// sticky.json, so it survives a restart and an agent can read it.
func (m *Model) switchTab(i int) {
	if i < 0 || i >= len(m.tabList) || i == m.tab {
		return
	}
	if err := m.store.SetTab(m.tabList[i].Name); err != nil {
		m.status = say(tr.ArrangementNotSaved, map[string]any{"Err": err.Error()})
		return
	}
	m.focus = ""
	m.reload()
	m.reveal = revealNote
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

// minPane is the least a note gets on the screen, borders included, before
// the notes after it go to the next screen.
const minPane = 10
