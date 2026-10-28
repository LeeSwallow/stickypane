package main

import (
	"os"
	"regexp"
	"testing"
)

// The app has one version: build/config.yml's info.version, which the
// macOS bundles repeat in CFBundleShortVersionString and CFBundleVersion
// (docs/overlay/09 §9.1).
func TestVersionsAgree(t *testing.T) {
	cfg, err := os.ReadFile("build/config.yml")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^info:\n(?:[ \t].*\n)*?[ \t]+version:\s*"([^"]+)"`).FindSubmatch(cfg)
	if m == nil {
		t.Fatal("build/config.yml has no info.version")
	}
	want := string(m[1])
	key := regexp.MustCompile(`<key>(CFBundleShortVersionString|CFBundleVersion)</key>\s*<string>([^<]*)</string>`)
	for _, p := range []string{"build/darwin/Info.plist", "build/darwin/Info.dev.plist"} {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		found := key.FindAllSubmatch(b, -1)
		if len(found) != 2 {
			t.Fatalf("%s: found %d of CFBundleShortVersionString and CFBundleVersion, want 2", p, len(found))
		}
		for _, f := range found {
			if string(f[2]) != want {
				t.Errorf("%s: %s is %s, build/config.yml says %s", p, f[1], f[2], want)
			}
		}
	}
}
