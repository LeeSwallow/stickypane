package initcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMain points the agents' home folders at empty places, so that the
// plugins installed on the machine running the tests do not change what
// init does. Tests that need a plugin install it with claudePlugin or
// codexPlugin.
func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "initcmd-home")
	if err != nil {
		panic(err)
	}
	os.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(tmp, "claude"))
	os.Setenv("CODEX_HOME", filepath.Join(tmp, "codex"))
	os.Unsetenv("TMUX")
	code := m.Run()
	os.RemoveAll(tmp)
	os.Exit(code)
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// claudePlugin installs the board plugin for Claude Code, for every project
// (projectPath "") or for one.
func claudePlugin(t *testing.T, projectPath string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	entry := `{"scope": "user", "installPath": "/x", "version": "0.1.0"}`
	if projectPath != "" {
		entry = fmt.Sprintf(`{"scope": "project", "projectPath": %q, "installPath": "/x"}`, projectPath)
	}
	write(t, filepath.Join(dir, "plugins", "installed_plugins.json"),
		`{"version": 2, "plugins": {"other@market": [{"scope": "user"}], "board@stickypane": [`+entry+`]}}`)
}

// codexPlugin installs the board plugin for Codex, with the given lines
// under its table in config.toml.
func codexPlugin(t *testing.T, lines string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("CODEX_HOME", dir)
	write(t, filepath.Join(dir, "config.toml"), "model = \"x\"\n\n[plugins.\"board@stickypane\"]\n"+lines+"\n[plugins.\"other@market\"]\nenabled = false\n")
}

func TestRunLeavesClaudeMDToTheClaudePlugin(t *testing.T) {
	root := t.TempDir()
	claudePlugin(t, "")
	claude := filepath.Join(root, "CLAUDE.md")
	write(t, claude, "# Rules\n\nUse tabs.\n\n"+Guide())
	out := run(t, root, Options{})
	if got := read(t, claude); got != "# Rules\n\nUse tabs.\n" {
		t.Errorf("the guide should come out of CLAUDE.md when the plugin teaches the board:\n%q", got)
	}
	if exists(filepath.Join(root, "AGENTS.md")) {
		t.Error("with the Claude plugin, AGENTS.md should not be created")
	}
	if !strings.Contains(out, "plugin") {
		t.Errorf("the output should say why CLAUDE.md was left alone: %q", out)
	}
}

func TestRunRemovesTheSkillWhenThePluginIsInstalled(t *testing.T) {
	root := t.TempDir()
	run(t, root, Options{Skill: true})
	claudePlugin(t, "")
	run(t, root, Options{})
	if exists(filepath.Join(root, skillPath)) {
		t.Error("the project skill duplicates the plugin and should be removed")
	}
}

func TestRunCountsAPluginInstalledForThisProjectOnly(t *testing.T) {
	root := t.TempDir()
	claudePlugin(t, root)
	claude := filepath.Join(root, "CLAUDE.md")
	write(t, claude, "rules\n")
	run(t, root, Options{})
	if got := read(t, claude); got != "rules\n" {
		t.Errorf("a plugin installed for this project teaches the board: %q", got)
	}
}

func TestRunIgnoresAPluginInstalledForAnotherProject(t *testing.T) {
	root := t.TempDir()
	claudePlugin(t, filepath.Join(root, "elsewhere"))
	claude := filepath.Join(root, "CLAUDE.md")
	write(t, claude, "rules\n")
	run(t, root, Options{})
	if got := read(t, claude); !strings.Contains(got, Guide()) {
		t.Errorf("a plugin of another project does not teach this one: %q", got)
	}
}

func TestRunLeavesAgentsMDToTheCodexPlugin(t *testing.T) {
	root := t.TempDir()
	codexPlugin(t, "enabled = true\n")
	agents := filepath.Join(root, "AGENTS.md")
	write(t, agents, Guide())
	run(t, root, Options{})
	if exists(agents) {
		t.Errorf("an AGENTS.md that held only the guide should go: %q", read(t, agents))
	}
}

