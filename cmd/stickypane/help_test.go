package main

import (
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/i18n"
)

var hangul = regexp.MustCompile(`[\x{AC00}-\x{D7A3}]`)

// english pins the language, so the tests do not depend on the machine's.
func english(t *testing.T) { t.Helper(); t.Setenv("STICKYPANE_LANG", "en") }

// `help card` prints card's help alone, with an example, not the whole page.
func TestHelpOnACommandPrintsThatCommandOnly(t *testing.T) {
	english(t)
	t.Chdir(t.TempDir())
	code, out, errOut := exec(t, "help", "card")
	if code != 0 || errOut != "" {
		t.Fatalf("code = %d, stderr = %q", code, errOut)
	}
	for _, want := range []string{"stickypane card <name> add|move", "Examples:", "stickypane card work move login --to Done", "Related:"} {
		if !strings.Contains(out, want) {
			t.Errorf("help card should contain %q:\n%s", want, out)
		}
	}
	for _, not := range []string{"stickypane mcp", "stickypane bridge", "stickypane wait"} {
		if strings.Contains(out, not) {
			t.Errorf("help card should not mention %q:\n%s", not, out)
		}
	}
}

// An unknown name suggests the closest one and is a usage error.
func TestHelpOnAnUnknownNameSuggests(t *testing.T) {
	english(t)
	t.Chdir(t.TempDir())
	for word, want := range map[string]string{"carb": "card", "chekclist": "checklist", "setpu": "setup", "star": "start"} {
		code, out, errOut := exec(t, "help", word)
		if code != exitUsage || out != "" || !strings.Contains(errOut, `did you mean "`+want+`"`) {
			t.Errorf("help %s: code = %d, out = %q, stderr = %q", word, code, out, errOut)
		}
	}
	if code, _, errOut := exec(t, "help", "zzzzzzzz"); code != exitUsage || strings.Contains(errOut, "did you mean") || !strings.Contains(errOut, "stickypane help") {
		t.Errorf("nothing near: code = %d, stderr = %q", code, errOut)
	}
}

// `help <shape>` prints what `kinds <shape>` prints.
func TestHelpOnAShapePrintsTheFormat(t *testing.T) {
	english(t)
	t.Chdir(t.TempDir())
	for _, shape := range []string{"checklist", "board", "form", "tab", "book", "script"} {
		_, want, _ := exec(t, "kinds", shape)
		code, out, _ := exec(t, "help", shape)
		if code != 0 || out != want || want == "" {
			t.Errorf("help %s differs from kinds %s:\n%s\n---\n%s", shape, shape, out, want)
		}
	}
	if _, out, _ := exec(t, "help", "checklist"); !strings.Contains(out, "type: checklist") || !strings.Contains(out, "- [ ]") {
		t.Errorf("the format and an example:\n%s", out)
	}
}

// A name that is both a command and a shape is the command: `help log`.
func TestHelpPrefersTheCommandOverTheShape(t *testing.T) {
	english(t)
	t.Chdir(t.TempDir())
	if _, out, _ := exec(t, "help", "log"); !strings.Contains(out, "stickypane log <name>") {
		t.Errorf("help log:\n%s", out)
	}
}

// The way to start is the first thing `help` says where there is no board,
// and is not repeated where there is one.
func TestHelpStart(t *testing.T) {
	english(t)
	t.Chdir(t.TempDir())
	code, start, _ := exec(t, "help", "start")
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	last := -1
	for _, step := range []string{"stickypane init", "stickypane write", "stickypane\n", "stickypane setup", "desktop app"} {
		i := strings.Index(start, step)
		if i < 0 || i < last {
			t.Errorf("%q should come in order in:\n%s", step, start)
		}
		last = i
	}
	if _, out, _ := exec(t, "help"); !strings.HasPrefix(out, start) {
		t.Errorf("help without a board starts with the way to start:\n%s", out)
	}
	project(t)
	english(t)
	if _, out, _ := exec(t, "help"); strings.Contains(out, "The way to start") || !strings.Contains(out, "Notes") {
		t.Errorf("help in a board is the overview:\n%s", out)
	}
}

