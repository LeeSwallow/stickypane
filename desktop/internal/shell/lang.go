package shell

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

// The app's own words — the tray, the menu bar, the folder picker, a
// window's title — are in the home window's language: "ko" or "en", or
// with "auto" the system's, read as the terminal reads it (the board's
// "auto" is the same): STICKYPANE_LANG, LC_ALL, LC_MESSAGES, LANG, then
// what the system is set to. The windows are told the system's answer
// (WithSystemLanguage), so that every part of the app says one language.

// localeVars are read in order; the first that names a language wins.
var localeVars = []string{"STICKYPANE_LANG", "LC_ALL", "LC_MESSAGES", "LANG"}

// systemLocaleTimeout bounds asking macOS for its locale.
const systemLocaleTimeout = 2 * time.Second

// detectLanguage is "ko" or "en" for the environment getenv reads, then
// for the system's locale.
func detectLanguage(getenv func(string) string, system func() string) string {
	for _, v := range localeVars {
		if l := langOf(getenv(v)); l != "" {
			return l
		}
	}
	if l := langOf(system()); l != "" {
		return l
	}
	return "en"
}

// langOf is "ko" for a Korean locale ("ko_KR.UTF-8", "ko-KR", "ko"), "en"
// for any other, and "" for none ("", "C", "POSIX").
func langOf(locale string) string {
	l := strings.TrimSpace(locale)
	if i := strings.IndexAny(l, ".@"); i >= 0 {
		l = l[:i]
	}
	l = strings.ToLower(strings.ReplaceAll(l, "_", "-"))
	switch {
	case l == "" || l == "c" || l == "posix":
		return ""
	case l == "ko" || strings.HasPrefix(l, "ko-"):
		return "ko"
	}
	return "en"
}

// systemLocale is what macOS is set to ("ko_KR"); "" elsewhere.
func systemLocale() string {
	if runtime.GOOS != "darwin" {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), systemLocaleTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "defaults", "read", "-g", "AppleLocale").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

var (
	systemLangOnce sync.Once
	systemLang     string
)

// SystemLanguage is "ko" or "en": what "auto" comes to on this computer,
// read once.
func SystemLanguage() string {
	systemLangOnce.Do(func() { systemLang = detectLanguage(os.Getenv, systemLocale) })
	return systemLang
}

// Lang is the language of the app's own words: the home window's choice,
// or the system's for "auto".
func (s *Shell) Lang() string {
	if l := s.Prefs().Language; l == "ko" || l == "en" {
		return l
	}
	if s.cfg.SystemLanguage != nil {
		return s.cfg.SystemLanguage()
	}
	return SystemLanguage()
}

// words picks the English or the Korean of a phrase for lang.
func words(lang, en, ko string) string {
	if lang == "ko" {
		return ko
	}
	return en
}
