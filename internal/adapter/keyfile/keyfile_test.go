package keyfile

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPathIsInTheHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("STICKYPANE_HOME", home)
	if got := Path(); got != filepath.Join(home, "keys.json") {
		t.Errorf("Path = %q", got)
	}
}

// A missing file asks for nothing; a written one reads back as it was,
// sorted and indented, with [] for an action unbound.
func TestWriteReadsBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "home", "keys.json")
	if keys, err := Read(path); err != nil || keys != nil {
		t.Fatalf("missing: %v, %v", keys, err)
	}
	want := map[string][]string{"undo": {"U"}, "note-archive": {"X", "ctrl+x"}, "jot": nil}
	if err := Write(path, want); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	if s := string(b); s != "{\n  \"jot\": [],\n  \"note-archive\": [\n    \"X\",\n    \"ctrl+x\"\n  ],\n  \"undo\": [\n    \"U\"\n  ]\n}\n" {
		t.Errorf("the file:\n%s", s)
	}
	got, err := Read(path)
	want["jot"] = []string{}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("read back %v, %v", got, err)
	}
	if err := Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := Remove(path); err != nil {
		t.Errorf("removing it again: %v", err)
	}
}

func TestABrokenFileIsAnError(t *testing.T) {
	dir := t.TempDir()
	for _, body := range []string{`{"jot": ["N"`, `{"jot": "N"}`, `["jot"]`, `{} {}`} {
		path := filepath.Join(dir, "keys.json")
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Read(path); err == nil || !strings.Contains(err.Error(), "key lists") {
			t.Errorf("%s: err = %v", body, err)
		}
	}
}

func TestShownStartsWithATilde(t *testing.T) {
	h, err := os.UserHomeDir()
	if err != nil {
		t.Skip(err)
	}
	if got := Shown(filepath.Join(h, ".stickypane", "keys.json")); got != filepath.Join("~", ".stickypane", "keys.json") {
		t.Errorf("Shown = %q", got)
	}
	if got := Shown("/elsewhere/keys.json"); got != "/elsewhere/keys.json" {
		t.Errorf("Shown = %q", got)
	}
}
