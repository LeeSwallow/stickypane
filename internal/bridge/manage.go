package bridge

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/LeeSwallow/stickypane/internal/theme"
	"strconv"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/api"
	"github.com/LeeSwallow/stickypane/internal/arrange"
	"github.com/LeeSwallow/stickypane/internal/doc"
	environment "github.com/LeeSwallow/stickypane/internal/env"
	"github.com/LeeSwallow/stickypane/internal/i18n"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// derive is called after a request changed the notes: a chart computed
// from them follows, as it does after every change the terminal makes.
func (s *server) derive() { _, _ = s.api.Derive() }

// noteArrange arranges a note on the screen as `stickypane set` does:
// open, size, rows, pin and color go to sticky.json, and so does the title
// of a note that has no front matter; the title of a Markdown note goes to
// its front matter. An empty size, color or title, and rows 0, give the
// choice back. A page is arranged by its book, but has its own title.
func (s *server) noteArrange(params json.RawMessage) (any, error) {
	var p arrangeParams
	if err := decode(params, &p); err != nil {
		return nil, err
	}
	var view []string
	add := func(key, value string) { view = append(view, key+"="+value) }
	if p.Open != nil {
		add("open", strconv.FormatBool(*p.Open))
	}
	if p.Pin != nil {
		add("pin", strconv.FormatBool(*p.Pin))
	}
	if p.Size != nil {
		add("size", *p.Size)
	}
	if p.Rows != nil {
		if *p.Rows < 0 {
			return nil, badRequest("rows cannot be negative")
		}
		rows := ""
		if *p.Rows > 0 {
			rows = strconv.Itoa(*p.Rows)
		}
		add("rows", rows)
	}
	if p.Color != nil {
		add("color", *p.Color)
	}
	for _, kv := range view {
		key, value, _ := strings.Cut(kv, "=")
		if _, _, err := arrange.Change(key, value); err != nil {
			return nil, badRequest(err.Error())
		}
	}
	var title []string
	if p.Title != nil {
		if strings.ContainsAny(*p.Title, "\r\n") {
			return nil, badRequest("a title is one line")
		}
		title = []string{"title=" + *p.Title}
	}
	if len(view)+len(title) == 0 {
		return nil, badRequest("note.arrange needs one of open, size, rows, pin, color or title")
	}
	n, owner, err := s.find(p.Note)
	if err != nil {
		return nil, err
	}
	// A page is arranged by its book, by the book's folder name, but has
	// its own title.
	switch {
	case owner.Name == n.Name:
		_, err = s.api.Set(n.Name, append(view, title...))
	default:
		if len(view) > 0 {
			_, err = s.api.Set(owner.Name, view)
		}
		if err == nil && len(title) > 0 {
			_, err = s.api.Set(n.Name, title)
		}
	}
	if err != nil {
		return nil, err
	}
	s.derive()
	return empty{}, nil
}

// tabOrder puts a tab's notes in the order the screen drew them, as the
// terminal does when it swaps a note with its neighbour: by full name.
// A name left out follows the others by name.
func (s *server) tabOrder(params json.RawMessage) (any, error) {
	var p orderParams
	if err := decode(params, &p); err != nil {
		return nil, err
	}
	if len(p.Order) == 0 {
		return nil, badRequest("params.order is empty")
	}
	if err := s.api.Order(p.Tab, p.Order); errors.Is(err, api.ErrBadOrder) {
		return nil, badRequest(err.Error())
	} else if err != nil {
		return nil, err
	}
	return empty{}, nil
}

// noteMove moves a note into a folder or a book ("docs/"), out to the top
// level (".") or under a new name; it answers the name the note now has.
func (s *server) noteMove(params json.RawMessage) (any, error) {
	var p moveParams
	if err := decode(params, &p); err != nil {
		return nil, err
	}
	if strings.TrimSpace(p.To) == "" {
		return nil, badRequest("say where to move the note: a folder, a new name, or . for the top level")
	}
	n, _, err := s.find(p.Note)
	if err != nil {
		return nil, err
	}
	moved, err := s.api.Move(n.Name, p.To) // a book is moved by its folder's name
	if err != nil {
		return nil, err
	}
	return map[string]string{"note": moved.To}, nil
}

// noteArchive moves a note, a page or a book out of sight into the archive.
func (s *server) noteArchive(params json.RawMessage) (any, error) {
	return s.manage(params, s.api.Archive)
}

// noteTrash moves a note, a page or a book to the trash.
func (s *server) noteTrash(params json.RawMessage) (any, error) {
	return s.manage(params, s.api.Remove)
}

// manage does to a note on the board what api does to it, a book by its
// folder's name as the terminal does.
func (s *server) manage(params json.RawMessage, do func(name string) (api.Moved, error)) (any, error) {
	var p noteParams
	if err := decode(params, &p); err != nil {
		return nil, err
	}
	n, _, err := s.find(p.Note)
	if err != nil {
		return nil, err
	}
	if _, err := do(n.Name); err != nil {
		return nil, err
	}
	return empty{}, nil
}

