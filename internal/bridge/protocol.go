package bridge

import (
	"encoding/json"
	"errors"
	"io/fs"

	"github.com/LeeSwallow/stickypane/internal/api"
	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// Protocol is the version of the messages. A screen that speaks another
// one asks the user to update; hello says which this is.
const Protocol = 1

// Version is the program's version, which hello reports. The program sets
// it before it serves.
var Version = "dev"

// The codes an error answer carries.
const (
	CodeBadRequest = "bad_request" // the request makes no sense: not JSON, no such method, a bad value
	CodeNotFound   = "not_found"   // no such note or tab
	CodeStale      = "stale"       // the note changed since the screen read it
	CodeFailed     = "failed"      // a sound request that could not be done
)

// request is one line from the screen. A request without an id is
// answered all the same, with a null id.
type request struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

// response is the answer to one request: a result or an error.
type response struct {
	ID     json.RawMessage `json:"id"`
	Result any             `json:"result,omitempty"`
	Error  *Error          `json:"error,omitempty"`
}

// notification is a line the bridge sends of its own accord.
type notification struct {
	Method string `json:"method"`
	Params any    `json:"params"`
}

// Error is an error answer.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string { return e.Message }

func badRequest(msg string) *Error { return &Error{CodeBadRequest, msg} }
func notFound(msg string) *Error   { return &Error{CodeNotFound, msg} }

// answer turns what a method returned into an error answer, sorting the
// errors the layers below give into the protocol's codes.
func answer(err error) *Error {
	var e *Error
	switch {
	case errors.As(err, &e):
		return e
	case errors.Is(err, api.ErrStale):
		return &Error{CodeStale, err.Error()}
	case errors.Is(err, api.ErrBadName), errors.Is(err, api.ErrUnknownType), errors.Is(err, widget.ErrUnknownAction), errors.Is(err, widget.ErrBadOption):
		return badRequest(err.Error())
	case errors.Is(err, fs.ErrNotExist), errors.Is(err, doc.ErrConflict):
		return notFound(err.Error())
	}
	return &Error{CodeFailed, err.Error()}
}

// empty is the result of a method that has nothing to say but that it
// was done: {}.
type empty struct{}

// The parameters of each method.
type (
	actParams struct {
		Note    string         `json:"note"`
		Version string         `json:"version"`
		Action  *widget.Action `json:"action"`
	}
	createParams struct {
		Tab   string `json:"tab"`
		Kind  string `json:"kind"`
		Title string `json:"title"`
		Body  string `json:"body"`
		// Options, when given, are the kind's create options (kinds.list
		// says which): the kind's own code writes the note from them, in
		// place of a body.
		Options map[string]json.RawMessage `json:"options"`
	}
	noteParams struct {
		Note string `json:"note"`
	}
	arrangeParams struct {
		Note  string  `json:"note"`
		Open  *bool   `json:"open"`
		Size  *string `json:"size"`
		Rows  *int    `json:"rows"`
		Pin   *bool   `json:"pin"`
		Color *string `json:"color"`
		Title *string `json:"title"`
	}
	orderParams struct {
		Tab   string   `json:"tab"`
		Order []string `json:"order"`
	}
	moveParams struct {
		Note string `json:"note"`
		To   string `json:"to"`
	}
	sendParams struct {
		Note    string          `json:"note"`
		Request json.RawMessage `json:"request"` // its title or part, or its number from 1
	}
	settingsParams struct {
		Theme    *string `json:"theme"`
		Style    *string `json:"style"`
		View     *string `json:"view"`
		Language *string `json:"language"`
	}
)
