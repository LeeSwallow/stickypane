// Package harness reads the plugin's skills for the front ends that cannot
// install a plugin: the MCP server hands them out as prompts and through
// its guide tool, and stickypane guide prints them. There is one copy of
// each skill, the plugin's own.
package harness

import (
	"io/fs"
	"path"
	"strings"

	"github.com/LeeSwallow/stickypane"
)

// Skill is one skill of the plugin.
type Skill struct {
	Name, Description string
	// Body is the skill without its front matter.
	Body string
}

// Skills lists the skills, the way in first.
func Skills() []Skill {
	entries, _ := fs.ReadDir(stickypane.Harness, "skills")
	var out []Skill
	for _, e := range entries {
		b, err := fs.ReadFile(stickypane.Harness, path.Join("skills", e.Name(), "SKILL.md"))
		if err != nil {
			continue
		}
		s := parse(e.Name(), string(b))
		if s.Name == First {
			out = append([]Skill{s}, out...)
		} else {
			out = append(out, s)
		}
	}
	return out
}

// First is the skill to read before the others.
const First = "using-the-board"

// parse splits a SKILL.md into its front matter and its body.
func parse(dir, text string) Skill {
	s := Skill{Name: dir, Body: text}
	rest, ok := strings.CutPrefix(text, "---\n")
	if !ok {
		return s
	}
	head, body, ok := strings.Cut(rest, "\n---\n")
	if !ok {
		return s
	}
	s.Body = strings.TrimLeft(body, "\n")
	for _, l := range strings.Split(head, "\n") {
		k, v, _ := strings.Cut(l, ":")
		switch strings.TrimSpace(k) {
		case "name":
			s.Name = strings.TrimSpace(v)
		case "description":
			s.Description = strings.TrimSpace(v)
		}
	}
	return s
}

// Lookup finds a skill by its name.
func Lookup(name string) (Skill, bool) {
	name = strings.TrimPrefix(strings.TrimSpace(name), "board:")
	for _, s := range Skills() {
		if s.Name == name {
			return s, true
		}
	}
	return Skill{}, false
}

// Read is a skill to follow away from the plugin: its body, then each
// file it points to (its references and the shared rules), so nothing is
// a path the reader cannot open.
func Read(s Skill) string {
	var b strings.Builder
	b.WriteString(s.Body)
	seen := map[string]bool{}
	for _, file := range pointsTo(s) {
		if seen[file] {
			continue
		}
		seen[file] = true
		text, err := fs.ReadFile(stickypane.Harness, file)
		if err != nil {
			continue
		}
		b.WriteString("\n\n---\n\n<!-- " + file + " -->\n\n")
		b.Write(text)
	}
	return b.String()
}

// pointsTo is the files a skill names: references/... in its own folder,
// and the plugin's rules/....
func pointsTo(s Skill) []string {
	var out []string
	for _, field := range strings.FieldsFunc(s.Body, func(r rune) bool { return r == '`' || r == ' ' || r == '\n' || r == '(' || r == ')' }) {
		field = strings.TrimRight(field, ".,:;")
		switch {
		case strings.HasPrefix(field, "references/") && strings.HasSuffix(field, ".md"):
			out = append(out, path.Join("skills", s.Name, field))
		case strings.HasPrefix(field, "${CLAUDE_PLUGIN_ROOT}/rules/"):
			out = append(out, strings.TrimPrefix(field, "${CLAUDE_PLUGIN_ROOT}/"))
		}
	}
	return out
}

// Index is the list of skills with when to use each, for a reader that
// will pick one.
func Index() string {
	var b strings.Builder
	for _, s := range Skills() {
		b.WriteString("- " + s.Name + ": " + s.Description + "\n")
	}
	return b.String()
}
