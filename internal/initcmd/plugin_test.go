package initcmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// repoRoot is the repository, two levels up from this package.
var repoRoot = filepath.Join("..", "..")

// The repository is also a Claude Code and Codex plugin. These tests keep
// its files honest: the manifests parse, the skill that teaches the board
// carries the same guide init installs, and the hook script runs.

func TestPluginManifestsParse(t *testing.T) {
	for _, f := range []string{".claude-plugin/plugin.json", ".claude-plugin/marketplace.json", ".codex-plugin/plugin.json", ".agents/plugins/marketplace.json", "hooks/claude-hooks.json", "hooks/codex-hooks.json"} {
		b, err := os.ReadFile(filepath.Join(repoRoot, f))
		if err != nil {
			t.Fatal(err)
		}
		var v map[string]any
		if err := json.Unmarshal(b, &v); err != nil {
			t.Errorf("%s: %v", f, err)
		}
	}
	for _, f := range []string{"commands/show.md", "commands/status.md", "commands/ask.md", "commands/setup.md"} {
		b, err := os.ReadFile(filepath.Join(repoRoot, f))
		if err != nil || !strings.HasPrefix(string(b), "---\ndescription: ") {
			t.Errorf("%s should start with a description: %v", f, err)
		}
	}
}

func TestTheSkillsAreCompleteAndCarryTheGuide(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot, "skills", "using-the-board", "references", "formats.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), GuideText()) {
		t.Error("skills/using-the-board/references/formats.md should contain the guide word for word; regenerate it from `stickypane guide`")
	}
	for _, name := range []string{"using-the-board", "asking-the-user", "tracking-progress"} {
		dir := filepath.Join(repoRoot, "skills", name)
		b, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
		if err != nil || !strings.HasPrefix(string(b), "---\nname: "+name+"\ndescription: Use when") {
			t.Errorf("skill %s needs its name and a description that starts with when to use it: %v", name, err)
		}
		if !strings.Contains(string(b), "## Loading instructions") {
			t.Errorf("skill %s should say what to read for what", name)
		}
		for _, ref := range []string{"rules.md", "flow.md"} {
			if _, err := os.Stat(filepath.Join(dir, "references", ref)); err != nil {
				t.Errorf("skill %s is missing references/%s", name, ref)
			}
		}
	}
}

func TestSessionStartHookIsQuietWithoutABoard(t *testing.T) {
	script, err := filepath.Abs(filepath.Join(repoRoot, "scripts", "session-start.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh")
	}
	cmd := exec.Command("sh", script)
	cmd.Dir = t.TempDir()
	out, err := cmd.CombinedOutput()
	if err != nil || len(out) != 0 {
		t.Errorf("without a board the hook should say nothing: %q, %v", out, err)
	}
}
