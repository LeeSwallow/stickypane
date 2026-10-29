package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The README sections are the help's text, so they cannot drift from it:
// the getting started (stickypane help start) and the overview of the
// commands, which stickypane help prints after it where there is no board.
func TestReadmeUsageIsTheHelp(t *testing.T) {
	for file, lang := range map[string]string{"README.md": "en", "README.ko.md": "ko"} {
		b, err := os.ReadFile(filepath.Join(repoRoot, file))
		if err != nil {
			t.Fatal(err)
		}
		start := run(t, lang, "help", "start")
		overview, ok := strings.CutPrefix(run(t, lang, "help"), start)
		if !ok || strings.TrimSpace(start) == "" || strings.TrimSpace(overview) == "" {
			t.Fatalf("%s: without a board, stickypane help should open with stickypane help start", lang)
		}
		for _, text := range []string{start, overview} {
			if !strings.Contains(string(b), strings.TrimSpace(text)) {
				t.Errorf("%s does not contain the help text:\n%s", file, text)
			}
		}
	}
}
