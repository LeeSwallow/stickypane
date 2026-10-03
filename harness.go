// Package stickypane holds the files the plugin is made of: the skills,
// the rules they share and the commands that open them. The program carries
// them too, so that an agent over MCP or at the command line reads the same
// skills a plugin install gives it.
package stickypane

import "embed"

// Harness is the plugin's skills, rules and commands.
//
//go:embed skills rules commands
var Harness embed.FS
