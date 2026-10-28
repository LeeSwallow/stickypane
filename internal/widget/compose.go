package widget

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// The types of an Option, as a screen draws its field.
const (
	OptText   = "text"   // one line
	OptLines  = "lines"  // several lines, one value a line (or a text of several lines)
	OptChoice = "choice" // one of Choices
	OptNumber = "number" // a number, as strconv.ParseFloat reads it
	OptBool   = "bool"   // true or false
)

// Option is one thing a screen may ask before it makes a note of a kind: a
// board's columns, a chart's view. Label is the English phrase shown for
// it, and the key of its translation in the "labels" of a language file.
// Default is what the note gets when the option is not given; a lines
// option's values are separated by "\n".
type Option struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Label   string   `json:"label"`
	Choices []string `json:"choices,omitempty"`
	Default string   `json:"default"`
}

// ErrBadOption is the error for options a kind does not take, or values
// it cannot use.
var ErrBadOption = errors.New("bad option")

// Make writes a new note of this kind called title from options a screen
// filled in: each option of k.Options, checked and with its default where
// it is not given, handed to k.Compose. A kind without Compose takes no
// options and is made from its Template. The result is the whole file, as
// Template's is; nothing is written.
func (k Kind) Make(title string, given map[string]string) ([]byte, error) {
	if k.Compose == nil {
		if len(given) > 0 {
			return nil, fmt.Errorf("%w: a %s takes no options", ErrBadOption, k.Name)
		}
		return k.Template(title), nil
	}
	opts, err := k.checkOptions(given)
	if err != nil {
		return nil, err
	}
	return k.Compose(title, opts)
}

// checkOptions returns every option of the kind with its value: the one
// given, checked against its type, else the default.
func (k Kind) checkOptions(given map[string]string) (map[string]string, error) {
	known := map[string]Option{}
	var names []string
	for _, o := range k.Options {
		known[o.Name] = o
		names = append(names, o.Name)
	}
	var unknown []string
	for name := range given {
		if _, ok := known[name]; !ok {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return nil, fmt.Errorf("%w: a %s has no option %q; it takes %s", ErrBadOption, k.Name, unknown[0], strings.Join(names, ", "))
	}
	out := map[string]string{}
	for _, o := range k.Options {
		v, ok := given[o.Name]
		if !ok {
			v = o.Default
		}
		v, err := o.Check(v)
		if err != nil {
			return nil, fmt.Errorf("%w: %s of a %s: %v", ErrBadOption, o.Name, k.Name, err)
		}
		out[o.Name] = v
	}
	return out, nil
}

// Check returns v as the option keeps it, or says why it is not a value of
// the option's type.
func (o Option) Check(v string) (string, error) {
	switch o.Type {
	case OptText:
		if strings.ContainsAny(v, "\r\n") {
			return "", errors.New("is one line")
		}
		return strings.TrimSpace(v), nil
	case OptLines:
		return strings.ReplaceAll(v, "\r\n", "\n"), nil
	case OptChoice:
		for _, c := range o.Choices {
			if strings.EqualFold(strings.TrimSpace(v), c) {
				return c, nil
			}
		}
		return "", fmt.Errorf("%q is not one of %s", v, strings.Join(o.Choices, ", "))
	case OptNumber:
		v = strings.TrimSpace(v)
		if v == "" {
			return "", nil
		}
		if _, err := strconv.ParseFloat(v, 64); err != nil {
			return "", fmt.Errorf("%q is not a number", v)
		}
		return v, nil
	case OptBool:
		b, err := strconv.ParseBool(strings.TrimSpace(v))
		if err != nil {
			return "", fmt.Errorf("%q is not true or false", v)
		}
		return strconv.FormatBool(b), nil
	}
	return "", fmt.Errorf("the option's type %q is unknown", o.Type)
}

// Lines splits a lines option into its values, without blank ones and
// without the spaces around each.
func Lines(v string) []string {
	var out []string
	for _, l := range strings.Split(v, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}
