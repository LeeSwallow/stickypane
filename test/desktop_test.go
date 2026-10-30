package test

import (
	"strings"
	"testing"
)

// The desktop app takes a stickypane for one that has the bridge when
// `stickypane help --all` lists `stickypane bridge`
// (desktop/internal/bridgeclient's Probe). This pins that contract on the
// program's side: a change to the help that drops the line would make the
// app call every stickypane too old.
func TestHelpAllListsTheBridgeForTheDesktopApp(t *testing.T) {
	for _, lang := range []string{"en", "ko"} {
		if out := run(t, lang, "help", "--all"); !strings.Contains(out, "stickypane bridge") {
			t.Errorf("STICKYPANE_LANG=%s stickypane help --all does not list `stickypane bridge`:\n%s", lang, out)
		}
	}
}
