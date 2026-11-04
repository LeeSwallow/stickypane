package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/usecase/initcmd"
)

// What a person reads speaks the board's language, here Korean, all of
// it: the status of setup, its plan and its refusals, the settings and
// their refusals, the keys' warnings, init and an unknown command. What is
// typed (commands, flags, scopes, versions, keys, values) stays as it is.
func TestThePersonsCommandsSpeakKorean(t *testing.T) {
	project(t)
	t.Setenv("STICKYPANE_LANG", "ko")
	_, f := stubSetup(t, "claude")
	f.agent("claude").Plugin["user"] = initcmd.PluginStatus{Version: "0.2.0", State: initcmd.StateOutdated}
	f.agent("claude").State = initcmd.StateOutdated

	_, out, _ := exec(t, "setup", "--status")
	for _, want := range []string{"보드 플러그인 버전: 0.2.1", "Claude Code: 업데이트 있음", "플러그인", "user 0.2.0 (구버전)", "MCP 서버", "없음", "Codex: 없음 (PATH에 codex 명령이 없어요)", "`stickypane setup --agents claude`로 업데이트해요."} {
		if !strings.Contains(out, want) {
			t.Errorf("setup --status should say %q:\n%s", want, out)
		}
	}
	_, out, _ = exec(t, "setup", "--agents", "claude", "--dry-run")
	if !strings.Contains(out, "stickypane setup이 실행할 명령:\n  claude plugin") {
		t.Errorf("setup --dry-run:\n%s", out)
	}
	_, out, _ = exec(t, "setup", "--agents", "claude")
	if !strings.Contains(out, "`stickypane setup --yes`") || !hangul.MatchString(lastLine(out)) {
		t.Errorf("setup without --yes:\n%s", out)
	}
	for _, c := range []struct {
		args []string
		says string
	}{
		{[]string{"setup", "--scope", "galaxy"}, "stickypane: --scope에는 user, project, local만 쓸 수 있어요: \"galaxy\"\n사용법:\n  stickypane setup "},
		{[]string{"config", "columns", "9"}, "stickypane: columns에는 3, 4, 1, 2만 쓸 수 있어요: \"9\"\n"},
		{[]string{"config", "colour", "x"}, "stickypane: 없는 설정이에요: \"colour\". 설정: theme,"},
		{[]string{"theme", "no-such-theme"}, "stickypane: 없는 테마예요: \"no-such-theme\". 테마 목록:\n* auto  (터미널을 따름:"},
		{[]string{"language", "xx"}, "stickypane: 번역이 없는 언어예요: \"xx\". 언어 목록:\n"},
		{[]string{"frobnicate", "x"}, "stickypane: 없는 명령이에요: frobnicate\n"},
		{[]string{"help", "setpu"}, "stickypane: 없는 명령이나 모양이에요: \"setpu\". 혹시 \"setup\" 말인가요?"},
		{[]string{"keys", "reset", "nope"}, "stickypane: 없는 동작이에요: \"nope\"."},
	} {
		if code, _, errOut := exec(t, c.args...); code == 0 || !strings.Contains(errOut, c.says) {
			t.Errorf("%v: code = %d, stderr = %q, want %q", c.args, code, errOut, c.says)
		}
	}

	_, out, _ = exec(t, "config")
	if !strings.Contains(out, "(기본값)") || !strings.Contains(out, "\n그 밖의 설정:\n") || !strings.Contains(out, "화면 색") {
		t.Errorf("config:\n%s", out)
	}
	_, out, _ = exec(t, "help", "--all")
	if !strings.Contains(out, "  stickypane setup [--scope") || !strings.Contains(out, "설치") {
		t.Errorf("help --all:\n%s", out)
	}
}

// The warnings of stickypane keys end in the board's language too.
func TestKeysWarnInKorean(t *testing.T) {
	keys := keysHome(t)
	t.Setenv("STICKYPANE_LANG", "ko")
	if err := os.MkdirAll(keys, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(keys, "keys.json"), []byte(`{"note-archive": ["q"], "nope": ["x"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, errOut := exec(t, "keys")
	for _, line := range strings.Split(strings.TrimSpace(errOut), "\n") {
		if !strings.HasSuffix(line, "이 동작은 기본 키를 써요.") {
			t.Errorf("a warning of keys ends in Korean: %q", line)
		}
	}
	if err := os.WriteFile(filepath.Join(keys, "keys.json"), []byte(`{`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, errOut := exec(t, "keys"); !strings.Contains(errOut, "그래서 기본 키를 써요.") {
		t.Errorf("an unreadable keys.json: %q", errOut)
	}
	if _, _, errOut := exec(t, "keys", "set", "jot", "J"); !strings.Contains(errOut, "stickypane keys reset으로 처음부터 다시 만드세요.") {
		t.Errorf("keys set on an unreadable keys.json: %q", errOut)
	}
}

func TestInitSpeaksTheBoardsLanguage(t *testing.T) {
	repo(t)
	t.Setenv("STICKYPANE_LANG", "en")
	_, out, _ := exec(t, "init", "--language", "ko")
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 || lines[0] != ".sticky/ 폴더를 만들고 환영 노트를 넣었어요." || !strings.HasPrefix(lastLine(out), "다음: ") {
		t.Errorf("init --language ko:\n%s", out)
	}
	for _, l := range lines {
		if !hangul.MatchString(l) {
			t.Errorf("init says a line in English: %q", l)
		}
	}
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	return lines[len(lines)-1]
}
