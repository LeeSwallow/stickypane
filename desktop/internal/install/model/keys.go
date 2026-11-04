package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// A Binding is one action of the terminal board as `stickypane keys
// --json` gives it: its name, the keys it has, what it does in the
// language asked for, whether keys.json changed it, the keys it comes
// with and its group. A stickypane older than the groups leaves the last
// two out.
type Binding struct {
	Name     string   `json:"name"`
	Keys     []string `json:"keys"`
	Words    string   `json:"words"`
	Changed  bool     `json:"changed"`
	Defaults []string `json:"defaults,omitempty"`
	Group    string   `json:"group,omitempty"`
}

// ErrNoKeys: the stickypane run printed no list of keys; it is older
// than `stickypane keys`.
var ErrNoKeys = errors.New("this stickypane cannot list the board's keys; update it")

// KeysArgs are the arguments that list the keys.
func KeysArgs() []string { return []string{"keys", "--json"} }

// SetKeysArgs are the arguments that give action keys; none unbinds it.
// The keys come after "--", so that a key such as "-" is not a flag.
func SetKeysArgs(action string, keys []string) []string {
	return append([]string{"keys", "set", action, "--"}, keys...)
}

// ResetKeysArgs are the arguments that take action's change back, or
// every change when action is "".
func ResetKeysArgs(action string) []string {
	if action == "" {
		return []string{"keys", "reset"}
	}
	return []string{"keys", "reset", action}
}

// ParseKeys reads what `stickypane keys --json` printed, its output and
// errors together: the list is its last line that starts with "[", and
// the lines before it say what of keys.json the board left out.
func ParseKeys(out string) ([]Binding, []string, error) {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	at := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.HasPrefix(lines[i], "[") {
			at = i
			break
		}
	}
	if at < 0 {
		return nil, nil, ErrNoKeys
	}
	var bs []Binding
	if err := json.Unmarshal([]byte(lines[at]), &bs); err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrNoKeys, err)
	}
	var problems []string
	for _, l := range lines[:at] {
		if l = strings.TrimSpace(l); l != "" {
			problems = append(problems, l)
		}
	}
	return bs, problems, nil
}

// RunError is the sentence a failed stickypane run printed first,
// without the "stickypane: " it starts with; the usage after it is left
// out.
func RunError(out string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(out), "\n")
	return strings.TrimPrefix(strings.TrimSpace(line), Name+": ")
}
