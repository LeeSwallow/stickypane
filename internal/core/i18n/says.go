package i18n

import (
	"errors"
	"regexp"
	"slices"
	"strings"
)

// says is what the command line says to a person, in English, by key:
// tables, plans, confirmations, questions and refusals. A language lays
// its own over them under "says"; a key it leaves out keeps the English.
// What only an agent or a script reads (--json, the note commands' lines,
// MCP, the bridge) and what goes to a log is not here: it stays English.
var says = map[string]string{
	// Lists inside a sentence.
	"list.and": "{{.Rest}} and {{.Last}}",
	"list.or":  "{{.Rest}} or {{.Last}}",

	// Any command.
	"error.noBoard":        "No .sticky folder here or above. Run `stickypane init` in your project to make one.",
	"error.unknownCommand": "unknown command {{.Name}}",
	"usage.more":           "More: stickypane help {{.Cmd}}",
	"help.unknownNear":     "no command or shape is called {{.Name}}; did you mean {{.Near}}? (stickypane help {{.Cmd}})",
	"help.unknown":         "no command or shape is called {{.Name}}; stickypane help lists the commands, stickypane kinds the shapes",
	"help.all":             "Every command",

	// stickypane init, and the board it makes.
	"init.badView":     "--view is terminal, app or both, not {{.Value}}",
	"init.badStyle":    "--style is doodle or neat, not {{.Value}}",
	"init.badLanguage": "--language is {{.Languages}} or auto, not {{.Value}}",
	"init.rootFolder":  "will not make a board here: {{.Dir}} is the root folder; run stickypane init in a project",
	"init.homeFolder":  "will not make a board here: {{.Dir}} is the home folder, and every project below it would find the board; run stickypane init in a project",
	"init.made":        "Made {{.Dir}}/ with a welcome note.",
	"init.exists":      "{{.Dir}}/ already exists.",
	"init.added":       "added the agent guide to {{.Files}}",
	"init.current":     "the agent guide in {{.Files}} is up to date",
	"init.removed":     "took the agent guide out of {{.Files}}: the board plugin teaches it",
	"init.plugin":      "the board plugin teaches your agent",
	"init.join":        "{{.Before}}; {{.After}}",
	"init.end":         "{{.Line}}.",
	"init.nextSplit":   "Next: `{{.Split}}` opens the board next to your agent.",
	"init.next":        "Next: run `stickypane` in a pane next to your agent.",

	// The desktop app, from the terminal.
	"app.opened":     "opened {{.Dir}} in the stickypane app",
	"app.missing":    "this board shows in the desktop app, which is not installed (get it from {{.URL}}); showing it here",
	"app.cannotOpen": "could not open the stickypane app: {{.Err}}",

	// stickypane setup.
	"setup.jsonNeedsStatus": "--json goes with --status",
	"setup.badScope":        "--scope is user, project or local, not {{.Value}}",
	"setup.badCommand":      "--command is a full path, not {{.Value}}",
	"setup.pluginNeedsMCP":  "--plugin=false leaves only the MCP server: add --mcp",
	"setup.badAgent":        "--agents names claude or codex, not {{.Value}}",
	"setup.askScope":        "Install for every project (user), this repository shared (project) or just you here (local)?",
	"setup.askMCP":          "Also register stickypane as an MCP server, for agents that prefer tools?",
	"setup.askRun":          "Run these?",
	"setup.noAgent":         "No agent found: no {{.Commands}} command on the PATH. Install Claude Code or Codex first, or give any other agent the guide with `stickypane init` in a project.",
	"setup.nothing":         "Nothing to do: the board plugin is set up for every agent found.",
	"setup.runs":            "stickypane setup runs:",
	"setup.alreadyDone":     "(already done)",
	"setup.toRun":           "Run `stickypane setup --yes` to do it, with the same flags.",
	"setup.next":            "Next: `stickypane init` in a project makes its board; in a git repository the agent's first `stickypane show` does too.",
	"setup.carries":         "This stickypane carries the board plugin {{.Version}}.",
	"setup.notFound":        "{{.Name}}: not found (no {{.Command}} command on the PATH)",
	"setup.state.off":       "not connected",
	"setup.state.ok":        "connected",
	"setup.state.outdated":  "an update is waiting",
	"setup.state.broken":    "broken",
	"setup.state.missing":   "not found",
	"setup.word.missing":    "missing",
	"setup.word.off":        "off",
	"setup.word.outdated":   "outdated",
	"setup.word.broken":     "broken",
	"setup.plugin":          "plugin",
	"setup.mcp":             "MCP server",
	"setup.marketplace":     "marketplace",
	"setup.error":           "error",
	"setup.none":            "none",
	"setup.folderGone":      "(the folder is gone)",
	"setup.fixGone":         "`stickypane setup --agents {{.Agent}}` points the marketplace at GitHub again and installs the plugin anew.",
	"setup.fixUpdate":       "`stickypane setup --agents {{.Agent}}` updates it.",

	// stickypane config, theme and language, the S panel and :set.
	"config.default":     "(default)",
	"config.more":        "more:",
	"setting.unknown":    "there is no setting {{.Name}}; there are {{.Settings}}",
	"setting.noTheme":    "there is no theme {{.Value}}",
	"setting.noLanguage": "there is no translation for {{.Value}}: use {{.Values}}",
	"setting.badStyle":   "style {{.Value}} is not doodle or neat",
	"setting.badView":    "view {{.Value}} is not terminal, overlay or both",
	"setting.refused":    "{{.Key}} is one of {{.Values}}, not {{.Value}}",
	"theme.missing":      "there is no theme {{.Value}}. The themes are:{{.List}}",
	"theme.auto":         "follows the terminal: {{.Dark}} or {{.Light}}",
	"theme.dark":         "dark",
	"theme.light":        "light",
	"language.missing":   "there is no translation for {{.Value}}. The languages are:{{.List}}",
	"language.auto":      "follows STICKYPANE_LANG, LC_ALL, LC_MESSAGES, LANG; now {{.Now}}",

	// stickypane keys.
	"keys.unreadable":      "{{.File}}: {{.Err}}; the keys are as they come",
	"keys.problem":         "{{.File}}: {{.Action}}: {{.Why}}; it keeps the keys it comes with",
	"keys.setUnreadable":   "{{.File}}: {{.Err}}; stickypane keys reset starts it again",
	"keys.resetUnreadable": "{{.File}}: {{.Err}}; stickypane keys reset removes it",
	"keys.removed":         "{{.File}} removed; the keys are as they come",
	"keys.noAction":        "there is no action {{.Action}}; stickypane keys lists them",

	// stickypane env, after each row's name.
	"env.split":      "{{.Pane}}: `{{.Split}}` opens the board next to your agent",
	"env.noPane":     "no tmux, Zellij, WezTerm or Windows Terminal here: run `stickypane` in a terminal beside your agent",
	"env.langNone":   "nothing set, so English",
	"env.langFrom":   "from {{.Var}}={{.Value}}",
	"env.langSystem": "from the system's locale {{.Locale}}",
	"env.language":   "{{.Language}} ({{.From}}); to change it: `{{.Set}}`, or `stickypane config language {{.Other}}`",
	"env.noScript":   "not available: {{.Err}}",
}

