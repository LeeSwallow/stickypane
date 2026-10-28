package main

import (
	"fmt"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/core/when"
)

// editCmd runs a command that changes one thing in a note: todo, card,
// chart, log, say or set. The note is made when it is missing.
func editCmd(cmd string) func(env, []string) error {
	return func(e env, args []string) error {
		fs := newFlags(cmd)
		var to string
		var stamp bool
		switch cmd {
		case "card":
			fs.StringVar(&to, "to", "", "the column")
		case "log":
			fs.BoolVar(&stamp, "time", false, "start the line with the time")
		case "say":
			fs.StringVar(&to, "as", "agent", "who says it")
		}
		least := map[string]int{"todo": 3, "card": 3, "chart": 4, "log": 2, "say": 2, "set": 2}[cmd]
		words, err := parse(fs, cmd, args, least, -1)
		if err != nil {
			return err
		}
		name, rest := words[0], words[1:]
		if cmd == "card" && rest[0] == "move" && to == "" {
			return usagef(cmd, "move needs --to <column>")
		}
		a, err := open(e, true)
		if err != nil {
			return err
		}
		var out string
		switch cmd {
		case "todo":
			out, err = said(a.Todo(name, rest[0], strings.Join(rest[1:], " ")))(status)
		case "card":
			out, err = said(a.Card(name, rest[0], strings.Join(rest[1:], " "), to))(status)
		case "chart":
			out, err = said(a.Chart(name, rest[0], strings.Join(rest[1:len(rest)-1], " "), rest[len(rest)-1]))(valued)
		case "log":
			line := strings.Join(rest, " ")
			if stamp {
				line = e.now().Format(when.Clock) + " " + line
			}
			out, err = said(a.Log(name, line))(status)
		case "say":
			out, err = said(a.Say(name, to, strings.Join(rest, " ")))(status)
		default: // set
			out, err = said(a.Set(name, rest))(keyed)
		}
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(e.stdout, out)
		return err
	}
}

// manageCmd runs a command that shows, hides, removes, restores or moves a
// note.
func manageCmd(cmd string) func(env, []string) error {
	least, most := 1, 1
	switch cmd {
	case "mv":
		least, most = 2, 2
	case "link":
		most = 2
	}
	return func(e env, args []string) error {
		words, err := parse(newFlags(cmd), cmd, args, least, most)
		if err != nil {
			return err
		}
		a, err := open(e, cmd == "show" || cmd == "link")
		if err != nil {
			return err
		}
		var out string
		switch cmd {
		case "rm":
			out, err = said(a.Remove(words[0]))(removed)
		case "restore":
			out, err = said(a.Restore(words[0]))(restored)
		case "archive":
			out, err = said(a.Archive(words[0]))(movedTo)
		case "show":
			out, err = said(a.Show(words[0]))(shown)
		case "hide":
			out, err = said(a.Hide(words[0]))(shown)
		case "link":
			name := ""
			if len(words) == 2 {
				name = words[1]
			}
			out, err = said(a.Link(words[0], name))(linked)
		default: // mv
			out, err = said(a.Move(words[0], words[1]))(movedTo)
		}
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(e.stdout, out)
		return err
	}
}
