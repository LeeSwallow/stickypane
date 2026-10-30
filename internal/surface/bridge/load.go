package bridge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/LeeSwallow/stickypane/internal/adapter/env"
	"github.com/LeeSwallow/stickypane/internal/adapter/store"
	"github.com/LeeSwallow/stickypane/internal/core/i18n"
	"github.com/LeeSwallow/stickypane/internal/core/theme"
	"github.com/LeeSwallow/stickypane/internal/core/widget"
	"github.com/LeeSwallow/stickypane/internal/usecase/arrange"
	"github.com/LeeSwallow/stickypane/internal/usecase/prefs"
)

// Board is what board.load answers: the tabs with their notes, the
// settings and the colors to draw with.
type Board struct {
	Title    string   `json:"title"`
	Tabs     []Tab    `json:"tabs"`
	Settings Settings `json:"settings"`
	Palette  Palette  `json:"palette"`
}

// Tab is one tab of the board; Name is "" for the root.
type Tab struct {
	Name  string `json:"name"`
	Title string `json:"title"`
	Notes []Note `json:"notes"`
}

// Note is one note as a screen draws it: how it is arranged, what it
// holds (its kind's Data) and its version, which an action sends back. A
// book has no file, so no version or data of its own; its pages are notes
// of the same shape.
type Note struct {
	Name  string `json:"name"`
	Tab   string `json:"tab"`
	Title string `json:"title"`
	Kind  string `json:"kind"`
	Icon  string `json:"icon"`
	Open  bool   `json:"open"`
	// Size is a fraction of the screen ("1/4" … "1") or "card" since #14
	// (unreleased): no longer "half" or "page", which are read as "1/2"
	// and "1". Sized is whether the size was chosen for the note, not its
	// kind's default: a sized note keeps its width, one without a size
	// takes the room its row leaves over (internal/core/layout.Rows).
	// Rows is 0 for a note as tall as its content.
	Size    string          `json:"size"`
	Sized   bool            `json:"sized"`
	Rows    int             `json:"rows"`
	Color   string          `json:"color"`
	Pinned  bool            `json:"pinned"`
	Version string          `json:"version"`
	Summary string          `json:"summary"`
	Data    json.RawMessage `json:"data"`
	ModTime time.Time       `json:"modTime"`
	Linked  bool            `json:"linked"`           // a symbolic link to something outside the notes folder
	Target  string          `json:"target,omitempty"` // where the link leads
	Error   string          `json:"error,omitempty"`  // why the file cannot be shown
	Pages   []Note          `json:"pages,omitempty"`
}

// Settings are the board's settings as a screen needs them.
type Settings struct {
	Theme    string   `json:"theme"` // the choice: a theme's name, or "auto"
	Style    string   `json:"style"`
	View     string   `json:"view"`
	Language Language `json:"language"`
	Tab      string   `json:"tab"`    // the tab the terminal is on, "" for the root
	Time     string   `json:"time"`   // how times are written: auto, ko, en-US...
	Editor   string   `json:"editor"` // what the terminal's E opens: auto or an editor
	Stale    string   `json:"stale"`  // when a card that has not moved is flagged
}

// Language is the language the user chose ("auto", "ko", "en") and the
// one that choice comes to here.
type Language struct {
	Choice   string `json:"choice"`
	Resolved string `json:"resolved"`
}

// Palette is the chosen theme's colors, by role, and the six note colors
// in the order of theme.NoteNames.
type Palette struct {
	Name   string    `json:"name"`
	Dark   bool      `json:"dark"`
	Base   string    `json:"base"`
	Text   string    `json:"text"`
	Muted  string    `json:"muted"`
	Accent string    `json:"accent"`
	Select string    `json:"select"`
	Good   string    `json:"good"`
	Warn   string    `json:"warn"`
	Bad    string    `json:"bad"`
	Info   string    `json:"info"`
	Notes  [6]string `json:"notes"`
}

// load reads the whole board for a screen.
func (s *server) load(b store.Board) Board {
	out := Board{Title: filepath.Base(s.project()), Settings: settingsOf(b.Settings), Tabs: []Tab{}}
	views := b.Settings.Views()
	for _, t := range b.Tabs {
		tab := Tab{Name: t.Name, Title: t.Title, Notes: []Note{}}
		if t.Name == "" && t.Title != "" {
			out.Title = t.Title
		}
		for _, n := range inOrder(t, b.Settings, views) {
			tab.Notes = append(tab.Notes, s.note(t.Name, n, views))
		}
		out.Tabs = append(out.Tabs, tab)
	}
	out.Palette = palette(out.Settings.Theme)
	return out
}

// inOrder is a tab's notes in the order the terminal draws them: the
// pinned ones first, then those the user put in an order of their own
// (tab.order) as they put them, then the rest by name.
func inOrder(t store.Tab, set store.Settings, views store.Views) []store.Note {
	place := map[string]int{}
	for i, name := range set.Order(t.Name) {
		place[name] = i + 1
	}
	rank := func(n store.Note) int {
		if p, ok := place[n.Name]; ok {
			return p
		}
		return len(place) + 1
	}
	out := slices.Clone(t.Notes)
	sort.SliceStable(out, func(i, j int) bool {
		if pi, pj := arrange.Pinned(views[out[i].Name], out[i].Doc), arrange.Pinned(views[out[j].Name], out[j].Doc); pi != pj {
			return pi
		}
		return rank(out[i]) < rank(out[j])
	})
	return out
}

