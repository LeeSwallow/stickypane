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

	"github.com/LeeSwallow/stickypane/internal/when"

	"github.com/LeeSwallow/stickypane/internal/arrange"
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
	// Modified is when the file last changed: for a book, its newest page.
	Modified time.Time `json:"modified"`
	// Created is when the note was made, when that is known.
	Created time.Time `json:"created,omitzero"`
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

// Store is the notes folder the API works on, for callers that read or
// change the board's settings.
func (a *API) Store() *store.Store { return a.st }

// fileName turns a note name into its file name, adding ".md" when it has
// no extension. A name may have folders in front: a tab, and a book in it
// ("deploy/docs/plan"). Anything that could leave the notes folder or hide
// the file is rejected, and so is a file the board does not show.
func fileName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || strings.Contains(name, `\`) || strings.ContainsFunc(name, unicode.IsControl) {
		return "", ErrBadName
	}
	parts := strings.Split(name, "/")
	if len(parts) > 3 {
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

// List describes every note, ordered by file name.
func (a *API) List() ([]Info, error) {
	board, err := a.st.Load()
	if err != nil {
		return nil, err
	}
	var notes []store.Note
	for _, t := range board.Tabs {
		notes = append(notes, t.Notes...)
	}
	views := board.Settings.Views() // a broken file arranges nothing
	infos := make([]Info, 0, len(notes))
	// describe lists a file. Where it is on the screen is said by
	// sticky.json for the note it belongs to (a page belongs to its book),
	// else by the note's own front matter, else by its kind.
	describe := func(n store.Note, owner store.Note) {
		kind := a.reg.For(n.Name, n.Doc)
		v := views[owner.Name]
		open, _ := arrange.Open(v, owner.Doc)
		info := Info{
			Name:     n.Name,
			Type:     kind.Name,
			Title:    arrange.Title(views[n.Name], n.Doc, n.Name),
			Open:     open,
			Size:     arrange.Size(v, owner.Doc, kind, n.Doc),
			Pinned:   arrange.Pinned(v, owner.Doc),
			Modified: n.ModTime,
			Created:  n.Created,
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

// Cat returns a note's file content.
func (a *API) Cat(name string) ([]byte, error) {
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
		if !arrange.ValidSize(opts.Size) {
			return "", fmt.Errorf("unknown size %q: use page, half or card", opts.Size)
		}
		d = d.Set("size", opts.Size)
	}
	old, err := a.st.Read(file)
	if err == nil {
		was := doc.Parse(old)
		for _, key := range append([]string{"created"}, viewKeys...) {
			if _, set := d.Get(key); set {
				continue
			}
			if v, ok := was.Get(key); ok {
				d = d.Set(key, v)
			}
		}
	}
	var before doc.Document
	if err == nil {
		before = doc.Parse(old)
	} else if isMarkdown(file) {
		d = created(d)
	}
	d = a.tend(file, before, d)
	if err := a.st.Write(file, d.Bytes()); err != nil {
		return file, err
	}
	_, _ = a.Derive() // charts computed from this note follow it
	return file, nil
}

// isMarkdown reports whether a note may carry front matter.
func isMarkdown(file string) bool { return strings.EqualFold(filepath.Ext(file), ".md") }

// created says in a new note when stickypane made it.
func created(d doc.Document) doc.Document {
	if _, ok := d.Get("created"); ok {
		return d
	}
	return d.Set("created", time.Now().Format(when.Stamp))
}

// tend writes what the note's kind fills in by itself, such as the time an
// item was ticked, so that the agent does not have to.
func (a *API) tend(file string, before, d doc.Document) doc.Document {
	if k := a.reg.For(file, d); k.Tend != nil {
		if op := k.Tend(before, d, time.Now()); op != nil {
			if out, err := op.Apply(d); err == nil {
				return out
			}
		}
	}
	return d
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
