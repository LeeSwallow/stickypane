package api

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// Event is something that happened on the board: in which note, of which
// kind, when, and what, in the kind's own words (widget.Event). Notes that
// appear and go are "note.created" and "note.removed"; a change a kind has
// no words for is "note.changed".
type Event struct {
	Time time.Time `json:"time"`
	Note string    `json:"note"`
	Kind string    `json:"kind"`
	widget.Event
}

// Filter picks events. Notes are names, with or without ".md"; a folder
// name takes its notes too. Types are event types or their first part
// ("card" takes card.added, card.moved, card.removed). Empty takes all.
type Filter struct {
	Notes []string
	Types []string
}

// Match reports whether the filter takes e.
func (f Filter) Match(e Event) bool {
	ok := len(f.Notes) == 0
	for _, n := range f.Notes {
		n = strings.TrimSuffix(strings.TrimSpace(n), "/")
		if e.Note == n || e.Note == n+".md" || strings.HasPrefix(e.Note, n+"/") {
			ok = true
		}
	}
	if !ok {
		return false
	}
	if len(f.Types) == 0 {
		return true
	}
	for _, t := range f.Types {
		if e.Type == t || strings.HasPrefix(e.Type, t+".") {
			return true
		}
	}
	return false
}

// Changes says what happened between two readings of the board, note by
// note, in name order.
func (a *API) Changes(before, after store.Board) []Event {
	was, now := files(before), files(after)
	names := make([]string, 0, len(was)+len(now))
	for n := range was {
		names = append(names, n)
	}
	for n := range now {
		if _, ok := was[n]; !ok {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	var out []Event
	for _, name := range names {
		b, hadB := was[name]
		n, hasN := now[name]
		switch {
		case !hadB:
			out = append(out, Event{Time: n.ModTime, Note: name, Kind: a.reg.For(name, n.Doc).Name, Event: widget.Event{Type: "note.created"}})
		case !hasN:
			out = append(out, Event{Time: time.Now(), Note: name, Kind: a.reg.For(name, b.Doc).Name, Event: widget.Event{Type: "note.removed"}})
		case string(b.Doc.Bytes()) != string(n.Doc.Bytes()):
			k := a.reg.For(name, n.Doc)
			var evs []widget.Event
			if k.Events != nil && k.Name == a.reg.For(name, b.Doc).Name {
				evs = k.Events(b.Doc, n.Doc)
			}
			if len(evs) == 0 {
				evs = []widget.Event{{Type: "note.changed"}}
			}
			for _, e := range evs {
				out = append(out, Event{Time: n.ModTime, Note: name, Kind: k.Name, Event: e})
			}
		}
	}
	return out
}

// files is every file of a board by name: notes, and the pages of books.
func files(b store.Board) map[string]store.Note {
	out := map[string]store.Note{}
	for _, t := range b.Tabs {
		for _, n := range t.Notes {
			if !n.Book() {
				out[n.Name] = n
			}
			for _, p := range n.Pages {
				out[p.Name] = p
			}
		}
	}
	return out
}

// Watch follows the board and hands each event the filter takes to emit,
// until ctx ends or emit fails. ready, when given, is closed once the
// board is being watched, so a caller knows nothing will be missed.
func (a *API) Watch(ctx context.Context, f Filter, emit func(Event) error, ready chan<- struct{}) error {
	board, err := a.st.Load()
	if err != nil {
		return err
	}
	changed, err := a.st.Watch(ctx)
	if err != nil {
		return err
	}
	if ready != nil {
		close(ready)
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case _, ok := <-changed:
			if !ok {
				return nil
			}
			next, err := a.st.Load()
			if err != nil {
				return err
			}
			for _, e := range a.Changes(board, next) {
				if f.Match(e) {
					if err := emit(e); err != nil {
						return err
					}
				}
			}
			board = next
		}
	}
}
