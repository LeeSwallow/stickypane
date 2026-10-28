package api

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/i18n"
	"github.com/LeeSwallow/stickypane/internal/kinds"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/widget"
	"github.com/LeeSwallow/stickypane/internal/widget/note"
)

// TestAnEditSaysTheSummaryInTheEnvsWords pins what an edit answers when the
// API speaks Korean: the summary is the note's, in the Env's words.
func TestAnEditSaysTheSummaryInTheEnvsWords(t *testing.T) {
	ko, err := i18n.Load("ko")
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), store.DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	a := NewWithEnv(store.Open(dir), kinds.Default(note.Plain), widget.Env{Translate: ko.L})
	steps := []struct {
		do   func() (string, error)
		want string
	}{
		{func() (string, error) { return a.Todo("plan", "add", "테스트 작성") }, "plan.md: 0/1"},
		{func() (string, error) { return a.Card("work", "add", "로그인", "") }, "work.md: 카드 1개"},
		{func() (string, error) { return a.Card("work", "add", "결제", "") }, "work.md: 카드 2개"},
		{func() (string, error) { return a.Chart("tokens", "set", "input", "1,200") }, "tokens.md: input = 1,200"},
		{func() (string, error) { return a.Log("worklog", "테스트 통과") }, "worklog.md: 1줄"},
		{func() (string, error) { return a.Log("worklog", "리뷰 끝") }, "worklog.md: 2줄"},
		{func() (string, error) { return a.Say("chat", "", "안녕") }, "chat.md: 메시지 1개"},
		{func() (string, error) { return a.Say("chat", "me", "응") }, "chat.md: 메시지 2개"},
	}
	for i, s := range steps {
		if got, err := s.do(); err != nil || got != s.want {
			t.Errorf("step %d = %q, %v; want %q", i, got, err, s.want)
		}
	}
}
