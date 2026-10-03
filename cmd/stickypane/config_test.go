package main

import (
	"strings"
	"testing"
)

// stickypane config lists every setting with its value and whether that is
// the default, prints one, or sets one.
func TestConfigListsGetsAndSets(t *testing.T) {
	project(t)
	code, out, _ := exec(t, "config")
	if code != 0 || !strings.Contains(out, "theme") || !strings.Contains(out, "stale") || !strings.Contains(out, "(default)") {
		t.Fatalf("config lists the settings: %d\n%s", code, out)
	}
	if code, out, _ := exec(t, "config", "stale", "1h"); code != 0 || out != "stale: 1h\n" {
		t.Errorf("set: %d %q", code, out)
	}
	if code, out, _ := exec(t, "config", "stale"); code != 0 || out != "1h\n" {
		t.Errorf("get: %d %q", code, out)
	}
	if code, _, errOut := exec(t, "config", "stale", "never"); code != 1 || !strings.Contains(errOut, "30m") {
		t.Errorf("a bad value lists the values: %d %q", code, errOut)
	}
}
