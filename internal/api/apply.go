package api

import (
	"errors"
	"fmt"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
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
	op, err := k.Act(d, o.act)
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
