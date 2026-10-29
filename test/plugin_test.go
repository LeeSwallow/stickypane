package test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/usecase/initcmd"
)

// The repository is also a Claude Code and Codex marketplace. These tests keep
// its files honest: the manifests parse, the skill that teaches the board
// carries the same guide init installs, and the hook script runs.

func TestPluginManifestsParse(t *testing.T) {
	for _, f := range []string{".claude-plugin/marketplace.json", ".agents/plugins/marketplace.json", "plugin/.claude-plugin/plugin.json", "plugin/.codex-plugin/plugin.json", "plugin/hooks/claude-hooks.json", "plugin/hooks/codex-hooks.json"} {
		b, err := os.ReadFile(filepath.Join(repoRoot, f))
		if err != nil {
			t.Fatal(err)
		}
		var v map[string]any
		if err := json.Unmarshal(b, &v); err != nil {
			t.Errorf("%s: %v", f, err)
		}
	}
	// /board:status also lists the notes, as /board:index did; showing a
	// note and the shapes are a command away (stickypane show, help <shape>).
	for _, f := range []string{"commands/status.md", "commands/ask.md", "commands/setup.md", "commands/track.md", "commands/say.md"} {
		b, err := os.ReadFile(filepath.Join(pluginRoot, f))
		if err != nil || !strings.HasPrefix(string(b), "---\ndescription: ") {
			t.Errorf("%s should start with a description: %v", f, err)
		}
	}
}

