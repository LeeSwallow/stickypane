package shell

import (
	"testing"
	"time"

	"github.com/LeeSwallow/stickypane/desktop/internal/index"
)

// timedIndex records the times the shell asks the index at.
type timedIndex struct {
	fakeIndex
	pruned, listed time.Time
}

func (ix *timedIndex) Prune(now time.Time) ([]index.Project, error) {
	ix.mu.Lock()
	ix.pruned = now
	ix.mu.Unlock()
	return nil, nil
}

func (ix *timedIndex) Sessions(now time.Time) (index.Groups, error) {
	ix.mu.Lock()
	ix.listed = now
	ix.mu.Unlock()
	return index.Groups{}, nil
}

// The home list is made at the time of the shell's clock, one time for
// the tidying and the list.
func TestSessionsTakeTheTimeFromTheShellsClock(t *testing.T) {
	at := time.Date(2026, 10, 28, 14, 2, 11, 0, time.UTC)
	ix := &timedIndex{}
	r := newRig(t, func(c *Config) {
		c.OpenIndex = func(string) (Index, error) { return ix, nil }
		c.Now = func() time.Time { return at }
	})
	r.s.Start()
	if _, err := r.s.Sessions(); err != nil {
		t.Fatal(err)
	}
	ix.mu.Lock()
	defer ix.mu.Unlock()
	if !ix.pruned.Equal(at) || !ix.listed.Equal(at) {
		t.Errorf("pruned at %v, listed at %v; want %v", ix.pruned, ix.listed, at)
	}
}
