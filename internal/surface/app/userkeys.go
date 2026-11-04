package app

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/LeeSwallow/stickypane/internal/core/i18n"
)

// A KeyProblem is a part of the user's key file the board could not use:
// an action it does not have, a key it cannot read, or a key two actions
// want. The action keeps the keys it comes with.
type KeyProblem struct {
	Action string // the action, as the file names it
	Why    string // in English
	err    *KeyError
}

func (p KeyProblem) String() string { return p.Action + ": " + p.Why }

// Say is why, in tr's language.
func (p KeyProblem) Say(tr i18n.Strings) string {
	if p.err == nil {
		return p.Why
	}
	return p.err.Say(tr)
}

func problem(action string, e *KeyError) KeyProblem {
	return KeyProblem{Action: action, Why: e.Error(), err: e}
}

// A KeyError is why a key cannot be had, in any language: the sentence of
// i18n.Strings that pick chooses, filled with vals. Error is the English.
type KeyError struct {
	pick  func(i18n.Strings) string
	vals  map[string]any
	inner *KeyError // the key of a sequence that is wrong
}

func keyErr(pick func(i18n.Strings) string, vals map[string]any) *KeyError {
	return &KeyError{pick: pick, vals: vals}
}

func (e *KeyError) Error() string { return e.Say(i18n.English()) }

// Say is e in tr's language. The action called Other is named by its
// words as OtherWords.
func (e *KeyError) Say(tr i18n.Strings) string {
	vals := maps.Clone(e.vals)
	if vals == nil {
		vals = map[string]any{}
	}
	if e.inner != nil {
		vals["Why"] = e.inner.Say(tr)
	}
	if other, ok := vals["Other"].(string); ok {
		vals["OtherWords"] = other
		if a, ok := defaultKeys.byName[other]; ok {
			vals["OtherWords"] = defaultKeys.actions[a].describe(tr)
		}
	}
	return i18n.Fill(e.pick(tr), vals)
}

// SayKeyError is err in tr's language when it is a KeyError, else as it is.
func SayKeyError(err error, tr i18n.Strings) string {
	var ke *KeyError
	if errors.As(err, &ke) {
		return ke.Say(tr)
	}
	return err.Error()
}

// A Binding is one of the board's actions as stickypane keys lists it.
type Binding struct {
	Name     string   `json:"name"`
	Keys     []string `json:"keys"`
	Words    string   `json:"words"`
	Changed  bool     `json:"changed"`  // the key file gave it these keys
	Defaults []string `json:"defaults"` // the keys it comes with
	Group    string   `json:"group"`    // what it is about (keyGroups)
}

// Bindings is every action of the board with the keys it has once want,
// a key file, is laid over the keys it comes with; its words are in tr's
// language. The problems are what of want was left out.
func Bindings(tr i18n.Strings, want map[string][]string) ([]Binding, []KeyProblem) {
	km, changed, problems := bindKeys(want)
	out := make([]Binding, len(km.actions))
	for i, a := range km.actions {
		out[i] = Binding{
			Name: a.name, Keys: append([]string{}, a.keys...), Words: a.describe(tr), Changed: changed[a.name],
			Defaults: append([]string{}, defaultKeys.keysOf(a.name)...), Group: groupOf(a.name),
		}
	}
	return out, problems
}

// UseKeys gives the board the keys of the key file at file: want is what
// it asks for, err why it could not be read. What could not be used keeps
// the keys it comes with, and the line returned says so, naming the file,
// for the bottom line; it is "" when all of it was used.
func (m *Model) UseKeys(file string, want map[string][]string, err error) string {
	if err != nil {
		m.keys = defaultKeys
		return m.keyed(m.tr.KeysUnreadable, map[string]any{"File": file})
	}
	km, _, problems := bindKeys(want)
	m.keys = km
	if len(problems) == 0 {
		return ""
	}
	var names []string
	for _, p := range problems {
		if !slices.Contains(names, p.Action) {
			names = append(names, p.Action)
		}
	}
	return m.keyed(m.tr.KeysIgnored, map[string]any{"File": file, "Names": strings.Join(names, ", ")})
}

