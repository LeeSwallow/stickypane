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

func TestTheBoardSkillCarriesTheGuide(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot, "skills", "using-the-board", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	skill := string(b)
	if !strings.HasPrefix(skill, "---\nname: using-the-board\ndescription: ") {
		t.Fatalf("the skill needs a name and a description:\n%.120s", skill)
	}
	if !strings.Contains(skill, GuideText()) {
		t.Error("skills/using-the-board/SKILL.md should contain the guide word for word; regenerate it from `stickypane guide`")
	}
	for _, name := range []string{"asking-the-user", "tracking-progress"} {
		b, err := os.ReadFile(filepath.Join(repoRoot, "skills", name, "SKILL.md"))
		if err != nil || !strings.Contains(string(b), "name: "+name+"\n") {
			t.Errorf("skill %s: %v", name, err)
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
