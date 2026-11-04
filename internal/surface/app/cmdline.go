package app

import (
	"path"
	"slices"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/core/widget"
	"github.com/LeeSwallow/stickypane/internal/surface/lineedit"
	"github.com/LeeSwallow/stickypane/internal/usecase/api"
	"github.com/LeeSwallow/stickypane/internal/usecase/prefs"
)

// The command line is vi's: : opens a line at the bottom, a command and
// its arguments are typed, enter runs it. A command may be shortened to
// any start of its name that no other shares (:col 3, :th nord). Every
// command does what a key or the settings panel does, through the same
// code; it is a way to say it by name.

func init() {
	handlers[modeCommand] = commandUpdate
	footers[modeCommand] = func(m *Model) string {
		line := m.env.Styles.Bold.Render(":") + m.cmdLine.View()
		if m.cmdHint != "" {
			line += "  " + m.env.Styles.Faint.Render(m.cmdHint)
		}
		return line
	}
}

// cmdState is the command line: the line being typed, what tab offered,
// and the commands run this session, the newest last.
type cmdState struct {
	cmdLine    *lineedit.Line
	cmdHint    string   // what tab could complete to, when it was more than one
	cmdHistory []string // the lines run, oldest first
	cmdAt      int      // the line of the history shown; len(cmdHistory) is the one being typed
	cmdDraft   string   // the line being typed, kept while the history is shown
}

// A command is one thing the command line does, under a name that stays
// English in every language, like the keys.
type command struct {
	name     string
	alias    []string // other whole names: q for quit, col for columns
	args     string   // what it takes, as the help writes it
	english  string   // what it does; i18n.Strings.Actions has it as "cmd.<name>"
	run      func(m *Model, args []string, bang bool) tea.Cmd
	complete func(m *Model, before []string) []string // the values the next argument may take, given those before it
}

// commands is every command, in the order the help lists them.
func commands() []command {
	setting := func(key string) func(*Model, []string, bool) tea.Cmd {
		return func(m *Model, args []string, _ bool) tea.Cmd {
			m.setting(key, strings.Join(args, " "))
			return nil
		}
	}
	values := func(key string) func(*Model, []string) []string {
		return func(_ *Model, before []string) []string {
			if len(before) > 0 {
				return nil
			}
			return prefs.Lookup(key).Values()
		}
	}
	return []command{
		{name: "quit", alias: []string{"q", "wq"}, english: "quit", run: func(*Model, []string, bool) tea.Cmd { return tea.Quit }},
		{name: "write", alias: []string{"w"}, english: "saved as you go", run: func(m *Model, _ []string, _ bool) tea.Cmd {
			m.status = m.tr.SavesAsYouGo
			return nil
		}},
		{name: "set", args: "[setting [value]]", english: "see or set a setting", run: func(m *Model, args []string, _ bool) tea.Cmd {
			m.set(args)
			return nil
		}, complete: func(_ *Model, before []string) []string {
			switch len(before) {
			case 0:
				return prefKeys()
			case 1:
				if key, _ := pick(before[0], prefKeys(), nil); key != "" {
					return prefs.Lookup(key).Values()
				}
			}
			return nil
		}},
		{name: "theme", args: "[name]", english: "see or set the theme", run: setting("theme"), complete: values("theme")},
		{name: "columns", alias: []string{"col"}, args: "[n]", english: "see or set columns", run: setting("columns"), complete: values("columns")},
		{name: "language", alias: []string{"lang"}, args: "[code]", english: "see or set language", run: setting("language"), complete: values("language")},
		{name: "edit", alias: []string{"e"}, args: "[note]", english: "zoom a note by name", run: func(m *Model, args []string, _ bool) tea.Cmd {
			if len(args) == 0 {
				return openIndex(m)
			}
			m.editByName(strings.Join(args, " "))
			return nil
		}, complete: func(m *Model, before []string) []string {
			if len(before) > 0 {
				return nil
			}
			return m.noteNames()
		}},
		{name: "new", args: "[shape] [title]", english: "add a note", run: newNote, complete: func(m *Model, before []string) []string {
			if len(before) > 0 {
				return nil
			}
			var out []string
			for _, k := range m.reg {
				out = append(out, k.Name)
			}
			return out
		}},
		{name: "jot", args: "[text]", english: "jot a note", run: func(m *Model, args []string, _ bool) tea.Cmd {
			if len(args) == 0 {
				return jot(m)
			}
			text := strings.Join(args, " ")
			m.create(text, "", []byte(text+"\n"))
			return nil
		}},
		{name: "tab", args: "<n|name>", english: "go to a tab", run: func(m *Model, args []string, _ bool) tea.Cmd {
			m.goToTab(strings.Join(args, " "))
			return nil
		}, complete: func(m *Model, before []string) []string {
			if len(before) > 0 {
				return nil
			}
			var out []string
			for _, t := range m.tabList {
				if t.Name != "" {
					out = append(out, t.Name)
				}
			}
			return out
		}},
		{name: "archive", english: "move to archive", run: onNote(archive)},
		{name: "delete", english: "delete (to .trash)", run: onNote(remove)},
		{name: "restore", alias: []string{"undo"}, english: "undo archive, delete", run: func(m *Model, _ []string, _ bool) tea.Cmd { return undoMove(m) }},
		{name: "keys", english: "the keys", run: func(m *Model, _ []string, _ bool) tea.Cmd { return openHelp(m) }},
		{name: "help", args: "[command]", english: "help on a command", run: helpCommand, complete: func(_ *Model, before []string) []string {
			if len(before) > 0 {
				return nil
			}
			return commandNames()
		}},
	}
}

