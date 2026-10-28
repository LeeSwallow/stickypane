package prefs

import "testing"

func TestHomeLanguageDefaultsToAuto(t *testing.T) {
	if got := Default().Language; got != "auto" {
		t.Fatalf("default language = %q, want auto", got)
	}
	for in, want := range map[string]string{
		"": "auto", "auto": "auto", "ko": "ko", "en": "en", "KO": "ko", " en ": "en", "fr": "auto",
	} {
		if got := (Prefs{Language: in}).HomeLanguage(); got != want {
			t.Errorf("HomeLanguage(%q) = %q, want %q", in, got, want)
		}
	}
}
