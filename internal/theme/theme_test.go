package theme

import (
	"regexp"
	"strings"
	"testing"
)

var hex = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func TestEveryThemeIsComplete(t *testing.T) {
	if len(All()) < 6 {
		t.Fatalf("only %d themes", len(All()))
	}
	names := map[string]bool{}
	for _, th := range All() {
		if names[th.Name] {
			t.Errorf("theme %q twice", th.Name)
		}
		names[th.Name] = true
		for role, c := range th.roles() {
			if !hex.MatchString(c) {
				t.Errorf("%s: %s = %q is not a color", th.Name, role, c)
			}
		}
		for i, c := range th.Notes {
			if !hex.MatchString(c) {
				t.Errorf("%s: note color %d = %q", th.Name, i, c)
			}
		}
		if th.Source == "" {
			t.Errorf("%s: say where the palette comes from", th.Name)
		}
	}
	dark, light := 0, 0
	for _, th := range All() {
		if th.Dark {
			dark++
		} else {
			light++
		}
	}
	if dark < 4 || light < 2 {
		t.Errorf("want dark and light themes, got %d dark, %d light", dark, light)
	}
}

func TestLookupIsForgiving(t *testing.T) {
	for _, name := range []string{"Catppuccin Mocha", "catppuccin-mocha", "CATPPUCCIN_MOCHA", " catppuccin mocha "} {
		if th, ok := Lookup(name); !ok || th.Name != "catppuccin-mocha" {
			t.Errorf("Lookup(%q) = %q, %v", name, th.Name, ok)
		}
	}
	if _, ok := Lookup("nothing like this"); ok {
		t.Error("an unknown theme is not found")
	}
}

func TestAutoFollowsTheTerminal(t *testing.T) {
	if th := Pick("auto", true); !th.Dark || th.Name != DefaultDark {
		t.Errorf("auto on a dark terminal = %q", th.Name)
	}
	if th := Pick("auto", false); th.Dark || th.Name != DefaultLight {
		t.Errorf("auto on a light terminal = %q", th.Name)
	}
	if th := Pick("", false); th.Name != DefaultLight {
		t.Errorf("no choice is auto: %q", th.Name)
	}
	if th := Pick("nord", false); th.Name != "nord" {
		t.Errorf("a chosen theme is used whatever the terminal: %q", th.Name)
	}
	if th := Pick("no such theme", true); th.Name != DefaultDark {
		t.Errorf("an unknown theme falls back: %q", th.Name)
	}
}

func TestNextCycles(t *testing.T) {
	all := All()
	first := all[0].Name
	seen := map[string]bool{first: true}
	name := first
	for i := 0; i < len(all)-1; i++ {
		name = Next(name)
		if seen[name] {
			t.Fatalf("Next came back to %q after %d steps", name, i+1)
		}
		seen[name] = true
	}
	if Next(name) != first {
		t.Errorf("Next should wrap around to %q", first)
	}
	if Next("unknown") != first {
		t.Error("Next of an unknown theme starts over")
	}
}

// Markdown's colors come from the theme too: TestMarkdownFollowsTheTheme in
// internal/kinds.
func TestStylesComeFromTheTheme(t *testing.T) {
	th, _ := Lookup("tokyonight-night")
	st := th.Styles()
	if !strings.Contains(st.Accent.Render("x"), "x") || st.Good.Render("ok") == "ok" {
		t.Errorf("styles should color their text: %q", st.Good.Render("ok"))
	}
}