// onNote runs f on the focused note, as a key does.
func onNote(f func(*Model, item) tea.Cmd) func(*Model, []string, bool) tea.Cmd {
	return func(m *Model, _ []string, _ bool) tea.Cmd { return onFocused(f)(m) }
}

func commandNames() []string {
	var out []string
	for _, c := range commands() {
		out = append(out, c.name)
	}
	return out
}

func prefKeys() []string {
	var out []string
	for _, p := range prefs.All {
		out = append(out, p.Key)
	}
	return out
}

// pick finds what word names among names, with aliases as other whole
// names for them: the name it is, else the one name it starts. When it
// starts more than one, those are returned, and the name is "".
func pick(word string, names []string, alias map[string]string) (string, []string) {
	word = strings.ToLower(word)
	if slices.Contains(names, word) {
		return word, nil
	}
	if n, ok := alias[word]; ok {
		return n, nil
	}
	var starts []string
	for _, n := range names {
		if strings.HasPrefix(n, word) {
			starts = append(starts, n)
		}
	}
	for a, n := range alias {
		if strings.HasPrefix(a, word) && !slices.Contains(starts, n) {
			starts = append(starts, n)
		}
	}
	if len(starts) == 1 {
		return starts[0], nil
	}
	slices.Sort(starts)
	return "", starts
}

// findCommand is the command word names, or what is wrong with it, said
// on the bottom line.
func (m *Model) findCommand(word string) (command, string) {
	all := commands()
	alias := map[string]string{}
	for _, c := range all {
		for _, a := range c.alias {
			alias[a] = c.name
		}
	}
	name, starts := pick(word, commandNames(), alias)
	if name == "" {
		if len(starts) > 1 {
			return command{}, say(m.tr.CommandAmbiguous, map[string]any{"Name": word, "Choices": strings.Join(starts, ", ")})
		}
		return command{}, say(m.tr.NotACommand, map[string]any{"Name": word})
	}
	for _, c := range all {
		if c.name == name {
			return c, ""
		}
	}
	return command{}, ""
}

// openCommandLine opens the command line on the bottom line.
func openCommandLine(m *Model) tea.Cmd {
	m.cmdLine = lineedit.New("", max(m.width-2, 1))
	m.cmdHint, m.cmdAt, m.cmdDraft = "", len(m.cmdHistory), ""
	m.back, m.mode = m.mode, modeCommand
	return nil
}

// commandUpdate edits the line. Esc closes it, enter runs it, the arrows
// go through the lines run before and tab completes the word being typed.
// What is typed is taken as it is: a jamo stays a jamo here.
func commandUpdate(m *Model, msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.PasteMsg:
		m.cmdLine.Type(msg.Content)
	case tea.KeyPressMsg:
		k := msg.String()
		if k != "tab" {
			m.cmdHint = ""
		}
		switch k {
		case "esc":
			m.mode = m.back
		case "enter":
			line := strings.TrimSpace(m.cmdLine.Value())
			m.mode = m.back
			if line == "" {
				return nil
			}
			if n := len(m.cmdHistory); n == 0 || m.cmdHistory[n-1] != line {
				m.cmdHistory = append(m.cmdHistory, line)
			}
			return m.runCommand(line)
		case "up", "ctrl+p":
			m.recall(-1)
		case "down", "ctrl+n":
			m.recall(1)
		case "tab":
			m.completeLine()
		case "backspace":
			// Backspace on an empty line closes it, as in vi.
			if m.cmdLine.Value() == "" {
				m.mode = m.back
				return nil
			}
			m.cmdLine.Key(k)
		default:
			if !m.cmdLine.Key(k) && msg.Text != "" {
				m.cmdLine.Type(msg.Text)
			}
		}
	}
	return nil
}

