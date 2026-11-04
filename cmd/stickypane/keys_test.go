package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// keysHome is a stickypane home of the test's own, which keys writes to;
// it does not exist yet.
func keysHome(t *testing.T) string {
	t.Helper()
	english(t)
	t.Chdir(t.TempDir())
	home := filepath.Join(t.TempDir(), "home")
	t.Setenv("STICKYPANE_HOME", home)
	return home
}

// keys lists every action with its keys and words, marking what keys.json
// changed; what it could not use goes to standard error.
func TestKeysListsEveryAction(t *testing.T) {
	home := keysHome(t)
	code, out, errOut := exec(t, "keys")
	if code != 0 || errOut != "" {
		t.Fatalf("code = %d, stderr = %q", code, errOut)
	}
	for _, re := range []string{`(?m)^note-archive +x +move to archive$`, `(?m)^jot +N n +jot a note$`, `(?m)^tab-3 +3 +tab 3$`} {
		if !regexp.MustCompile(re).MatchString(out) {
			t.Errorf("no line %s in:\n%s", re, out)
		}
	}
	if _, err := os.Stat(home); err == nil {
		t.Error("listing made the home folder")
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "keys.json"), []byte(`{"note-archive": ["X"], "nope": ["Q"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errOut = exec(t, "keys")
	if code != 0 || !regexp.MustCompile(`(?m)^note-archive +X +\(keys.json\) +move to archive$`).MatchString(out) {
		t.Errorf("code = %d:\n%s", code, out)
	}
	if !strings.Contains(errOut, "nope: there is no such action") {
		t.Errorf("stderr = %q", errOut)
	}
	_, out, _ = exec(t, "keys", "--json")
	var got []struct {
		Name    string
		Keys    []string
		Words   string
		Changed bool
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil || len(got) < 40 {
		t.Fatalf("%v: %s", err, out)
	}
	if got[0].Name != "focus-left" || got[0].Keys[0] != "h" || got[0].Changed {
		t.Errorf("%+v", got[0])
	}
	t.Setenv("STICKYPANE_LANG", "ko")
	if _, out, _ := exec(t, "keys"); !strings.Contains(out, "보관") {
		t.Errorf("the words follow the board's language:\n%s", out)
	}
}

// keys set writes keys.json, keeping its other entries, sorted; what the
// board could not use is refused and nothing is written. keys reset takes
// one change back, then the file.
func TestKeysSetAndReset(t *testing.T) {
	home := keysHome(t)
	file := filepath.Join(home, "keys.json")
	read := func() string {
		b, err := os.ReadFile(file)
		if err != nil {
			return ""
		}
		return string(b)
	}
	if code, out, errOut := exec(t, "keys", "set", "note-archive", "X", "shift+x", "ctrl+x"); code != 0 || out != "note-archive: X ctrl+x\n" {
		t.Fatalf("code = %d, out = %q, stderr = %q", code, out, errOut)
	}
	if code, _, _ := exec(t, "keys", "set", "jot"); code != 0 {
		t.Fatal("set with no keys")
	}
	want := "{\n  \"jot\": [],\n  \"note-archive\": [\n    \"X\",\n    \"ctrl+x\"\n  ]\n}\n"
	if got := read(); got != want {
		t.Errorf("keys.json:\n%s", got)
	}
	for _, c := range []struct {
		args []string
		why  string
	}{
		{[]string{"note-pin", "u"}, "u is undo's key too"},
		{[]string{"nope", "Q"}, "there is no such action"},
		{[]string{"note-pin", "1 g"}, "starts with a digit"},
		{[]string{"note-pin", "g esc"}, "goes on after esc"},
		{[]string{"note-pin", "ctrl+c"}, "always quits"},
	} {
		code, _, errOut := exec(t, append([]string{"keys", "set"}, c.args...)...)
		if code == 0 || !strings.Contains(errOut, c.why) {
			t.Errorf("keys set %q: code = %d, stderr = %q", c.args, code, errOut)
		}
	}
	if got := read(); got != want {
		t.Errorf("a refused set wrote keys.json:\n%s", got)
	}
	if code, out, _ := exec(t, "keys", "reset", "note-archive"); code != 0 || out != "note-archive: x\n" {
		t.Errorf("reset note-archive: %d %q", code, out)
	}
	if got := read(); got != "{\n  \"jot\": []\n}\n" {
		t.Errorf("keys.json:\n%s", got)
	}
	if code, out, _ := exec(t, "keys", "reset"); code != 0 || !strings.Contains(out, "removed") {
		t.Errorf("reset: %d %q", code, out)
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Errorf("keys.json is still there: %v", err)
	}
	if code, _, _ := exec(t, "keys", "reset", "nope"); code == 0 {
		t.Error("reset of no action")
	}
	if code, _, _ := exec(t, "keys", "frob"); code != exitUsage {
		t.Error("keys frob is a usage error")
	}
}

// A key with spaces is a sequence: keys.json keeps it with its spaces,
// and keys shows it as the board does, side by side.
func TestKeysSetASequence(t *testing.T) {
	home := keysHome(t)
	if code, out, errOut := exec(t, "keys", "set", "note-archive", "d x", "ㅌ"); code != 0 || out != "note-archive: dx x\n" {
		t.Fatalf("code = %d, out = %q, stderr = %q", code, out, errOut)
	}
	b, err := os.ReadFile(filepath.Join(home, "keys.json"))
	if err != nil || !strings.Contains(string(b), `"d x"`) {
		t.Errorf("keys.json: %s %v", b, err)
	}
	if _, out, _ := exec(t, "keys"); !regexp.MustCompile(`(?m)^note-archive +dx x +\(keys.json\)`).MatchString(out) {
		t.Errorf("keys:\n%s", out)
	}
}

// A broken keys.json is not overwritten by set; keys lists the defaults
// and says why; reset removes it.
func TestKeysWithABrokenFile(t *testing.T) {
	home := keysHome(t)
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(home, "keys.json")
	if err := os.WriteFile(file, []byte(`{"jot": [`), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errOut := exec(t, "keys")
	if code != 0 || !strings.Contains(errOut, "the keys are as they come") || !regexp.MustCompile(`(?m)^jot +N n `).MatchString(out) {
		t.Errorf("code = %d, stderr = %q:\n%s", code, errOut, out)
	}
	if code, _, errOut := exec(t, "keys", "set", "jot", "Y"); code == 0 || !strings.Contains(errOut, "keys reset") {
		t.Errorf("set over a broken file: %d %q", code, errOut)
	}
	if b, _ := os.ReadFile(file); string(b) != `{"jot": [` {
		t.Errorf("the broken file was written: %q", b)
	}
	if code, _, _ := exec(t, "keys", "reset"); code != 0 {
		t.Error("reset")
	}
}
