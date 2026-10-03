// Package env tells what stickypane is running in: the system, the shell,
// the tool that can open a pane next to the agent, and the user's locale.
// Everything that differs between macOS, Linux and Windows, or between
// tmux, Zellij, WezTerm and Windows Terminal, is decided here, so the rest
// of the code asks instead of guessing.
//
// The files: env.go detects, pane.go knows the pane tools, shell.go the
// shells and how scripts run, powershell.go what is PowerShell's own, and
// locale_*.go how each system names its locale when no variable does.
package env

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
)

// Env is what Detect found.
type Env struct {
	OS         string // runtime.GOOS: "darwin", "linux", "windows"
	Shell      Shell
	Pane       Pane
	Locale     string // as found, such as "ko_KR.UTF-8" or "ko-KR"; "" for none
	LocaleFrom string // the variable it came from, or "system"

	editor   []string // the settings' choice, over $VISUAL and $EDITOR
	getenv   func(string) string
	lookPath func(string) (string, error)
}

// Detect looks at the environment and returns what it found. Variables are
// read on every call, since they are cheap and tests change them; asking
// the system for its locale, which may start a program, happens once.
func Detect() Env { return detect(os.Getenv, runtime.GOOS, cachedSystemLocale) }

var (
	localeOnce sync.Once
	sysLocale  string
)

func cachedSystemLocale() string {
	localeOnce.Do(func() { sysLocale = systemLocale() })
	return sysLocale
}

// localeVars are read in order; the first that names a language wins. "C"
// and "POSIX" name none.
var localeVars = []string{"STICKYPANE_LANG", "LC_ALL", "LC_MESSAGES", "LANG"}

func detect(getenv func(string) string, goos string, system func() string) Env {
	e := Env{OS: goos, getenv: getenv, lookPath: exec.LookPath}
	e.Shell = detectShell(getenv, goos)
	e.Pane = detectPane(getenv)
	for _, v := range localeVars {
		if l := strings.TrimSpace(getenv(v)); l != "" && l != "C" && l != "POSIX" {
			e.Locale, e.LocaleFrom = l, v
			return e
		}
	}
	if l := strings.TrimSpace(system()); l != "" {
		e.Locale, e.LocaleFrom = l, "system"
	}
	return e
}
