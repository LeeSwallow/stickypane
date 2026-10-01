package api

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const formNote = "---\ntype: form\ntitle: Deploy\n---\n## Where?\n- (x) staging\n- ( ) production\n\n[ Go ]\n"

func TestAnswersReadsAForm(t *testing.T) {
	a, _ := newAPI(t, map[string]string{"f.md": formNote, "n.md": "plain\n"})
	got, err := a.Answers("f")
	if err != nil || got.Submitted || len(got.Answers) != 1 || got.Answers[0].Question != "Where?" || strings.Join(got.Answers[0].Values, ",") != "staging" {
		t.Errorf("Answers = %+v, %v", got, err)
	}
	if _, err := a.Answers("n"); !errors.Is(err, ErrNotForm) {
		t.Errorf("a plain note has no answers, got %v", err)
	}
	if _, err := a.Answers("missing"); err == nil {
		t.Error("a missing note should be an error")
	}
}

func TestWaitReturnsWhenAButtonIsPressed(t *testing.T) {
	a, dir := newAPI(t, map[string]string{"f.md": formNote})
	go func() {
		time.Sleep(30 * time.Millisecond)
		pressed := strings.Replace(formNote, "title: Deploy\n", "title: Deploy\nsubmitted: Go\n", 1)
		_ = os.WriteFile(filepath.Join(dir, "f.md"), []byte(pressed), 0o644)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	got, err := a.Wait(ctx, "f", 5*time.Millisecond)
	if err != nil || !got.Submitted || got.Button != "Go" {
		t.Errorf("Wait = %+v, %v", got, err)
	}
}

func TestWaitGivesUpWhenItsTimeIsOver(t *testing.T) {
	a, _ := newAPI(t, map[string]string{"f.md": formNote})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	got, err := a.Wait(ctx, "f", 5*time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) || got.Submitted || len(got.Answers) != 1 {
		t.Errorf("Wait should return the answers so far and the deadline: %+v, %v", got, err)
	}
}

func TestARewriteClearsTheSubmission(t *testing.T) {
	pressed := strings.Replace(formNote, "title: Deploy\n", "title: Deploy\nopen: true\nsubmitted: Go\nsubmitted_at: 2026-10-02T14:03:05Z\n", 1)
	a, dir := newAPI(t, map[string]string{"f.md": pressed})
	if _, err := a.Write("f", Options{Type: "form"}, []byte("## Again?\n- ( ) yes\n")); err != nil {
		t.Fatal(err)
	}
	if got := read(t, dir, "f.md"); strings.Contains(got, "submitted") || !strings.Contains(got, "open: true") {
		t.Errorf("a new question starts unanswered and stays on the screen:\n%s", got)
	}
}
