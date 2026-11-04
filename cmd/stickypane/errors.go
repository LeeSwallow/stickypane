package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/core/i18n"
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

// errNoBoard is returned when there is no notes folder to work on.
var errNoBoard = i18n.Err("error.noBoard", nil)

// usageError says the arguments make no sense. cmd names the command whose
// usage is printed after msg; an empty cmd prints the whole usage. said,
// when there is one, is msg as the person reads it, in their language.
type usageError struct {
	cmd, msg string
	said     *i18n.Message
}

func (e usageError) Error() string { return e.msg }

// usagef is a usage error for cmd, in English: the note commands', which
// agents read.
func usagef(cmd, format string, a ...any) error {
	return usageError{cmd: cmd, msg: fmt.Sprintf(format, a...)}
}

// usageSay is a usage error for cmd that a person reads: the sentence
// under key, said in their language.
func usageSay(cmd, key string, values map[string]any) error {
	m := i18n.Err(key, values)
	return usageError{cmd: cmd, msg: m.Error(), said: m}
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
	if err == nil {
		return exitOK
	}
	lang := helpLanguage()
	tr, _ := i18n.Load(lang)
	switch {
	case errors.As(err, &usage):
		msg := usage.msg
		if usage.said != nil {
			msg = usage.said.Say(tr)
		}
		if msg != "" {
			fmt.Fprintln(stderr, "stickypane:", msg)
		}
		fmt.Fprint(stderr, usageIn(lang, tr, usage.cmd))
		return exitUsage
	case errors.Is(err, errNoBoard):
		fmt.Fprintln(stderr, errNoBoard.Say(tr))
		return exitFailed
	case errors.As(err, &timeout):
		fmt.Fprintln(stderr, "stickypane:", timeout.msg)
		return exitTimeout
	case errors.Is(err, context.Canceled):
		return exitInterrupted
	}
	fmt.Fprintln(stderr, "stickypane:", i18n.Speak(err, tr))
	return exitFailed
}

// usageIn is what a usage error prints after its message: in English the
// lines of the usage text about cmd (usageOf); in another language, the
// command's usage line and where its help is, or the overview.
func usageIn(lang string, tr i18n.Strings, cmd string) string {
	if lang == "en" {
		return usageOf(cmd)
	}
	c, ok := lookup(cmd)
	if !ok {
		return overview(tr)
	}
	return tr.T("label.usage", "Usage") + ":\n  " + c.help.syntax + "\n" + tr.Say("usage.more", map[string]any{"Cmd": c.names[0]}) + "\n"
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
