package i18n

import (
	"github.com/LeeSwallow/stickypane/internal/env"

	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestEnglishHasEveryString(t *testing.T) {
	v := reflect.ValueOf(English())
	for i := 0; i < v.NumField(); i++ {
		f := v.Type().Field(i)
		if f.Type.Kind() == reflect.String && v.Field(i).String() == "" {
			t.Errorf("English().%s is empty", f.Name)
		}
	}
}

func TestEveryTranslationLoadsOverEnglish(t *testing.T) {
	langs := Languages()
	if langs[0] != "en" || len(langs) < 2 {
		t.Fatalf("Languages = %v", langs)
	}
	for _, code := range langs {
		s, err := Load(code)
		if err != nil {
			t.Fatalf("%s: %v", code, err)
		}
		if s.Conflict == "" || s.Help == "" {
			t.Errorf("%s: a missing field should keep its English text", code)
		}
		for _, line := range strings.Split(s.Help, "\n") {
			if w := ansi.StringWidth(line); w > 40 {
				t.Errorf("%s: help line %q is %d cells wide, want at most 40", code, line, w)
			}
		}
	}
	if _, err := Load("xx"); err == nil {
		t.Error("a language without a translation is an error")
	}
}

func TestHelpNamesTheResizeKeys(t *testing.T) {
	for _, code := range Languages() {
		s, err := Load(code)
		if err != nil {
			t.Fatal(err)
		}
		for _, k := range []string{"+ -", "shift+up down", "shift+pgup pgdn", "0"} {
			listed := false
			for _, line := range strings.Split(s.Help, "\n") {
				listed = listed || strings.HasPrefix(strings.TrimSpace(line), k+"  ")
			}
			if !listed {
				t.Errorf("%s: the help should list %q", code, k)
			}
		}
		if !strings.Contains(s.Help, "\n  ") || strings.Count(s.Help, "\n") < strings.Count(English().Help, "\n") {
			t.Errorf("%s: the help should have every line the English one has", code)
		}
		for _, phrase := range []string{"width", "content height"} {
			if code != "en" && s.L(phrase) == phrase {
				t.Errorf("%s: %q is not translated", code, phrase)
			}
		}
	}
}

func TestKoreanTranslatesWhatTheUserSees(t *testing.T) {
	ko, err := Load("ko")
	if err != nil {
		t.Fatal(err)
	}
	if ko.NothingOpen == English().NothingOpen || ko.L("next") == "next" || ko.L("%d cards") == "%d cards" {
		t.Errorf("Korean should translate the screen: %q, %q, %q", ko.NothingOpen, ko.L("next"), ko.L("%d cards"))
	}
	if !strings.Contains(ko.Help, "tab") || !strings.Contains(ko.Help, "shift+tab") {
		t.Error("key names are never translated")
	}
	if got := Fill(ko.ConfirmDelete, map[string]any{"Name": "plan"}); !strings.Contains(got, "plan") || strings.Contains(got, "{{") {
		t.Errorf("Fill = %q", got)
	}
}

func TestDetectReadsTheEnvironment(t *testing.T) {
	for _, v := range []string{"STICKYPANE_LANG", "LC_ALL", "LC_MESSAGES", "LANG"} {
		t.Setenv(v, "")
	}
	// With no variable the system's own locale decides (macOS, Windows),
	// and English when it says nothing either.
	want := normalize(env.Detect().Locale)
	if want == "" {
		want = "en"
	}
	if got := Detect(here()); got != want {
		t.Errorf("nothing set = %q, want the system's %q", got, want)
	}
	t.Setenv("LANG", "ko_KR.UTF-8")
	if got := Detect(here()); got != "ko" {
		t.Errorf("LANG=ko_KR.UTF-8 = %q", got)
	}
	t.Setenv("LC_ALL", "C")
	if got := Detect(here()); got != "ko" {
		t.Errorf("C is not a language and does not hide LANG: %q", got)
	}
	t.Setenv("LC_ALL", "en_US.UTF-8")
	if got := Detect(here()); got != "en" {
		t.Errorf("LC_ALL wins over LANG: %q", got)
	}
	t.Setenv("STICKYPANE_LANG", "Ko-KR")
	if got := Detect(here()); got != "ko" {
		t.Errorf("STICKYPANE_LANG wins over all: %q", got)
	}
	if Pick("auto", here()).NothingOpen != Pick("ko", here()).NothingOpen || Pick("nope", here()).NothingOpen != Pick("ko", here()).NothingOpen || Pick("", here()).NothingOpen != Pick("ko", here()).NothingOpen {
		t.Error("Pick: auto, nothing and an unknown code all follow the environment")
	}
	t.Setenv("STICKYPANE_LANG", "en")
	if Pick("ko", here()).NothingOpen == English().NothingOpen {
		t.Error("a chosen language wins over the environment")
	}
}

// Resolve is the one rule for which language a choice comes to: the choice
// when there is a translation for it, else the environment's when there is
// one for that, else English. Pick speaks what Resolve says.
func TestResolveIsTheChoiceThenTheEnvironmentThenEnglish(t *testing.T) {
	for _, v := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		t.Setenv(v, "")
	}
	t.Setenv("STICKYPANE_LANG", "ko")
	for _, c := range []struct{ choice, want string }{
		{"ko", "ko"},
		{"KO_kr", "ko"},
		{"en", "en"},
		{"auto", "ko"},
		{"", "ko"},
		{"tlh", "ko"}, // no translation: the environment's
	} {
		if got := Resolve(c.choice, here()); got != c.want {
			t.Errorf("Resolve(%q) = %q, want %q", c.choice, got, c.want)
		}
		want, _ := Load(c.want)
		if got := Pick(c.choice, here()); got.NothingOpen != want.NothingOpen {
			t.Errorf("Pick(%q) does not speak %q", c.choice, c.want)
		}
	}
	t.Setenv("STICKYPANE_LANG", "fr") // no translation either: English
	if got := Resolve("tlh", here()); got != "en" {
		t.Errorf("Resolve with nothing translated = %q, want en", got)
	}
}

func TestFillLeavesABrokenTemplateReadable(t *testing.T) {
	if got := Fill("Delete {{.Name}}", nil); got != "Delete <no value>" && got != "Delete {{.Name}}" {
		t.Errorf("Fill with no values = %q", got)
	}
	if got := Fill("Delete {{.Name", map[string]any{"Name": "x"}); got != "Delete {{.Name" {
		t.Errorf("a broken template is returned as it is: %q", got)
	}
}

// here is the locale the environment names, read as the callers read it.
func here() string { return env.Detect().Locale }
