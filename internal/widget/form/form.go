// Package form is the document that asks: Markdown with options to choose,
// lines to fill in and buttons to press. The answers are written into the
// same file, so the agent that wrote the questions reads them back.
package form

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
	"github.com/LeeSwallow/stickypane/internal/widget/note"
)

const (
	// defaultButton is the button of a form that has none of its own.
	defaultButton = "Submit"
	fieldWidth    = 48 // widest a text field gets, borders included
	placeholder   = "enter to type"

	keySubmitted   = "submitted"
	keySubmittedAt = "submitted_at"
)

// now is the clock, replaced in tests.
var now = time.Now

var (
	// optionRe splits "- ( ) text" and "- [x] text": the list marker with up
	// to three spaces before it, the opening bracket, the mark, the closing
	// bracket, a space, the text. The space is required, so that a link
	// written "- [x](url)" is not a ticked box.
	optionRe = regexp.MustCompile(`^( {0,3}[-*] )([(\[])([ xX])([)\]])(?:( )(.*))?$`)
	// fieldRe matches a line to fill in: ">" and the answer so far.
	fieldRe = regexp.MustCompile(`^>(.*)$`)
	// buttonsRe matches a line made only of "[ Label ]" buttons.
	buttonsRe = regexp.MustCompile(`^\s*(\[ [^\[\]]+ \]\s*)+$`)
	buttonRe  = regexp.MustCompile(`\[ ([^\[\]]+) \]`)
)

// NewKind returns the form kind. Its text is drawn with render, the same
// way a plain note is.
func NewKind(render note.Renderer) widget.Kind {
	return widget.Kind{
		Name:    "form",
		Label:   "Form",
		Icon:    "◉",
		Hint:    []string{"j k", "move", "enter", "choose or press"},
		Blurb:   "A document that asks: options to choose, lines to fill in, buttons to press. The answers land in the file.",
		Example: "## Where to deploy?\n- (x) staging\n- ( ) production\n\n## Note\n> after lunch\n\n[ Deploy ] [ Cancel ]\n",
		Keys:    "j k up down h l left right space enter",
		Size:    func(doc.Document) string { return widget.SizeHalf },
		Template: func(title string) []byte {
			return widget.NewFile("form", title, "## Question\n- ( ) Yes\n- ( ) No\n\n[ "+defaultButton+" ]\n")
		},
		Parse: func(d doc.Document) widget.Widget { return parse(d, render) },
	}
}

type lineKind int

const (
	prose   lineKind = iota // anything that is not a control
	option                  // "- ( ) text" or "- [ ] text"
	detail                  // an indented line under an option
	field                   // "> text"
	buttons                 // "[ A ] [ B ]"
)

// line is one line of the body, read as part of a form.
type line struct {
	kind     lineKind
	text     string   // the option's text, the field's answer, or the line itself
	radio    bool     // an option written with ( )
	on       bool     // a chosen option
	question int      // options and fields: which question this answers
	labels   []string // buttons
}

// scan reads every line of body. Line i of the result is line i of
// doc.Lines(body). It also returns what each question is called: the
// heading above it, else the line above it, else its number.
func scan(body string) (lines []line, questions []string) {
	var heading, above string
	var fence string // the run of ` or ~ that closes the code block we are in
	grouped, canDetail, radio := false, false, false
	ask := func() int {
		name := heading
		if name == "" {
			name = above
		}
		if name == "" {
			name = fmt.Sprintf("question %d", len(questions)+1)
		}
		heading, above = "", ""
		questions = append(questions, name)
		return len(questions) - 1
	}
	for _, raw := range doc.Lines(body) {
		raw = strings.TrimSuffix(raw, "\r")
		trimmed := strings.TrimSpace(raw)
		l := line{kind: prose, text: raw}
		switch m := optionRe.FindStringSubmatch(raw); {
		case fence != "": // inside a code block nothing is a control
			if strings.HasPrefix(trimmed, fence) && strings.Trim(trimmed, fence[:1]) == "" {
				fence = ""
			}
		case strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~"):
			// The block ends at a line of at least as many of the same
			// character, so a longer fence can quote a shorter one.
			fence = trimmed[:len(trimmed)-len(strings.TrimLeft(trimmed, trimmed[:1]))]
			grouped, canDetail = false, false
		case m != nil && (m[2] == "(") == (m[4] == ")"):
			l = line{kind: option, text: strings.TrimSpace(m[6]), radio: m[2] == "(", on: m[3] != " "}
			// Options of one kind in a row are one question.
			if !grouped || radio != l.radio {
				l.question = ask()
			} else {
				l.question = len(questions) - 1
			}
			grouped, canDetail, radio = true, true, l.radio
		case trimmed == "":
			// A blank line ends the question: the screen shows a gap
			// there, so what follows must not share one answer with it.
			grouped, canDetail = false, false
		case len(buttonsOf(raw)) > 0:
			l = line{kind: buttons, labels: buttonsOf(raw)}
			grouped, canDetail = false, false
		case canDetail && (strings.HasPrefix(raw, " ") || strings.HasPrefix(raw, "\t")):
			l = line{kind: detail, text: trimmed}
		case fieldRe.MatchString(raw):
			l = line{kind: field, text: strings.TrimSpace(raw[1:]), question: ask()}
			grouped, canDetail = false, false
		default:
			if strings.HasPrefix(trimmed, "#") {
				heading = strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
			} else {
				above = trimmed
			}
			grouped, canDetail = false, false
		}
		lines = append(lines, l)
	}
	return lines, questions
}