// recall shows the line run step lines before or after the one shown; past
// the newest is the line that was being typed.
func (m *Model) recall(step int) {
	at := m.cmdAt + step
	if at < 0 || at > len(m.cmdHistory) {
		return
	}
	if m.cmdAt == len(m.cmdHistory) {
		m.cmdDraft = m.cmdLine.Value()
	}
	m.cmdAt = at
	text := m.cmdDraft
	if at < len(m.cmdHistory) {
		text = m.cmdHistory[at]
	}
	m.cmdLine = lineedit.New(text, m.cmdLine.Width)
}

// completeLine completes the word at the end of the line: a command's
// name, or the argument the command takes there. One way to go fills it
// in; several fill in what they share and are shown beside the line.
func (m *Model) completeLine() {
	text := m.cmdLine.Value()
	fields := strings.Fields(text)
	word, before := "", fields
	if !strings.HasSuffix(text, " ") && len(fields) > 0 {
		word, before = fields[len(fields)-1], fields[:len(fields)-1]
	}
	var offered []string
	if len(before) == 0 {
		offered = commandNames()
	} else if c, bad := m.findCommand(before[0]); bad == "" && c.complete != nil {
		offered = c.complete(m, before[1:])
	}
	var fit []string
	for _, o := range offered {
		if strings.HasPrefix(strings.ToLower(o), strings.ToLower(word)) {
			fit = append(fit, o)
		}
	}
	if len(fit) == 0 {
		return
	}
	head := strings.Join(before, " ")
	if head != "" {
		head += " "
	}
	if len(fit) == 1 {
		m.cmdLine = lineedit.New(head+fit[0]+" ", m.cmdLine.Width)
		return
	}
	m.cmdLine = lineedit.New(head+commonStart(fit), m.cmdLine.Width)
	m.cmdHint = strings.Join(fit, " ")
}

// commonStart is the longest start every one of words has.
func commonStart(words []string) string {
	start := words[0]
	for _, w := range words[1:] {
		for !strings.HasPrefix(strings.ToLower(w), strings.ToLower(start)) {
			start = start[:len(start)-1]
		}
	}
	return start
}

// runCommand runs a line of the command line: a number goes to that note
// of the tab, as a line number goes to a line in vi; anything else is a
// command and its arguments. What goes wrong is said on the bottom line.
func (m *Model) runCommand(line string) tea.Cmd {
	fields := strings.Fields(line)
	if n, err := strconv.Atoi(fields[0]); err == nil && len(fields) == 1 {
		m.goToNumber(n)
		return nil
	}
	word, bang := strings.CutSuffix(fields[0], "!")
	c, bad := m.findCommand(word)
	if bad != "" {
		m.status = bad
		return nil
	}
	return c.run(m, fields[1:], bang)
}

// goToNumber focuses the nth note of the tab, counting from 1 along the
// title bar.
func (m *Model) goToNumber(n int) {
	if n < 1 || n > len(m.items) {
		m.status = say(m.tr.NoNoteNumber, map[string]any{"N": n})
		return
	}
	m.mode = modeBoard
	m.setFocus(m.items[n-1].note.Name)
}

// set is :set. Alone it lists every setting with its value; with a
// setting it says its value; with a value too it changes it. vi's
// setting=value is read as the two.
func (m *Model) set(args []string) {
	if len(args) == 1 {
		if k, v, ok := strings.Cut(args[0], "="); ok {
			args = []string{k, v}
		}
	}
	if len(args) == 0 {
		set := m.store.Settings()
		var parts []string
		for _, p := range prefs.All {
			parts = append(parts, p.Key+"="+prefs.Get(set, p.Key))
		}
		m.status = strings.Join(parts, "  ")
		return
	}
	key, starts := pick(args[0], prefKeys(), nil)
	if key == "" {
		m.status = say(m.tr.NotASetting, map[string]any{"Name": args[0], "Choices": strings.Join(orAll(starts, prefKeys()), ", ")})
		return
	}
	m.setting(key, strings.Join(args[1:], " "))
}

func orAll(some, all []string) []string {
	if len(some) > 0 {
		return some
	}
	return all
}

// setting says a setting's value, or changes it as the settings panel
// does, when value is not "".
func (m *Model) setting(key, value string) {
	if value == "" {
		m.status = say(m.tr.SettingIs, map[string]any{"Key": key, "Value": prefs.Get(m.store.Settings(), key)})
		return
	}
	if err := m.calls().ChangeSettings(api.Setting{Key: key, Value: value}); err != nil {
		m.status = err.Error()
		return
	}
	m.reload()
	m.status = say(m.tr.SettingChanged, map[string]any{"Key": key, "Value": prefs.Get(m.store.Settings(), key)})
}