// noteRestore brings back the note of that name that was trashed last. The
// note is not on the board, so its name is checked as a name only.
func (s *server) noteRestore(params json.RawMessage) (any, error) {
	var p noteParams
	if err := decode(params, &p); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(p.Note)
	if err := checkName(name); err != nil {
		return nil, err
	}
	// A book was trashed as a folder, whatever dots its name has; api
	// takes it by that name.
	if _, err := s.api.Restore(name); err != nil {
		return nil, err
	}
	return empty{}, nil
}

// sendResult is what note.send answers: how the request ended, and the log
// its answer is in.
type sendResult struct {
	Status int    `json:"status"` // the HTTP status, 0 when none came
	Text   string `json:"text"`   // "200 OK"
	Took   int64  `json:"took"`   // milliseconds
	Failed bool   `json:"failed"` // not sent, an error status, a hook or a check that failed
	End    string `json:"end"`    // the last line of its log, as the terminal says it
	Log    string `json:"log"`    // the note that holds the log
}

// noteSend sends one request of a .http note and waits for its answer, as
// the terminal does after the user said yes. The log is written by the
// same function (api.SendPart); the app reads it from the board.
func (s *server) noteSend(params json.RawMessage) (any, error) {
	var p sendParams
	if err := decode(params, &p); err != nil {
		return nil, err
	}
	want := ""
	var number int
	switch {
	case json.Unmarshal(p.Request, &number) == nil && len(p.Request) > 0:
		want = "#" + strconv.Itoa(number)
	case json.Unmarshal(p.Request, &want) == nil && strings.TrimSpace(want) != "":
	default:
		return nil, badRequest("which request? params.request is its title, or its number from 1")
	}
	n, _, err := s.find(p.Note)
	if err != nil {
		return nil, err
	}
	if k := s.reg.For(n.Name, n.Doc); n.Book() || k.Name != "api" {
		return nil, badRequest(fmt.Sprintf("%s has no requests to send", p.Note))
	}
	part, err := s.api.Request(n.Name, want)
	if err != nil {
		return nil, badRequest(err.Error())
	}
	res, err := s.api.SendPart(s.ctx, n.Name, part, "")
	if err != nil {
		return nil, err
	}
	return sendResult{
		Status: res.Status, Text: res.Text, Took: res.Took.Milliseconds(), Failed: res.Failed,
		End: res.End, Log: res.Log,
	}, nil
}

// kindInfo is one shape of note as the catalog shows it: what it is, and
// an example whole file with what that example holds, drawn from Data like
// any note of the shape.
type kindInfo struct {
	Name    string          `json:"name"`
	Icon    string          `json:"icon"`
	Label   string          `json:"label"`
	Blurb   string          `json:"blurb"`
	Example string          `json:"example"`
	Data    json.RawMessage `json:"data"`
	// Options are what note.create takes for this shape, so a screen
	// draws its create form from them: [] when it takes none.
	Options []widget.Option `json:"options"`
}

// kindsList is the catalog: every shape, in the registry's order, with its
// example in a language: the one asked for ({"language": "ko"}, the
// language the screen draws in), else the board's.
func (s *server) kindsList(params json.RawMessage) (any, error) {
	var p kindsParams
	if err := decode(params, &p); err != nil {
		return nil, err
	}
	choice := p.Language
	if choice == "" {
		choice = s.api.Store().Settings().Language()
	}
	words := i18n.Pick(choice, environment.Detect().Locale)
	env := s.api.Env()
	env.Translate = words.L
	out := []kindInfo{}
	for _, k := range s.reg {
		example := k.ExampleFileIn(words)
		info := kindInfo{Name: k.Name, Icon: k.Icon, Label: k.Label, Blurb: k.Blurb, Example: example, Data: json.RawMessage("null"), Options: []widget.Option{}}
		info.Options = append(info.Options, k.Options...)
		if k.Data != nil {
			info.Data = data("example"+k.New, k.Data(doc.Parse([]byte(example)), env))
		}
		out = append(out, info)
	}
	return out, nil
}

// themesList is every theme's palette, dark ones first, as theme.All
// orders them: a screen lists them to choose from, and draws the other
// half of a dark and light pair without a copy of its own.
func (s *server) themesList(json.RawMessage) (any, error) {
	out := []Palette{}
	for _, t := range theme.All() {
		out = append(out, paletteOf(t))
	}
	return map[string]any{"themes": out}, nil
}

// boardIndex is api.Index: every note in a line, with the notes it links
// to and the notes that link to it.
func (s *server) boardIndex(json.RawMessage) (any, error) {
	entries, err := s.api.Index()
	if err != nil {
		return nil, err
	}
	if entries == nil {
		entries = []api.Entry{}
	}
	return map[string]any{"entries": entries}, nil
}
