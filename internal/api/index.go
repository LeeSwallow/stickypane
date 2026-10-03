package api

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/LeeSwallow/stickypane/internal/arrange"
	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// Entry is one note in the index: enough to know what it is and whether to
// look, without reading it.
type Entry struct {
	Tab      string    `json:"tab"`       // "" for the root
	TabTitle string    `json:"tab_title"` // the tab's name on the screen
	Name     string    `json:"name"`
	Kind     string    `json:"kind"`
	Icon     string    `json:"icon"`
	Title    string    `json:"title"`
	Summary  string    `json:"summary,omitempty"` // what the widget counts: 2/5, 7 cards
	Gist     string    `json:"gist,omitempty"`    // a line: its summary, or what to look at first
	Open     bool      `json:"open"`
	Modified time.Time `json:"modified"`
}

// Index lists every note of every tab in a line, the way a wiki's index
// page does: for a person scanning a board with many notes, and for an
// agent, which reads it instead of every note.
func (a *API) Index() ([]Entry, error) {
	b, err := a.st.Load()
	if err != nil {
		return nil, err
	}
	views := b.Settings.Views()
	var out []Entry
	for _, t := range b.Tabs {
		for _, n := range t.Notes {
			e := Entry{Tab: t.Name, TabTitle: t.Title, Name: n.Name, Modified: n.ModTime}
			v := views[n.Name]
			e.Open, _ = arrange.Open(v, n.Doc)
			if n.Book() {
				e.Kind, e.Icon = "book", "▤"
				e.Title = arrange.Title(v, n.Doc, n.Name)
				e.Summary = fmt.Sprintf("%d pages", len(n.Pages))
				if len(n.Pages) > 0 {
					e.Gist = arrange.Title(store.View{}, n.Pages[0].Doc, n.Pages[0].Name)
				}
				out = append(out, e)
				continue
			}
			k := a.reg.For(n.Name, n.Doc)
			e.Kind, e.Icon = k.Name, k.Icon
			e.Title = arrange.Title(v, n.Doc, n.Name)
			if n.Err == nil {
				e.Summary = k.Parse(n.Doc).Summary()
				e.Gist = gist(k, n.Doc)
			}
			out = append(out, e)
		}
	}
	return out, nil
}

var (
	openItemRe = regexp.MustCompile(`^\s*[-*] \[ \]\s+(.*)$`)
	leadRe     = regexp.MustCompile(`^\s*(#+|[-*+]|\d+[.)]|>|\[[ xX]\])\s+`)
	marksRe    = regexp.MustCompile("\\*\\*|__|`|~~")
)

// gist is a note's line in the index: its front matter summary when it has
// one, else what its shape says to look at first.
func gist(k widget.Kind, d doc.Document) string {
	if s, ok := d.Get("summary"); ok && strings.TrimSpace(s) != "" {
		return clip(s)
	}
	lines := doc.Lines(d.Body)
	switch k.Name {
	case "checklist":
		for _, l := range lines {
			if m := openItemRe.FindStringSubmatch(strings.TrimRight(l, "\r")); m != nil {
				return clip(m[1])
			}
		}
		return ""
	case "log":
		for i := len(lines) - 1; i >= 0; i-- {
			if l := strings.TrimSpace(lines[i]); l != "" {
				return clip(l)
			}
		}
		return ""
	}
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" || l == "---" || (k.Name == "script" && strings.HasPrefix(l, "#")) || strings.HasPrefix(l, "```") {
			continue
		}
		for leadRe.MatchString(l) {
			l = leadRe.ReplaceAllString(l, "")
		}
		if l = strings.TrimSpace(marksRe.ReplaceAllString(l, "")); l != "" {
			return clip(l)
		}
	}
	return ""
}

// clip keeps a gist to one short line.
func clip(s string) string {
	s = widget.Clean(strings.Join(strings.Fields(s), " "))
	if r := []rune(s); len(r) > 80 {
		return string(r[:79]) + "…"
	}
	return s
}

// IndexMarkdown writes the index as a page: a heading per tab, a line per
// note with its title, file, count and gist.
func IndexMarkdown(entries []Entry, project string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n%d notes. Open ones are marked ●.\n", project, len(entries))
	tab := "\x00"
	for _, e := range entries {
		if e.Tab != tab {
			tab = e.Tab
			title := e.TabTitle
			if tab == "" {
				title = project
			}
			fmt.Fprintf(&b, "\n## %s\n\n", title)
		}
		open := " "
		if e.Open {
			open = "●"
		}
		fmt.Fprintf(&b, "- %s %s **%s** `%s`", open, e.Icon, e.Title, e.Name)
		if e.Summary != "" {
			fmt.Fprintf(&b, " · %s", e.Summary)
		}
		if e.Gist != "" && e.Gist != e.Title {
			fmt.Fprintf(&b, " — %s", e.Gist)
		}
		b.WriteString("\n")
	}
	return b.String()
}