// note describes a note, or a book and its pages. Where a page is on the
// screen is its book's to say, as on the terminal.
func (s *server) note(tab string, n store.Note, views store.Views) Note {
	v := views[n.Name]
	if !n.Book() {
		return s.file(tab, n, n, v)
	}
	open, _ := arrange.Open(v, n.Doc)
	out := Note{
		Name: n.Name, Tab: tab, Title: arrange.Title(v, n.Doc, n.Name),
		Kind: "book", Icon: "▤", Open: open, Size: widget.SizeFull,
		Color: arrange.ColorName(n.Name, arrange.Color(v, n.Doc)), Pinned: arrange.Pinned(v, n.Doc),
		ModTime: n.ModTime, Data: json.RawMessage("null"), Pages: []Note{},
	}
	out.Linked, out.Target = s.link(n.Name)
	for _, p := range n.Pages {
		out.Pages = append(out.Pages, s.file(tab, p, n, v))
		if p.ModTime.After(out.ModTime) {
			out.ModTime = p.ModTime
		}
	}
	if len(n.Pages) > 0 {
		k := s.reg.For(n.Pages[0].Name, n.Pages[0].Doc)
		out.Size = arrange.Size(v, n.Doc, k, n.Pages[0].Doc)
		out.Rows = arrange.Rows(v, n.Doc, k)
		out.Sized = arrange.Sized(v, n.Doc)
	}
	out.Summary = fmt.Sprintf("%d pages", len(n.Pages)) // as the index says
	return out
}

// file describes one file: a note, or a page of the book owner (which is
// the note itself when it is not a page), arranged by the view v.
func (s *server) file(tab string, n, owner store.Note, v store.View) Note {
	k := s.reg.For(n.Name, n.Doc)
	open, _ := arrange.Open(v, owner.Doc)
	out := Note{
		Name: n.Name, Tab: tab, Kind: k.Name, Icon: k.Icon, Open: open,
		Size:    arrange.Size(v, owner.Doc, k, n.Doc),
		Sized:   arrange.Sized(v, owner.Doc),
		Rows:    arrange.Rows(v, owner.Doc, k),
		Color:   arrange.ColorName(owner.Name, arrange.Color(v, owner.Doc)),
		Pinned:  arrange.Pinned(v, owner.Doc),
		ModTime: n.ModTime,
		Data:    json.RawMessage("null"),
	}
	out.Linked, out.Target = s.link(n.Name)
	if owner.Name == n.Name {
		out.Title = arrange.Title(v, n.Doc, n.Name)
	} else {
		out.Title = arrange.Title(store.View{}, n.Doc, n.Name)
	}
	if n.Err != nil {
		out.Error = n.Err.Error()
		return out
	}
	out.Version = widget.Version(n.Doc)
	out.Summary = k.Parse(n.Doc, s.api.Env()).Summary()
	if k.Data != nil {
		out.Data = data(n.Name, k.Data(n.Doc, s.api.Env()))
	}
	return out
}

// link says whether a note is a symbolic link, and to what.
func (s *server) link(name string) (bool, string) {
	st := s.api.Store() // read only: whether the file is a link, and where it is
	if !st.Linked(name) {
		return false, ""
	}
	full := filepath.Join(st.Dir, filepath.FromSlash(name))
	if target, err := filepath.EvalSymlinks(full); err == nil {
		return true, target
	}
	if target, err := os.Readlink(full); err == nil { // a link to nothing
		return true, target
	}
	return true, ""
}

// data encodes what a kind's Data says. A document does not know its
// file's name, so where the value says which language it runs in ("lang"),
// one rule decides, the runner's: the extension when it names one (".sh"
// is "sh", ".ps1" is "ps1"), else the value's own guess from the shebang.
func data(name string, v any) json.RawMessage {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return json.RawMessage("null")
	}
	raw := json.RawMessage(bytes.TrimSpace(b.Bytes()))
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || fields["lang"] == nil {
		return raw
	}
	switch strings.ToLower(path.Ext(name)) {
	case ".sh":
		fields["lang"] = json.RawMessage(`"sh"`)
	case ".ps1":
		fields["lang"] = json.RawMessage(`"ps1"`)
	default:
		return raw
	}
	if out, err := json.Marshal(fields); err == nil {
		return out
	}
	return raw
}

// settings reads the root settings.
func (s *server) settings() Settings { return settingsOf(s.api.Settings()) }

// settingsOf is what the settings files say, with the defaults.
func settingsOf(set store.Settings) Settings {
	th := set.Theme()
	if th == "" {
		th = "auto"
	}
	lang := strings.ToLower(set.Language())
	if lang == "" {
		lang = "auto"
	}
	return Settings{
		Theme: th, Style: set.Style(), View: set.ViewMode(),
		Language: Language{lang, i18n.Resolve(lang, env.Detect().Locale)},
		Tab:      set.Tab(),
		Time:     prefs.Get(set, "time"),
		Editor:   prefs.Get(set, "editor"),
		Stale:    prefs.Get(set, "stale"),
	}
}

// palette is the colors of the chosen theme. The bridge has no terminal
// to ask whether its background is dark, so "auto" is the dark default.
func palette(choice string) Palette { return paletteOf(theme.Pick(choice, true)) }

// paletteOf is a theme's colors in the shape a screen reads.
func paletteOf(t theme.Theme) Palette {
	return Palette{
		Name: t.Name, Dark: t.Dark, Base: t.Base, Text: t.Text, Muted: t.Muted,
		Accent: t.Accent, Select: t.Select, Good: t.Good, Warn: t.Warn, Bad: t.Bad,
		Info: t.Info, Notes: t.Notes,
	}
}
