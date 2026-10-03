package main

import (
	"errors"
	"flag"
	"io"
	"regexp"
	"slices"
)

// negative matches a word that is a negative number, not a flag.
var negative = regexp.MustCompile(`^-[0-9][0-9,_.]*$`)

// newFlags returns a flag set for a command that reports nothing itself:
// errors come back to exitCode, which prints them with the usage.
func newFlags(cmd string) *flag.FlagSet {
	fs := flag.NewFlagSet("stickypane "+cmd, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

// parse reads the flags of cmd wherever they are among its words, and
// checks that between least and most words are left (most < 0: any). The
// standard flag package stops at the first word; looping lets an agent
// write `stickypane card work move login --to Done` and `--to Done` first
// alike. A flag the command does not have is a usage error, not text.
//
// A negative number ("-5", for a chart) is a word, and everything after
// "--" is words as written.
func parse(fs *flag.FlagSet, cmd string, args []string, least, most int) ([]string, error) {
	var words, rest []string
	if i := slices.Index(args, "--"); i >= 0 {
		args, rest = args[:i], args[i+1:]
	}
	for {
		if len(args) > 0 && negative.MatchString(args[0]) {
			words, args = append(words, args[0]), args[1:]
			continue
		}
		if err := fs.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil, usageError{cmd: cmd}
			}
			return nil, usagef(cmd, "%v", err)
		}
		if fs.NArg() == 0 {
			break
		}
		words = append(words, fs.Arg(0))
		args = fs.Args()[1:]
	}
	words = append(words, rest...)
	if len(words) < least || (most >= 0 && len(words) > most) {
		return nil, usageError{cmd: cmd}
	}
	return words, nil
}