// bindKeys lays want over the board's keys. An action want does not name
// keeps its keys; one it names gets the keys it lists, none for []. What
// cannot be used is a problem, and its action keeps its keys: a name that
// is no action, a key that is not one, and a key the action would share
// with another (the one the file changed gives it up).
func bindKeys(want map[string][]string) (*keymap, map[string]bool, []KeyProblem) {
	acts := actions()
	at := map[string]int{}
	for i, a := range acts {
		at[a.name] = i
	}
	changed := map[string]bool{}
	var problems []KeyProblem
	for _, name := range slices.Sorted(maps.Keys(want)) {
		i, ok := at[name]
		if !ok {
			problems = append(problems, problem(name, keyErr(func(t i18n.Strings) string { return t.KeyNoAction }, map[string]any{"Action": name})))
			continue
		}
		keys, err := NormalKeys(want[name])
		if err != nil {
			problems = append(problems, problem(name, err.(*KeyError)))
			continue
		}
		acts[i].keys, changed[name] = keys, true
	}
	// A key two actions want: the one the file changed keeps the keys it
	// comes with. Defaults never share a key, so this ends.
	for {
		i, j, key := sharedKey(acts)
		if i < 0 {
			break
		}
		back, other := j, i
		if !changed[acts[j].name] {
			back, other = i, j
		}
		problems = append(problems, problem(acts[back].name, keyErr(func(t i18n.Strings) string { return t.KeyShared },
			map[string]any{"Plain": key, "Key": strconv.Quote(key), "Other": acts[other].name})))
		acts[back].keys = defaultKeys.keysOf(acts[back].name)
		delete(changed, acts[back].name)
	}
	km, err := newKeymap(acts)
	if err != nil { // cannot happen: no key is shared now
		return defaultKeys, nil, append(problems, KeyProblem{Action: "keys", Why: err.Error()})
	}
	return km, changed, problems
}

// sharedKey finds a key two actions have: the first action, the second,
// and the key; -1 when there is none.
func sharedKey(acts []action) (int, int, string) {
	owner := map[string]int{}
	for j, a := range acts {
		for _, k := range a.keys {
			if i, ok := owner[k]; ok && i != j {
				return i, j, k
			}
			owner[k] = j
		}
	}
	return -1, -1, ""
}

// NormalKeys is keys as the terminal names them (see NormalKey), each
// once, in their order; or why one of them is not a key.
func NormalKeys(keys []string) ([]string, error) {
	out := []string{}
	for _, k := range keys {
		n, err := NormalKey(k)
		if err != nil {
			return nil, err
		}
		if !slices.Contains(out, n) {
			out = append(out, n)
		}
	}
	return out, nil
}

// modifiers are the modifiers a key may have, in the order the terminal
// writes them: "ctrl+alt+x", never "alt+ctrl+x".
var modifiers = []string{"ctrl", "alt", "shift", "meta", "hyper", "super"}

// namedKeys are the keys the terminal names with a word.
var namedKeys = func() map[string]bool {
	ks := map[string]bool{}
	for _, k := range strings.Fields("enter tab backspace esc space up down left right home end pgup pgdown insert delete begin find select") {
		ks[k] = true
	}
	for n := 1; n <= 63; n++ {
		ks[fmt.Sprintf("f%d", n)] = true
	}
	return ks
}()

// keyAliases are other words for a named key.
var keyAliases = map[string]string{"pgdn": "pgdown", "escape": "esc", "return": "enter", "del": "delete"}

// NormalKey is key as the terminal names it when it is pressed, which is
// how the board matches it: a character ("x", "X", "?", "가"), or a named
// key ("enter", "pgdown", "f5"), after modifiers ("ctrl+x", "shift+tab").
// Modifiers are put in the terminal's order, "pgdn" is "pgdown", and
// "shift+x" is "X", which is what the terminal says for it; a jamo of the
// Korean 2-set layout is the Latin key it is on ("ㅓ" is "j"). ctrl+c is
// not a key the board can give: it always quits.
//
// Keys written with spaces between are a sequence, pressed one after the
// other ("g g"). A sequence cannot start with a digit, which is read as a
// count, nor go on after esc, which drops the keys before it.
func NormalKey(key string) (string, error) {
	parts := strings.Fields(key)
	if len(parts) < 2 {
		return normalKey(key)
	}
	for i, p := range parts {
		n, err := normalKey(p)
		if err != nil {
			e := keyErr(func(t i18n.Strings) string { return t.KeyInSeq }, map[string]any{"Seq": strconv.Quote(key)})
			e.inner = err.(*KeyError)
			return "", e
		}
		switch {
		case i == 0 && isDigit(n):
			return "", keyErr(func(t i18n.Strings) string { return t.KeySeqDigit }, map[string]any{"Key": strconv.Quote(key)})
		case i > 0 && n == "esc":
			return "", keyErr(func(t i18n.Strings) string { return t.KeySeqEsc }, map[string]any{"Key": strconv.Quote(key)})
		}
		parts[i] = n
	}
	return strings.Join(parts, " "), nil
}

