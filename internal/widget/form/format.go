package form

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/doc"
)

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
