package chart

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/doc"
)

// put writes value for label: over the old number when the label is there,
// as a new line at the end when it is not.
func put(d doc.Document, label, value string) (doc.Document, error) {
	label = strings.TrimSpace(label)
	if label == "" || strings.ContainsAny(label, "\r\n") {
		return d, errors.New("a value needs a label on one line")
	}
	lines := doc.Lines(d.Body)
	if i, from, to := find(lines, label); i >= 0 {
		lines[i] = lines[i][:from] + value + lines[i][to:]
		d.Body = doc.Join(lines)
		return d, nil
	}
	eol := doc.EOL(d.Body)
	if d.Body != "" && !strings.HasSuffix(d.Body, "\n") {
		d.Body += eol + "\n"
	}
	d.Body += label + ": " + value + eol + "\n"
	return d, nil
}

// Set gives the value called Label the number Value, written as given. A
// label the chart does not have yet is added at the end.
type Set struct{ Label, Value string }

// Apply implements doc.Op.
func (o Set) Apply(d doc.Document) (doc.Document, error) {
	value := strings.TrimSpace(o.Value)
	if !numberRe.MatchString(value) {
		return d, fmt.Errorf("%q is not a number", o.Value)
	}
	return put(d, o.Label, value)
}

// Add counts the value called Label up by Delta, or down when Delta is
// negative. A label the chart does not have yet starts from zero. A value
// written with commas keeps them.
type Add struct {
	Label string
	Delta float64
}

// Apply implements doc.Op.
func (o Add) Apply(d doc.Document) (doc.Document, error) {
	lines := doc.Lines(d.Body)
	old, grouped := 0.0, false
	if i, from, to := find(lines, strings.TrimSpace(o.Label)); i >= 0 {
		text := lines[i][from:to]
		v, err := number(text)
		if err != nil {
			return d, fmt.Errorf("the value of %q is not a number: %s", o.Label, text)
		}
		old, grouped = v, strings.Contains(text, ",")
	}
	sum := math.Round((old+o.Delta)*1e6) / 1e6
	if math.IsNaN(sum) || math.IsInf(sum, 0) {
		return d, errors.New("the result is not a number")
	}
	value := strconv.FormatFloat(sum, 'f', -1, 64)
	if grouped {
		value = trim(sum)
	}
	return put(d, o.Label, value)
}
