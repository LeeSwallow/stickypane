// Package api lists, reads and writes notes for callers that do not edit the
// files themselves: the command line and the MCP server. Editing the files
// directly stays the primary way to use stickypane; this is a thin layer over
// the same files.
package api

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/widget"
	"github.com/LeeSwallow/stickypane/internal/widget/form"
)

// ErrBadName rejects names that are not a note in the notes folder.
var ErrBadName = errors.New(`a note name is a file name such as "plan", "build.log" or, for a page of a book, "docs/plan"`)

// waitRetries is how many failed looks in a row Wait rides out.
const waitRetries = 10

// ErrNotForm reports that a note has no answers to read.
var ErrNotForm = errors.New("that note is not a form: only a note with type: form has answers")

// Info describes a note without its content.
type Info struct {
	Name   string `json:"name"`
	Title  string `json:"title"`
	Type   string `json:"type"`
	Open   bool   `json:"open"`
	Size   string `json:"size"`
	Pinned bool   `json:"pinned"`
}

// Options are the front matter keys Write sets. Zero values set nothing.
type Options struct {
	Type  string
	Title string
	Size  string
	Open  bool
}

// API works on one notes folder.
type API struct {
	st  *store.Store
	reg widget.Registry
}

// New returns the API for a notes folder. reg decides which types exist and
// what size a note has when it does not say.
func New(st *store.Store, reg widget.Registry) *API { return &API{st: st, reg: reg} }

// fileName turns a note name into its file name, adding ".md" when it has
// no extension. A name may have one folder in front, for a page of a book.
// Anything that could leave the notes folder or hide the file is rejected,
// and so is a file the board does not show.
func fileName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || strings.Contains(name, `\`) || strings.ContainsFunc(name, unicode.IsControl) {
		return "", ErrBadName
	}
	parts := strings.Split(name, "/")
	if len(parts) > 2 {
		return "", ErrBadName
	}
	for _, p := range parts {
		if p == "" || strings.HasPrefix(p, ".") {
			return "", ErrBadName
		}
	}
	if len(parts) == 2 && parts[0] == store.ArchiveDir {
		return "", ErrBadName
	}
	if filepath.Ext(name) == "" {
		name += ".md"
	}
	if !store.IsNote(name) {
		return "", ErrBadName
	}
	return name, nil
}

func validSize(size string) bool {
	return size == widget.SizePage || size == widget.SizeHalf || size == widget.SizeCard
}

// List describes every note, ordered by file name.
func (a *API) List() ([]Info, error) {
	notes, err := a.st.Scan()
	if err != nil {
		return nil, err
	}
	views, _ := a.st.Views() // a broken file arranges nothing
	infos := make([]Info, 0, len(notes))
	// describe lists a file. Where it is on the screen is said by
	// sticky.json for the note it belongs to (a page belongs to its book),
	// else by the note's own front matter, else by its kind.
	describe := func(n store.Note, owner store.Note) {
		kind := a.reg.For(n.Name, n.Doc)
		v := views[owner.Name]
		info := Info{Name: n.Name, Type: kind.Name, Pinned: owner.Doc.Pinned()}
		if v.Pin != nil {
			info.Pinned = *v.Pin
		}
		if info.Title, _ = n.Doc.Get("title"); info.Title == "" {
			base := filepath.Base(n.Name)
			info.Title = strings.TrimSuffix(base, filepath.Ext(base))
		}
		open, _ := owner.Doc.Get("open")
		info.Open = strings.EqualFold(open, "true")
		if v.Open != nil {
			info.Open = *v.Open
		}
		size, _ := owner.Doc.Get("size")
		switch {
		case validSize(strings.ToLower(v.Size)):
			info.Size = strings.ToLower(v.Size)
		case validSize(strings.ToLower(size)):
			info.Size = strings.ToLower(size)
		case kind.Size != nil:
			info.Size = kind.Size(n.Doc)
		}
		infos = append(infos, info)
	}
	for _, n := range notes {
		if !n.Book() {
			describe(n, n)
		}
		for _, p := range n.Pages {
			describe(p, n)
		}
	}
	return infos, nil
}

// Show returns a note's file content.
func (a *API) Show(name string) ([]byte, error) {
	file, err := fileName(name)
	if err != nil {
		return nil, err
	}
	return a.st.Read(file)
}

// viewKeys are how the user arranged a note on the screen.
var viewKeys = []string{"open", "size", "rows", "color", "pin"}

// Write creates the note or replaces its content with body, then sets the
// front matter keys given in opts. It returns the file name. The text that
// was in the note before is gone, so read a note before rewriting it. How the
// user arranged the note is kept: a rewrite that says nothing about open,
// size, rows, color or pin leaves them as they were, so that an agent
// updating a note does not make it vanish from the user's screen.
func (a *API) Write(name string, opts Options, body []byte) (string, error) {
	file, err := fileName(name)
	if err != nil {
		return "", err
	}
	d := doc.Parse(body)
	if opts.Type != "" {
		if a.reg.Lookup(opts.Type).Name != opts.Type {
			return "", fmt.Errorf("unknown type %q", opts.Type)
		}
		d = d.Set("type", opts.Type)
	}
	if opts.Title != "" {
		d = d.Set("title", opts.Title)
	}
	if opts.Open {
		d = d.Set("open", "true")
	}
	if opts.Size != "" {
		if !validSize(opts.Size) {
			return "", fmt.Errorf("unknown size %q: use page, half or card", opts.Size)
		}
		d = d.Set("size", opts.Size)
	}
	if old, err := a.st.Read(file); err == nil {
		was := doc.Parse(old)
		for _, key := range viewKeys {
			if _, set := d.Get(key); set {
				continue
			}
			if v, ok := was.Get(key); ok {
				d = d.Set(key, v)
			}
		}
	}
	return file, a.st.Write(file, d.Bytes())
}

// Answers reads what the user chose and wrote in a form, and which button
// they pressed.
func (a *API) Answers(name string) (form.Answers, error) {
	file, err := fileName(name)
	if err != nil {
		return form.Answers{}, err
	}
	b, err := a.st.Read(file)
	if err != nil {
		return form.Answers{}, err
	}
	d := doc.Parse(b)
	if d.Type() != "form" {
		return form.Answers{}, ErrNotForm
	}
	return form.Read(d), nil
}

// Wait returns a form's answers once a button has been pressed, looking at
// the file every interval. When ctx ends first it returns the answers so far
// together with the context's error.
//
// A form that cannot be read at the first look is an error. After that, a
// few failed looks in a row are ridden out: an editor or an agent replacing
// the file may leave it empty for a moment.
func (a *API) Wait(ctx context.Context, name string, every time.Duration) (form.Answers, error) {
	tick := time.NewTicker(every)
	defer tick.Stop()
	var last form.Answers
	for failed, first := 0, true; ; first = false {
		got, err := a.Answers(name)
		switch {
		case err == nil && got.Submitted:
			return got, nil
		case err == nil:
			last, failed = got, 0
		case first || failed >= waitRetries:
			return got, err
		default:
			failed++
		}
		got = last
		select {
		case <-ctx.Done():
			return got, ctx.Err()
		case <-tick.C:
		}
	}
}