func TestRunTreatsADisabledCodexPluginAsMissing(t *testing.T) {
	root := t.TempDir()
	codexPlugin(t, "enabled = false\n")
	run(t, root, Options{})
	if got := read(t, filepath.Join(root, "AGENTS.md")); got != Guide() {
		t.Errorf("AGENTS.md = %q", got)
	}
}

func TestRunWritesASkillForAClaudeProjectWithoutCLAUDEMD(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	run(t, root, Options{})
	if skill := read(t, filepath.Join(root, skillPath)); skill != Skill() {
		t.Errorf("Claude Code reads skills, not AGENTS.md; skill = %.100q", skill)
	}
	if exists(filepath.Join(root, "AGENTS.md")) {
		t.Error("AGENTS.md should not be created when the guide went to a skill")
	}
}

func TestRunWritesBothFilesWhenBothExist(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "CLAUDE.md"), "c\n")
	write(t, filepath.Join(root, "AGENTS.md"), "a\n")
	out := run(t, root, Options{})
	for _, f := range []string{"CLAUDE.md", "AGENTS.md"} {
		if !strings.Contains(read(t, filepath.Join(root, f)), Guide()) {
			t.Errorf("%s should carry the guide", f)
		}
	}
	if !strings.Contains(out, "CLAUDE.md and AGENTS.md") {
		t.Errorf("one line should name both files: %q", out)
	}
}

func TestRunSaysWhatItDidInAtMostThreeLines(t *testing.T) {
	root := t.TempDir()
	first := run(t, root, Options{})
	again := run(t, root, Options{})
	for _, out := range []string{first, again} {
		lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
		if len(lines) > 3 {
			t.Errorf("%d lines, want at most 3:\n%s", len(lines), out)
		}
		if last := lines[len(lines)-1]; !strings.HasPrefix(last, "Next: ") || !strings.Contains(last, "`stickypane`") {
			t.Errorf("the last line should be the one next step: %q", last)
		}
	}
	if !strings.Contains(first, "Made .sticky/") || !strings.Contains(first, "AGENTS.md") {
		t.Errorf("first run: %q", first)
	}
	if !strings.Contains(again, "already") || !strings.Contains(again, "up to date") {
		t.Errorf("a second run should say nothing changed: %q", again)
	}
}

func TestRunNamesTheTmuxCommandInTmux(t *testing.T) {
	t.Setenv("TMUX", "/tmp/tmux-501/default,1,0")
	out := run(t, t.TempDir(), Options{NoAgentDocs: true})
	if !strings.Contains(out, "tmux split-window -h stickypane") {
		t.Errorf("in tmux the next step is the split command: %q", out)
	}
}

func TestProjectRoot(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := ProjectRoot(sub); got != "" {
		t.Errorf("outside a repository ProjectRoot = %q, want none", got)
	}
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := ProjectRoot(sub); got != root {
		t.Errorf("ProjectRoot = %q, want %q", got, root)
	}
	t.Setenv("HOME", root)
	if got := ProjectRoot(sub); got != "" {
		t.Errorf("a repository at the home folder is not a project: %q", got)
	}
}

func TestMakeBoard(t *testing.T) {
	root := t.TempDir()
	dir, made, err := MakeBoard(root)
	if err != nil || !made || dir != filepath.Join(root, ".sticky") {
		t.Fatalf("MakeBoard = %q, %v, %v", dir, made, err)
	}
	if !exists(filepath.Join(dir, "welcome.md")) || !exists(filepath.Join(dir, ".gitignore")) {
		t.Error("a new board has a welcome note and keeps its trash out of git")
	}
	if _, made, err := MakeBoard(root); made || err != nil {
		t.Errorf("a second MakeBoard should find the board: %v, %v", made, err)
	}
}