// normalKey is NormalKey of one key.
func normalKey(key string) (string, error) {
	k := latinKey(strings.TrimSpace(key))
	q := map[string]any{"Key": strconv.Quote(k)}
	with := func(more map[string]any) map[string]any {
		m := maps.Clone(q)
		maps.Copy(m, more)
		return m
	}
	if k == "" {
		return "", keyErr(func(t i18n.Strings) string { return t.KeyEmpty }, nil)
	}
	base, mods := k, []string{}
	if i := strings.LastIndex(k[:len(k)-1], "+"); i >= 0 {
		base = k[i+1:]
		for _, mod := range strings.Split(k[:i], "+") {
			mod = strings.ToLower(mod)
			if !slices.Contains(modifiers, mod) {
				return "", keyErr(func(t i18n.Strings) string { return t.KeyBadModifier }, with(map[string]any{"Mod": strconv.Quote(mod), "Mods": strings.Join(modifiers, ", ")}))
			}
			if slices.Contains(mods, mod) {
				return "", keyErr(func(t i18n.Strings) string { return t.KeyModTwice }, with(map[string]any{"Mod": mod}))
			}
			mods = append(mods, mod)
		}
	}
	if utf8.RuneCountInString(base) > 1 {
		base = strings.ToLower(base)
		if a, ok := keyAliases[base]; ok {
			base = a
		}
		if !namedKeys[base] {
			return "", keyErr(func(t i18n.Strings) string { return t.KeyUnknownName }, with(map[string]any{"Name": strconv.Quote(base)}))
		}
	} else if r, _ := utf8.DecodeRuneInString(base); !unicode.IsPrint(r) {
		return "", keyErr(func(t i18n.Strings) string { return t.KeyNotAKey }, q)
	}
	if len(mods) == 1 && mods[0] == "shift" && utf8.RuneCountInString(base) == 1 {
		r, _ := utf8.DecodeRuneInString(base)
		if !unicode.IsLetter(r) {
			return "", keyErr(func(t i18n.Strings) string { return t.KeyShiftChar }, q)
		}
		base, mods = string(unicode.ToUpper(r)), nil
	}
	// With ctrl or alt a capital is the letter with shift: "ctrl+X" is
	// "ctrl+shift+x".
	if r, _ := utf8.DecodeRuneInString(base); len(mods) > 0 && utf8.RuneCountInString(base) == 1 && unicode.IsUpper(r) {
		base = string(unicode.ToLower(r))
		if !slices.Contains(mods, "shift") {
			mods = append(mods, "shift")
		}
	}
	slices.SortFunc(mods, func(a, b string) int { return slices.Index(modifiers, a) - slices.Index(modifiers, b) })
	out := strings.Join(append(mods, base), "+")
	if out == "ctrl+c" {
		return "", keyErr(func(t i18n.Strings) string { return t.KeyCtrlC }, nil)
	}
	return out, nil
}

// ShowKey is a key as the board shows it: a sequence's characters side by
// side, "g g" as gg.
func ShowKey(key string) string { return showKey(key) }

// keyed fills a message's placeholders, and the keys it names: an action's
// name in CamelCase ({{.Undo}}, {{.ThemeNext}}) is its first key, "" when
// it has none.
func (m *Model) keyed(s string, values map[string]any) string {
	all := map[string]any{}
	for _, a := range m.keys.actions {
		all[camel(a.name)] = m.keys.key(a.name)
	}
	maps.Copy(all, values)
	return i18n.Fill(s, all)
}

// camel is a kebab-case name in CamelCase: "theme-next" is "ThemeNext".
func camel(name string) string {
	parts := strings.Split(name, "-")
	for i, p := range parts {
		if p != "" {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}
