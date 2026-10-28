package initcmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/adapter/filelock"
	"github.com/LeeSwallow/stickypane/internal/core/doc"
	"github.com/LeeSwallow/stickypane/internal/core/i18n"
)

// The welcome note is written in the board's language: the one init was
// given, else the environment's.
func TestWelcomeNoteIsInTheBoardsLanguage(t *testing.T) {
	welcome := func(root string) doc.Document {
		t.Helper()
		return doc.Parse([]byte(read(t, filepath.Join(root, ".sticky", "welcome.md"))))
	}
	korean := func(d doc.Document) bool {
		title, _ := d.Get("title")
		return strings.Contains(title, "환영") && strings.Contains(d.Body, "+ 새 노트") && strings.Contains(d.Body, "터미널과 에이전트")
	}

	t.Setenv("STICKYPANE_LANG", "ko")
	root := t.TempDir()
	run(t, root, Options{NoAgentDocs: true})
	if w := welcome(root); !korean(w) || !w.Pinned() {
		t.Errorf("the environment asks for Korean, so the welcome note is Korean and pinned: %q", w.Body)
	}
	root = t.TempDir()
	run(t, root, Options{NoAgentDocs: true, Language: "en"})
	if w := welcome(root); korean(w) || !strings.Contains(w.Body, "+ New note") {
		t.Errorf("--language en wins over the environment: %q", w.Body)
	}

	t.Setenv("STICKYPANE_LANG", "en")
	root = t.TempDir()
	run(t, root, Options{NoAgentDocs: true, Language: "ko"})
	if w := welcome(root); !korean(w) {
		t.Errorf("--language ko wins over the environment: %q", w.Body)
	}
	root = t.TempDir()
	run(t, root, Options{NoAgentDocs: true, Language: "auto"})
	if w := welcome(root); korean(w) {
		t.Errorf("--language auto follows the environment (English): %q", w.Body)
	}
}

func TestLanguagePrefersTheChoiceThenTheBoardThenTheEnvironment(t *testing.T) {
	t.Setenv("STICKYPANE_LANG", "en")
	dir := filepath.Join(t.TempDir(), ".sticky")
	if got := Language(dir, ""); got != "en" {
		t.Errorf("no board, no choice: the environment's, got %q", got)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sticky.json"), []byte(`{"language": "ko"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Language(dir, ""); got != "ko" {
		t.Errorf("the board's setting wins over the environment, got %q", got)
	}
	if got := Language(dir, "en"); got != "en" {
		t.Errorf("a choice wins over the board's setting, got %q", got)
	}
	if got := Language(dir, "xx"); got != "en" {
		t.Errorf("a choice with no translation follows the environment, got %q", got)
	}
}

