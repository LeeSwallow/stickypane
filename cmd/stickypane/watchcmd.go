package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	osexec "os/exec"
	"strings"
	"time"

	"github.com/LeeSwallow/stickypane/internal/api"
	environment "github.com/LeeSwallow/stickypane/internal/env"
)

// errEnough ends a watch that has what it waited for.
var errEnough = errors.New("enough")

// watchCmd prints what happens on the board as it happens, one event a
// line, and with --exec runs a command for each: a chart that counts what
// a checklist ticks, a chat that answers a form, a sound when a card
// reaches Done. It never runs anything the user did not start.
func watchCmd(e env, args []string) error {
	fs := newFlags("watch")
	asJSON := fs.Bool("json", false, "print each event as JSON")
	notes := fs.String("note", "", "only these notes or folders, comma-separated")
	types := fs.String("type", "", "only these events or their kinds (card, item.ticked), comma-separated")
	run := fs.String("exec", "", "run this command for each event, with STICKY_* variables")
	once := fs.Bool("once", false, "end after the first event")
	timeout := fs.Duration("timeout", 0, "give up after this long, such as 10m")
	if _, err := parse(fs, "watch", args, 0, 0); err != nil {
		return err
	}
	if *timeout < 0 {
		return usagef("watch", "--timeout cannot be negative")
	}
	a, err := open(e, false)
	if err != nil {
		return err
	}
	ctx := e.ctx
	if *timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *timeout)
		defer cancel()
	}
	f := api.Filter{Notes: list(*notes), Types: list(*types)}
	err = a.Watch(ctx, f, func(ev api.Event) error {
		if *asJSON {
			if err := printJSON(e.stdout, ev); err != nil {
				return err
			}
		} else {
			fmt.Fprintln(e.stdout, describe(ev))
		}
		if *run != "" {
			react(e, *run, ev)
		}
		if *once {
			return errEnough
		}
		return nil
	}, nil)
	switch {
	case errors.Is(err, errEnough):
		return nil
	case errors.Is(err, context.DeadlineExceeded):
		return timeoutError{fmt.Sprintf("nothing happened within %s", *timeout)}
	}
	return err
}

// describe writes an event for a person: when, where, what.
func describe(ev api.Event) string {
	line := ev.Time.Local().Format("15:04:05") + "  " + ev.Note + "  " + ev.Type
	if ev.Item != "" {
		line += "  " + ev.Item
	}
	if ev.From != "" || ev.To != "" {
		line += "  " + ev.From + " → " + ev.To
	}
	return line
}

// react runs the user's command for an event in their own shell, with the
// event in its environment: STICKY_EVENT (all of it, as JSON), STICKY_NOTE,
// STICKY_KIND, STICKY_TYPE, STICKY_ITEM, STICKY_FROM and STICKY_TO. A
// command that fails is reported and the watch goes on.
func react(e env, line string, ev api.Event) {
	all, _ := json.Marshal(ev)
	name, args := environment.Detect().Shell.Command(line)
	c := osexec.CommandContext(e.ctx, name, args...)
	c.Stdout, c.Stderr = e.stdout, e.stderr
	c.Env = append(os.Environ(),
		"STICKY_EVENT="+string(all),
		"STICKY_NOTE="+ev.Note,
		"STICKY_KIND="+ev.Kind,
		"STICKY_TYPE="+ev.Type,
		"STICKY_ITEM="+ev.Item,
		"STICKY_FROM="+ev.From,
		"STICKY_TO="+ev.To,
		"STICKY_TIME="+ev.Time.Format(time.RFC3339),
	)
	if err := c.Run(); err != nil {
		fmt.Fprintf(e.stderr, "stickypane: %s: %v\n", line, err)
	}
}

// list splits a comma-separated flag.
func list(s string) []string {
	var out []string
	for _, v := range strings.Split(s, ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}
