package store

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/LeeSwallow/stickypane/internal/core/doc"
)

// appendLine adds a line to a note, slowly, so that two writers that do
// not wait for each other would both read the note before either wrote.
type appendLine string

func (l appendLine) Apply(d doc.Document) (doc.Document, error) {
	time.Sleep(time.Millisecond)
	d.Body += string(l) + "\n"
	return d, nil
}

const (
	helperDir   = "STICKYPANE_LOCK_HELPER_DIR"
	helperWho   = "STICKYPANE_LOCK_HELPER_WHO"
	linesEach   = 25
	helperProcs = 2
	helperGos   = 2
)

// appendLines applies linesEach appendLine ops to shared.md, from a
// store of its own.
func appendLines(dir, who string) error {
	s := Open(dir)
	for i := range linesEach {
		if err := s.Apply("shared.md", appendLine(fmt.Sprintf("%s-%d", who, i))); err != nil {
			return err
		}
	}
	return nil
}

// TestLockHelper is the other process of TestApplyAcrossProcessesLosesNothing.
func TestLockHelper(t *testing.T) {
	dir := os.Getenv(helperDir)
	if dir == "" {
		t.Skip("run by TestApplyAcrossProcessesLosesNothing")
	}
	if err := appendLines(dir, os.Getenv(helperWho)); err != nil {
		t.Fatal(err)
	}
}

// Two processes, and two goroutines in this one, add lines to one note at
// once. Every Apply reads the note as it is and writes it back: without a
// lock between them, one would write over a line another just added.
func TestApplyAcrossProcessesLosesNothing(t *testing.T) {
	if testing.Short() {
		t.Skip("starts processes")
	}
	s := newStore(t)
	write(t, s, "shared.md", "")
	var wg sync.WaitGroup
	errs := make(chan error, helperProcs+helperGos)
	for p := range helperProcs {
		cmd := exec.Command(os.Args[0], "-test.run=^TestLockHelper$", "-test.count=1")
		cmd.Env = append(os.Environ(), helperDir+"="+s.Dir, fmt.Sprintf("%s=proc%d", helperWho, p))
		wg.Go(func() {
			if out, err := cmd.CombinedOutput(); err != nil {
				errs <- fmt.Errorf("%v: %s", err, out)
			}
		})
	}
	for g := range helperGos {
		wg.Go(func() {
			if err := appendLines(s.Dir, fmt.Sprintf("go%d", g)); err != nil {
				errs <- err
			}
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	lines := strings.Fields(read(t, s, "shared.md"))
	if want := (helperProcs + helperGos) * linesEach; len(lines) != want {
		t.Fatalf("the note has %d lines, want %d: updates were lost", len(lines), want)
	}
}

// Two goroutines arrange different notes of one tab at once: every view
// is kept.
func TestSetViewAtOnceLosesNothing(t *testing.T) {
	s := newStore(t)
	const each = 20
	for i := range 2 * each {
		write(t, s, fmt.Sprintf("n%d.md", i), "x\n")
	}
	var wg sync.WaitGroup
	for g := range 2 {
		wg.Go(func() {
			other := Open(s.Dir)
			for i := g * each; i < (g+1)*each; i++ {
				if err := other.SetView(fmt.Sprintf("n%d.md", i), func(v *View) { v.Color = "red" }); err != nil {
					t.Error(err)
				}
			}
		})
	}
	wg.Wait()
	views, err := s.Views()
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 2*each {
		t.Fatalf("%d views kept, want %d", len(views), 2*each)
	}
}