// buttonsOf returns the labels of a line made only of buttons, and nothing
// for any other line. A button without a name is not a button.
func buttonsOf(raw string) []string {
	if !buttonsRe.MatchString(raw) {
		return nil
	}
	var labels []string
	for _, b := range buttonRe.FindAllStringSubmatch(raw, -1) {
		if label := strings.TrimSpace(b[1]); label != "" {
			labels = append(labels, label)
		}
	}
	return labels
}

// buttonLabels returns every button of the form, or the built-in one when
// the form has none.
func buttonLabels(lines []line) []string {
	var labels []string
	for _, l := range lines {
		labels = append(labels, l.labels...)
	}
	if len(labels) == 0 {
		return []string{defaultButton}
	}
	return labels
}

// control is something the cursor can stand on: an option, a field, or one
// button of a button line.
type control struct {
	line   int
	button int
}

// Form is the widget for a form note.
type Form struct {
	render    note.Renderer
	lines     []line
	questions []string
	controls  []control
	cursor    int
	submitted string // the pressed button, "" before that
	at        string
	drawn     []area // where each control was in the last Draw
}

// area is the cells a control took in the last Draw: its lines and, for a
// button, its columns.
type area struct {
	lines  widget.Span
	x0, x1 int
}

// anyCol is the column range of a control that owns its whole lines.
const anyCol = 1 << 30

func parse(d doc.Document, render note.Renderer) *Form {
	f := &Form{render: render}
	f.lines, f.questions = scan(d.Body)
	f.submitted, _ = d.Get(keySubmitted)
	f.at, _ = d.Get(keySubmittedAt)
	hasButtons := false
	for _, l := range f.lines {
		hasButtons = hasButtons || l.kind == buttons
	}
	if !hasButtons {
		// A form can always be submitted. The built-in button is drawn
		// after the last line and is never written into the file.
		f.lines = append(f.lines, line{kind: prose}, line{kind: buttons, labels: []string{defaultButton}})
	}
	for i, l := range f.lines {
		switch l.kind {
		case option, field:
			f.controls = append(f.controls, control{line: i})
		case buttons:
			for b := range l.labels {
				f.controls = append(f.controls, control{line: i, button: b})
			}
		}
	}
	return f
}

func (f *Form) clamp() { f.cursor = max(min(f.cursor, len(f.controls)-1), 0) }

// nth is how many controls of the same kind and text come before control c.
func (f *Form) nth(c control) int {
	n := 0
	for i := 0; i < c.line; i++ {
		l, target := f.lines[i], f.lines[c.line]
		if l.kind == target.kind && (target.kind == field || l.text == target.text) {
			n++
		}
	}
	return n
}

