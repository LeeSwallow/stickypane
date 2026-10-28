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
// API speaks Korean: the summary is the note's, in the Env's words; the
// rest of what it answers is what was asked.
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
		do   func() (Edited, error)
		want Edited
	}{
		{func() (Edited, error) { return a.Todo("plan", "add", "테스트 작성") }, Edited{Note: "plan.md", Action: "add", Item: "테스트 작성", Summary: "0/1"}},
		{func() (Edited, error) { return a.Card("work", "add", "로그인", "") }, Edited{Note: "work.md", Action: "add", Item: "로그인", Summary: "카드 1개"}},
		{func() (Edited, error) { return a.Card("work", "add", "결제", "") }, Edited{Note: "work.md", Action: "add", Item: "결제", Summary: "카드 2개"}},
		{func() (Edited, error) { return a.Chart("tokens", "set", "input", "1,200") }, Edited{Note: "tokens.md", Action: "set", Item: "input", Value: "1,200", Summary: "값 1개"}},
		{func() (Edited, error) { return a.Log("worklog", "테스트 통과") }, Edited{Note: "worklog.md", Action: "log", Item: "테스트 통과", Summary: "1줄"}},
		{func() (Edited, error) { return a.Log("worklog", "리뷰 끝") }, Edited{Note: "worklog.md", Action: "log", Item: "리뷰 끝", Summary: "2줄"}},
		{func() (Edited, error) { return a.Say("chat", "", "안녕") }, Edited{Note: "chat.md", Action: "say", Item: "안녕", Who: "agent", Summary: "메시지 1개"}},
		{func() (Edited, error) { return a.Say("chat", "me", "응") }, Edited{Note: "chat.md", Action: "say", Item: "응", Who: "me", Summary: "메시지 2개"}},
	}
	for i, s := range steps {
		if got, err := s.do(); err != nil || got != s.want {
			t.Errorf("step %d = %+v, %v; want %+v", i, got, err, s.want)
		}
	}
}
