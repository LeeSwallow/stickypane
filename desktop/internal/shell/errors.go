package shell

import (
	"errors"
	"fmt"
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

func noWindow(id uint) error      { return fmt.Errorf("%w %d", ErrNoWindow, id) }
func noHost(name string) error    { return fmt.Errorf("%w %q", ErrNoHost, name) }
func hostTaken(name string) error { return fmt.Errorf("%w %q", ErrHostTaken, name) }

// hostOpenError is ErrHostOpen with the host's name in its sentence:
// close the boards of "dev" before renaming it.
type hostOpenError struct{ name string }

func (e *hostOpenError) Error() string {
	return fmt.Sprintf("close the boards of %q before renaming it", e.name)
}
func (e *hostOpenError) Is(target error) bool { return target == ErrHostOpen }