// The overview is short, grouped by task, and names every command.
func TestHelpOverviewNamesEveryCommand(t *testing.T) {
	english(t)
	project(t)
	english(t)
	_, out, _ := exec(t, "help")
	if n := strings.Count(out, "\n"); n > 60 {
		t.Errorf("the overview is %d lines", n)
	}
	for _, g := range helpGroups {
		if !strings.Contains(out, "\n"+g.title+"\n") {
			t.Errorf("group %q missing", g.title)
		}
	}
	for _, c := range commands {
		if !regexp.MustCompile(`(?m)^  ` + regexp.QuoteMeta(c.names[0]) + ` `).MatchString(out) {
			t.Errorf("the overview does not list %s", c.names[0])
		}
	}
}

// Every command has help, in a known group, with a usage line that agrees
// with the usage text, and at least one example.
func TestEveryCommandHasHelp(t *testing.T) {
	groups := map[string]bool{}
	for _, g := range helpGroups {
		groups[g.key] = true
	}
	flag := regexp.MustCompile(`--[a-z-]+`)
	for _, c := range commands {
		name, h := c.names[0], c.help
		if !groups[h.group] || h.brief == "" || h.what == "" || len(h.examples) == 0 {
			t.Errorf("%s: help is incomplete: %+v", name, h)
		}
		if h.syntax != "stickypane "+name && !strings.HasPrefix(h.syntax, "stickypane "+name+" ") {
			t.Errorf("%s: syntax %q", name, h.syntax)
		}
		// The flags of the help and of the usage text are the same.
		got := flag.FindAllString(h.syntax, -1)
		want := flag.FindAllString(usageOf(name), -1)
		if name == "help" || name == "version" {
			continue
		}
		slices.Sort(got)
		slices.Sort(want)
		if !slices.Equal(slices.Compact(got), slices.Compact(want)) {
			t.Errorf("%s: help has flags %v, the usage text %v", name, got, want)
		}
		for _, r := range h.related {
			if _, ok := lookup(r); !ok && r != "start" && !isShape(r) {
				t.Errorf("%s: related %q is nothing", name, r)
			}
		}
	}
}

func isShape(name string) bool {
	k, f, ok := findShape(name)
	return ok && (k != nil || f != nil)
}

// Every example parses: it runs without a usage error. The few that would
// reach outside the test (an agent's installer, a server, a wait) are only
// checked to name a real command.
func TestEveryExampleParses(t *testing.T) {
	english(t)
	t.Setenv("HOME", t.TempDir())
	project(t)
	english(t)
	notRun := map[string]bool{"setup": true, "watch": true, "mcp": true, "bridge": true, "wait": true, "api": true}
	for _, c := range commands {
		for _, ex := range c.help.examples {
			stdin, line := "", ex
			if rest, ok := strings.CutPrefix(ex, "echo "); ok {
				pipe, cmd, _ := strings.Cut(rest, " | ")
				stdin, line = strings.Join(splitWords(pipe), " ")+"\n", cmd
			}
			words := splitWords(line)
			if len(words) < 2 || words[0] != "stickypane" {
				t.Errorf("%q does not start with stickypane", ex)
				continue
			}
			if _, ok := lookup(words[1]); !ok {
				t.Errorf("%q: no command %s", ex, words[1])
				continue
			}
			if words[1] != c.names[0] && !strings.Contains(ex, "| stickypane "+c.names[0]) {
				t.Errorf("the example of %s runs %s: %q", c.names[0], words[1], ex)
			}
			if notRun[words[1]] {
				continue
			}
			if code, _, errOut := execIn(t, stdin, words[1:]...); code == exitUsage {
				t.Errorf("%q is a usage error: %s", ex, errOut)
			}
		}
	}
}

