// Package theme is the design system: a small set of color roles that every
// part of the screen draws with, and the palettes that fill them. A theme
// never sets the background: the terminal's own stays, so a theme is a set
// of foreground colors that go well with a dark or a light one.
package theme

import (
	"sort"
	"strings"

	"charm.land/glamour/v2/ansi"
	"charm.land/glamour/v2/styles"
	"charm.land/lipgloss/v2"
)

// Theme is one palette, by role. Every color is a "#rrggbb" string.
type Theme struct {
	Name   string
	Dark   bool   // made for a dark terminal
	Source string // where the colors come from, with the license

	Base   string // the background the theme is made for; text on an accent ribbon
	Text   string // text drawn in a color of its own, such as a selection
	Muted  string // hints, details, rules, inactive borders
	Accent string // focus: the focused tab and border, buttons, scroll bars
	Select string // the background of the selected line and tab
	Good   string // done, success
	Warn   string // warnings, messages that need a look
	Bad    string // errors
	Info   string // links, headings

	// Notes are the colors a note can have: yellow, pink, blue, green,
	// purple, orange, in that order.
	Notes [6]string
}

// NoteNames are the names of the note colors, in the order of Notes.
var NoteNames = [6]string{"yellow", "pink", "blue", "green", "purple", "orange"}

// roles returns the single colors by role name, for checks.
func (t Theme) roles() map[string]string {
	return map[string]string{
		"base": t.Base, "text": t.Text, "muted": t.Muted, "accent": t.Accent, "select": t.Select,
		"good": t.Good, "warn": t.Warn, "bad": t.Bad, "info": t.Info,
	}
}

// Styles is what the screen draws with, made from a theme.
type Styles struct {
	Bold, Faint, Struck     lipgloss.Style
	Selected                lipgloss.Style // the selected line
	Accent, Good, Warn, Bad lipgloss.Style
	Info                    lipgloss.Style
	Border                  lipgloss.Style // an unfocused frame
	Key, Label              lipgloss.Style // a key hint and what it does
}

// Styles makes the styles of the theme.
func (t Theme) Styles() Styles {
	c := func(hex string) lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color(hex)) }
	return Styles{
		Bold:     lipgloss.NewStyle().Bold(true),
		Faint:    c(t.Muted),
		Struck:   c(t.Muted).Strikethrough(true),
		Selected: lipgloss.NewStyle().Background(lipgloss.Color(t.Select)).Foreground(lipgloss.Color(t.Text)),
		Accent:   c(t.Accent),
		Good:     c(t.Good),
		Warn:     c(t.Warn),
		Bad:      c(t.Bad),
		Info:     c(t.Info),
		Border:   c(t.Muted),
		Key:      c(t.Accent).Bold(true),
		Label:    c(t.Muted),
	}
}

// Markdown makes the Glamour style of the theme: the stock dark or light
// style with the theme's colors in the roles that show, and without the
// page margin, so short notes stay compact.
func (t Theme) Markdown() ansi.StyleConfig {
	cfg := styles.LightStyleConfig
	if t.Dark {
		cfg = styles.DarkStyleConfig
	}
	var zero uint
	cfg.Document.Margin = &zero
	cfg.Document.BlockPrefix = ""
	cfg.Document.BlockSuffix = ""
	cfg.Document.Color = nil // the terminal's own text color
	set := func(p *ansi.StylePrimitive, hex string) {
		h := hex
		p.Color = &h
	}
	set(&cfg.Heading.StylePrimitive, t.Info)
	set(&cfg.H1.StylePrimitive, t.Info)
	cfg.H1.BackgroundColor = nil
	set(&cfg.H2.StylePrimitive, t.Info)
	set(&cfg.H3.StylePrimitive, t.Info)
	set(&cfg.H4.StylePrimitive, t.Info)
	set(&cfg.H5.StylePrimitive, t.Info)
	set(&cfg.H6.StylePrimitive, t.Muted)
	set(&cfg.Link, t.Accent)
	set(&cfg.LinkText, t.Accent)
	set(&cfg.Code.StylePrimitive, t.Warn)
	cfg.Code.BackgroundColor = nil
	set(&cfg.BlockQuote.StylePrimitive, t.Muted)
	set(&cfg.HorizontalRule, t.Muted)
	set(&cfg.Enumeration, t.Accent)
	set(&cfg.Item, t.Accent)
	set(&cfg.Emph, t.Text)
	set(&cfg.Strong, t.Text)
	set(&cfg.Table.StylePrimitive, t.Text)
	return cfg
}

// Default names the theme used when the user chose none.
const (
	DefaultDark  = "stickypane-dark"
	DefaultLight = "stickypane-light"
)

// Lookup finds a theme by name. Case, spaces, "-" and "_" do not matter.
func Lookup(name string) (Theme, bool) {
	want := key(name)
	for _, t := range all {
		if key(t.Name) == want {
			return t, true
		}
	}
	return Theme{}, false
}

func key(name string) string {
	return strings.NewReplacer(" ", "", "-", "", "_", "").Replace(strings.ToLower(strings.TrimSpace(name)))
}

// Pick returns the theme for a choice: the named one, or for "auto" (or no
// choice, or a name that is not a theme) the default for a dark or a light
// terminal.
func Pick(choice string, dark bool) Theme {
	if t, ok := Lookup(choice); ok && key(choice) != "auto" {
		return t
	}
	name := DefaultLight
	if dark {
		name = DefaultDark
	}
	t, _ := Lookup(name)
	return t
}

// Next returns the theme after the named one, wrapping around, for a key
// that cycles through them.
func Next(name string) string {
	list := All()
	for i, t := range list {
		if key(t.Name) == key(name) {
			return list[(i+1)%len(list)].Name
		}
	}
	return list[0].Name
}

// All returns every theme, dark ones first, by name.
func All() []Theme {
	out := append([]Theme(nil), all...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Dark != out[j].Dark {
			return out[i].Dark
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// Holder is the theme in use, shared by the parts of the program that draw.
type Holder struct{ t Theme }

// NewHolder returns a holder with the theme.
func NewHolder(t Theme) *Holder { return &Holder{t} }

// Get returns the theme in use.
func (h *Holder) Get() Theme { return h.t }

// Set changes the theme in use.
func (h *Holder) Set(t Theme) { h.t = t }
