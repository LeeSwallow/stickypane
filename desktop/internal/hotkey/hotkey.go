// Package hotkey says what a global hotkey is: it reads an accelerator as
// the settings' key recorder writes it ("Option+Space", "Cmd+Shift+K"),
// writes it one way, and refuses what cannot be a global hotkey — a key
// with no modifier, a key the system keeps for itself. Whether another
// program holds it is the desktop's to say (the shell tries it). Pure.
package hotkey

import (
	"fmt"
	"strings"
)

// The ways a hotkey is refused, for the screen to put in words.
const (
	CodeEmpty    = "empty"    // nothing was pressed
	CodeKey      = "key"      // the key is not one a hotkey can be
	CodeModifier = "modifier" // it needs Cmd, Option or Ctrl
	CodeReserved = "reserved" // the system or the app keeps it
)

// Error is a hotkey refused; Code is one of the Code constants.
type Error struct {
	Code   string
	Accel  string
	Detail string
}

func (e *Error) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("hotkey %q: %s", e.Accel, e.Detail)
	}
	return fmt.Sprintf("hotkey %q: %s", e.Accel, e.Code)
}

// The modifiers, in the order macOS writes them.
var order = []string{"Ctrl", "Option", "Shift", "Cmd"}

var modifiers = map[string]string{
	"ctrl": "Ctrl", "control": "Ctrl",
	"alt": "Option", "option": "Option", "opt": "Option", "optionoralt": "Option",
	"shift": "Shift",
	"cmd": "Cmd", "command": "Cmd", "cmdorctrl": "Cmd", "super": "Cmd", "meta": "Cmd",
}

// named are the keys with names, as Wails reads them, written one way.
var named = map[string]string{
	"space": "Space", "tab": "Tab", "return": "Return", "enter": "Return",
	"escape": "Escape", "esc": "Escape", "backspace": "Backspace", "delete": "Delete",
	"left": "Left", "right": "Right", "up": "Up", "down": "Down",
	"home": "Home", "end": "End", "page up": "Page Up", "page down": "Page Down",
	"plus": "Plus",
}

// reserved are hotkeys macOS or the app keep for themselves.
var reserved = map[string]string{
	"Cmd+Space":         "Spotlight",
	"Ctrl+Space":        "the input source",
	"Ctrl+Option+Space": "the input source",
	"Cmd+Tab":           "switching apps",
	"Cmd+Q":             "quitting an app",
	"Cmd+W":             "closing a window",
	"Cmd+H":             "hiding an app",
	"Cmd+M":             "minimising a window",
	"Cmd+`":             "switching windows",
	"Cmd+,":             "an app's settings",
	"Option+Cmd+Escape": "Force Quit",
	"Shift+Cmd+3":       "screenshots",
	"Shift+Cmd+4":       "screenshots",
	"Shift+Cmd+5":       "screenshots",
}

// key writes a key one way: a name, an F-key, or one printable character
// in upper case. ok is false when it cannot be a hotkey's key.
func key(k string) (string, bool) {
	l := strings.ToLower(strings.TrimSpace(k))
	if n, ok := named[l]; ok {
		return n, true
	}
	if len(l) >= 2 && l[0] == 'f' {
		n := 0
		for _, c := range l[1:] {
			if c < '0' || c > '9' {
				return "", false
			}
			n = n*10 + int(c-'0')
		}
		if n >= 1 && n <= 20 {
			return fmt.Sprintf("F%d", n), true
		}
		return "", false
	}
	r := []rune(l)
	if len(r) != 1 || r[0] <= ' ' || r[0] > '~' || r[0] == '+' {
		return "", false
	}
	return strings.ToUpper(l), true
}

// isF says whether a key written by key is an F-key.
func isF(k string) bool { return len(k) >= 2 && k[0] == 'F' && k[1] >= '0' && k[1] <= '9' }

// Normalize reads an accelerator and writes it one way: its modifiers in
// macOS's order, then its key ("option+space" is "Option+Space").
func Normalize(accel string) (string, error) {
	a := strings.TrimSpace(accel)
	if a == "" {
		return "", &Error{Code: CodeEmpty, Accel: accel}
	}
	parts := strings.Split(a, "+")
	last := parts[len(parts)-1]
	k, ok := key(last)
	if !ok {
		return "", &Error{Code: CodeKey, Accel: accel, Detail: fmt.Sprintf("%q is not a key a hotkey can use", last)}
	}
	seen := map[string]bool{}
	for _, m := range parts[:len(parts)-1] {
		name, ok := modifiers[strings.ToLower(strings.TrimSpace(m))]
		if !ok {
			return "", &Error{Code: CodeKey, Accel: accel, Detail: fmt.Sprintf("%q is not a modifier", m)}
		}
		seen[name] = true
	}
	var out []string
	for _, m := range order {
		if seen[m] {
			out = append(out, m)
		}
	}
	return strings.Join(append(out, k), "+"), nil
}

// Check says whether accel can be a global hotkey, and answers it written
// one way. A hotkey needs Cmd, Option or Ctrl (Shift alone would take a
// capital letter from every program), except an F-key, which may stand
// alone; and it may not be one the system keeps.
func Check(accel string) (string, error) {
	n, err := Normalize(accel)
	if err != nil {
		return "", err
	}
	parts := strings.Split(n, "+")
	k := parts[len(parts)-1]
	strong := false
	for _, m := range parts[:len(parts)-1] {
		if m != "Shift" {
			strong = true
		}
	}
	if !strong && !isF(k) {
		return "", &Error{Code: CodeModifier, Accel: n, Detail: "needs Cmd, Option or Ctrl"}
	}
	if why, ok := reserved[n]; ok {
		return "", &Error{Code: CodeReserved, Accel: n, Detail: "kept for " + why}
	}
	return n, nil
}
