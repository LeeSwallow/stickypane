package api

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/adapter/store"
)

// MaxSameName is how many notes of one name CreateFrom makes before it
// gives up.
const MaxSameName = 1000

// CreateFrom makes a new note named after text, in a tab or a book (in,
// "" for the top level; a book is "tab/book"), with the extension ext (".md" when empty), and returns its file
// name. It is the one way a screen makes a note: the name is text's slug
// on the API's clock (store.Slug), with "-2", "-3"... after it when a note
// has it already, each made exclusively so that nothing is written over;
// the note is stamped and tended as Create does. A page goes where the
// book's file names put it. A tab or a book that is not there is
// fs.ErrNotExist.
func (a *API) CreateFrom(in, text, ext string, opts Options, body []byte) (string, error) {
	if ext == "" {
		ext = store.MarkdownExt
	}
	if in = strings.Trim(in, "/"); in != "" {
		if _, err := a.folder(in); err != nil {
			return "", err
		}
	}
	stem := store.Slug(text, a.now())
	if in != "" {
		stem = in + "/" + stem
	}
	for i := 1; i <= MaxSameName; i++ {
		name := stem + ext
		if i > 1 {
			name = fmt.Sprintf("%s-%d%s", stem, i, ext)
		}
		file, err := a.Create(name, opts, body)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		return file, err
	}
	return "", fmt.Errorf("there are %d notes called %q already", MaxSameName, stem)
}

// folder finds a tab ("deploy") or a book ("deploy/docs") on the board,
// with the notes in it.
func (a *API) folder(name string) ([]store.Note, error) {
	tabs, err := a.st.Tabs()
	if err != nil {
		return nil, err
	}
	for _, t := range tabs {
		if t.Name == "" {
			continue
		}
		if t.Name == name {
			return t.Notes, nil
		}
		if i := slices.IndexFunc(t.Notes, func(n store.Note) bool { return n.Book() && n.Name == name }); i >= 0 {
			return t.Notes[i].Pages, nil
		}
	}
	return nil, missing(fmt.Sprintf("there is no tab or book %q", name))
}

// Pages lists the pages of a book ("deploy/docs"), in the order the board
// shows them. Anything that is not a book is fs.ErrNotExist.
func (a *API) Pages(book string) ([]Info, error) {
	book = strings.Trim(strings.TrimSpace(book), "/")
	if !strings.Contains(book, "/") {
		return nil, missing(fmt.Sprintf("there is no book %q: a book is a folder in a tab, such as deploy/docs", book))
	}
	if _, err := a.folder(book); err != nil {
		return nil, err
	}
	infos, err := a.List()
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(infos, func(n Info) bool { return path.Dir(n.Name) != book }), nil
}