// noteNames are the notes of every tab as :edit takes them: by file name,
// a Markdown note without its .md.
func (m *Model) noteNames() []string {
	idx, err := m.calls().Index()
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range idx {
		if n := strings.TrimSuffix(e.Name, ".md"); !slices.Contains(out, n) {
			out = append(out, n)
		}
	}
	return out
}

// editByName zooms the note name finds, in whatever tab it is: the one
// whose file or title it is, else the first the index's filter would
// show for it.
func (m *Model) editByName(name string) {
	idx, err := m.calls().Index()
	if err != nil {
		m.status = err.Error()
		return
	}
	want := strings.ToLower(name)
	found := -1
	for i, e := range idx {
		file := strings.ToLower(e.Name)
		if file == want || strings.TrimSuffix(file, ".md") == want || strings.TrimSuffix(path.Base(file), ".md") == want || strings.ToLower(e.Title) == want {
			found = i
			break
		}
	}
	if found < 0 {
		for i, e := range idx {
			if matches(e, strings.Fields(want)) {
				found = i
				break
			}
		}
	}
	if found < 0 {
		m.status = say(m.tr.NoNoteMatches, map[string]any{"Name": name})
		return
	}
	m.goTo(idx[found])
	if i := m.index(idx[found].Name); i >= 0 {
		m.zoom(m.items[i])
	}
}

// newNote is :new. Alone it opens the shapes; a shape's name and a title
// make that note, a shape alone asks for the title, and any other words
// are jotted as a plain note.
func newNote(m *Model, args []string, _ bool) tea.Cmd {
	if len(args) == 0 {
		return openCatalog(m)
	}
	for _, k := range m.reg {
		if !strings.EqualFold(k.Name, args[0]) || k.Template == nil {
			continue
		}
		build := func(title string) { m.create(title, k.New, k.Template(title, m.env)) }
		if len(args) == 1 {
			m.ask(m.tr.Title, "", build)
		} else {
			build(strings.Join(args[1:], " "))
		}
		return nil
	}
	text := strings.Join(args, " ")
	m.create(text, "", []byte(text+"\n"))
	return nil
}

// goToTab shows the tab a number (from 1) or a name stands for.
func (m *Model) goToTab(which string) {
	if which == "" {
		m.status = say(m.tr.CommandUsage, map[string]any{"Usage": ":tab <n|name>"})
		return
	}
	if n, err := strconv.Atoi(which); err == nil {
		if n < 1 || n > len(m.tabList) {
			m.status = say(m.tr.NoSuchTab, map[string]any{"Name": which})
			return
		}
		m.mode = modeBoard
		m.switchTab(n - 1)
		return
	}
	var names []string
	for _, t := range m.tabList {
		names = append(names, strings.ToLower(t.Name))
	}
	alias := map[string]string{}
	for _, t := range m.tabList {
		alias[strings.ToLower(t.Title)] = strings.ToLower(t.Name)
	}
	name, _ := pick(which, names, alias)
	for i, t := range m.tabList {
		if name != "" && strings.ToLower(t.Name) == name {
			m.mode = modeBoard
			m.switchTab(i)
			return
		}
	}
	m.status = say(m.tr.NoSuchTab, map[string]any{"Name": which})
}

// helpCommand is :help: alone the help screen, with a command what that
// command takes and does.
func helpCommand(m *Model, args []string, _ bool) tea.Cmd {
	if len(args) == 0 {
		return openHelp(m)
	}
	c, bad := m.findCommand(args[0])
	if bad != "" {
		m.status = bad
		return nil
	}
	m.status = strings.TrimSpace(":"+c.name+" "+c.args) + "  " + c.describe(m)
	return nil
}

// describe is what the command does, in the screen's language.
func (c command) describe(m *Model) string { return m.tr.Action("cmd."+c.name, c.english) }

// commandHelp is the help's section of commands: each with what it takes
// and what it does.
func (m *Model) commandHelp() []string {
	lines := []string{m.tr.Action("help.commands", "Command line")}
	lines = append(lines, "  :3"+strings.Repeat(" ", helpKeyColumn-2)+m.tr.Action("cmd.number", "the third note"))
	for _, c := range commands() {
		usage := ":" + c.name
		if len(c.alias) > 0 {
			usage = ":" + c.alias[0]
		}
		lines = append(lines, "  "+usage+strings.Repeat(" ", max(helpKeyColumn-widget.Width(usage), 1))+c.describe(m))
	}
	return lines
}
