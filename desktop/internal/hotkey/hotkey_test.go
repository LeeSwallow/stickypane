package hotkey

import (
	"errors"
	"testing"
)

func TestNormalize(t *testing.T) {
	for in, want := range map[string]string{
		"option+space":       "Option+Space",
		"Alt+Space":          "Option+Space",
		"Cmd+Shift+k":        "Shift+Cmd+K",
		"cmdorctrl+shift+K":  "Shift+Cmd+K",
		"ctrl+option+cmd+f5": "Ctrl+Option+Cmd+F5",
		"Ctrl+Enter":         "Ctrl+Return",
		"Cmd+;":              "Cmd+;",
		"Cmd+plus":           "Cmd+Plus",
		" Option + Space ":   "Option+Space",
	} {
		got, err := Normalize(in)
		if err != nil || got != want {
			t.Errorf("Normalize(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
}

func TestCheckRefuses(t *testing.T) {
	for in, code := range map[string]string{
		"":            CodeEmpty,
		"K":           CodeModifier,
		"Shift+K":     CodeModifier,
		"Cmd+Space":   CodeReserved,
		"command+q":   CodeReserved,
		"Ctrl+Space":  CodeReserved,
		"Cmd+Shift+4": CodeReserved,
		"Cmd+Banana":  CodeKey,
		"Hyper+K":     CodeKey,
		"Option+F99":  CodeKey,
		"Option+한":    CodeKey,
	} {
		_, err := Check(in)
		var e *Error
		if !errors.As(err, &e) || e.Code != code {
			t.Errorf("Check(%q) = %v, want code %s", in, err, code)
		}
	}
}

func TestCheckTakes(t *testing.T) {
	for in, want := range map[string]string{
		"Option+Space": "Option+Space",
		"F13":          "F13",
		"Shift+F2":     "Shift+F2",
		"ctrl+alt+n":   "Ctrl+Option+N",
		"Cmd+Option+=": "Option+Cmd+=",
	} {
		got, err := Check(in)
		if err != nil || got != want {
			t.Errorf("Check(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
}
