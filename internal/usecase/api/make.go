package api

import "fmt"

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
		body, err := k.Make(title, given)
		if err != nil {
			return nil, "", err
		}
		if k.New != "" {
			return body, k.New, nil
		}
		return body, ".md", nil
	}
	return nil, "", fmt.Errorf("%w %q", ErrUnknownType, kind)
}
