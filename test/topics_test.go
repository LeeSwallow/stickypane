package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/core/kinds"
	"github.com/LeeSwallow/stickypane/internal/core/widget/note"
)

// The skills are the guide's topics, a situation each, and their recipes
// are the use cases the triage table kept (docs/history/specs/
// 2026-11-23-usecase-triage.md). These tests keep the two in step.

// frontMatter is the key: value lines between a file's first two "---".
func frontMatter(text string) map[string]string {
	out := map[string]string{}
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || lines[0] != "---" {
		return out
	}
	for _, l := range lines[1:] {
		if l == "---" {
			break
		}
		k, v, _ := strings.Cut(l, ":")
		out[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return out
}

// triage is the table's rows: key, verdict and topic.
func triage(t *testing.T) (rows []struct{ key, verdict, topic string }) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot, "docs", "history", "specs", "2026-11-23-usecase-triage.md"))
	if err != nil {
		t.Fatal(err)
	}
	row := regexp.MustCompile(`(?m)^\| ([a-z]+-[A-Z]+\d+) \| [^|]* \| (topic|example|exclude) \| ?([a-z-]*) ?\|`)
	for _, m := range row.FindAllStringSubmatch(string(b), -1) {
		rows = append(rows, struct{ key, verdict, topic string }{m[1], m[2], m[3]})
	}
	return rows
}

// recipes are a topic's references with a case: the use cases it carries
// as recipes, by file name.
func recipes(t *testing.T, topic string) map[string]map[string]string {
	t.Helper()
	out := map[string]map[string]string{}
	files, _ := filepath.Glob(filepath.Join(pluginRoot, "skills", topic, "references", "*.md"))
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		fm := frontMatter(string(b))
		fm["text"] = string(b)
		out[filepath.Base(f)] = fm
	}
	return out
}

// Every case the triage table kept is in the guide: a topic case in its
// topic's body (named under cases: in its front matter), an example as a
// recipe in its topic's references/. Nothing else is: a recipe that no row
// asks for fails too.
func TestEveryKeptCaseIsInItsTopic(t *testing.T) {
	rows := triage(t)
	count := map[string]int{}
	want := map[string]string{} // key -> "topic/verdict"
	for _, r := range rows {
		count[r.verdict]++
		if r.verdict != "exclude" {
			want[r.key] = r.topic + "/" + r.verdict
		}
	}
	if len(rows) != 120 || count["topic"] != 29 || count["example"] != 44 || count["exclude"] != 47 {
		t.Fatalf("the triage table has %d rows: %v", len(rows), count)
	}
	found := map[string]string{}
	for _, topic := range skillNames {
		b, err := os.ReadFile(filepath.Join(pluginRoot, "skills", topic, "SKILL.md"))
		if err != nil {
			t.Fatal(err)
		}
		cases := regexp.MustCompile(`(?m)^  cases: (.*)$`).FindStringSubmatch(string(b))
		if cases != nil {
			for _, k := range strings.Fields(cases[1]) {
				found[k] = topic + "/topic"
			}
		}
		for file, fm := range recipes(t, topic) {
			if fm["case"] == "" {
				if topic != "using-the-board" || file != "kinds.md" {
					t.Errorf("%s/references/%s is not a recipe of a case", topic, file)
				}
				continue
			}
			found[fm["case"]] = topic + "/example"
			if !strings.HasPrefix(fm["description"], "Use when") {
				t.Errorf("recipe %s/%s should say when to use it", topic, file)
			}
			body := fm["text"][strings.Index(fm["text"], "\n---\n")+5:]
			if !strings.HasPrefix(strings.TrimLeft(body, "\n"), "# ") || !strings.Contains(body, "\nShapes: ") || !strings.Contains(body, "\nDone when: ") || !strings.Contains(body, "stickypane ") {
				t.Errorf("recipe %s/%s: a title, Shapes:, the commands, Done when:", topic, file)
			}
			if n := strings.Count(fm["text"], "\n"); n > 40 {
				t.Errorf("recipe %s/%s is %d lines, at most 40", topic, file, n)
			}
			if !strings.Contains(string(b), "`references/"+file+"`") {
				t.Errorf("%s should list its recipe references/%s", topic, file)
			}
		}
	}
	for k, w := range want {
		if found[k] != w {
			t.Errorf("%s belongs in %s; the guide has it in %q", k, w, found[k])
		}
	}
	for k, f := range found {
		if want[k] == "" {
			t.Errorf("%s is in %s, but the triage table does not keep it", k, f)
		}
	}
}

