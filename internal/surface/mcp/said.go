package mcp

import (
	"fmt"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/usecase/api"
)

// The text the tools that change the board answer. api answers with
// what happened; these are the MCP server's words for it, read by agents.

// status says where a note stands after an edit: its name and its summary.
func status(e api.Edited) string {
	if e.Summary == "" {
		return e.Note
	}
	return e.Note + ": " + e.Summary
}

// valued says a chart's value after an edit.
func valued(e api.Edited) string { return fmt.Sprintf("%s: %s = %s", e.Note, e.Item, e.Value) }

// keyed says which keys a set changed.
func keyed(k api.Keyed) string {
	var parts []string
	if len(k.Set) > 0 {
		parts = append(parts, "set "+strings.Join(k.Set, ", "))
	}
	if len(k.Removed) > 0 {
		parts = append(parts, "removed "+strings.Join(k.Removed, ", "))
	}
	return k.Note + ": " + strings.Join(parts, "; ")
}

// shown says whether a note is on the screen, and what was linked to put
// it there.
func shown(s api.Shown) string {
	switch {
	case !s.Open:
		return "hidden " + s.Note
	case s.Linked != "":
		return fmt.Sprintf("showing %s (linked to %s)", s.Note, s.Linked)
	}
	return "showing " + s.Note
}

// linked says what a link points to.
func linked(l api.Linked) string { return fmt.Sprintf("linked %s to %s", l.Note, l.Target) }

// movedTo says where a note went.
func movedTo(m api.Moved) string { return fmt.Sprintf("moved %s to %s", m.From, m.To) }

// removed says where a removed note went and how to bring it back.
func removed(m api.Moved) string {
	return fmt.Sprintf("%s; `stickypane restore %s` brings it back", movedTo(m), strings.TrimSuffix(m.From, ".md"))
}

// restored says what came back.
func restored(m api.Moved) string { return "restored " + m.To }

// said words what api answered, when it did not fail: the agent reads the
// sentence, api only says what happened.
func said[T any](v T, err error) func(func(T) string) (string, error) {
	return func(say func(T) string) (string, error) {
		if err != nil {
			return "", err
		}
		return say(v), nil
	}
}
