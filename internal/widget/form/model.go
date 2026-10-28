package form

import (
	"fmt"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
	"github.com/LeeSwallow/stickypane/internal/widget/note"
)

// control is something the cursor can stand on: an option, a field, or one
// button of a button line.
type control struct {
	line   int
	button int
}

// content is what a form note says: the reader's side, with no screen
// state in it.
type content struct {
	lines     []line
	questions []string
	controls  []control
	submitted string // the pressed button, "" before that
	at        string
}

func parse(d doc.Document, render note.Renderer, e widget.Env) *Form {
	return &Form{content: read(d), render: render, env: e}
}

// read reads a form note.
func read(d doc.Document) content {
	var f content
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
	a := Answers{}
	a.Button, _ = d.Get(keySubmitted)
	a.Submitted = a.Button != ""
	if a.Submitted {
		a.At, _ = d.Get(keySubmittedAt)
	}
	a.Answers = collect(lines, questions)
	return a
}

// collect gathers what was chosen or written, question by question.
func collect(lines []line, questions []string) []Answer {
	answers := make([]Answer, len(questions))
	for i, q := range questions {
		answers[i] = Answer{Question: q, Values: []string{}}
	}
	for _, l := range lines {
		if (l.kind == option && l.on) || (l.kind == field && l.text != "") {
			answers[l.question].Values = append(answers[l.question].Values, l.text)
		}
	}
	return answers
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

// Line is one line of a form as another screen draws it. Kind is one of
// what the form tells apart: "text" (anything that is not a control),
// "option" (Radio for ( ), Checked when chosen), "detail" (an indented line
// under an option), "field" (Text is the answer so far) and "buttons"
// (Labels). Group names the question an option or a field answers.
type Line struct {
	Kind    string   `json:"kind"`
	Text    string   `json:"text"`
	Group   string   `json:"group,omitempty"`
	Radio   bool     `json:"radio,omitempty"`
	Checked bool     `json:"checked,omitempty"`
	Labels  []string `json:"labels,omitempty"`
}

// Page is what Data says of a form: every line, the pressed button and when.
type Page struct {
	Lines     []Line `json:"lines"`
	Submitted string `json:"submitted"`
	At        string `json:"at"`
}

// kindNames are the Line kinds, by the line kinds of the reader.
var kindNames = map[lineKind]string{prose: "text", option: "option", detail: "detail", field: "field", buttons: "buttons"}

// data is the lines of the form, from the same read the board draws. When
// the file has no button, the built-in one is among them, because it can
// be pressed.
func data(d doc.Document, _ widget.Env) any {
	c := read(d)
	p := Page{Lines: make([]Line, 0, len(c.lines)), Submitted: c.submitted}
	if c.submitted != "" {
		p.At = c.at
	}
	for _, l := range c.lines {
		out := Line{Kind: kindNames[l.kind], Text: l.text, Labels: l.labels}
		if l.kind == option || l.kind == field {
			out.Group = c.questions[l.question]
		}
		if l.kind == option {
			out.Radio, out.Checked = l.radio, l.on
		}
		p.Lines = append(p.Lines, out)
	}
	return p
}
