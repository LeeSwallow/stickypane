package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/adapter/store"
)

func TestPageAddAndList(t *testing.T) {
	root := project(t)
	docs := filepath.Join(root, store.DirName, "deploy", "docs")
	if err := os.MkdirAll(docs, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"01-intro.md", "02-usage.md"} {
		if err := os.WriteFile(filepath.Join(docs, f), []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if code, out, errOut := exec(t, "page", "add", "deploy/docs", "Release notes"); code != 0 || out != "added deploy/docs/release-notes.md\n" {
		t.Fatalf("page add: code = %d, out = %q, stderr = %q", code, out, errOut)
	}
	if b, err := os.ReadFile(filepath.Join(docs, "release-notes.md")); err != nil || !strings.HasSuffix(string(b), "Release notes\n") {
		t.Errorf("the page holds %q, %v", b, err)
	}
	code, out, errOut := exec(t, "page", "list", "deploy/docs")
	if code != 0 || !strings.Contains(out, "deploy/docs/01-intro.md") || strings.Index(out, "02-usage") > strings.Index(out, "release-notes") {
		t.Fatalf("page list: code = %d, out = %q, stderr = %q", code, out, errOut)
	}
	code, out, _ = exec(t, "page", "list", "deploy/docs", "--json")
	var pages []struct{ Name string }
	if code != 0 || json.Unmarshal([]byte(out), &pages) != nil || len(pages) != 3 || pages[2].Name != "deploy/docs/release-notes.md" {
		t.Errorf("page list --json: code = %d, %q", code, out)
	}
	for _, args := range [][]string{{"page", "add", "deploy/nope", "x"}, {"page", "list", "deploy"}, {"page", "add", "deploy", "x"}, {"page", "add"}} {
		if code, _, _ := exec(t, args...); code == 0 {
			t.Errorf("%v succeeded", args)
		}
	}
}
