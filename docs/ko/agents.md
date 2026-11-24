# 에이전트와 함께 쓰기

에이전트가 보드를 알게 되는 방법과, 에이전트가 쓰는 명령입니다. 영문
[agents.md](../agents.md)가 원본이고 더 자세합니다(`setup` 옵션, MCP 도구 표).
[README로 돌아가기](../../README.ko.md)

에이전트는 파일만 고쳐도 됩니다. 명령과 MCP 도구는 노트를 먼저 읽지 않고 하나만
바꾸고 싶을 때 씁니다.

## 한 번만 알려 주면 됩니다

| 방법 | 명령 | 대상 |
| --- | --- | --- |
| 플러그인 | `stickypane setup` | Claude Code, Codex: 스킬 다섯 개, 슬래시 명령, 세션 훅 |
| 안내문 | `stickypane init`이 `CLAUDE.md`나 `AGENTS.md`에 넣습니다 | 그 파일을 읽는 모든 에이전트 |
| MCP | `stickypane setup --mcp` | MCP를 쓰는 모든 클라이언트 |

`setup`은 보드를 만들지 않고 안내문 파일도 쓰지 않으니 어느 폴더에서 실행해도 됩니다.
보드는 프로젝트에서 `stickypane init`으로 만듭니다. `setup --undo`는 플러그인을 지우고,
`--mcp`를 붙이면 MCP 서버도 지웁니다. 보드와 안내문은 그대로 둡니다.

`stickypane setup --status`는 에이전트마다, 범위(user, project, local)마다 플러그인
버전과 MCP 서버를 보여 줍니다. Codex는 user 범위만 있습니다. setup을 다시 실행하면
오래된 플러그인은 업데이트하고, 사라진 폴더를 가리키는 마켓플레이스는 GitHub로 다시
잇습니다.

플러그인은 계층으로 되어 있습니다. 입구 스킬 `using-the-board`가 무엇을 어떤 모양으로
보여 줄지와 상황에 맞는 스킬 열다섯 개를 알려 줍니다: 진행 추적(`tracking-the-work`),
일하며 대화(`talking-while-working`), 그림 그리기(`drawing-a-diagram`), 코드 전 계획
(`planning-before-code`), 막힘 보고(`reporting-a-blocker`), 고르기
(`choosing-between-options`), 위험한 일의 승인(`approving-a-risky-action`), 숫자
(`showing-numbers`), 명령 넘기기(`handing-over-a-command`), API 확인
(`checking-an-api`), 긴 글(`reading-long-material`), 노트 연결(`connecting-notes`),
워크트리(`working-in-a-worktree`), 이어받기와 넘기기(`resuming-and-handing-off`),
마무리(`finishing-the-task`). 스킬마다 사례 하나씩을 담은 레시피가 `references/`에
있고, 필요할 때만 `stickypane guide <스킬>/<레시피>`로 읽습니다. 에이전트 안내문
(`stickypane guide`)은 영어로 남습니다.

## 에이전트용 명령

```sh
stickypane note show plan                   # 띄우기 (경로를 주면 링크해서 띄움)
stickypane todo add plan "테스트 작성"      # plan.md: 0/1
stickypane todo check plan 테스트           # plan.md: 1/1
stickypane card move work 로그인 --to Done  # work.md: 2 cards
stickypane chart add tokens input 800       # tokens.md: input = 2,000
stickypane log add worklog --time "테스트 통과"
stickypane note set plan open=true size=1/2
stickypane note mv plan docs/   /   note rm plan   /   note restore plan
stickypane wait deploy --timeout 10m        # 폼의 버튼이 눌릴 때까지
```

없는 노트는 만들어지고, 항목·카드·칸은 글자 일부나 `#2` 같은 순번으로 가리킬 수
있습니다. `wait`는 시간이 다 되면 종료 코드 3으로 끝납니다. `stickypane mcp`는
같은 명령을 MCP 도구로 내놓습니다.

```sh
claude mcp add stickypane -- stickypane mcp
```

## 브리지

`stickypane bridge`는 에이전트나 사람이 쓰는 명령이 아닙니다. 만들고 있는 데스크톱
앱(#10)에 보드를 표준 입출력의 JSON 줄로 내놓습니다. 입력이 닫히면 끝나고, 띄운
스크립트도 멈춥니다.
