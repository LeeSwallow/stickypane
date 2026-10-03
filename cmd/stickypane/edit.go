package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/LeeSwallow/stickypane/internal/when"
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
			out, err = a.Todo(name, rest[0], strings.Join(rest[1:], " "))
		case "card":
			out, err = a.Card(name, rest[0], strings.Join(rest[1:], " "), to)
		case "chart":
			out, err = a.Chart(name, rest[0], strings.Join(rest[1:len(rest)-1], " "), rest[len(rest)-1])
		case "log":
			line := strings.Join(rest, " ")
			if stamp {
				line = time.Now().Format(when.Clock) + " " + line
			}
			out, err = a.Log(name, line)
		case "say":
			out, err = a.Say(name, to, strings.Join(rest, " "))
		default: // set
			out, err = a.Set(name, rest)
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
			out, err = a.Remove(words[0])
		case "restore":
			out, err = a.Restore(words[0])
		case "archive":
			out, err = a.Archive(words[0])
		case "show":
			out, err = a.Show(words[0])
		case "hide":
			out, err = a.Hide(words[0])
		case "link":
			name := ""
			if len(words) == 2 {
				name = words[1]
			}
			out, err = a.Link(words[0], name)
		default: // mv
			out, err = a.Move(words[0], words[1])
		}
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(e.stdout, out)
		return err
	}
}
