package main

import (
	"context"
	"fmt"
	"io"

	"github.com/LeeSwallow/stickypane/internal/initcmd"
)

// env is what every command gets: the context, which ends when the user
// interrupts, and the standard streams. Tests hand in buffers.
type env struct {
	ctx    context.Context
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

// command is one word of the command line. Its usage is the lines of the
// usage text that start with "stickypane <name>", so the help and the table
// cannot drift apart.
type command struct {
	names []string // the name and its aliases
	run   func(e env, args []string) error
}

// commands is the table run looks words up in, in the order of the usage
// text. It is filled in init, because help reads it.
var commands []command

func init() {
	commands = []command{
		{[]string{"init"}, initCmd},
		{[]string{"setup"}, setupCmd},
		{[]string{"theme"}, themeCmd},
		{[]string{"language"}, languageCmd},
		{[]string{"config"}, configCmd},
		{[]string{"kinds"}, kindsCmd},
		{[]string{"env"}, envCmd},
		{[]string{"guide"}, guideCmd},
		{[]string{"version", "--version", "-v"}, func(e env, _ []string) error { _, err := fmt.Fprintln(e.stdout, "stickypane", version); return err }},
		{[]string{"help", "--help", "-h"}, func(e env, _ []string) error { _, err := io.WriteString(e.stdout, usage); return err }},
		{[]string{"list"}, listCmd},
		{[]string{"index"}, indexCmd},
		{[]string{"show"}, manageCmd("show")},
		{[]string{"hide"}, manageCmd("hide")},
		{[]string{"cat"}, catCmd},
		{[]string{"write"}, writeCmd},
		{[]string{"todo"}, editCmd("todo")},
		{[]string{"card"}, editCmd("card")},
		{[]string{"chart"}, editCmd("chart")},
		{[]string{"log"}, editCmd("log")},
		{[]string{"say"}, editCmd("say")},
		{[]string{"set"}, editCmd("set")},
		{[]string{"mv"}, manageCmd("mv")},
		{[]string{"link"}, manageCmd("link")},
		{[]string{"rm"}, manageCmd("rm")},
		{[]string{"restore"}, manageCmd("restore")},
		{[]string{"archive"}, manageCmd("archive")},
		{[]string{"answers"}, answersCmd(false)},
		{[]string{"wait"}, answersCmd(true)},
		{[]string{"api"}, apiCmd},
		{[]string{"watch"}, watchCmd},
		{[]string{"mcp"}, mcpCmd},
	}
}

// lookup finds the command a word names.
func lookup(word string) (command, bool) {
	for _, c := range commands {
		for _, n := range c.names {
			if n == word {
				return c, true
			}
		}
	}
	return command{}, false
}

// initCmd makes the board and puts the agent guide where agents read it.
func initCmd(e env, args []string) error {
	fs := newFlags("init")
	noDocs := fs.Bool("no-agent-docs", false, "leave AGENTS.md and CLAUDE.md alone")
	skill := fs.Bool("skill", false, "install the guide as a Claude Code skill instead")
	if _, err := parse(fs, "init", args, 0, 0); err != nil {
		return err
	}
	return initcmd.Run(".", initcmd.Options{NoAgentDocs: *noDocs, Skill: *skill}, e.stdout)
}
