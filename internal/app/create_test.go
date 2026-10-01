package app

import (
	"os"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/widget/board"
)

func TestJotCreatesAPlainNote(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": "one\n"})
	press(m, "n")
	if s := screen(m); !strings.Contains(s, "Jot:") {
		t.Fatalf("the jot prompt should show:\n%s", s)
	}
	typeText(m, "Check env before deploy")
	press(m, "enter")
	if got := readFile(t, dir, "check-env-before-deploy.md"); got != "Check env before deploy\n" {
		t.Errorf("file = %q", got)
	}
	if m.focus != "check-env-before-deploy.md" {
		t.Errorf("focus = %q, want the new note", m.focus)
	}
	s := screen(m)
	if !strings.Contains(s, "Check env before deploy") || strings.Contains(s, "●") {
		t.Errorf("the new note should show and not be marked as changed:\n%s", s)
	}
}

func TestJotKeepsKoreanInTheFileName(t *testing.T) {
	m, dir := newModel(t, nil)
	press(m, "n")
	typeText(m, "배포 전에 확인")
	press(m, "enter")
	if got := readFile(t, dir, "배포-전에-확인.md"); got != "배포 전에 확인\n" {
		t.Errorf("file = %q", got)
	}
}

func TestEmptyJotCreatesNothing(t *testing.T) {
	m, dir := newModel(t, nil)
	press(m, "n", "enter")
	press(m, "n", "esc")
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("no file should exist, got %v", entries)
	}
}

func TestCatalogCreatesTheChosenShape(t *testing.T) {
	m, dir := newModel(t, nil)
	press(m, "a")
	s := screen(m)
	for _, want := range []string{"Add a note", "Note", "Board (kanban)", "Checklist", "Log"} {
		if !strings.Contains(s, want) {
			t.Fatalf("catalog should list %q:\n%s", want, s)
		}
	}
	press(m, "j", "enter")
	if s := screen(m); !strings.Contains(s, "Title:") {
		t.Fatalf("choosing a shape should ask for a title:\n%s", s)
	}
	typeText(m, "Auth work")
	press(m, "enter")
	if got := readFile(t, dir, "auth-work.md"); got != string(board.Kind.Template("Auth work")) {
		t.Errorf("file = %q", got)
	}
	if m.focus != "auth-work.md" || m.mode != modeBoard {
		t.Errorf("focus = %q, mode = %v", m.focus, m.mode)
	}
}

func TestCatalogCanBeCancelled(t *testing.T) {
	m, dir := newModel(t, nil)
	press(m, "a", "j", "esc")
	if m.mode != modeBoard {
		t.Errorf("mode = %v", m.mode)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("no file should exist, got %v", entries)
	}
}