// Say is the sentence under key in s's language, filled with values; the
// English when the language has none, and the key itself when there is no
// such sentence at all (a test keeps that from happening).
func (s Strings) Say(key string, values map[string]any) string {
	t := s.Says[key]
	if t == "" {
		t = says[key]
	}
	if t == "" {
		return key
	}
	return Fill(t, values)
}

// SaysKeys are the keys of every sentence of the command line.
func SaysKeys() []string {
	keys := make([]string, 0, len(says))
	for k := range says {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// List says items as one phrase: "a", "a and b", "a, b and c"; with or,
// "a or b".
func (s Strings) List(items []string, or bool) string {
	if len(items) < 2 {
		return strings.Join(items, "")
	}
	key := "list.and"
	if or {
		key = "list.or"
	}
	return s.Say(key, map[string]any{"Rest": strings.Join(items[:len(items)-1], ", "), "Last": items[len(items)-1]})
}

// A Sayer is an error that can say itself in any language. Its Error is
// the English, for logs and for what wraps it.
type Sayer interface {
	error
	Say(s Strings) string
}

// A Message is an error that is one of the command line's sentences: its
// key and its values. A value that is an error is said in the same
// language, when it can be. Cause, when there is one, is what errors.Is
// finds under it.
type Message struct {
	Key    string
	Values map[string]any
	Cause  error
}

// Err is the error that says the sentence under key with values.
func Err(key string, values map[string]any) *Message {
	return &Message{Key: key, Values: values}
}

func (m *Message) Error() string { return m.Say(Strings{}) }

// Say is m in s's language.
func (m *Message) Say(s Strings) string {
	values := make(map[string]any, len(m.Values))
	for k, v := range m.Values {
		if err, ok := v.(error); ok {
			v = Speak(err, s)
		}
		values[k] = v
	}
	return s.Say(m.Key, values)
}

func (m *Message) Unwrap() error { return m.Cause }

// Speak is err as a person reads it in s's language: a Sayer in it is
// said in that language, and what wraps it before it (a "file: " in front)
// stays as it is. Any other error is its English.
func Speak(err error, s Strings) string {
	var sayer Sayer
	if !errors.As(err, &sayer) {
		return err.Error()
	}
	if before, ok := strings.CutSuffix(err.Error(), sayer.Error()); ok {
		return before + sayer.Say(s)
	}
	return err.Error()
}

var placeholder = regexp.MustCompile(`\{\{[^}]*?\.([A-Z][A-Za-z]*)`)

// Placeholders are the names a sentence fills, sorted, each once.
func Placeholders(text string) []string {
	var names []string
	for _, m := range placeholder.FindAllStringSubmatch(text, -1) {
		names = append(names, m[1])
	}
	slices.Sort(names)
	return slices.Compact(names)
}
