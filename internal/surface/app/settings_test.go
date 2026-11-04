package app

import (
	"strings"
	"testing"
	"time"

	"github.com/LeeSwallow/stickypane/internal/usecase/prefs"
)

// S opens the settings: theme, language and columns, each with its value,
// the defaults said as such. j and k pick one, l and h (or enter) change
// it, at once, and the change is kept in sticky.json; esc closes.
func TestTheSettingsPanelChangesWhatItSays(t *testing.T) {
	t.Parallel()
	m, _ := newModel(t, map[string]string{"a.md": opened("note\n")})
	press(m, "S")
	s := screen(m)
	if m.mode != modeSettings {
		t.Fatalf("S opens the settings:\n%s", s)
	}
	for _, want := range []string{"Settings", "Theme", "auto", "Language", "Notes per row", "stickypane config"} {
		if !strings.Contains(s, want) {
			t.Errorf("the panel shows %q:\n%s", want, s)
		}
	}
	press(m, "j", "j") // Notes per row
	press(m, "l")      // 3 -> 4
	if got := prefs.Get(m.store.Settings(), "columns"); got != "4" || m.columns != 4 {
		t.Fatalf("columns = %q (%d on the screen), want 4", got, m.columns)
	}
	press(m, "h") // back to 3: the default leaves no trace
	if got := m.store.Settings().Root("columns"); got != "" {
		t.Errorf("the default is not written: %q", got)
	}
	press(m, "esc")
	if m.mode != modeBoard {
		t.Errorf("esc closes the settings, mode = %v", m.mode)
	}
}

// The time format, the editor and when a card stalls are left off the
// panel, and still apply when stickypane config sets them.
func TestSettingsOffThePanelStillApply(t *testing.T) {
	t.Parallel()
	m, _ := newModel(t, map[string]string{"a.md": opened("note\n")})
	press(m, "S")
	s := screen(m)
	for _, gone := range []string{"Time format", "Editor", "Cards stall after"} {
		if strings.Contains(s, gone) {
			t.Errorf("the panel leaves off %q:\n%s", gone, s)
		}
	}
	if err := prefs.Set(m.store, "time", "ko"); err != nil {
		t.Fatal(err)
	}
	if err := prefs.Set(m.store, "stale", "1h"); err != nil {
		t.Fatal(err)
	}
	m.reload()
	if got := m.env.Short(time.Date(2020, 1, 2, 3, 4, 0, 0, time.Local)); got != "2020. 1. 2." {
		t.Errorf("the time format applies: %q", got)
	}
	if m.env.StaleAfter != time.Hour {
		t.Errorf("when a card stalls applies: %v", m.env.StaleAfter)
	}
}
