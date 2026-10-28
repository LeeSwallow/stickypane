// Package prefs holds the app's own preferences that belong to no project.
// The shell reads and writes them in ~/.stickypane/overlay.json; this
// package only says what they mean.
package prefs

import "strings"

// Prefs is the part of overlay.json the frontend's language needs.
type Prefs struct {
	// Language is what the home window speaks: auto, ko or en.
	Language string `json:"language"`
}

// Default is Prefs before the user has chosen anything.
func Default() Prefs { return Prefs{Language: "auto"} }

// HomeLanguage is the language choice to use: a value that is not auto, ko
// or en (a missing or hand-edited field) comes to auto.
func (p Prefs) HomeLanguage() string {
	switch l := strings.ToLower(strings.TrimSpace(p.Language)); l {
	case "ko", "en", "auto":
		return l
	}
	return "auto"
}
