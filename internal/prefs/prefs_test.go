package prefs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/store"
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
