package model

import (
	"errors"
	"slices"
	"testing"
)

func TestParseKeysReadsTheListAndWhatWasLeftOut(t *testing.T) {
	out := "keys.json: nope: there is no such action; it keeps the keys it comes with\n" +
		`[{"name":"note-archive","keys":["X"],"words":"move to archive","changed":true,"defaults":["x"],"group":"notes"},` +
		`{"name":"jot","keys":["N","n"],"words":"jot a note","changed":false}]` + "\n"
	bs, problems, err := ParseKeys(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(bs) != 2 || bs[0].Name != "note-archive" || !bs[0].Changed || !slices.Equal(bs[0].Defaults, []string{"x"}) || bs[0].Group != "notes" {
		t.Errorf("%+v", bs)
	}
	if bs[1].Group != "" || bs[1].Defaults != nil {
		t.Errorf("an older stickypane has no groups: %+v", bs[1])
	}
	if len(problems) != 1 || problems[0] != "keys.json: nope: there is no such action; it keeps the keys it comes with" {
		t.Errorf("problems %q", problems)
	}
}

func TestParseKeysOfAStickypaneWithoutKeys(t *testing.T) {
	for _, out := range []string{"stickypane: unknown command keys\nusage: ...", "[not json"} {
		if _, _, err := ParseKeys(out); !errors.Is(err, ErrNoKeys) {
			t.Errorf("%q: %v", out, err)
		}
	}
}

func TestKeysArguments(t *testing.T) {
	if got := SetKeysArgs("note-narrower", []string{"-", "g g"}); !slices.Equal(got, []string{"keys", "set", "note-narrower", "--", "-", "g g"}) {
		t.Errorf("%q", got)
	}
	if got := SetKeysArgs("theme-next", nil); !slices.Equal(got, []string{"keys", "set", "theme-next", "--"}) {
		t.Errorf("unbound: %q", got)
	}
	if got := ResetKeysArgs(""); !slices.Equal(got, []string{"keys", "reset"}) {
		t.Errorf("%q", got)
	}
	if got := ResetKeysArgs("jot"); !slices.Equal(got, []string{"keys", "reset", "jot"}) {
		t.Errorf("%q", got)
	}
}

func TestRunErrorIsTheFirstSentence(t *testing.T) {
	out := "stickypane: note-archive: \"q\" is quit's key too; keys.json is left as it was\n"
	if got := RunError(out); got != `note-archive: "q" is quit's key too; keys.json is left as it was` {
		t.Errorf("%q", got)
	}
	if got := RunError("stickypane: unknown command keys\nusage: stickypane ..."); got != "unknown command keys" {
		t.Errorf("%q", got)
	}
}
