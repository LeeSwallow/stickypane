package shell

import "strconv"

// The Wails events the shell sends the frontend. Their names are the
// frontend's as well (src/lib/shell.ts and the windows that listen):
// a name changes on both sides at once.

// Events every window, or one window of its kind, hears.
const (
	// NoticesEvent tells the home window its notices changed.
	NoticesEvent = "home.notices"
	// PrefsEvent carries the app's settings (Prefs) to every window when
	// they change; SettingsEvent tells the settings window its board changed.
	PrefsEvent    = "prefs.changed"
	SettingsEvent = "settings.board"
	// HostsEvent tells every window the host list changed.
	HostsEvent = "hosts.changed"
)

// The events of one window are the name, a colon and the window's id.
const (
	changedPrefix   = "board.changed:"
	restartedPrefix = "bridge.restarted:"
	statusPrefix    = "project.status:"
	canvasPrefix    = "canvas.changed:"
)

func forWindow(prefix string, window uint) string {
	return prefix + strconv.FormatUint(uint64(window), 10)
}

// ChangedEvent is the Wails event that carries a project window's
// board.changed; RestartedEvent its bridge.restarted; StatusEvent a change
// of its Status.
func ChangedEvent(window uint) string   { return forWindow(changedPrefix, window) }
func RestartedEvent(window uint) string { return forWindow(restartedPrefix, window) }
func StatusEvent(window uint) string    { return forWindow(statusPrefix, window) }

// CanvasEvent tells a project window its layout changed outside it: a note
// torn off, put back or gone.
func CanvasEvent(window uint) string { return forWindow(canvasPrefix, window) }
