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

	"github.com/LeeSwallow/stickypane/internal/core/when"

	"github.com/LeeSwallow/stickypane/internal/adapter/store"
	"github.com/LeeSwallow/stickypane/internal/core/doc"
	"github.com/LeeSwallow/stickypane/internal/core/widget"
	"github.com/LeeSwallow/stickypane/internal/core/widget/form"
	"github.com/LeeSwallow/stickypane/internal/usecase/arrange"
)

// ErrBadName rejects names that are not a note in the notes folder.
var ErrBadName = errors.New(`a note name is a file name such as "plan", "build.log" or, for a page of a book, "docs/plan"`)

// ErrUnknownType is what Write and Create answer for a type that is not a
// shape of note. The message names the type.
var ErrUnknownType = errors.New("unknown type")

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
	now func() time.Time // the clock every time the API writes is read from
	env widget.Env       // what widgets read and act with: the same clock
}

// New returns the API for a notes folder. reg decides which types exist and
// what size a note has when it does not say.
func New(st *store.Store, reg widget.Registry) *API { return NewWithClock(st, reg, time.Now) }

// NewWithClock is New with the clock the API reads when it writes a time:
// when a note was made, an item ticked, a message said. A nil clock is
// time.Now.
func NewWithClock(st *store.Store, reg widget.Registry, now func() time.Time) *API {
	if now == nil {
		now = time.Now
	}
	return NewWithEnv(st, reg, widget.Env{Clock: now, StaleAfter: widget.StaleAfter})
}

// NewWithEnv is New with the Env the API hands the widgets it reads and
// acts through: a summary in the user's words, the time an action writes.
// The API's clock is the Env's.
func NewWithEnv(st *store.Store, reg widget.Registry, e widget.Env) *API {
	return &API{st: st, reg: reg, now: e.Now, env: e}
}

// Now is the time on the API's clock, for callers that write a time
// beside what the API writes.
func (a *API) Now() time.Time { return a.now() }

// Env is what the API hands widgets: its clock, and the words and settings
// it was made with. A screen that draws a note's data reads it with this.
func (a *API) Env() widget.Env { return a.env }

// Store is the notes folder the API works on, for callers that only read
// it (where a file is, whether it is a link), watch it, or hand it to the
// runner. A change to a note or a setting goes through the API, so every
// front end keeps one rule for it.
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
	return a.st.Read(a.existing(name, file))
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
	d, err := a.shape(doc.Parse(body), opts)
	if err != nil {
		return "", err
	}
	// What is kept from the note is read in the same step that replaces
	// it, so nothing written in between is lost.
	err = a.st.Rewrite(file, func(old []byte, ok bool) ([]byte, error) {
		d := d
		var before doc.Document
		if ok {
			before = doc.Parse(old)
			for _, key := range append([]string{"created"}, viewKeys...) {
				if _, set := d.Get(key); set {
					continue
				}
				if v, ok := before.Get(key); ok {
					d = d.Set(key, v)
				}
			}
		} else if isMarkdown(file) {
			d = created(d, a.now())
		}
		return a.tend(file, before, d).Bytes(), nil
	})
	if err != nil {
		return file, err
	}
	_, _ = a.Derive() // charts computed from this note follow it
	return file, nil
}

// shape sets the front matter keys opts asks for.
func (a *API) shape(d doc.Document, opts Options) (doc.Document, error) {
	if opts.Type != "" {
		if a.reg.Lookup(opts.Type).Name != opts.Type {
			return d, fmt.Errorf("%w %q", ErrUnknownType, opts.Type)
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
		size, ok := widget.NormalSize(opts.Size)
		if !ok {
			return d, fmt.Errorf("unknown size %q: use %s", opts.Size, arrange.SizeNames)
		}
		d = d.Set("size", size)
	}
	return d, nil
}

// Create makes a new note and returns its file name. It is Write for a
// name nobody has: where a file is already, nothing is written and the
// error is fs.ErrExist, even if the file came a moment ago, so that two
// callers choosing the same name never write over each other.
func (a *API) Create(name string, opts Options, body []byte) (string, error) {
	file, err := fileName(name)
	if err != nil {
		return "", err
	}
	d, err := a.shape(doc.Parse(body), opts)
	if err != nil {
		return "", err
	}
	if isMarkdown(file) {
		d = created(d, a.now())
	}
	d = a.tend(file, doc.Document{}, d)
	if err := a.st.WriteNew(file, d.Bytes()); err != nil {
		return file, err
	}
	_, _ = a.Derive() // charts computed from this note follow it
	return file, nil
}

// isMarkdown reports whether a note may carry front matter.
func isMarkdown(file string) bool { return strings.EqualFold(filepath.Ext(file), ".md") }

// created says in a new note when stickypane made it.
func created(d doc.Document, now time.Time) doc.Document {
	if _, ok := d.Get("created"); ok {
		return d
	}
	return d.Set("created", now.Format(when.Stamp))
}

// tend writes what the note's kind fills in by itself, such as the time an
// item was ticked, so that the agent does not have to.
func (a *API) tend(file string, before, d doc.Document) doc.Document {
	if k := a.reg.For(file, d); k.Tend != nil {
		if op := k.Tend(before, d, a.now()); op != nil {
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
	b, err := a.st.Read(a.existing(name, file))
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
	return a.wait(ctx, name, tick.C)
}

// wait is Wait looking again on every tick, which a test can send one by one.
func (a *API) wait(ctx context.Context, name string, tick <-chan time.Time) (form.Answers, error) {
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
		case <-tick:
		}
	}
}
