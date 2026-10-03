package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
)

// The exit codes. They are part of the command line's contract: scripts and
// agents branch on them.
const (
	exitOK          = 0
	exitFailed      = 1   // something went wrong; stderr says what
	exitUsage       = 2   // the arguments make no sense
	exitTimeout     = 3   // wait gave up before a button was pressed
	exitInterrupted = 130 // the user pressed Ctrl-C, as shells count it
)

const noBoard = "No .sticky folder here or above. Run `stickypane init` in your project to make one."

// errNoBoard is returned when there is no notes folder to work on.
var errNoBoard = errors.New(noBoard)

// usageError says the arguments make no sense. cmd names the command whose
// usage is printed after msg; an empty cmd prints the whole usage.
type usageError struct{ cmd, msg string }

func (e usageError) Error() string { return e.msg }

// usagef is a usage error for cmd.
func usagef(cmd, format string, a ...any) error {
	return usageError{cmd: cmd, msg: fmt.Sprintf(format, a...)}
}

// timeoutError says wait gave up.
type timeoutError struct{ msg string }

func (e timeoutError) Error() string { return e.msg }

// exitCode prints what err says and returns the exit code for it. It is
// the one place where errors become codes, so every command reports the
// same way.
func exitCode(err error, stderr io.Writer) int {
	var usage usageError
	var timeout timeoutError
	switch {
	case err == nil:
		return exitOK
	case errors.As(err, &usage):
		if usage.msg != "" {
			fmt.Fprintln(stderr, "stickypane:", usage.msg)
		}
		fmt.Fprint(stderr, usageOf(usage.cmd))
		return exitUsage
	case errors.Is(err, errNoBoard):
		fmt.Fprintln(stderr, noBoard)
		return exitFailed
	case errors.As(err, &timeout):
		fmt.Fprintln(stderr, "stickypane:", timeout.msg)
		return exitTimeout
	case errors.Is(err, context.Canceled):
		return exitInterrupted
	}
	fmt.Fprintln(stderr, "stickypane:", err)
	return exitFailed
}

// usageOf returns the lines of the usage text about one command: the line
// that starts with "stickypane <cmd>" and the indented lines under it. An
// empty or unknown command gets the whole text.
func usageOf(cmd string) string {
	if cmd == "" {
		return usage
	}
	var out []string
	in := false
	for _, line := range strings.Split(usage, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "stickypane "+cmd || strings.HasPrefix(trimmed, "stickypane "+cmd+" "):
			in = true
			out = append(out, line)
		case in && strings.HasPrefix(line, "      ") && !strings.HasPrefix(trimmed, "stickypane "):
			out = append(out, line)
		default:
			in = false
		}
	}
	if len(out) == 0 {
		return usage
	}
	return "usage:\n" + strings.Join(out, "\n") + "\n"
}