// Draw implements widget.Widget. Text is drawn like a plain note; options,
// fields and buttons are drawn as controls where they are written.
func (f *Form) Draw(width int, active bool) (string, widget.Span) {
	f.clamp()
	var out []string
	at := widget.NoSpan
	f.drawn = f.drawn[:0]
	sel := f.controls[f.cursor]
	for i := 0; i < len(f.lines); i++ {
		l := f.lines[i]
		selected := active && sel.line == i
		from := len(out)
		switch l.kind {
		case prose:
			end := i
			for end < len(f.lines) && f.lines[end].kind == prose {
				end++
			}
			out = append(out, f.text(f.lines[i:end], width, i > 0, end < len(f.lines))...)
			i = end - 1
		case option:
			out = append(out, drawOption(l, width, selected)...)
			for i+1 < len(f.lines) && f.lines[i+1].kind == detail {
				i++
				for _, d := range widget.Wrap(widget.Clean(f.lines[i].text), max(width-4, 1)) {
					out = append(out, "    "+widget.Faint.Render(d))
				}
			}
		case detail: // a detail with no option above it cannot happen; show it
			out = append(out, widget.Wrap(widget.Clean(l.text), max(width, 1))...)
		case field:
			out = append(out, drawField(l.text, width, selected)...)
		case buttons:
			focus := -1
			if selected {
				focus = sel.button
			}
			rows, places := f.drawButtons(l.labels, width, focus)
			starts := make([]int, len(rows))
			for r, row := range rows {
				starts[r] = len(out)
				out = append(out, row...)
			}
			for b, p := range places {
				lines := widget.Span{Start: starts[p.row], End: starts[p.row] + len(rows[p.row])}
				f.drawn = append(f.drawn, area{lines: lines, x0: p.x0, x1: p.x1})
				if b == focus {
					at = lines
				}
			}
			continue
		}
		if l.kind == option || l.kind == field {
			lines := widget.Span{Start: from, End: len(out)}
			f.drawn = append(f.drawn, area{lines: lines, x1: anyCol})
			if selected {
				at = lines
			}
		}
	}
	if f.submitted != "" {
		sent := "sent"
		if t, err := time.Parse(time.RFC3339, f.at); err == nil {
			sent += " " + t.Local().Format("15:04")
		}
		out = append(out, widget.Good.Render("✓ ")+widget.Faint.Render(widget.Truncate(sent, max(width-2, 0))))
	}
	return widget.Fit(strings.Join(out, "\n"), width), at
}

// text draws a run of prose lines. The blank lines at its ends are the
// form's to decide, not the renderer's: they become one blank line each, and
// only where something is drawn on both sides. before and after report
// whether anything comes before and after the run.
func (f *Form) text(lines []line, width int, before, after bool) []string {
	first, last := 0, len(lines)
	for first < last && strings.TrimSpace(lines[first].text) == "" {
		first++
	}
	for last > first && strings.TrimSpace(lines[last-1].text) == "" {
		last--
	}
	if first == last {
		if before && after {
			return []string{""}
		}
		return nil
	}
	raw := make([]string, 0, last-first)
	for _, l := range lines[first:last] {
		raw = append(raw, widget.Clean(l.text))
	}
	drawn := strings.Split(f.render(strings.Join(raw, "\n")+"\n", width), "\n")
	blank := func(s string) bool { return strings.TrimSpace(ansi.Strip(s)) == "" }
	for len(drawn) > 0 && blank(drawn[0]) {
		drawn = drawn[1:]
	}
	for len(drawn) > 0 && blank(drawn[len(drawn)-1]) {
		drawn = drawn[:len(drawn)-1]
	}
	var out []string
	if before && first > 0 {
		out = append(out, "")
	}
	out = append(out, drawn...)
	if after && last < len(lines) {
		out = append(out, "")
	}
	return out
}

func drawOption(l line, width int, selected bool) []string {
	glyph := "☐ "
	switch {
	case l.radio && l.on:
		glyph = "◉ "
	case l.radio:
		glyph = "○ "
	case l.on:
		glyph = "☑ "
	}
	var out []string
	for j, text := range widget.Wrap(widget.Clean(l.text), max(width-4, 1)) {
		lead := "    "
		if j == 0 {
			lead = "  " + glyph
			if selected {
				lead = "› " + glyph
			}
		}
		switch {
		case selected:
			out = append(out, widget.Selected.Render(lead+text))
		case l.on:
			out = append(out, widget.Good.Render(lead)+widget.Bold.Render(text))
		default:
			out = append(out, lead+text)
		}
	}
	return out
}

// border is the six pieces of a box: corners, the horizontal and the
// vertical line.
type border struct{ tl, tr, bl, br, h, v string }

var (
	thin  = border{"╭", "╮", "╰", "╯", "─", "│"}
	heavy = border{"┏", "┓", "┗", "┛", "━", "┃"}
)

// boxed draws lines inside a box whose inside is inner cells wide. The
// border and the text are styled separately so a focused box can stand out.
func boxed(lines []string, inner int, b border, edge, text lipgloss.Style) []string {
	rule := strings.Repeat(b.h, inner+2)
	out := []string{edge.Render(b.tl + rule + b.tr)}
	for _, l := range lines {
		out = append(out, edge.Render(b.v)+text.Render(" "+widget.Pad(l, inner)+" ")+edge.Render(b.v))
	}
	return append(out, edge.Render(b.bl+rule+b.br))
}

