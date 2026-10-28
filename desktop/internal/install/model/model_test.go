package model

import (
	"reflect"
	"testing"
)

func TestLinksGoToTheUsersBinOrTheSystemsOnlyWhenPicked(t *testing.T) {
	if got := LinkPath(TargetUser, "/home/a"); got != "/home/a/.local/bin/stickypane" {
		t.Errorf("user link = %s", got)
	}
	if got := LinkPath(ParseTarget("system"), "/home/a"); got != "/usr/local/bin/stickypane" {
		t.Errorf("system link = %s", got)
	}
	if ParseTarget("") != TargetUser || ParseTarget("/etc") != TargetUser {
		t.Error("anything but system is the user's folder")
	}
}

func TestIsAppsKnowsTheAppsOwnCopies(t *testing.T) {
	b := "/Applications/Stickypane.app/Contents/Helpers/stickypane"
	for real, want := range map[string]bool{
		b: true,
		"/Users/a/Downloads/Stickypane.app/Contents/Helpers/stickypane": true,
		"/opt/homebrew/Cellar/stickypane/0.1.1/bin/stickypane":          false,
		"/Users/a/go/bin/stickypane":                                    false,
		"":                                                              false,
	} {
		if got := IsApps(real, b); got != want {
			t.Errorf("IsApps(%q) = %v", real, got)
		}
	}
}

func TestFixForSaysHowTheCopyCame(t *testing.T) {
	cask := FixFor(Copy{Path: "/opt/homebrew/bin/stickypane", Real: "/opt/homebrew/Caskroom/stickypane/0.1.1/stickypane"}, "/h/.local/bin")
	if cask.Kind != "cask" || cask.Command != "brew uninstall --cask stickypane" || cask.After != "/opt/homebrew/bin" || cask.Before != "/h/.local/bin" {
		t.Errorf("cask: %+v", cask)
	}
	if f := FixFor(Copy{Path: "/usr/local/bin/stickypane", Real: "/usr/local/Cellar/stickypane/0.1.1/bin/stickypane"}, ""); f.Kind != "brew" || f.Command != "brew uninstall stickypane" {
		t.Errorf("formula: %+v", f)
	}
	if f := FixFor(Copy{Path: "/h/go bin/stickypane", Real: "/h/go bin/stickypane"}, ""); f.Kind != "remove" || f.Command != "rm '/h/go bin/stickypane'" {
		t.Errorf("by hand: %+v", f)
	}
}

func TestRCFileByShell(t *testing.T) {
	for shell, want := range map[string][2]string{
		"/bin/zsh":      {"/h/.zshrc", `export PATH="$HOME/.local/bin:$PATH"`},
		"/bin/bash":     {"/h/.bashrc", `export PATH="$HOME/.local/bin:$PATH"`},
		"/usr/bin/fish": {"/h/.config/fish/config.fish", "fish_add_path /h/.local/bin"},
		"":              {"/h/.profile", `export PATH="$HOME/.local/bin:$PATH"`},
	} {
		f, l := RCFile(shell, "/h", "/h/.local/bin")
		if f != want[0] || l != want[1] {
			t.Errorf("%q: %s %s", shell, f, l)
		}
	}
}

func TestScriptsQuoteTheirPaths(t *testing.T) {
	got := LinkScript("/Volumes/x y/Stickypane.app/Contents/Helpers/stickypane", "/usr/local/bin/stickypane")
	want := "mkdir -p /usr/local/bin && ln -sfn '/Volumes/x y/Stickypane.app/Contents/Helpers/stickypane' /usr/local/bin/stickypane"
	if got != want {
		t.Errorf("LinkScript =\n%s\nwant\n%s", got, want)
	}
	if got := ShQuote("it's"); got != `'it'\''s'` {
		t.Errorf("ShQuote = %s", got)
	}
	if got := UnlinkScript("/usr/local/bin/stickypane"); got != "rm -f /usr/local/bin/stickypane" {
		t.Errorf("UnlinkScript = %s", got)
	}
}

func TestSetupNeverRunsInTheRootOrHome(t *testing.T) {
	for dir, want := range map[string]bool{"/": false, "/h": false, "/h/": false, "": false, ".": false, "rel": false, "/tmp/x": true, "/h/p": true} {
		if got := SafeWorkDir(dir, "/h"); got != want {
			t.Errorf("SafeWorkDir(%q) = %v", dir, got)
		}
	}
}

func TestSetupArgs(t *testing.T) {
	if got := SetupArgs(true, true, "/a/stickypane"); !reflect.DeepEqual(got, []string{"setup", "--scope", "user", "--mcp", "--command", "/a/stickypane", "--dry-run"}) {
		t.Errorf("dry with mcp: %q", got)
	}
	if got := SetupArgs(false, false, "/a/stickypane"); !reflect.DeepEqual(got, []string{"setup", "--scope", "user", "--yes"}) {
		t.Errorf("run without mcp: %q", got)
	}
}

func TestReadProbeIgnoresWhatTheShellPrints(t *testing.T) {
	v, ok := ReadProbe("welcome!\n" + Marker + "/a:/b" + Marker + "bye")
	if !ok || v != "/a:/b" {
		t.Errorf("ReadProbe = %q %v", v, ok)
	}
	if _, ok := ReadProbe("nothing"); ok {
		t.Error("no marker reads as a PATH")
	}
	if got := SplitPath(" /a::/b "); !reflect.DeepEqual(got, []string{"/a", "/b"}) {
		t.Errorf("SplitPath = %q", got)
	}
	if got := JoinPath([]string{"/a", "/b"}, []string{"/b/", "/c"}); got != "/a:/b:/c" {
		t.Errorf("JoinPath = %q", got)
	}
	if !OnPath("/h/.local/bin/", []string{"/x", "/h/.local/bin"}) || OnPath("/y", []string{"/x"}) {
		t.Error("OnPath")
	}
}
