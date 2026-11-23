package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/core/when"
)

// editCmd runs a command that changes one thing in a note: todo, card,
// chart, log, chat or note set. The note is made when it is missing.
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
		case "chat":
			fs.StringVar(&to, "as", "agent", "who says it")
		}
		words, err := parse(fs, cmd, args, 0, -1)
		if err != nil {
			return err
		}
		words, ok := oldOrder(cmd, words)
		least := map[string]int{"todo": 3, "card": 3, "chart": 4, "log": 2, "chat": 2, "note set": 2}[cmd]
		if !ok || len(words) < least {
			return usageError{cmd: cmd}
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
		case "chat":
			out, err = said(a.Say(name, to, strings.Join(rest, " ")))(status)
		default: // note set
			out, err = said(a.Set(name, rest))(keyed)
		}
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(e.stdout, out)
		return err
	}
}

// shapeVerbs are the verbs of each shape command.
var shapeVerbs = map[string][]string{
	"todo": {"add", "check", "uncheck"}, "card": {"add", "move"},
	"chart": {"set", "add"}, "log": {"add"}, "chat": {"add"},
}

// oldOrder reads the words of a shape command, in the order
// <verb> <name> ... or the old <name> <verb> ..., into the old order the
// code takes: <name> <verb> ..., or <name> ... for log and chat, which had
// no verb. A first word that is a verb of the noun means the new order,
// even when the second is one too (#149): todo add check x adds x to the
// note check. chat has no old order (say was its old form); without its
// verb it is not ok.
func oldOrder(cmd string, words []string) ([]string, bool) {
	switch {
	case len(words) < 2 || !slices.Contains(shapeVerbs[cmd], words[0]):
		return words, cmd != "chat"
	case cmd == "log" || cmd == "chat":
		return words[1:], true
	}
	return append([]string{words[1], words[0]}, words[2:]...), true
}

// manageCmd runs a command that shows, hides, removes, restores or moves a
// note: cmd is note show, note hide, and so on.
func manageCmd(cmd string) func(env, []string) error {
	verb := strings.TrimPrefix(cmd, "note ")
	least, most := 1, 1
	switch verb {
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
		a, err := open(e, verb == "show" || verb == "link")
		if err != nil {
			return err
		}
		var out string
		switch verb {
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
