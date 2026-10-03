package api

import (
	"fmt"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/harness"
)

// Guide is the agent's way in, in layers. With no topic it is overview
// (the short guide), then the skills with when to follow each and the
// shapes a note takes. A topic is a skill's name, which gives the whole
// skill with the files it points to, or a shape's name, which gives its
// format with an example.
func (a *API) Guide(topic, overview string) (string, error) {
	topic = strings.ToLower(strings.TrimSpace(topic))
	if topic == "" {
		var b strings.Builder
		b.WriteString(strings.TrimRight(overview, "\n"))
		b.WriteString("\n\n## Skills\n\nFor a bigger job, read the skill: the guide with its name as the topic.\n\n")
		b.WriteString(harness.Index())
		b.WriteString("\n## Shapes\n\nA shape's file format and an example: the guide with its name as the topic.\n\n")
		b.WriteString(strings.Join(a.shapes(), ", ") + "\n")
		return b.String(), nil
	}
	if s, ok := harness.Lookup(topic); ok {
		return harness.Read(s), nil
	}
	if text, ok := a.Shape(topic); ok {
		return text, nil
	}
	var skills []string
	for _, s := range harness.Skills() {
		skills = append(skills, s.Name)
	}
	return "", fmt.Errorf("no topic %q: a skill (%s) or a shape (%s)", topic, strings.Join(skills, ", "), strings.Join(a.shapes(), ", "))
}

// shapes names the kinds of note.
func (a *API) shapes() []string {
	names := make([]string, 0, len(a.reg))
	for _, k := range a.reg {
		names = append(names, k.Name)
	}
	return names
}

// Shape is what a kind of note is for, how it works, and an example file.
func (a *API) Shape(name string) (string, bool) {
	for _, k := range a.reg {
		if k.Name != name {
			continue
		}
		var b strings.Builder
		fmt.Fprintf(&b, "# %s\n\n%s\n\n%s\n\n```sh\n%s\n```\n", k.Name, k.Blurb, k.Usage, k.Command)
		if k.Example != "" {
			fmt.Fprintf(&b, "\nAn example file:\n\n```\n%s```\n", k.ExampleFile())
		}
		return b.String(), true
	}
	return "", false
}