// drawField draws a line to fill in as a box with the answer inside.
func drawField(text string, width int, selected bool) []string {
	inner := min(width, fieldWidth) - 4
	if inner < 1 {
		return []string{widget.Truncate("✎ "+widget.Clean(text), width)}
	}
	lines, st := widget.Wrap(widget.Clean(text), inner), lipgloss.NewStyle()
	if text == "" {
		lines, st = []string{""}, widget.Faint
		if selected {
			lines = []string{widget.Truncate(placeholder, inner)}
		}
	}
	if selected {
		return boxed(lines, inner, heavy, widget.Bold, st)
	}
	return boxed(lines, inner, thin, widget.Faint, st)
}

// place is where a button was drawn: its row and its columns.
type place struct{ row, x0, x1 int }

// drawButtons draws the buttons of one line side by side, on more rows when
// they do not fit. It returns the rows and where each button is.
func (f *Form) drawButtons(labels []string, width, focus int) (rows [][]string, places []place) {
	var row []string
	used := 0
	flush := func() {
		if row != nil {
			rows = append(rows, row)
		}
		row, used = nil, 0
	}
	for i, label := range labels {
		text := widget.Clean(label)
		edge, body := widget.Faint, lipgloss.NewStyle()
		if label == f.submitted {
			text, edge, body = "✓ "+text, widget.Good, widget.Good
		}
		text = widget.Truncate(text, max(width-4, 1))
		b := thin
		if i == focus {
			b, edge, body = heavy, widget.Bold, widget.Selected
		}
		button := boxed([]string{text}, widget.Width(text), b, edge, body)
		w := widget.Width(text) + 4
		if used > 0 && used+1+w > width {
			flush()
		}
		if row == nil {
			row = button
		} else {
			for j := range row {
				row[j] += " " + button[j]
			}
			used++
		}
		places = append(places, place{row: len(rows), x0: used, x1: used + w})
		used += w
	}
	flush()
	return rows, places
}

// Summary implements widget.Widget: the pressed button, or how many
// questions have an answer.
func (f *Form) Summary() string {
	if f.submitted != "" {
		return "✓ " + widget.Clean(f.submitted)
	}
	if len(f.questions) == 0 {
		return ""
	}
	answered := make([]bool, len(f.questions))
	for _, l := range f.lines {
		if (l.kind == option && l.on) || (l.kind == field && l.text != "") {
			answered[l.question] = true
		}
	}
	n := 0
	for _, a := range answered {
		if a {
			n++
		}
	}
	return fmt.Sprintf("%d/%d", n, len(f.questions))
}

// Update implements widget.Widget.
func (f *Form) Update(key string) (widget.Widget, widget.Result) {
	var res widget.Result
	f.clamp()
	switch key {
	case "j", "down", "l", "right":
		f.cursor++
	case "k", "up", "h", "left":
		f.cursor--
	case "space", "enter":
		c := f.controls[f.cursor]
		switch l := &f.lines[c.line]; l.kind {
		case option:
			res.Op = Choose{Text: l.text, Nth: f.nth(c), On: !l.on}
			if !l.on && l.radio {
				for i := range f.lines {
					if o := &f.lines[i]; o.kind == option && o.question == l.question {
						o.on = false
					}
				}
			}
			l.on = !l.on
			f.submitted = ""
		case field:
			nth, old := f.nth(c), l.text
			res.Prompt = &widget.Prompt{
				Label:   widget.Clean(f.questions[l.question]),
				Initial: old,
				Empty:   true,
				Submit:  func(text string) doc.Op { return Fill{Nth: nth, Old: old, Text: text} },
			}
		case buttons:
			f.submitted, f.at = l.labels[c.button], now().Format(time.RFC3339)
			res.Op = Submit{Label: f.submitted, At: f.at}
		}
	}
	f.clamp()
	return f, res
}

// Click implements widget.Clicker: a press on a control does what enter
// does there.
func (f *Form) Click(line, col int) (widget.Widget, widget.Result, bool) {
	for i, a := range f.drawn {
		if i < len(f.controls) && line >= a.lines.Start && line < a.lines.End && col >= a.x0 && col < a.x1 {
			f.cursor = i
			w, res := f.Update("enter")
			return w, res, true
		}
	}
	return f, widget.Result{}, false
}

// Sync implements widget.Widget.
func (f *Form) Sync(d doc.Document) widget.Widget {
	nf := parse(d, f.render)
	nf.cursor = f.cursor
	nf.clamp()
	return nf
}

// unsubmit takes the submission back: the answers are changing again.
func unsubmit(d doc.Document) doc.Document {
	return d.Unset(keySubmitted).Unset(keySubmittedAt)
}

