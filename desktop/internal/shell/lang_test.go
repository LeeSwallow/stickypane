package shell

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"testing"

	installsvc "github.com/LeeSwallow/stickypane/desktop/internal/install/service"
	"github.com/LeeSwallow/stickypane/desktop/internal/remote"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// "auto" is read as the terminal reads it: the variables in order, then
// the system.
func TestDetectLanguageReadsTheEnvironmentThenTheSystem(t *testing.T) {
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	sys := func(l string) func() string { return func() string { return l } }
	cases := []struct {
		env  map[string]string
		sys  string
		want string
	}{
		{map[string]string{"STICKYPANE_LANG": "ko", "LANG": "en_US.UTF-8"}, "", "ko"},
		{map[string]string{"LC_ALL": "C", "LANG": "ko_KR.UTF-8"}, "en_US", "ko"},
		{map[string]string{"LANG": "en_US.UTF-8"}, "ko_KR", "en"},
		{nil, "ko_KR", "ko"},
		{nil, "fr_FR", "en"},
		{nil, "", "en"},
	}
	for _, c := range cases {
		if got := detectLanguage(env(c.env), sys(c.sys)); got != c.want {
			t.Errorf("%v, system %q: %q, want %q", c.env, c.sys, got, c.want)
		}
	}
}

// The app's own words follow the home window's choice, and the system's
// for auto.
func TestLangIsTheChoiceOrTheSystems(t *testing.T) {
	r := newRig(t, func(c *Config) { c.SystemLanguage = func() string { return "ko" } })
	if got := r.s.Lang(); got != "ko" {
		t.Errorf("auto: %q, want the system's ko", got)
	}
	if err := r.s.SetPrefs(Prefs{Language: "en"}); err != nil {
		t.Fatal(err)
	}
	if got := r.s.Lang(); got != "en" {
		t.Errorf("chosen en: %q", got)
	}
	r.p.mu.Lock()
	said := r.p.lang
	r.p.mu.Unlock()
	if said != "en" {
		t.Errorf("the menu bar was told %q, want en", said)
	}
}

// Changing the language says the tray again, in the new one.
func TestTheTrayFollowsTheLanguage(t *testing.T) {
	r := newRig(t, func(c *Config) { c.SystemLanguage = func() string { return "en" } })
	r.s.Start()
	if err := r.s.SetPrefs(Prefs{Language: "ko"}); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "a Korean tray", func() bool {
		r.p.mu.Lock()
		defer r.p.mu.Unlock()
		return r.p.tray.Lang == "ko"
	})
	if !sameTray(Tray{Lang: "ko"}, Tray{Lang: "ko"}) || sameTray(Tray{Lang: "ko"}, Tray{Lang: "en"}) {
		t.Error("a tray in another language is another tray")
	}
	if home, quit := trayWords("ko"); home != "홈 열기" || quit != "stickypane 종료" {
		t.Errorf("ko tray: %q %q", home, quit)
	}
}

func TestTheSettingsWindowIsTitledInTheLanguage(t *testing.T) {
	r := newRig(t, func(c *Config) { c.SystemLanguage = func() string { return "ko" } })
	r.s.OpenSettings(0, "board")
	r.p.mu.Lock()
	defer r.p.mu.Unlock()
	var title string
	for _, w := range r.p.windows {
		if w.opts.Kind == KindSettings {
			title = w.opts.Title
		}
	}
	if title != "설정" {
		t.Errorf("settings title %q, want 설정", title)
	}
}

// A window is told what auto is here, so that it speaks what the tray does.
func TestAWindowIsToldTheSystemLanguage(t *testing.T) {
	u, err := url.Parse(WithSystemLanguage(pageSettings+"&section=keys", "ko"))
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if q.Get("syslang") != "ko" || q.Get("window") != "settings" || q.Get("section") != "keys" {
		t.Errorf("url %q", u)
	}
}

func TestTheMenuBarAndThePickerSpeakKorean(t *testing.T) {
	m := application.NewMenu()
	edit := m.AddSubmenu("Edit")
	edit.Add("Copy")
	edit.Add("Something of our own")
	relabel(m, menuKo)
	if l := m.ItemAt(0).Label(); l != "편집" {
		t.Errorf("Edit: %q", l)
	}
	if l := edit.ItemAt(0).Label(); l != "복사하기" {
		t.Errorf("Copy: %q", l)
	}
	if l := edit.ItemAt(1).Label(); l != "Something of our own" {
		t.Errorf("an item with no Korean keeps its words: %q", l)
	}
	if msg, button := pickWords("ko"); msg == "" || button != "선택" {
		t.Errorf("picker: %q %q", msg, button)
	}
	if _, button := pickWords("en"); button != "Choose" {
		t.Errorf("picker en: %q", button)
	}
}

// An error the shell knows reaches the window as a code and values, which
// the window says in its own language; its English stays for the log.
func TestKnownErrorsGoToTheWindowAsCodes(t *testing.T) {
	cases := []struct {
		err    error
		code   string
		values map[string]string
	}{
		{fmt.Errorf("%w (closed)", ErrNoBoard), "no_board", nil},
		{fmt.Errorf("%w: run stickypane init there", ErrRemote), "remote", nil},
		{fmt.Errorf("%w in the root folder", ErrUnsafeFolder), "unsafe_folder", nil},
		{fmt.Errorf("%w: /", installsvc.ErrUnsafeDir), "unsafe_folder", nil},
		{noHost("dev"), "no_host", map[string]string{"name": "dev"}},
		{hostTaken("box"), "host_taken", map[string]string{"name": "box"}},
		{hostOpen("dev"), "host_open", map[string]string{"name": "dev"}},
		{ErrNoAutostart, "no_autostart", nil},
		{remote.ValidName("a b"), "host_name", map[string]string{"value": "a b"}},
		{remote.ValidPath("rel"), "folder_absolute", map[string]string{"value": "rel"}},
	}
	for _, c := range cases {
		w, ok := WordsOf(c.err)
		if !ok || w.Code != c.code || fmt.Sprint(w.Values) != fmt.Sprint(c.values) {
			t.Errorf("%v: %+v %v, want %s %v", c.err, w, ok, c.code, c.values)
		}
		var back ErrorWords
		if err := json.Unmarshal(MarshalError(c.err), &back); err != nil || back.Code != c.code {
			t.Errorf("%v marshals to %+v (%v)", c.err, back, err)
		}
	}
	if _, ok := WordsOf(errors.New("exit status 1")); ok {
		t.Error("an error the shell does not know has no code")
	}
	if MarshalError(errors.New("exit status 1")) != nil {
		t.Error("an unknown error is marshalled as Wails does")
	}
}
