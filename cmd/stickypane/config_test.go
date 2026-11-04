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

// config lists the settings S changes first, then the rest under "more:".
func TestConfigListsThePanelsSettingsFirst(t *testing.T) {
	project(t)
	_, out, _ := exec(t, "config")
	first, more, ok := strings.Cut(out, "\nmore:\n")
	if !ok {
		t.Fatalf("no more: heading\n%s", out)
	}
	for _, key := range []string{"theme", "language", "columns"} {
		if !strings.Contains(first, key+" ") {
			t.Errorf("%s is listed first:\n%s", key, out)
		}
	}
	for _, key := range []string{"time", "editor", "stale"} {
		if !strings.Contains(more, key+" ") || strings.Contains(first, key+" ") {
			t.Errorf("%s is listed under more:\n%s", key, out)
		}
	}
}

// theme and language are short for config theme and config language: the
// overview leaves them out, and their help is config's.
func TestThemeAndLanguageAreShortForConfig(t *testing.T) {
	english(t)
	project(t)
	english(t)
	_, out, _ := exec(t, "help")
	if strings.Contains(out, "  theme ") || strings.Contains(out, "  language ") {
		t.Errorf("the overview lists theme or language:\n%s", out)
	}
	for _, name := range []string{"theme", "language"} {
		if code, out, _ := exec(t, "help", name); code != 0 || !strings.HasPrefix(out, "stickypane config - ") {
			t.Errorf("help %s: %d\n%s", name, code, out)
		}
	}
	if code, out, _ := exec(t, "theme", "nord"); code != 0 || out != "theme: nord\n" {
		t.Errorf("theme nord: %d %q", code, out)
	}
	if code, out, _ := exec(t, "config", "theme"); code != 0 || out != "nord\n" {
		t.Errorf("config theme after theme nord: %d %q", code, out)
	}
}