// splitWords splits a command line the way a shell does, for the quotes the
// examples use.
func splitWords(s string) []string {
	var words []string
	var cur strings.Builder
	var quote rune
	started := false
	for _, r := range s {
		switch {
		case quote != 0 && r == quote:
			quote = 0
		case quote != 0:
			cur.WriteRune(r)
		case r == '"' || r == '\'':
			quote, started = r, true
		case r == ' ':
			if started || cur.Len() > 0 {
				words = append(words, cur.String())
				cur.Reset()
				started = false
			}
		default:
			cur.WriteRune(r)
		}
	}
	if started || cur.Len() > 0 {
		words = append(words, cur.String())
	}
	return words
}

// Under STICKYPANE_LANG=ko the help speaks Korean and keeps the commands
// and the file formats as they are; English has no Korean in it.
func TestHelpInKorean(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("STICKYPANE_LANG", "ko")
	for _, args := range [][]string{{"help"}, {"help", "start"}, {"help", "card"}, {"help", "init"}, {"help", "checklist"}} {
		code, out, _ := exec(t, args...)
		if code != 0 || !hangul.MatchString(out) {
			t.Errorf("%v: code = %d, no Korean in:\n%s", args, code, out)
		}
	}
	_, out, _ := exec(t, "help", "card")
	for _, want := range []string{"stickypane card <name> add|move", "예시:", "stickypane card work move login --to Done", "--to"} {
		if !strings.Contains(out, want) {
			t.Errorf("Korean help card should contain %q:\n%s", want, out)
		}
	}
	_, out, _ = exec(t, "help")
	for _, c := range commands {
		if !strings.Contains(out, c.names[0]+" ") {
			t.Errorf("the Korean overview does not name %s", c.names[0])
		}
	}
	t.Setenv("STICKYPANE_LANG", "en")
	for _, args := range [][]string{{"help"}, {"help", "card"}, {"help", "checklist"}} {
		if _, out, _ := exec(t, args...); hangul.MatchString(out) {
			t.Errorf("%v in English has Korean in it:\n%s", args, out)
		}
	}
}

// The board's own language choice wins over the environment.
func TestHelpFollowsTheBoardsLanguage(t *testing.T) {
	project(t)
	t.Setenv("STICKYPANE_LANG", "en")
	if code, _, _ := exec(t, "language", "ko"); code != 0 {
		t.Fatal("language ko failed")
	}
	if _, out, _ := exec(t, "help", "card"); !hangul.MatchString(out) {
		t.Errorf("the board speaks Korean:\n%s", out)
	}
}

// Every Korean help key belongs to something that exists, so a renamed
// command does not leave a stale translation behind; and every command is
// translated.
func TestKoreanHelpKeysAreReal(t *testing.T) {
	ko, err := i18n.Load("ko")
	if err != nil {
		t.Fatal(err)
	}
	known := map[string]bool{"start": true, "open.brief": true, "overview.head": true, "overview.foot": true}
	for _, k := range []string{"usage", "options", "examples", "related", "name"} {
		known["label."+k] = true
	}
	for _, g := range helpGroups {
		known["group."+g.key] = true
	}
	for _, k := range []string{"make", "file", "behaves", "keys"} {
		known["shape."+k] = true
	}
	for _, c := range commands {
		n := c.names[0]
		for _, k := range []string{n + ".brief", n + ".what"} {
			known[k] = true
			if ko.CLI[k] == "" {
				t.Errorf("ko.json has no %q", k)
			}
		}
		if c.help.args != "" {
			known[n+".args"] = true
			if ko.CLI[n+".args"] == "" {
				t.Errorf("ko.json has no %q", n+".args")
			}
		}
	}
	for k := range ko.CLI {
		if !known[k] {
			t.Errorf("ko.json has the stale key %q", k)
		}
	}
}

// The README sections are the help's text, so they cannot drift from it.
func TestReadmeUsageIsTheHelp(t *testing.T) {
	for file, code := range map[string]string{"../../README.md": "en", "../../README.ko.md": "ko"} {
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		s, _ := i18n.Load(code)
		for _, text := range []string{s.T("start", startText), overview(s)} {
			if !strings.Contains(string(b), strings.TrimSpace(text)) {
				t.Errorf("%s does not contain the help text:\n%s", file, text)
			}
		}
	}
}
