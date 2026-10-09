package harness

import (
	"strings"
	"testing"
)

func TestSkillsComeFromThePluginWithTheWayInFirst(t *testing.T) {
	skills := Skills()
	if len(skills) < 5 || skills[0].Name != First {
		t.Fatalf("skills = %v", skills)
	}
	for _, s := range skills {
		if !strings.HasPrefix(s.Description, "Use when") || strings.HasPrefix(s.Body, "---") || !strings.HasPrefix(s.Body, "# ") {
			t.Errorf("skill %s: %q / %.40q", s.Name, s.Description, s.Body)
		}
	}
}

func TestReadCarriesTheFilesASkillPointsTo(t *testing.T) {
	s, ok := Lookup("board:using-the-board")
	if !ok {
		t.Fatal("no way in")
	}
	text := Read(s)
	for _, want := range []string{"# Using the board", "<!-- skills/using-the-board/references/kinds.md -->", "## Checklist", "<!-- rules/touching-the-board.md -->", "<!-- rules/writing-notes.md -->"} {
		if !strings.Contains(text, want) {
			t.Errorf("Read should carry %q", want)
		}
	}
	if strings.Count(text, "<!-- rules/touching-the-board.md -->") != 1 {
		t.Error("a file named twice is carried once")
	}
	if _, ok := Lookup("nope"); ok {
		t.Error("an unknown skill is not found")
	}
}

func TestIndexSaysWhenToUseEach(t *testing.T) {
	if got := Index(); !strings.HasPrefix(got, "- using-the-board: Use when") || !strings.Contains(got, "- talking-in-chat: Use when") {
		t.Errorf("Index = %q", got)
	}
}

// A skill saved with Windows line ends reads as one with Unix line ends.
func TestParseReadsFrontMatterWithCRLF(t *testing.T) {
	for _, nl := range []string{"\n", "\r\n"} {
		text := strings.ReplaceAll("---\nname: asking\ndescription: Use when asked.\n---\n\n# Asking\n", "\n", nl)
		if s := parse("dir", text); s.Name != "asking" || s.Description != "Use when asked." || s.Body != "# Asking"+nl {
			t.Errorf("%q: parse = %+v", nl, s)
		}
	}
	if s := parse("dir", "---\r\nname: open\r\n"); s.Name != "dir" || s.Body != "---\r\nname: open\r\n" {
		t.Errorf("front matter that is never closed is body: %+v", s)
	}
}
