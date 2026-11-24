package app

import (
	"errors"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/adapter/git"
	"github.com/LeeSwallow/stickypane/internal/adapter/store"
	"github.com/LeeSwallow/stickypane/internal/core/doc"
	"github.com/LeeSwallow/stickypane/internal/core/widget"
	"github.com/LeeSwallow/stickypane/internal/core/widget/diff"
	"github.com/LeeSwallow/stickypane/internal/usecase/diffbook"
)

// A diff note (type: diff) is a book whose pages come from git, not from
// files: its own page is the list of worktrees, and every file the chosen
// worktree changed is a page after it, its diff read only when the page
// is drawn. Nothing of it is written but the choice of worktree and base,
// in sticky.json.

// diffFresh is how long what was read of git is shown before a reload
// reads it again; "r" reads it at once.
const diffFresh = 2 * time.Second

// diffState is what the screen last read of git for one diff note.
type diffState struct {
	key    string // the worktree and base it was read for
	at     time.Time
	book   diffbook.Book
	bodies map[string]diffBody // the diffs of the pages drawn since
	prPath string              // the worktree the pull request was asked for
	pr     *diff.PR
}

// diffBody is one file's diff as git gave it.
type diffBody struct {
	text string
	cut  bool
	err  error
}

// prMsg brings the pull request gh found for a worktree, or nil.
type prMsg struct {
	note, path string
	pr         *diff.PR
}

// isDiff says whether the note is a diff note.
func (m *Model) isDiff(n store.Note) bool {
	return n.Err == nil && !n.Book() && m.reg.For(n.Name, n.Doc).Name == diff.Kind.Name
}

// diffPages are the pages of the diff note n: its own page, the list of
// worktrees, unless its front matter pins one; then a page for each file
// the chosen worktree changed.
func (m *Model) diffPages(n store.Note, old map[string]page) []page {
	v := m.views[n.Name]
	pinned, _ := n.Doc.Get("worktree")
	chosen, base := v.Worktree, v.Base
	if pinned != "" {
		chosen = pinned
	}
	if base == "" {
		base, _ = n.Doc.Get("base")
	}
	st := m.readDiff(n.Name, chosen, base)
	var pages []page
	if pinned == "" {
		o := st.book.Overview
		if st.prPath == st.book.Path {
			o.PR = st.pr
		}
		cursor := ""
		if l, ok := old[n.Name].w.(*diff.List); ok {
			cursor = l.Cursor()
		}
		pages = append(pages, page{n, diff.ListKind, diff.NewList(o, cursor, m.env)})
	}
	for _, c := range st.book.Changes {
		title := c.Path
		if c.Where != git.Committed {
			title += " · " + m.tr.L(c.Where)
		}
		name, c := n.Name+":"+c.Where+":"+c.Path, c
		pages = append(pages, page{
			note: store.Note{Name: name, Doc: doc.Document{}.Set("title", title)},
			kind: diff.Kind,
			w:    &sourcePage{load: func() diffBody { return m.diffOf(n.Name, c) }, env: m.env},
		})
	}
	if len(pages) == 0 { // a pinned worktree that changed nothing
		pages = append(pages, page{n, diff.Kind, diff.NewPage("", m.env)})
	}
	return pages
}

// readDiff reads git for a diff note, unless what was read for the same
// choice is fresh. A new worktree's pull request is asked of gh in the
// background; the screen does not wait for it.
func (m *Model) readDiff(note, chosen, base string) *diffState {
	key := chosen + "\x00" + base
	prev := m.diffs[note]
	if prev != nil && prev.key == key && m.now().Sub(prev.at) < diffFresh {
		return prev
	}
	book, err := diffbook.Read(filepath.Dir(m.store.Dir), chosen, base)
	// Outside a repository the list says there are no worktrees.
	if err != nil && len(book.Overview.Worktrees) == 0 && !errors.Is(err, git.ErrNoRepository) {
		m.status = say(m.tr.CannotReadChanges, map[string]any{"Err": err.Error()})
	}
	st := &diffState{key: key, at: m.now(), book: book, bodies: map[string]diffBody{}}
	if prev != nil && prev.prPath == book.Path {
		st.prPath, st.pr = prev.prPath, prev.pr
	} else if path := book.Path; path != "" {
		st.prPath = path
		m.pending = tea.Batch(m.pending, func() tea.Msg { return prMsg{note, path, diffbook.PR(path)} })
	}
	if m.diffs == nil {
		m.diffs = map[string]*diffState{}
	}
	m.diffs[note] = st
	return st
}