// The plugin's manifests name the release they ship with: the newest
// version the changelog lists.
func TestPluginManifestsNameTheRelease(t *testing.T) {
	log, err := os.ReadFile(filepath.Join(repoRoot, "CHANGELOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^## \[(\d+\.\d+\.\d+)\]`).FindSubmatch(log)
	if m == nil {
		t.Fatal("the changelog lists no release")
	}
	release := string(m[1])
	versions := func(f string) []string {
		b, err := os.ReadFile(filepath.Join(repoRoot, f))
		if err != nil {
			t.Fatal(err)
		}
		var v struct {
			Version string `json:"version"`
			Plugins []struct {
				Version string `json:"version"`
			} `json:"plugins"`
		}
		if err := json.Unmarshal(b, &v); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		out := []string{}
		if v.Version != "" {
			out = append(out, v.Version)
		}
		for _, p := range v.Plugins {
			out = append(out, p.Version)
		}
		return out
	}
	for _, f := range []string{".claude-plugin/marketplace.json", ".agents/plugins/marketplace.json", "plugin/.claude-plugin/plugin.json", "plugin/.codex-plugin/plugin.json"} {
		got := versions(f)
		if len(got) == 0 {
			t.Errorf("%s names no version", f)
		}
		for _, v := range got {
			if v != release {
				t.Errorf("%s says %s; the release is %s", f, v, release)
			}
		}
	}
}

// skillNames are the plugin's skills.
var skillNames = []string{"using-the-board", "asking-the-user", "tracking-progress", "talking-in-chat", "connecting-notes"}

// harnessFiles are every file of the plugin an agent reads.
func harnessFiles(t *testing.T) map[string]string {
	t.Helper()
	files := map[string]string{}
	for _, dir := range []string{"skills", "commands", "scripts", "rules"} {
		err := filepath.WalkDir(filepath.Join(pluginRoot, dir), func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			b, err := os.ReadFile(path)
			files[path] = string(b)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return files
}

func TestTheSkillsAreShortAndOpenWithWhatToDo(t *testing.T) {
	for _, name := range skillNames {
		b, err := os.ReadFile(filepath.Join(pluginRoot, "skills", name, "SKILL.md"))
		if err != nil || !strings.HasPrefix(string(b), "---\nname: "+name+"\ndescription: Use when") {
			t.Errorf("skill %s needs its name and a description that starts with when to use it: %v", name, err)
			continue
		}
		if n := strings.Count(string(b), "\n"); n > 80 {
			t.Errorf("skill %s is %d lines; it should be skimmable, at most 80", name, n)
		}
		// The first thing after the title is something to run, not a list
		// of files to read first.
		_, body, _ := strings.Cut(string(b), "\n# ")
		if first := strings.Index(body, "stickypane "); first < 0 || first > 600 {
			t.Errorf("skill %s should show a command near the top", name)
		}
	}
}

func TestTheHarnessNamesOnlyFilesThatExist(t *testing.T) {
	ref := regexp.MustCompile("`((?:references|\\$\\{CLAUDE_PLUGIN_ROOT\\})/[^`\\s]+)`")
	for path, text := range harnessFiles(t) {
		for _, m := range ref.FindAllStringSubmatch(text, -1) {
			target := strings.Replace(m[1], "${CLAUDE_PLUGIN_ROOT}", pluginRoot, 1)
			if strings.HasPrefix(m[1], "references/") {
				target = filepath.Join(filepath.Dir(path), m[1])
			}
			if _, err := os.Stat(target); err != nil {
				t.Errorf("%s names %s, which is not there", path, m[1])
			}
		}
	}
}

func TestTheHarnessNamesOnlyRealCommands(t *testing.T) {
	real := map[string]bool{}
	for _, c := range strings.Fields("init setup env watch config index guide version help theme language list cat write answers wait mcp show hide todo card chart log say set rm restore archive mv link kinds api") {
		real[c] = true
	}
	cmd := regexp.MustCompile("stickypane ([a-z]+)")
	files := harnessFiles(t)
	files["guide"] = initcmd.Guide()
	for path, text := range files {
		for _, m := range cmd.FindAllStringSubmatch(text, -1) {
			if !real[m[1]] && m[1] != "board" && m[1] != "is" && m[1] != "notes" {
				t.Errorf("%s runs `stickypane %s`, which is not a command", path, m[1])
			}
		}
	}
}

func TestEveryShapeHasAShortEntry(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(pluginRoot, "skills", "using-the-board", "references", "kinds.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"## Note", "mermaid", "## Board", "type: board", "## Checklist", "type: checklist", "## Log", "type: log", ".log", "## Chart", "type: chart", "## Form", "type: form", "stickypane wait", "## Script", ".sh", "## Tab", "## Book", "stickypane help"} {
		if !strings.Contains(string(b), want) {
			t.Errorf("references/kinds.md should cover %q", want)
		}
	}
}

func TestSessionStartHookIsQuietWithoutABoard(t *testing.T) {
	script, err := filepath.Abs(filepath.Join(pluginRoot, "scripts", "session-start.sh"))
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

// Only .sticky is a board. ~/.stickypane is where the registry lives, so a
// project under it without a .sticky of its own has no board.
func TestBoardContextTakesOnlyStickyForABoard(t *testing.T) {
	script, err := filepath.Abs(filepath.Join(pluginRoot, "scripts", "board-context.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh")
	}
	if runtime.GOOS == "windows" {
		t.Skip("the hook runs under a Unix shell; Windows paths are not its input")
	}
	bin := t.TempDir()
	fake := "#!/bin/sh\necho 'stickypane 0.0.0'\n"
	if err := os.WriteFile(filepath.Join(bin, "stickypane"), []byte(fake), 0o755); err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	project := filepath.Join(home, "work", "app")
	for _, d := range []string{filepath.Join(home, ".stickypane"), project} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	run := func(dir string) (string, int) {
		cmd := exec.Command("sh", script)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "PWD="+dir)
		out, err := cmd.CombinedOutput()
		code := 0
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else if err != nil {
			t.Fatal(err)
		}
		return string(out), code
	}
	if out, code := run(project); code != 3 || !strings.Contains(out, "BOARD_DIR=\n") {
		t.Errorf("a folder named .stickypane is not a board: exit %d\n%s", code, out)
	}
	if err := os.Mkdir(filepath.Join(project, ".sticky"), 0o755); err != nil {
		t.Fatal(err)
	}
	if out, code := run(project); code != 0 || !strings.Contains(out, "BOARD_DIR="+filepath.Join(project, ".sticky")+"\n") {
		t.Errorf(".sticky is the board: exit %d\n%s", code, out)
	}
}

// TestTheLayersPointAtEachOther keeps the plugin a tree: every command
// follows a skill that exists, every skill the way in names is there, and
// every skill but the way in is named by it.
func TestTheLayersPointAtEachOther(t *testing.T) {
	named := regexp.MustCompile("`(?:board:)?([a-z]+(?:-[a-z]+)+)`")
	isSkill := map[string]bool{}
	for _, s := range skillNames {
		isSkill[s] = true
	}
	way, err := os.ReadFile(filepath.Join(pluginRoot, "skills", "using-the-board", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range skillNames[1:] {
		if !strings.Contains(string(way), "`"+s+"`") {
			t.Errorf("using-the-board should say when to follow %s", s)
		}
	}
	for path, text := range harnessFiles(t) {
		for _, m := range named.FindAllStringSubmatch(text, -1) {
			if strings.Contains(m[0], "board:") && !isSkill[m[1]] {
				t.Errorf("%s follows %s, which is not a skill", path, m[0])
			}
		}
	}
}

// Both marketplaces sit at the repository's root, so that
// `plugin marketplace add LeeSwallow/stickypane` keeps working, and point
// at plugin/, where the plugin is whole: its manifests, the files they name
// and the scripts its hooks and commands run. scripts/ at the root is for
// developing stickypane only.
func TestTheMarketplacesPointAtThePluginFolder(t *testing.T) {
	read := func(f string, v any) {
		b, err := os.ReadFile(filepath.Join(repoRoot, f))
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(b, v); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
	}
	var claude struct {
		Name    string
		Plugins []struct{ Name, Source string }
	}
	read(".claude-plugin/marketplace.json", &claude)
	if claude.Name != "stickypane" || len(claude.Plugins) != 1 || claude.Plugins[0].Name != "board" || claude.Plugins[0].Source != "./plugin" {
		t.Errorf("the Claude Code marketplace should be stickypane with board at ./plugin: %+v", claude)
	}
	type source struct{ Source, Path string }
	var codex struct {
		Name    string
		Plugins []struct {
			Name   string
			Source source
		}
	}
	read(".agents/plugins/marketplace.json", &codex)
	if codex.Name != "stickypane" || len(codex.Plugins) != 1 || codex.Plugins[0].Name != "board" || codex.Plugins[0].Source != (source{"local", "./plugin"}) {
		t.Errorf("the Codex marketplace should be stickypane with board at ./plugin: %+v", codex)
	}
	var claudePlugin struct{ Name, Hooks string }
	read("plugin/.claude-plugin/plugin.json", &claudePlugin)
	var codexPlugin struct{ Name, Skills, Hooks string }
	read("plugin/.codex-plugin/plugin.json", &codexPlugin)
	if claudePlugin.Name != "board" || codexPlugin.Name != "board" {
		t.Errorf("both plugins are board: %q, %q", claudePlugin.Name, codexPlugin.Name)
	}
	for _, p := range []string{claudePlugin.Hooks, codexPlugin.Skills, codexPlugin.Hooks} {
		if _, err := os.Stat(filepath.Join(pluginRoot, p)); err != nil {
			t.Errorf("a manifest names %s, which is not in plugin/", p)
		}
	}
	run := regexp.MustCompile(`\$\{(?:CLAUDE_)?PLUGIN_ROOT\}/([^"\\]+)`)
	for _, f := range []string{"hooks/claude-hooks.json", "hooks/codex-hooks.json"} {
		b, err := os.ReadFile(filepath.Join(pluginRoot, f))
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range run.FindAllStringSubmatch(string(b), -1) {
			if _, err := os.Stat(filepath.Join(pluginRoot, m[1])); err != nil {
				t.Errorf("%s runs %s, which is not in plugin/", f, m[1])
			}
		}
	}
	for _, old := range []string{"skills", "commands", "rules", "hooks", ".codex-plugin", ".claude-plugin/plugin.json", "scripts/session-start.sh", "scripts/board-context.sh"} {
		if _, err := os.Stat(filepath.Join(repoRoot, old)); err == nil {
			t.Errorf("%s belongs in plugin/", old)
		}
	}
}
