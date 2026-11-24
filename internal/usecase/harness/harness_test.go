package harness

import (
	"strings"
	"testing"
)

func TestSkillsComeFromThePluginWithTheWayInFirst(t *testing.T) {
	skills := Skills()
	if len(skills) != 16 || skills[0].Name != First {
		t.Fatalf("skills = %v", skills)
	}
	for _, s := range skills {
		if !strings.HasPrefix(s.Description, "Use when") || strings.HasPrefix(s.Body, "---") || !strings.HasPrefix(s.Body, "# ") {
			t.Errorf("skill %s: %q / %.40q", s.Name, s.Description, s.Body)
		}
	}
}

// Read carries the rules a skill points to, once each, and leaves its
// references to be read when needed: the shapes in full would triple the
// way in.
func TestReadCarriesTheRulesASkillPointsTo(t *testing.T) {
	s, ok := Lookup("board:using-the-board")
	if !ok {
		t.Fatal("no way in")
	}
	text := Read(s)
	for _, want := range []string{"# Using the board", "<!-- rules/touching-the-board.md -->", "<!-- rules/writing-notes.md -->"} {
		if !strings.Contains(text, want) {
			t.Errorf("Read should carry %q", want)
		}
	}
	if strings.Contains(text, "<!-- skills/using-the-board/references/kinds.md -->") {
		t.Error("Read should not carry the shapes reference")
	}
	if strings.Count(text, "<!-- rules/touching-the-board.md -->") != 1 {
		t.Error("a file named twice is carried once")
	}
	if _, ok := Lookup("nope"); ok {
		t.Error("an unknown skill is not found")
	}
}

func TestIndexSaysWhenToUseEach(t *testing.T) {
	if got := Index(); !strings.HasPrefix(got, "- using-the-board: Use when") || !strings.Contains(got, "- talking-while-working: Use when") {
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

// A recipe is a case written out, in a skill's references/: read by the
// skill's name and its own, and named at the end of the skill for a reader
// without the plugin.
func TestARecipeIsReadByItsSkillAndName(t *testing.T) {
	r, ok := Lookup("approving-a-risky-action/several-at-once")
	if !ok || r.Name != "approving-a-risky-action/several-at-once" || !strings.HasPrefix(r.Description, "Use when") || !strings.HasPrefix(r.Body, "# ") {
		t.Fatalf("recipe = %+v, %v", r, ok)
	}
	if len(Recipes("approving-a-risky-action")) < 2 || len(Recipes("connecting-notes")) != 0 {
		t.Error("a skill's recipes are its references that say when to use them")
	}
	for _, name := range []string{"using-the-board/kinds", "approving-a-risky-action/nope", "nope/several-at-once"} {
		if _, ok := Lookup(name); ok {
			t.Errorf("%s is not a recipe", name)
		}
	}
	s, _ := Lookup("approving-a-risky-action")
	if text := Read(s); !strings.Contains(text, "`stickypane guide approving-a-risky-action/<recipe>`") {
		t.Errorf("Read should say how to read a recipe: %.300q", text)
	}
	s, _ = Lookup("connecting-notes")
	if strings.Contains(Read(s), "<recipe>") {
		t.Error("a skill without recipes says nothing of them")
	}
}
