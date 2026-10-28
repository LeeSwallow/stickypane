package api

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/LeeSwallow/stickypane/internal/core/doc"
	"github.com/LeeSwallow/stickypane/internal/core/widget"
)

// ErrStale is what Apply answers when the note is not the one the screen
// drew its action from.
var ErrStale = errors.New("the note changed since it was read")

// acted is an Op that does what a screen asked, if the note is still the
// one it read. It checks the file as it is when written, so a change made
// in between is never overwritten.
type acted struct {
	api     *API
	file    string
	version string
	act     widget.Action
	after   *doc.Document
}

func (o acted) Apply(d doc.Document) (doc.Document, error) {
	if widget.Version(d) != o.version {
		return d, ErrStale
	}
	k := o.api.reg.For(o.file, d)
	if k.Act == nil {
		return d, fmt.Errorf("%s is a %s: it takes no action from a screen", o.file, k.Name)
	}
	op, err := k.Act(d, o.act, o.api.env)
	if err != nil {
		return d, err
	}
	out, err := op.Apply(d)
	if err != nil {
		return d, err
	}
	*o.after = o.api.tend(o.file, d, out)
	return *o.after, nil
}

// Apply does what a screen other than the terminal asked of a note. version
// is what the screen read (widget.Version); when the note has changed since,
// nothing is written and the answer is ErrStale. It returns the version of
// the note afterwards.
func (a *API) Apply(name, version string, act widget.Action) (string, error) {
	file, err := fileName(name)
	if err != nil {
		return "", err
	}
	file = a.existing(name, file)
	var after doc.Document
	if err := a.st.Apply(file, acted{a, file, version, act, &after}); err != nil {
		return "", err
	}
	_, _ = a.Derive() // charts computed from this note follow it
	return widget.Version(after), nil
}

// Do applies op, an intent a screen made from a key (tick this item, move
// that card), to the note called name, as it is on disk now, and fills in
// what the note's kind writes by itself. On a conflict nothing is written
// and the error is doc.ErrConflict.
func (a *API) Do(name string, op doc.Op) error {
	if err := a.st.Apply(name, tended{a, name, op}); err != nil {
		return err
	}
	_, _ = a.Derive() // charts computed from this note follow it
	return nil
}

// tended is an Op followed by what the note's kind fills in.
type tended struct {
	api  *API
	file string
	op   doc.Op
}

func (o tended) Apply(d doc.Document) (doc.Document, error) {
	out, err := o.op.Apply(d)
	if err != nil {
		return d, err
	}
	return o.api.tend(o.file, d, out), nil
}

// Save writes what the user wrote in an editor into the note called name,
// as it is: the text is theirs, and nothing is added to it. was is the
// file as the editor loaded it; when the file is not that any more, or
// cannot be read, someone else has changed it, and nothing is written:
// ErrStale. A nil was writes whatever is there.
func (a *API) Save(name string, was, body []byte) error {
	// Compared and written in one step: nothing written in between is
	// overwritten.
	err := a.st.Rewrite(name, func(now []byte, ok bool) ([]byte, error) {
		if was != nil && (!ok || !bytes.Equal(now, was)) {
			return nil, ErrStale
		}
		return body, nil
	})
	if err != nil {
		return err
	}
	_, _ = a.Derive()
	return nil
}