// replace rewrites line i of lines with text, keeping its line ending.
func replace(lines []string, i int, text string) {
	if strings.HasSuffix(lines[i], "\r") {
		text += "\r"
	}
	lines[i] = text
}

// Choose sets an option to state On. The option is the one with Text; when
// several share that text, Nth counts from zero to say which. Choosing an
// option written with ( ) clears the others of its question. An option that
// is gone, or already in that state, is a conflict.
type Choose struct {
	Text string
	Nth  int
	On   bool
}

// Apply implements doc.Op.
func (o Choose) Apply(d doc.Document) (doc.Document, error) {
	raw := doc.Lines(d.Body)
	lines, _ := scan(d.Body)
	set := func(i int, on bool) {
		m := optionRe.FindStringSubmatch(strings.TrimSuffix(raw[i], "\r"))
		mark := " "
		if on {
			mark = "x"
		}
		replace(raw, i, m[1]+m[2]+mark+m[4]+m[5]+m[6])
	}
	nth := o.Nth
	for i, l := range lines {
		if l.kind != option || l.text != o.Text {
			continue
		}
		if nth > 0 {
			nth--
			continue
		}
		if l.on == o.On {
			break // not the option the screen showed
		}
		set(i, o.On)
		if o.On && l.radio {
			for j, other := range lines {
				if j != i && other.kind == option && other.question == l.question && other.on {
					set(j, false)
				}
			}
		}
		d.Body = doc.Join(raw)
		return unsubmit(d), nil
	}
	return d, doc.ErrConflict
}

// Fill writes Text into the Nth field, counting from zero. A field that no
// longer reads Old is a conflict.
type Fill struct {
	Nth       int
	Old, Text string
}

// Apply implements doc.Op.
func (o Fill) Apply(d doc.Document) (doc.Document, error) {
	raw := doc.Lines(d.Body)
	lines, _ := scan(d.Body)
	nth := o.Nth
	for i, l := range lines {
		if l.kind != field {
			continue
		}
		if nth > 0 {
			nth--
			continue
		}
		if l.text != o.Old {
			break
		}
		if o.Text == o.Old {
			return d, nil // nothing changed, so nothing is taken back
		}
		replace(raw, i, "> "+o.Text)
		d.Body = doc.Join(raw)
		return unsubmit(d), nil
	}
	return d, doc.ErrConflict
}

// Submit records that the button Label was pressed at time At. A button the
// form no longer has is a conflict.
type Submit struct{ Label, At string }

// Apply implements doc.Op.
func (o Submit) Apply(d doc.Document) (doc.Document, error) {
	lines, _ := scan(d.Body)
	for _, label := range buttonLabels(lines) {
		if label == o.Label {
			return d.Set(keySubmitted, o.Label).Set(keySubmittedAt, o.At), nil
		}
	}
	return d, doc.ErrConflict
}

// Answer is what the user chose or wrote for one question.
type Answer struct {
	Question string   `json:"question"`
	Values   []string `json:"values"`
}

// Answers is a form as its author reads it back.
type Answers struct {
	Submitted bool     `json:"submitted"`
	Button    string   `json:"button,omitempty"` // the pressed button
	At        string   `json:"at,omitempty"`     // when, RFC 3339
	Answers   []Answer `json:"answers"`
}

// Read collects the answers of a form.
func Read(d doc.Document) Answers {
	lines, questions := scan(d.Body)
	a := Answers{Answers: make([]Answer, len(questions))}
	a.Button, _ = d.Get(keySubmitted)
	a.Submitted = a.Button != ""
	if a.Submitted {
		a.At, _ = d.Get(keySubmittedAt)
	}
	for i, q := range questions {
		a.Answers[i] = Answer{Question: q, Values: []string{}}
	}
	for _, l := range lines {
		if (l.kind == option && l.on) || (l.kind == field && l.text != "") {
			a.Answers[l.question].Values = append(a.Answers[l.question].Values, l.text)
		}
	}
	return a
}

// String prints the answers one question per line, for a person or an agent
// to read.
func (a Answers) String() string {
	var b strings.Builder
	if a.Submitted {
		fmt.Fprintf(&b, "submitted: %s\n", a.Button)
		if a.At != "" {
			fmt.Fprintf(&b, "at: %s\n", a.At)
		}
	} else {
		b.WriteString("submitted: no\n")
	}
	for _, q := range a.Answers {
		values := "-"
		if len(q.Values) > 0 {
			values = strings.Join(q.Values, "; ")
		}
		fmt.Fprintf(&b, "%s: %s\n", q.Question, values)
	}
	return b.String()
}
