// Package stickypane holds the files the plugin is made of: the skills,
// the rules they share, the commands that open them and the head of the
// agent guide, all under plugin/. The program carries them too, so that an
// agent over MCP or at the command line reads the same skills a plugin
// install gives it.
package stickypane

import (
	"embed"
	"io/fs"
)

//go:embed plugin/skills plugin/rules plugin/commands plugin/guide.md
var files embed.FS

// Harness is the plugin's skills, rules, commands and guide.md, at the
// paths they have under plugin/.
var Harness = func() fs.FS {
	sub, err := fs.Sub(files, "plugin")
	if err != nil {
		panic(err)
	}
	return sub
}()
