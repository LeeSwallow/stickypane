package main

import (
	"fmt"
)

// apiCmd lists the requests of a .http note, or sends one or all of them and
// prints their logs. It is the agent's way to send what it wrote, without
// the user's yes: the agent could send the same with curl.
func apiCmd(e env, args []string) error {
	fs := newFlags("api")
	envName := fs.String("env", "", "the environment to use")
	all := fs.Bool("all", false, "send every request, in order")
	words, err := parse(fs, "api", args, 1, 2)
	if err != nil {
		return err
	}
	a, err := open(e, false)
	if err != nil {
		return err
	}
	request := ""
	if len(words) == 2 {
		request = words[1]
	}
	out, _, err := a.HTTP(e.ctx, words[0], request, *envName, *all)
	fmt.Fprint(e.stdout, out)
	return err
}