// gotPR shows the pull request gh found, if the note still shows that
// worktree.
func (m *Model) gotPR(msg prMsg) {
	if st := m.diffs[msg.note]; st != nil && st.prPath == msg.path && msg.pr != nil {
		st.pr = msg.pr
		m.reload()
	}
}

// diffOf is the diff of one changed file, read from git the first time
// its page is drawn.
func (m *Model) diffOf(note string, c git.Change) diffBody {
	st := m.diffs[note]
	if st == nil {
		return diffBody{}
	}
	key := c.Where + ":" + c.Path
	if b, ok := st.bodies[key]; ok {
		return b
	}
	var b diffBody
	b.text, b.cut, b.err = st.book.Diff(c)
	st.bodies[key] = b
	return b
}

// sourcePage is the page of one changed file: its diff, read when the
// page is first drawn, so a book of many files reads only the ones looked
// at.
type sourcePage struct {
	load func() diffBody
	w    *diff.Page
	note string // what to say under it: that it was cut, or why it is not there
	env  widget.Env
}

func (p *sourcePage) page() *diff.Page {
	if p.w == nil {
		b := p.load()
		p.w = diff.NewPage(b.text, p.env)
		switch {
		case b.err != nil:
			p.note = b.err.Error()
		case b.cut:
			p.note = p.env.T("the rest is not shown: it is over 1 MiB")
		}
	}
	return p.w
}

// Draw implements widget.Widget.
func (p *sourcePage) Draw(width int, active bool) (string, widget.Span) {
	out, at := p.page().Draw(width, active)
	if p.note != "" {
		var more []string
		for _, l := range widget.Wrap(p.note, width) {
			more = append(more, p.env.Styles.Faint.Render(l))
		}
		out += "\n" + strings.Join(more, "\n")
	}
	return out, at
}

// Summary implements widget.Widget.
func (p *sourcePage) Summary() string { return p.page().Summary() }

// Update implements widget.Widget: a diff takes no keys.
func (p *sourcePage) Update(string) (widget.Widget, widget.Result) { return p, widget.Result{} }

// Sync implements widget.Widget: the page has no file.
func (p *sourcePage) Sync(doc.Document) widget.Widget { return p }

// fromSource says whether the page shown is one git made, which has no
// file to edit, move or delete, and says so on the bottom line.
func (m *Model) fromSource(it item) bool {
	if len(it.pages) > 0 && it.file().Path == "" {
		m.status = m.tr.L("read only")
		return true
	}
	return false
}

// writable wraps an action on a note's file so that a page git made
// refuses it.
func writable(f func(*Model, item) tea.Cmd) func(*Model, item) tea.Cmd {
	return func(m *Model, it item) tea.Cmd {
		if m.fromSource(it) {
			return nil
		}
		return f(m, it)
	}
}

// diffKey carries out enter and b on the list of worktrees: enter shows
// the worktree under the cursor, b asks for the base. Both are kept in the
// diff note's entry in sticky.json. It reports whether the key was one.
func (m *Model) diffKey(i int, key string) bool {
	it := m.items[i]
	l, ok := it.w.(*book)
	if !ok || key != "enter" && key != "b" {
		return false
	}
	list, ok := l.current().w.(*diff.List)
	if !ok {
		return false
	}
	note := l.current().note.Name
	if key == "b" {
		m.ask(m.tr.L("base"), m.views[note].Base, func(text string) {
			m.setView(note, func(v *store.View) { v.Base = strings.TrimSpace(text) })
		})
		m.inputEmpty = true
		return true
	}
	chosen := list.Cursor()
	m.setView(note, func(v *store.View) { v.Worktree = chosen })
	// Turn to the first changed file, when there is one.
	if j := m.index(it.note.Name); j >= 0 {
		for _, p := range m.items[j].pages {
			if strings.HasPrefix(p.note.Name, note+":") {
				m.focusFile(p.note.Name)
				break
			}
		}
	}
	return true
}
