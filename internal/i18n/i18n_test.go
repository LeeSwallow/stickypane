package i18n

import (
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
	if got := Detect(); got != "en" {
		t.Errorf("nothing set = %q", got)
	}
	t.Setenv("LANG", "ko_KR.UTF-8")
	if got := Detect(); got != "ko" {
		t.Errorf("LANG=ko_KR.UTF-8 = %q", got)
	}
	t.Setenv("LC_ALL", "C")
	if got := Detect(); got != "ko" {
		t.Errorf("C is not a language and does not hide LANG: %q", got)
	}
	t.Setenv("LC_ALL", "en_US.UTF-8")
	if got := Detect(); got != "en" {
		t.Errorf("LC_ALL wins over LANG: %q", got)
	}
	t.Setenv("STICKYPANE_LANG", "Ko-KR")
	if got := Detect(); got != "ko" {
		t.Errorf("STICKYPANE_LANG wins over all: %q", got)
	}
	if Pick("auto").NothingOpen != Pick("ko").NothingOpen || Pick("nope").NothingOpen != Pick("ko").NothingOpen || Pick("").NothingOpen != Pick("ko").NothingOpen {
		t.Error("Pick: auto, nothing and an unknown code all follow the environment")
	}
	t.Setenv("STICKYPANE_LANG", "en")
	if Pick("ko").NothingOpen == English().NothingOpen {
		t.Error("a chosen language wins over the environment")
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
