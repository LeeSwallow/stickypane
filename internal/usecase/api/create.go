package api

import (
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/adapter/store"
)

// MaxSameName is how many notes of one name CreateFrom makes before it
// gives up.
const MaxSameName = 1000

// CreateFrom makes a new note named after text, in tab ("" for the top
// level), with the extension ext (".md" when empty), and returns its file
// name. It is the one way a screen makes a note: the name is text's slug
// on the API's clock (store.Slug), with "-2", "-3"... after it when a note
// has it already, each made exclusively so that nothing is written over;
// the note is stamped and tended as Create does. A tab that is not there is
// fs.ErrNotExist.
func (a *API) CreateFrom(tab, text, ext string, opts Options, body []byte) (string, error) {
	if ext == "" {
		ext = ".md"
	}
	if tab = strings.Trim(tab, "/"); tab != "" {
		tabs, err := a.st.Tabs()
		if err != nil {
			return "", err
		}
		if !slices.ContainsFunc(tabs, func(t store.Tab) bool { return t.Name == tab }) {
			return "", missing(fmt.Sprintf("there is no tab %q", tab))
		}
	}
	stem := store.Slug(text, a.now())
	if tab != "" {
		stem = tab + "/" + stem
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
