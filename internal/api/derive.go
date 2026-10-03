package api

import (
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/arrange"
	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget/chart"
	"github.com/LeeSwallow/stickypane/internal/widget/form"
)

// A chart may say in its front matter where its values come from:
//
//	from: plan          one note: its own numbers
//	from: plan, work    several: one number each, named by the note's title
//
// stickypane computes them, by the source's shape, and writes them into
// the chart, so the file is true without stickypane and an agent reads it
// like any chart. What a shape gives is fixed, not a query language:
//
//	checklist   done and open; with several sources, percent done
//	board       cards per column; with several, cards in all
//	form        how many chose each answer; with several, answered questions
//	log         lines
//
// A chart computed from another computed chart is left alone, so values
// never chase each other.

var (
	tickRe = regexp.MustCompile(`^\s*[-*] \[([ xX])\]`)
	cardRe = regexp.MustCompile(`^[-*] `)
)

// Derive computes every chart with from: and writes those whose values
// changed. It returns how many it wrote.
func (a *API) Derive() (int, error) {
	b, err := a.st.Load()
	if err != nil {
		return 0, err
	}
	files := files(b)
	views := b.Settings.Views()
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	wrote := 0
	for name, n := range files {
		from, ok := n.Doc.Get("from")
		if !ok || a.reg.For(name, n.Doc).Name != "chart" {
			continue
		}
		sources := splitList(from)
		var values [][2]string
		usable := len(sources) > 0
		for _, s := range sources {
			src, ok := files[Resolve(s, names)]
			if _, derived := src.Doc.Get("from"); !ok || derived {
				usable = false
				break
			}
			kind := a.reg.For(src.Name, src.Doc).Name
			if len(sources) == 1 {
				values = reduce(kind, src.Doc)
			} else {
				// Named as the board names it: its title, else its file
				// name without the number that orders it.
				label := arrange.Title(views[src.Name], src.Doc, src.Name)
				values = append(values, [2]string{label, headline(kind, src.Doc)})
			}
		}
		if !usable || values == nil {
			continue
		}
		op := chart.Replace{Values: values}
		after, _ := op.Apply(n.Doc)
		if after.Body == n.Doc.Body {
			continue
		}
		if err := a.st.Apply(name, op); err != nil {
			return wrote, err
		}
		wrote++
	}
	return wrote, nil
}

// splitList reads "a, b" or "[a, b]".
func splitList(s string) []string {
	s = strings.Trim(strings.TrimSpace(s), "[]")
	var out []string
	for _, v := range strings.Split(s, ",") {
		if v = strings.Trim(strings.TrimSpace(v), `"'`); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// reduce is what one source gives a chart, by its shape.
func reduce(kind string, d doc.Document) [][2]string {
	switch kind {
	case "checklist":
		done, total := ticks(d)
		return [][2]string{{"done", fmt.Sprint(done)}, {"open", fmt.Sprint(total - done)}}
	case "board":
		var out [][2]string
		for _, c := range columns(d) {
			out = append(out, [2]string{c.title, fmt.Sprint(c.cards)})
		}
		return out
	case "form":
		count := map[string]int{}
		var order []string
		for _, q := range form.Read(d).Answers {
			for _, v := range q.Values {
				if count[v] == 0 {
					order = append(order, v)
				}
				count[v]++
			}
		}
		var out [][2]string
		for _, v := range order {
			out = append(out, [2]string{v, fmt.Sprint(count[v])})
		}
		return out
	}
	return [][2]string{{"lines", fmt.Sprint(lineCount(d))}}
}

// headline is the one number a source gives a chart of several.
func headline(kind string, d doc.Document) string {
	switch kind {
	case "checklist":
		done, total := ticks(d)
		if total == 0 {
			return "0"
		}
		return fmt.Sprint(int(math.Round(float64(done) * 100 / float64(total))))
	case "board":
		n := 0
		for _, c := range columns(d) {
			n += c.cards
		}
		return fmt.Sprint(n)
	case "form":
		n := 0
		for _, q := range form.Read(d).Answers {
			if len(q.Values) > 0 {
				n++
			}
		}
		return fmt.Sprint(n)
	}
	return fmt.Sprint(lineCount(d))
}

func ticks(d doc.Document) (done, total int) {
	for _, l := range doc.Lines(d.Body) {
		if m := tickRe.FindStringSubmatch(l); m != nil {
			total++
			if m[1] != " " {
				done++
			}
		}
	}
	return done, total
}

type column struct {
	title string
	cards int
}

func columns(d doc.Document) []column {
	var out []column
	for _, l := range doc.Lines(d.Body) {
		l = strings.TrimRight(l, "\r")
		switch {
		case strings.HasPrefix(l, "## "):
			out = append(out, column{title: strings.TrimSpace(l[3:])})
		case len(out) > 0 && cardRe.MatchString(l):
			out[len(out)-1].cards++
		}
	}
	return out
}

func lineCount(d doc.Document) int {
	n := 0
	for _, l := range doc.Lines(d.Body) {
		if strings.TrimSpace(l) != "" {
			n++
		}
	}
	return n
}
