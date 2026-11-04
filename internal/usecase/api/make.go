package api

import (
	"fmt"

	environment "github.com/LeeSwallow/stickypane/internal/adapter/env"
	"github.com/LeeSwallow/stickypane/internal/adapter/store"
	"github.com/LeeSwallow/stickypane/internal/core/i18n"
	"github.com/LeeSwallow/stickypane/internal/core/widget"
)

// Make writes a new note of kind called title from create options, by the
// kind's own Compose, and returns it with the extension a file of the kind
// takes: ".md", or the kind's own (".sh") for one that is a file of its
// own and has no front matter. No kind is the plain note. Nothing is
// written: hand the body to Create or Write.
func (a *API) Make(kind, title string, given map[string]string) (body []byte, ext string, err error) {
	if kind == "" {
		kind = "note"
	}
	for _, k := range a.reg {
		if k.Name != kind {
			continue
		}
		body, err := k.Make(title, given, a.words())
		if err != nil {
			return nil, "", err
		}
		if k.New != "" {
			return body, k.New, nil
		}
		return body, store.MarkdownExt, nil
	}
	return nil, "", fmt.Errorf("%w %q", ErrUnknownType, kind)
}

// words is the Env a new note is written with, so a shape's own words in
// it (a board's columns) are the board's: the API's Env, in the language
// the board chose (else the environment's) when the Env has no words of
// its own. The command line, MCP and the bridge give the API none, as
// what they print stays English; a screen gives it the words it draws in.
func (a *API) words() widget.Env {
	e := a.env
	if e.Translate == nil && a.st != nil {
		e.Translate = i18n.Pick(a.st.Language(), environment.Detect().Locale).L
	}
	return e
}
