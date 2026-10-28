package api

import (
	"errors"
	"fmt"

	"github.com/LeeSwallow/stickypane/internal/store"
)

// Board reads the whole board in one pass: every tab with its notes, and
// the settings. It only reads.
func (a *API) Board() (store.Board, error) { return a.st.Load() }

// Tabs reads the tabs with their notes. It only reads.
func (a *API) Tabs() ([]store.Tab, error) { return a.st.Tabs() }

// ErrBadOrder is what Order answers for an order it cannot keep: empty, a
// name that is not a note of the tab, or a name given twice.
var ErrBadOrder = errors.New("not an order of the tab's notes")

// Order puts a tab's notes in the order given, by full name, as the
// terminal does when it swaps a note with its neighbour and the desktop
// app when a note is dragged. A note left out follows the others by name.
// A tab that is not there is fs.ErrNotExist.
func (a *API) Order(tab string, names []string) error {
	tabs, err := a.st.Tabs()
	if err != nil {
		return err
	}
	var here map[string]bool
	for _, t := range tabs {
		if t.Name == tab {
			here = map[string]bool{}
			for _, n := range t.Notes {
				here[n.Name] = true
			}
		}
	}
	if here == nil {
		return missing(fmt.Sprintf("there is no tab %q", tab))
	}
	if len(names) == 0 {
		return badOrder("the order is empty")
	}
	seen := map[string]bool{}
	for _, name := range names {
		switch {
		case !here[name]:
			return badOrder(fmt.Sprintf("%q is not a note of tab %q", name, tab))
		case seen[name]:
			return badOrder(fmt.Sprintf("%q is twice in the order", name))
		}
		seen[name] = true
	}
	if err := a.st.SetOrder(tab, names); err != nil {
		return err
	}
	_, _ = a.Derive()
	return nil
}

// badOrder is an ErrBadOrder in its own words.
type badOrder string

func (b badOrder) Error() string        { return string(b) }
func (b badOrder) Is(target error) bool { return target == ErrBadOrder }
