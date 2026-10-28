package bridge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/LeeSwallow/stickypane/internal/arrange"
	"github.com/LeeSwallow/stickypane/internal/i18n"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/theme"
	"github.com/LeeSwallow/stickypane/internal/widget"
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
	Name    string          `json:"name"`
	Tab     string          `json:"tab"`
	Title   string          `json:"title"`
	Kind    string          `json:"kind"`
	Icon    string          `json:"icon"`
	Open    bool            `json:"open"`
	Size    string          `json:"size"`
	Rows    int             `json:"rows"`
	Color   string          `json:"color"`
	Pinned  bool            `json:"pinned"`
	Version string          `json:"version"`
	Summary string          `json:"summary"`
	Data    json.RawMessage `json:"data"`
	ModTime time.Time       `json:"modTime"`
	Error   string          `json:"error,omitempty"` // why the file cannot be shown
	Pages   []Note          `json:"pages,omitempty"`
}

// Settings are the board's settings as a screen needs them.
type Settings struct {
	Theme    string   `json:"theme"` // the choice: a theme's name, or "auto"
	Style    string   `json:"style"`
	View     string   `json:"view"`
	Language Language `json:"language"`
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
	out := Board{Title: filepath.Base(s.project()), Settings: s.settings(), Tabs: []Tab{}}
	views := b.Settings.Views()
	for _, t := range b.Tabs {
		tab := Tab{Name: t.Name, Title: t.Title, Notes: []Note{}}
		if t.Name == "" && t.Title != "" {
			out.Title = t.Title
		}
		for _, n := range t.Notes {
			tab.Notes = append(tab.Notes, s.note(t.Name, n, views))
		}
		out.Tabs = append(out.Tabs, tab)
	}
	out.Palette = palette(out.Settings.Theme)
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
		Kind: "book", Icon: "▤", Open: open, Size: widget.SizePage,
		Color: color(n.Name, arrange.Color(v, n.Doc)), Pinned: arrange.Pinned(v, n.Doc),
		ModTime: n.ModTime, Data: json.RawMessage("null"), Pages: []Note{},
	}
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
		Rows:    arrange.Rows(v, owner.Doc, k),
		Color:   color(owner.Name, arrange.Color(v, owner.Doc)),
		Pinned:  arrange.Pinned(v, owner.Doc),
		ModTime: n.ModTime,
		Data:    json.RawMessage("null"),
	}
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
	out.Summary = k.Parse(n.Doc).Summary()
	if k.Data != nil {
		out.Data = data(n.Name, k.Data(n.Doc))
	}
	return out
}

// data encodes what a kind's Data says. A document does not know its
// file's name, so where the value says which language it runs in ("lang"),
// the file's extension decides: "ps1" for a .ps1 file, "sh" otherwise.
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
	lang := `"sh"`
	if strings.EqualFold(path.Ext(name), ".ps1") {
		lang = `"ps1"`
	}
	fields["lang"] = json.RawMessage(lang)
	if out, err := json.Marshal(fields); err == nil {
		return out
	}
	return raw
}

// color is the name of a note's color: the one chosen, else one the name
// picks, the way the terminal picks it, so both screens agree.
func color(name, chosen string) string {
	for _, c := range theme.NoteNames {
		if strings.EqualFold(c, chosen) {
			return c
		}
	}
	h := fnv.New32a()
	h.Write([]byte(name))
	return theme.NoteNames[h.Sum32()%uint32(len(theme.NoteNames))]
}

// settings reads the root settings.
func (s *server) settings() Settings {
	st := s.api.Store()
	th := st.Theme()
	if th == "" {
		th = "auto"
	}
	lang := strings.ToLower(st.Language())
	if lang == "" {
		lang = "auto"
	}
	return Settings{Theme: th, Style: st.Style(), View: st.ViewMode(), Language: Language{lang, resolve(lang)}}
}

// resolve is the language a choice comes to: the choice when there is a
// translation for it, else what the environment asks for when there is
// one for that, else English.
func resolve(choice string) string {
	for _, c := range []string{choice, i18n.Detect()} {
		if c == "en" {
			return c
		}
		if _, err := i18n.Load(c); err == nil && c != "" && c != "auto" {
			return c
		}
	}
	return "en"
}

// palette is the colors of the chosen theme. The bridge has no terminal
// to ask whether its background is dark, so "auto" is the dark default.
func palette(choice string) Palette {
	t := theme.Pick(choice, true)
	return Palette{
		Name: t.Name, Dark: t.Dark, Base: t.Base, Text: t.Text, Muted: t.Muted,
		Accent: t.Accent, Select: t.Select, Good: t.Good, Warn: t.Warn, Bad: t.Bad,
		Info: t.Info, Notes: t.Notes,
	}
}
