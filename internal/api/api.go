// Package api lists, reads and writes notes for callers that do not edit the
// files themselves: the command line and the MCP server. Editing the files
// directly stays the primary way to use stickypane; this is a thin layer over
// the same files.
package api

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// ErrBadName rejects names that are not a plain Markdown file name.
var ErrBadName = errors.New(`a note name is a plain file name such as "plan" or "plan.md"`)

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

// fileName turns a note name into its file name, adding ".md" when missing.
// Anything that could leave the notes folder or hide the file is rejected.
func fileName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || strings.ContainsAny(name, `/\`) || strings.HasPrefix(name, ".") {
		return "", ErrBadName
	}
	switch ext := filepath.Ext(name); {
	case ext == "":
		name += ".md"
	case !strings.EqualFold(ext, ".md"):
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
	infos := make([]Info, 0, len(notes))
	for _, n := range notes {
		kind := a.reg.Lookup(n.Doc.Type())
		info := Info{Name: n.Name, Type: kind.Name, Pinned: n.Doc.Pinned()}
		if info.Title, _ = n.Doc.Get("title"); info.Title == "" {
			info.Title = strings.TrimSuffix(n.Name, filepath.Ext(n.Name))
		}
		open, _ := n.Doc.Get("open")
		info.Open = strings.EqualFold(open, "true")
		if size, _ := n.Doc.Get("size"); validSize(strings.ToLower(size)) {
			info.Size = strings.ToLower(size)
		} else if kind.Size != nil {
			info.Size = kind.Size(n.Doc)
		}
		infos = append(infos, info)
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

// Write creates the note or replaces all of it with body, then sets the
// front matter keys given in opts. It returns the file name. Whatever was in
// the note before is gone, so read a note before rewriting it.
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
	return file, a.st.Write(file, d.Bytes())
}