// Every welcome note says the same things as the English one: the same
// front matter, and the same keys, buttons and agents.
func TestEveryWelcomeMatchesTheEnglish(t *testing.T) {
	en := doc.Parse([]byte(Welcome("en")))
	for _, lang := range i18n.Languages() {
		w := doc.Parse([]byte(Welcome(lang)))
		if title, _ := w.Get("title"); title == "" || !w.Pinned() || strings.TrimSpace(w.Body) == "" {
			t.Errorf("%s: the welcome note needs a title, a pin and a body", lang)
		}
		for _, key := range []string{"open", "color"} {
			a, _ := en.Get(key)
			if b, _ := w.Get(key); a != b {
				t.Errorf("%s: %s = %q, the English has %q", lang, key, b, a)
			}
		}
		for _, k := range []string{"**tab**", "**enter**", "**N**", "**a**", "**o**", "**?**", "**×**", "`.sticky/`", "Claude Code", "Codex"} {
			if !strings.Contains(w.Body, k) {
				t.Errorf("%s: the welcome note should name %s", lang, k)
			}
		}
	}
	if Welcome("xx") != Welcome("en") {
		t.Error("a language without a welcome note gets the English one")
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func run(t *testing.T, root string, opts Options) string {
	t.Helper()
	var out bytes.Buffer
	if err := Run(root, opts, &out); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestGuideIsShortAndCoversEveryShape(t *testing.T) {
	g := Guide()
	if !strings.HasPrefix(g, startMark+"\n") || !strings.HasSuffix(g, endMark+"\n") {
		t.Error("the guide must be wrapped in the markers")
	}
	if n := strings.Count(g, "\n"); n > 55 {
		t.Errorf("the guide is %d lines, want at most 55", n)
	}
	for _, want := range []string{".sticky/", "type: board", "type: checklist", "type: log", "type: chart", "label: number", "mermaid", "type: form", "- ( ) ", "[ Deploy ]", "stickypane wait", "stickypane todo", "stickypane card", "stickypane chart", "stickypane log", "stickypane set", "- [ ]", "## Heading", "open: true", "sticky.json", "name.sh", "name.log", "stickypane mv", "stickypane show", "A folder", "stickypane help <shape>", "a book", "hide"} {
		if !strings.Contains(g, want) {
			t.Errorf("the guide should mention %q", want)
		}
	}
}

func TestRunCreatesBoardWelcomeNoteAndAgentsFile(t *testing.T) {
	t.Setenv("STICKYPANE_LANG", "en")
	root := t.TempDir()
	out := run(t, root, Options{})
	welcome := doc.Parse([]byte(read(t, filepath.Join(root, ".sticky", "welcome.md"))))
	if open, _ := welcome.Get("open"); open != "true" {
		t.Errorf("the welcome note should be open on first run, open = %q", open)
	}
	if !welcome.Pinned() || !strings.Contains(welcome.Body, "jots a note") || !strings.Contains(welcome.Body, "+ New note") {
		t.Errorf("the welcome note should be pinned and say how to start in the terminal and in the app: %q", welcome.Body)
	}
	if got := read(t, filepath.Join(root, "AGENTS.md")); got != Guide() {
		t.Errorf("AGENTS.md = %q, want exactly the guide", got)
	}
	if _, err := os.Stat(filepath.Join(root, "CLAUDE.md")); !errors.Is(err, os.ErrNotExist) {
		t.Error("CLAUDE.md should not be created when it did not exist")
	}
	if !strings.Contains(out, "AGENTS.md") || !strings.Contains(out, "stickypane") {
		t.Errorf("output should say what happened: %q", out)
	}
}

func TestRunAppendsToExistingAgentFilesOnly(t *testing.T) {
	root := t.TempDir()
	claude := filepath.Join(root, "CLAUDE.md")
	if err := os.WriteFile(claude, []byte("# Project rules\n\nUse tabs."), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, root, Options{})
	if got, want := read(t, claude), "# Project rules\n\nUse tabs.\n\n"+Guide(); got != want {
		t.Errorf("CLAUDE.md = %q\nwant %q", got, want)
	}
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); !errors.Is(err, os.ErrNotExist) {
		t.Error("AGENTS.md should not be created when CLAUDE.md exists")
	}
}

func TestRunTwiceChangesNothing(t *testing.T) {
	root := t.TempDir()
	run(t, root, Options{})
	agents := filepath.Join(root, "AGENTS.md")
	welcome := filepath.Join(root, ".sticky", "welcome.md")
	first := read(t, agents)
	if err := os.Remove(welcome); err != nil { // the user took the note down
		t.Fatal(err)
	}
	run(t, root, Options{})
	if got := read(t, agents); got != first {
		t.Errorf("a second run changed AGENTS.md:\n%q\nwas:\n%q", got, first)
	}
	if _, err := os.Stat(welcome); !errors.Is(err, os.ErrNotExist) {
		t.Error("a second run must not bring the welcome note back")
	}
}

func TestRunReplacesAnOldGuide(t *testing.T) {
	root := t.TempDir()
	agents := filepath.Join(root, "AGENTS.md")
	old := "intro\n\n" + startMark + "\nstale text\n" + endMark + "\n\noutro\n"
	if err := os.WriteFile(agents, []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, root, Options{})
	if got, want := read(t, agents), "intro\n\n"+Guide()+"\noutro\n"; got != want {
		t.Errorf("AGENTS.md = %q\nwant %q", got, want)
	}
}

func TestNoAgentDocs(t *testing.T) {
	root := t.TempDir()
	run(t, root, Options{NoAgentDocs: true})
	if _, err := os.Stat(filepath.Join(root, ".sticky")); err != nil {
		t.Errorf("the board should still be created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); !errors.Is(err, os.ErrNotExist) {
		t.Error("AGENTS.md should not be touched")
	}
}

func TestGuideTextHasNoMarkers(t *testing.T) {
	text := GuideText()
	if strings.Contains(text, "stickypane:start") || strings.Contains(text, "stickypane:end") {
		t.Errorf("GuideText should drop the markers: %q", text)
	}
	if !strings.HasPrefix(text, "## stickypane notes") || !strings.Contains(text, "type: board") {
		t.Errorf("GuideText should be the guide itself: %q", text)
	}
}

func TestSkillInstallsASkillInsteadOfTheGuide(t *testing.T) {
	root := t.TempDir()
	out := run(t, root, Options{Skill: true})
	skill := read(t, filepath.Join(root, ".claude", "skills", "stickypane", "SKILL.md"))
	if !strings.HasPrefix(skill, "---\nname: stickypane\ndescription: ") {
		t.Fatalf("a skill starts with its name and description:\n%.200s", skill)
	}
	head, body, _ := strings.Cut(strings.TrimPrefix(skill, "---\n"), "\n---\n")
	if strings.Count(head, "\n") != 1 || !strings.Contains(head, "board") {
		t.Errorf("the description should be one line that says when to use the skill: %q", head)
	}
	if !strings.Contains(body, "stickypane todo") || !strings.Contains(body, "type: form") || strings.Contains(body, startMark) {
		t.Errorf("the skill should carry the guide, without the markers:\n%s", body)
	}
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); !errors.Is(err, os.ErrNotExist) {
		t.Error("with a skill the guide is not added to AGENTS.md: the skill is loaded when it is needed")
	}
	if !strings.Contains(out, "SKILL.md") {
		t.Errorf("the output should name the file: %q", out)
	}
	run(t, root, Options{Skill: true})
	if again := read(t, filepath.Join(root, ".claude", "skills", "stickypane", "SKILL.md")); again != skill {
		t.Error("running again should change nothing")
	}
}

func TestRunKeepsTheTrashOutOfGit(t *testing.T) {
	root := t.TempDir()
	run(t, root, Options{NoAgentDocs: true})
	ignore := read(t, filepath.Join(root, ".sticky", ".gitignore"))
	if !strings.Contains(ignore, ".trash/") || !strings.Contains(ignore, filelock.Name) {
		t.Errorf(".sticky/.gitignore = %q", ignore)
	}
}

// No board in the home folder or the root: every project below would find
// it. A symlink to the home folder is the home folder.
func TestRefuseBoardInHomeAndRoot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	link := filepath.Join(t.TempDir(), "h")
	if err := os.Symlink(home, link); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{home, link, "/"} {
		if err := RefuseBoard(dir); !errors.Is(err, ErrNoBoardHere) {
			t.Errorf("RefuseBoard(%q) = %v", dir, err)
		}
	}
	if err := RefuseBoard(filepath.Join(home, "project")); err != nil {
		t.Errorf("a folder under home: %v", err)
	}
}
