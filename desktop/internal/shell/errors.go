package shell

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/LeeSwallow/stickypane/desktop/internal/install/model"
	installsvc "github.com/LeeSwallow/stickypane/desktop/internal/install/service"
	"github.com/LeeSwallow/stickypane/desktop/internal/remote"
)

// The shell's errors, for errors.Is. Each is the start of what the window
// shows: the rest names the window, the host or the reason.
var (
	// ErrNoWindow: no open board window, or torn-off note, has that id.
	ErrNoWindow = errors.New("no project window")
	// ErrNoCaller: a service was called from no window.
	ErrNoCaller = errors.New("no window made this call")
	// ErrNoBoard: the window has no running bridge (it is loading, failed,
	// or closed).
	ErrNoBoard = errors.New("no board in this window")
	// ErrNoNote: a tear-off named no note.
	ErrNoNote = errors.New("which note?")
	// ErrNoHost: no host of that name is in the app's list.
	ErrNoHost = errors.New("no host")
	// ErrHostTaken: another host has that name already.
	ErrHostTaken = errors.New("a host is already named")
	// ErrHostOpen: a host's boards are open, so it cannot be renamed.
	ErrHostOpen = errors.New("close the boards of the host before renaming it")
	// ErrUnsafeFolder: a board would be made where every project below
	// would find it.
	ErrUnsafeFolder = errors.New("will not make a board")
	// ErrIndexNotOpen and ErrIndexNotStarted: the index of boards cannot
	// be had (yet).
	ErrIndexNotOpen    = errors.New("index: not open")
	ErrIndexNotStarted = errors.New("index: not started")
)

// ErrRemote is what a remote board cannot do from here: open a note in
// this computer's editor, or be made with this computer's stickypane.
var ErrRemote = errors.New("this board is on another computer")

// ErrNoAutostart: this desktop cannot open the app at login.
var ErrNoAutostart = errors.New("opening at login is not supported here")

func noWindow(id uint) error { return fmt.Errorf("%w %d", ErrNoWindow, id) }
func noHost(name string) error {
	return &hostError{ErrNoHost, name, fmt.Sprintf("%v %q", ErrNoHost, name)}
}
func hostTaken(name string) error {
	return &hostError{ErrHostTaken, name, fmt.Sprintf("%v %q", ErrHostTaken, name)}
}
func hostOpen(name string) error {
	return &hostError{ErrHostOpen, name, fmt.Sprintf("close the boards of %q before renaming it", name)}
}

// hostError is one of the host errors with the host's name, which the
// window's sentence names too.
type hostError struct {
	is   error
	name string
	said string
}

func (e *hostError) Error() string        { return e.said }
func (e *hostError) Is(target error) bool { return target == e.is }

// ErrorWords is what a window says for an error the shell knows, in the
// window's own language: Code names the sentence (error.<code> in the
// frontend's catalogs) and Values fill it. It goes to the window as the
// failed call's cause (MarshalError); the message stays English, for the
// log.
type ErrorWords struct {
	Code   string            `json:"code"`
	Values map[string]string `json:"values,omitempty"`
}

// errorCodes are the errors a window says in its own words, by code. The
// first that err is wins.
var errorCodes = []struct {
	err  error
	code string
}{
	{ErrNoBoard, "no_board"},
	{ErrRemote, "remote"},
	{ErrUnsafeFolder, "unsafe_folder"},
	{installsvc.ErrUnsafeDir, "unsafe_folder"},
	{ErrNoHost, "no_host"},
	{ErrHostTaken, "host_taken"},
	{ErrHostOpen, "host_open"},
	{ErrNoAutostart, "no_autostart"},
	{ErrIndexNotOpen, "index"},
	{ErrIndexNotStarted, "index"},
	{installsvc.ErrNoBundled, "no_stickypane"},
	{model.ErrNoKeys, "old_stickypane"},
	{installsvc.ErrNoProject, "no_project"},
}

// WordsOf is the code and values of err, if the shell knows it.
func WordsOf(err error) (ErrorWords, bool) {
	var inv *remote.Invalid
	if errors.As(err, &inv) {
		return ErrorWords{Code: inv.Code, Values: map[string]string{"value": inv.Value}}, true
	}
	for _, c := range errorCodes {
		if !errors.Is(err, c.err) {
			continue
		}
		w := ErrorWords{Code: c.code}
		var he *hostError
		if errors.As(err, &he) {
			w.Values = map[string]string{"name": he.name}
		}
		return w, true
	}
	return ErrorWords{}, false
}

// MarshalError is the app's application.Options.MarshalError: an error
// the shell knows goes to the window as its ErrorWords; any other as
// Wails sends it (nil).
func MarshalError(err error) []byte {
	w, ok := WordsOf(err)
	if !ok {
		return nil
	}
	b, jerr := json.Marshal(w)
	if jerr != nil {
		return nil
	}
	return b
}