// helpVerbs are the verbs of each command that has them, from the command
// table (stickypane help --json): note show, todo add, log add...
func helpVerbs(t *testing.T) map[string][]string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot, "cmd", "stickypane", "testdata", "help.json"))
	if err != nil {
		t.Fatal(err)
	}
	var h struct {
		Commands []struct{ Name, Syntax string }
	}
	if err := json.Unmarshal(b, &h); err != nil {
		t.Fatal(err)
	}
	verbs := map[string][]string{}
	word := regexp.MustCompile(`^[a-z|]+$`)
	for _, c := range h.Commands {
		if noun, verb, ok := strings.Cut(c.Name, " "); ok {
			verbs[noun] = append(verbs[noun], verb)
			continue
		}
		if f := strings.Fields(c.Syntax); len(f) > 2 && word.MatchString(f[2]) {
			verbs[c.Name] = append(verbs[c.Name], strings.Split(f[2], "|")...)
		}
	}
	return verbs
}

// The topics write every command as <noun> <verb> <target>: the old forms
// (todo plan add, show, write, say) still run, but are not taught.
func TestTopicsWriteCommandsAsNounVerb(t *testing.T) {
	verbs := helpVerbs(t)
	if !slices.Contains(verbs["todo"], "check") || !slices.Contains(verbs["note"], "write") || !slices.Contains(verbs["chat"], "add") {
		t.Fatalf("verbs = %v", verbs)
	}
	cmd := regexp.MustCompile(`stickypane ([a-z]+)(?: ([a-z]+))?`)
	old := map[string]bool{}
	for _, c := range strings.Fields("show hide cat write set mv link rm restore archive list say") {
		old[c] = true
	}
	err := filepath.WalkDir(filepath.Join(pluginRoot, "skills"), func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(path)
		for _, m := range cmd.FindAllStringSubmatch(string(b), -1) {
			if old[m[1]] {
				t.Errorf("%s: `%s` is an old form", path, m[0])
			}
			if vs, ok := verbs[m[1]]; ok && m[2] != "" && !slices.Contains(vs, m[2]) {
				t.Errorf("%s: `%s`: %s takes %v", path, m[0], m[1], vs)
			}
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// The way in says which shape shows what, for every shape there is.
func TestTheShapeTableCoversEveryShape(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(pluginRoot, "skills", "using-the-board", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	var shapes []string
	for _, line := range strings.Split(string(b), "\n") {
		if cols := strings.Split(line, "|"); len(cols) > 3 && strings.HasPrefix(line, "| ") {
			shapes = append(shapes, cols[2])
		}
	}
	for _, k := range kinds.Default(note.Plain) {
		if !slices.ContainsFunc(shapes, func(s string) bool {
			return regexp.MustCompile(`\b` + k.Name + `\b`).MatchString(s)
		}) {
			t.Errorf("using-the-board's table names no shape %s", k.Name)
		}
	}
}

// The approval rule is written once, in plugin/rules/: the topics point to
// it and do not say it again.
func TestTheApprovalRuleIsWrittenOnce(t *testing.T) {
	for _, phrase := range []string{"exit code 3 is a no", "a fresh form"} {
		var in []string
		for path, text := range harnessFiles(t) {
			if strings.Contains(strings.ToLower(strings.Join(strings.Fields(text), " ")), phrase) {
				in = append(in, path)
			}
		}
		if len(in) != 1 || filepath.Base(filepath.Dir(in[0])) != "rules" {
			t.Errorf("%q should be written once, in rules/: %v", phrase, in)
		}
	}
}
