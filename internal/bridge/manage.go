package bridge

import (
	"encoding/json"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/api"
	"github.com/LeeSwallow/stickypane/internal/arrange"
	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/httpfile"
	"github.com/LeeSwallow/stickypane/internal/store"
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
	switch {
	case owner.Book():
		// api.Set names a book by a folder of the notes folder itself; a
		// book is also in a tab, so its view is written here, with the
		// same changes Set makes.
		if owner.Name == n.Name {
			view = append(view, title...)
		}
		err = s.setView(owner.Name, view)
		if err == nil && owner.Name != n.Name && len(title) > 0 {
			_, err = s.api.Set(n.Name, title)
		}
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

// setView writes "key=value" arrangements of a note that api.Set cannot
// name, as Set does: a title is the note's name in sticky.json.
func (s *server) setView(name string, pairs []string) error {
	var changes []func(*store.View)
	for _, kv := range pairs {
		key, value, _ := strings.Cut(kv, "=")
		if key == "title" {
			value = strings.TrimSpace(value)
			changes = append(changes, func(v *store.View) { v.Title = value })
			continue
		}
		change, _, err := arrange.Change(key, strings.TrimSpace(value))
		if err != nil {
			return badRequest(err.Error())
		}
		changes = append(changes, change)
	}
	return s.api.Store().SetView(name, func(v *store.View) {
		for _, change := range changes {
			change(v)
		}
	})
}

// tabOrder puts a tab's notes in the order the screen drew them, as the
// terminal does when it swaps a note with its neighbour: by full name.
// A name left out follows the others by name.
func (s *server) tabOrder(params json.RawMessage) (any, error) {
	var p orderParams
	if err := decode(params, &p); err != nil {
		return nil, err
	}
	tabs, err := s.api.Store().Tabs()
	if err != nil {
		return nil, err
	}
	var here map[string]bool
	for _, t := range tabs {
		if t.Name == p.Tab {
			here = map[string]bool{}
			for _, n := range t.Notes {
				here[n.Name] = true
			}
		}
	}
	if here == nil {
		return nil, notFound(fmt.Sprintf("there is no tab %q", p.Tab))
	}
	if len(p.Order) == 0 {
		return nil, badRequest("params.order is empty")
	}
	seen := map[string]bool{}
	for _, name := range p.Order {
		switch {
		case !here[name]:
			return nil, badRequest(fmt.Sprintf("%q is not a note of tab %q: use the names board.load gave", name, p.Tab))
		case seen[name]:
			return nil, badRequest(fmt.Sprintf("%q is twice in the order", name))
		}
		seen[name] = true
	}
	if err := s.api.Store().SetOrder(p.Tab, p.Order); err != nil {
		return nil, err
	}
	s.derive()
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
	dest, err := s.api.MoveTo(n.Name, p.To) // a book is moved by its folder's name
	if err != nil {
		return nil, err
	}
	s.derive()
	return map[string]string{"note": dest}, nil
}

// noteArchive moves a note out of sight into the archive.
func (s *server) noteArchive(params json.RawMessage) (any, error) {
	return s.manage(params, func(n string, book bool) error {
		if book {
			_, err := s.api.Store().Archive(n)
			return err
		}
		_, err := s.api.Archive(n)
		return err
	})
}

// noteTrash moves a note, a page or a book to the trash.
func (s *server) noteTrash(params json.RawMessage) (any, error) {
	return s.manage(params, func(n string, book bool) error {
		if book {
			_, err := s.api.Store().Trash(n)
			return err
		}
		_, err := s.api.Remove(n)
		return err
	})
}

func (s *server) manage(params json.RawMessage, do func(name string, book bool) error) (any, error) {
	var p noteParams
	if err := decode(params, &p); err != nil {
		return nil, err
	}
	n, _, err := s.find(p.Note)
	if err != nil {
		return nil, err
	}
	if err := do(n.Name, n.Book()); err != nil {
		return nil, err
	}
	s.derive()
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
	if path.Ext(name) == "" { // a book was trashed as a folder
		st := s.api.Store()
		from, ok := st.Trashed(name)
		if !ok {
			return nil, notFound(fmt.Sprintf("there is no %s in the trash", name))
		}
		if err := st.Restore(from, name); err != nil {
			return nil, err
		}
	} else if _, err := s.api.Restore(name); err != nil {
		return nil, err
	}
	s.derive()
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
// same function (httpfile.SendNote); the app reads it from the board.
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
	st := s.api.Store()
	data, err := st.Read(n.Name)
	if err != nil {
		return nil, err
	}
	f := httpfile.Parse(string(data))
	titles := make([]string, len(f.Requests))
	for i, r := range f.Requests {
		titles[i] = r.Title()
	}
	part, err := widget.Pick(titles, want, "request")
	if err != nil {
		return nil, badRequest(err.Error())
	}
	res, err := httpfile.SendNote(s.ctx, st, st.Dir, n.Name, part, "")
	if err != nil {
		return nil, err
	}
	return sendResult{
		Status: res.Status, Text: res.Text, Took: res.Took.Milliseconds(), Failed: res.Failed(),
		End: strings.Trim(httpfile.End(res), "[]"), Log: httpfile.LogName(n.Name),
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
}

// kindsList is the catalog: every shape, in the registry's order.
func (s *server) kindsList(json.RawMessage) (any, error) {
	out := []kindInfo{}
	for _, k := range s.reg {
		example := k.ExampleFile()
		info := kindInfo{Name: k.Name, Icon: k.Icon, Label: k.Label, Blurb: k.Blurb, Example: example, Data: json.RawMessage("null")}
		if k.Data != nil {
			info.Data = data("example"+k.New, k.Data(doc.Parse([]byte(example))))
		}
		out = append(out, info)
	}
	return out, nil
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
