package app

import (
	"strings"
	"testing"
	"time"

	"github.com/LeeSwallow/stickypane/internal/prefs"
	"github.com/LeeSwallow/stickypane/internal/when"
)

// S opens the settings: every setting with its value, the defaults said
// as such. j and k pick one, l and h (or enter) change it, at once, and
// the change is kept in sticky.json; esc closes.
func TestTheSettingsPanelChangesWhatItSays(t *testing.T) {
	t.Cleanup(func() { when.UseLocale("") })
	m, _ := newModel(t, map[string]string{"a.md": opened("note\n")})
	press(m, "S")
	s := screen(m)
	if m.mode != modeSettings {
		t.Fatalf("S opens the settings:\n%s", s)
	}
	for _, want := range []string{"Settings", "Theme", "auto", "Time format", "Editor", "Cards stall after", "30m"} {
		if !strings.Contains(s, want) {
			t.Errorf("the panel shows %q:\n%s", want, s)
		}
	}
	press(m, "j", "j") // Time format
	press(m, "l")      // auto -> ko
	if got := prefs.Get(m.store.Settings(), "time"); got != "ko" {
		t.Fatalf("time = %q, want ko", got)
	}
	if got := when.Short(time.Date(2020, 1, 2, 3, 4, 0, 0, time.Local), time.Now()); got != "2020. 1. 2." {
		t.Errorf("the change applies at once: %q", got)
	}
	press(m, "h") // back to auto: the default leaves no trace
	if got := m.store.Settings().Root("time"); got != "" {
		t.Errorf("the default is not written: %q", got)
	}
	press(m, "esc")
	if m.mode != modeBoard {
		t.Errorf("esc closes the settings, mode = %v", m.mode)
	}
}
