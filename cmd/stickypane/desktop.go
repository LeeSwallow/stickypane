package main

import (
	"errors"
	"fmt"
	"io"
	osexec "os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/adapter/store"
)

// The desktop app (desktop/): its bundle id on macOS and its program on
// Linux (docs/desktop/09 §9.1).
const (
	appBundleID = "io.github.leeswallow.stickypane"
	appProgram  = "stickypane-app"
	appGet      = "https://github.com/LeeSwallow/stickypane/releases"
)

// errNoApp: the desktop app is not installed here.
var errNoApp = errors.New("the stickypane app is not installed")

// openApp asks the desktop app to show the project at root: it starts the
// app, or hands the project to the one running. Tests replace it; nothing
// is launched in a test.
var openApp = launchApp

func launchApp(root string) error {
	arg := "--project=" + root
	switch runtime.GOOS {
	case "darwin":
		out, err := osexec.Command("/usr/bin/open", "-b", appBundleID, "--args", arg).CombinedOutput()
		if err != nil {
			// open says "Unable to find application" for a bundle id no app has
			return fmt.Errorf("%w (%s)", errNoApp, strings.TrimSpace(string(out)))
		}
		return nil
	case "linux":
		bin, err := osexec.LookPath(appProgram)
		if err != nil {
			return errNoApp
		}
		cmd := osexec.Command(bin, arg)
		if err := cmd.Start(); err != nil {
			return err
		}
		return cmd.Process.Release()
	}
	return errNoApp
}

// showInApp opens the board in the desktop app when its view asks for it
// ("overlay", shown as app, or "both"). It says whether the terminal board
// is still wanted: not when the app showed an app-only board.
func showInApp(dir string, stderr io.Writer) (terminal bool) {
	view := store.Open(dir).ViewMode()
	if view == store.ViewTerminal {
		return true
	}
	err := openApp(filepath.Dir(dir))
	switch {
	case err == nil:
		if view == store.ViewOverlay {
			fmt.Fprintln(stderr, "stickypane:", helpStrings().Say("app.opened", map[string]any{"Dir": filepath.Dir(dir)}))
			return false
		}
		return true
	case errors.Is(err, errNoApp) && view == store.ViewOverlay:
		fmt.Fprintln(stderr, "stickypane:", helpStrings().Say("app.missing", map[string]any{"URL": appGet}))
	case !errors.Is(err, errNoApp):
		fmt.Fprintln(stderr, "stickypane:", helpStrings().Say("app.cannotOpen", map[string]any{"Err": err}))
	}
	return true
}
