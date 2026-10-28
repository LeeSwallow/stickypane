package shell

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestChromeIsTrafficLightsOnMacAndOurOwnElsewhere(t *testing.T) {
	if got := ChromeFor("darwin"); got != ChromeMac {
		t.Errorf("darwin: %q", got)
	}
	if got := ChromeFor("linux"); got != ChromeOwn {
		t.Errorf("linux: %q", got)
	}
	cases := map[string]string{
		"/":                 "/?chrome=mac",
		"/?window=project":  "/?chrome=mac&window=project",
		"/?window=project&": "/?chrome=mac&window=project",
	}
	for in, want := range cases {
		if got := WithChrome(in, ChromeMac); got != want {
			t.Errorf("WithChrome(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTheMenuBarHasTheAppMenuWithQuit(t *testing.T) {
	// Wails' AppMenu role carries the standard Quit item (Cmd+Q).
	if !slices.Contains(AppMenuRoles, application.AppMenu) {
		t.Error("no app menu: Cmd+Q would not quit")
	}
	if !slices.Contains(AppMenuRoles, application.EditMenu) {
		t.Error("no edit menu: copy and paste in fields need it on macOS")
	}
}

func TestTrayAndMenuSayQuitStickypane(t *testing.T) {
	if _, quit := trayWords("ko"); quit != "stickypane 종료" {
		t.Errorf("ko quit = %q", quit)
	}
	if _, quit := trayWords("en"); quit != "Quit stickypane" {
		t.Errorf("en quit = %q", quit)
	}
}

func TestQuitEndsTheApp(t *testing.T) {
	r := newRig(t)
	(&ProjectService{S: r.s}).Quit()
	r.p.mu.Lock()
	quit := r.p.quit
	r.p.quit = false
	r.p.mu.Unlock()
	if !quit {
		t.Error("ProjectService.Quit did not quit")
	}
	(&HomeService{S: r.s}).Quit()
	r.p.mu.Lock()
	defer r.p.mu.Unlock()
	if !r.p.quit {
		t.Error("HomeService.Quit did not quit")
	}
}

func trayNotices(r *rig) int {
	n := 0
	for _, e := range r.p.emitted(NoticesEvent) {
		for _, x := range e.([]Notice) {
			if x.Code == NoticeStayInTray {
				n++
			}
		}
	}
	return n
}

func TestClosingAWindowSaysOnceThatTheAppStaysInTheTray(t *testing.T) {
	r := newRig(t)
	a := filepath.Join(r.home, "a")
	r.s.OpenProject(a)
	r.p.projectWindows()[0].Close()

	if n := trayNotices(r); n != 1 {
		t.Fatalf("%d tray notices after the first close, want 1", n)
	}
	// nothing else was open: the home comes forward to say it
	if hs := r.p.homeWindows(); len(hs) != 1 || !hs[0].Visible() {
		t.Error("the home window did not come forward with the notice")
	}
	st, err := LoadState(filepath.Join(r.home, StateFile))
	if err != nil {
		t.Fatal(err)
	}
	if !st.TrayHint {
		t.Error("the hint is not remembered in overlay.json")
	}

	r.s.OpenProject(a)
	r.p.projectWindows()[1].Close()
	if n := trayNotices(r); n != 1 {
		t.Errorf("the hint came again: %d notices", n)
	}
}

func TestTheTrayHintIsNotShownAgainAfterARestart(t *testing.T) {
	r := newRig(t)
	if err := (State{Hotkey: DefaultHotkey, TrayHint: true}).Save(filepath.Join(r.home, StateFile)); err != nil {
		t.Fatal(err)
	}
	r2 := newRig(t, func(c *Config) { c.Home = r.home })
	r2.s.OpenProject(filepath.Join(r.home, "a"))
	r2.p.projectWindows()[0].Close()
	if n := trayNotices(r2); n != 0 {
		t.Errorf("%d tray notices, want none: it was shown before", n)
	}
}

func TestClosingTheHomeFirstSaysItInTheHome(t *testing.T) {
	r := newRig(t)
	r.s.ShowHome()
	h := r.p.homeWindows()[0]
	h.Close() // the fake closes; Wails would hide it
	waitFor(t, "the tray notice", func() bool { return trayNotices(r) == 1 })
	waitFor(t, "the home again", func() bool {
		hs := r.p.homeWindows()
		return len(hs) > 0 && hs[len(hs)-1].Visible()
	})
}
