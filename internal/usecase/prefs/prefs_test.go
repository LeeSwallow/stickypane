package prefs

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/adapter/store"
)

func board(t *testing.T) *store.Store {
	t.Helper()
	dir := filepath.Join(t.TempDir(), store.DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return store.Open(dir)
}

// Every setting has a default, so nothing needs setting; what is set is
// kept in the root sticky.json.
func TestEverySettingHasADefault(t *testing.T) {
	st := board(t)
	for _, p := range All {
		if got := Get(st.Settings(), p.Key); got != p.Default || got == "" {
			t.Errorf("%s = %q, want its default %q", p.Key, got, p.Default)
		}
		if len(p.Values()) < 2 || p.Values()[0] != p.Default {
			t.Errorf("%s: its values start with the default: %q", p.Key, p.Values())
		}
	}
}

func TestSetKeepsAValueAndRefusesOthers(t *testing.T) {
	st := board(t)
	if err := Set(st, "stale", "1h"); err != nil {
		t.Fatal(err)
	}
	if got := Get(st.Settings(), "stale"); got != "1h" {
		t.Errorf("stale = %q", got)
	}
	if err := Set(st, "stale", "forever"); err == nil || !strings.Contains(err.Error(), "30m") {
		t.Errorf("an unknown value lists the values: %v", err)
	}
	if err := Set(st, "colour", "x"); err == nil || !strings.Contains(err.Error(), "theme") {
		t.Errorf("an unknown setting lists the settings: %v", err)
	}
	if err := Set(st, "stale", Lookup("stale").Default); err != nil || Get(st.Settings(), "stale") != "30m" {
		t.Errorf("setting the default back: %v", err)
	}
}

func TestNextGoesRoundTheValues(t *testing.T) {
	p := Lookup("stale")
	vals := p.Values()
	if p.Next(vals[len(vals)-1], 1) != vals[0] || p.Next(vals[0], -1) != vals[len(vals)-1] {
		t.Error("Next wraps around both ways")
	}
}

// One rule for every setting: the screen's view and style, the theme by any
// spelling of its name, a language that has a translation. Apply writes
// nothing when one value is refused, and a default leaves no trace.
func TestApplyChecksEveryChangeBeforeWritingAny(t *testing.T) {
	st := board(t)
	err := Apply(st, Change{"theme", "Nord"}, Change{"view", "hologram"})
	var refused *Refused
	if !errors.As(err, &refused) || refused.Key != "view" || err.Error() != `view "hologram" is not terminal, overlay or both` {
		t.Fatalf("a refused view: %v", err)
	}
	if st.Theme() != "" {
		t.Errorf("a refused change wrote the theme %q", st.Theme())
	}
	if err := Apply(st, Change{"theme", "Nord"}, Change{"view", "both"}, Change{"style", "NEAT"}, Change{"language", "KO"}); err != nil {
		t.Fatal(err)
	}
	if st.Theme() != "nord" || st.ViewMode() != "both" || st.Style() != "neat" || st.Language() != "ko" {
		t.Errorf("stored: %q %q %q %q", st.Theme(), st.ViewMode(), st.Style(), st.Language())
	}
	if err := Apply(st, Change{"theme", "auto"}, Change{"view", "terminal"}); err != nil {
		t.Fatal(err)
	}
	if v := st.Settings().Root("theme"); v != "" || Get(st.Settings(), "theme") != "auto" || Get(st.Settings(), "view") != "terminal" {
		t.Errorf("a default is no value: theme %q", v)
	}
}
