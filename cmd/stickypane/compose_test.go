package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/adapter/store"
)

// write --opt has the shape write the note from its create options, as
// the desktop app's create form does; a repeated option is several lines.
func TestWriteFromOptions(t *testing.T) {
	root := project(t)
	dir := filepath.Join(root, store.DirName)
	code, out, errOut := exec(t, "write", "release", "--type", "checklist", "--title", "Release", "--opt", "items=build", "--opt", "items=ship")
	if code != 0 || !strings.Contains(out, "release.md") {
		t.Fatalf("write: %d %q %q", code, out, errOut)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "release.md")); !strings.Contains(string(b), "- [ ] build\n- [ ] ship\n") || !strings.Contains(string(b), "type: checklist") {
		t.Errorf("release.md = %q", b)
	}
	if code, out, errOut := exec(t, "write", "build", "--type", "script", "--opt", "commands=go test ./..."); code != 0 || !strings.Contains(out, "build.sh") {
		t.Fatalf("script: %d %q %q", code, out, errOut)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "build.sh")); string(b) != "#!/bin/sh\ngo test ./...\n" {
		t.Errorf("build.sh = %q", b)
	}
	if code, _, errOut := exec(t, "write", "c", "--type", "chart", "--opt", "view=pie"); code == 0 || !strings.Contains(errOut, "pie") {
		t.Errorf("a bad option: %d %q", code, errOut)
	}
	if code, _, _ := exec(t, "write", "c", "--opt", "novalue"); code != exitUsage {
		t.Errorf("an option without = is a usage error: %d", code)
	}
}
