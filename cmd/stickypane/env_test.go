package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// `stickypane env` says what the board found about where it runs and what
// follows from it: how to open the board beside the agent, which language
// the screen speaks and why, and how scripts run. It needs no board.
func TestEnvSaysWhatItFound(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("WEZTERM_PANE", "3")
	t.Setenv("SHELL", "/bin/zsh")
	t.Setenv("STICKYPANE_LANG", "ko")
	code, out, errOut := exec(t, "env")
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, errOut)
	}
	for _, want := range []string{"system", "shell", "zsh", "wezterm cli split-pane --right -- stickypane", "ko", "STICKYPANE_LANG", "export STICKYPANE_LANG=", ".sh", ".ps1"} {
		if !strings.Contains(out, want) {
			t.Errorf("env should say %q:\n%s", want, out)
		}
	}
	code, out, _ = exec(t, "env", "--json")
	var got map[string]any
	if code != 0 || json.Unmarshal([]byte(out), &got) != nil || got["pane"] != "wezterm" || got["language"] != "ko" {
		t.Errorf("env --json = %d %s", code, out)
	}
}
