# proposal: 명령을 `<명사> <동사> <대상>` 한 규칙으로 맞춘다

라벨: proposal, needs triage
상위 이슈: #140
선행: 없음 (새 명령 `page`, `worktree`, `diff`가 이 규칙으로 들어간다)

## 문제

명령은 스무 개가량이고 문법이 반쯤만 맞는다.

- 모양별 명령은 이름이 먼저고 동사가 뒤에 온다. `todo plan add`, `card work move`, `chart tokens add`.
- 범용 명령은 동사가 먼저다. `write plan`, `mv plan deploy/`, `rm plan`.
- `log`, `say`는 동사가 아예 없다. `log worklog "x"`, `say chat "x"`.
- `link`만 이름 앞에 경로가 온다. `link <path> [name]`.

새로 들어올 명령(#141 워크트리, #143 diff, #148 페이지 추가)을 어느 문법에 맞출지 정할 수 없다.
`git worktree add`, `gh issue create`, `docker container ls`처럼 많이 쓰이는 `<도구> <명사> <동사> <대상>`을 따르면 외우는 부담이 줄고, 에이전트도 처음 보는 명령을 짐작하기 쉽다.
지침들은 순서를 하나로 정하고 끝까지 지키라고 한다([CLI Guidelines 요약](https://cdn.jsdelivr.net/npm/@hacxy/skills@0.1.0/skills/create-cli/references/cli-guidelines.md), [Microsoft 설계 안내](https://learn.microsoft.com/en-us/dotNET/standard/commandline/design-guidance)).
명사-동사와 동사-명사 중 어느 쪽이 낫다는 합의는 없다([Habitat](https://www.github.com/habitat-sh/habitat/blob/main/UX_PRINCIPLES.md)는 명사 먼저, [PatternFly](https://www.patternfly.org/content-design/writing-guides/cli-handbook)는 동사 먼저). 여기서는 둘 중 널리 쓰이는 쪽과 새 명령이 닿는 `git worktree`에 맞춘다.

## 제안

**규칙.** `stickypane <명사> <동사> <대상> [인자]`.

- 명사는 모양이나 대상이다. `note`, `page`, `worktree`, `diff`, `todo`, `card`, `chart`, `log`, `chat`, `form`.
- 동사는 `add`, `list`(별칭 `ls`), `show`, `set`, `mv`, `rm`으로 맞춘다. 모양별 동작(`check`, `uncheck`, `move`)은 그 모양의 명사 아래에 남긴다.
- 대상 이름은 동사 바로 뒤에 온다.
- 모든 `list`와 `show`는 `--json`을 받는다.

| 지금 | 새 형태 |
| --- | --- |
| `todo plan add "x"` | `todo add plan "x"` |
| `todo plan check x` | `todo check plan x` |
| `card work move login --to Done` | `card move work login --to Done` |
| `chart tokens add input 1200` | `chart add tokens input 1200` |
| `log worklog --time "x"` | `log add worklog --time "x"` |
| `say chat "x"` | `chat add chat "x"` |
| `write`, `cat`, `show`, `set`, `mv`, `rm`, `archive`, `link`, `hide`, `list` | `note write`, `note cat`, `note show`, `note set`, `note mv`, `note rm`, `note archive`, `note link`, `note hide`, `note list` |
| 없음 | `page add <책> [이름]`, `page list <책>` |
| 없음 | `worktree list`, `worktree show <이름>` |
| 없음 | `diff list [<워크트리>]`, `diff show <워크트리> <파일>` |

`say`가 `chat add chat`이 되는 어색함은 동사를 `add`로 통일해 푼다(`todo add`, `card add`, `log add`, `chart add`, `chat add`). 말맛은 줄지만 한 규칙이 된다.

**옛 형태는 계속 받는다.** 명령 표의 `names []string`이 이미 별칭을 받는다(`config`의 `theme`, `language`). 옛 순서는 숨은 별칭으로 둔다.

- 모양별 명령에서 두 번째 말이 동사(`add|check|uncheck|move|set`)이면 옛 순서(`todo plan add x`)로 읽고, 첫 말이 동사이면 새 순서로 읽는다. 노트 이름이 동사와 같은 경우(`todo add add x`)가 있을 수 있어 규칙을 정해야 한다 (확인 필요).
- 옛 형태는 경고 없이 동작한다. 없애는 시점은 #145의 계측 결과가 나온 뒤에 따로 정한다.
- 가이드와 스킬은 새 형태로 바꾼다. 가이드 55줄 한도는 그대로이고, 새 명령(`page`, `worktree`, `diff`)은 기본 안내가 아니라 `stickypane guide <주제>`에서만 알린다(#146).

**기계가 읽는 명령 목록.** `stickypane help --json`이 명령 이름, 인자, 예시를 안정된 JSON으로 낸다. 조사에서 에이전트용 CLI에 반복해서 나온 방식이다([Algolia](https://www.algolia.com/blog/engineering/we-rewrote-the-algolia-cli-for-ai-agents), [Poehnelt](https://justin.poehnelt.com/posts/rewrite-your-cli-for-ai-agents.md)). 둘 다 실무 글이고 표준은 아니다.
명령 하나의 결과를 한 줄로 주는 방식(`plan.md: 2/5`)은 토큰을 아끼므로 그대로 둔다.

## MCP와 플러그인도 같은 규칙을 따른다

규칙이 CLI에만 있으면 에이전트는 표면마다 다른 이름을 외운다. 세 표면이 같은 이름을 쓰게 한다.

**MCP 서버(`internal/surface/mcp/server.go`).** 지금 도구는 열네 개이고 이름 규칙이 섞여 있다.

| 지금 | 형태 | 새 이름 |
| --- | --- | --- |
| `todo`, `card`, `chart` (`name`, `action`) | 명사 + `action` 값 | 그대로. `action`이 CLI의 동사와 같은 값(`add`, `check`, `move`…)이다 |
| `log`, `say` | 명사, 동사 없음 | `log`는 그대로(`action: add`를 기본으로), `say`는 `chat`(`action: add`) |
| `read_note`, `write_note`, `set_keys`, `arrange_note` | 동사_명사, 제각각 | `note` 하나에 `action: read\|write\|set\|mv\|rm\|show\|hide\|archive\|restore` |
| `index` | 명사 | `note`의 `action: list`(요약) |
| `read_answers`, `wait_event`, `api`, `guide` | 제각각 | 이 이슈에서는 그대로 |
| 없음 | | `page`, `worktree`, `diff` (`action: add\|list\|show`) |

- 도구마다 `action` 값 하나로 동작을 가르는 지금 구조를 유지한다. 도구를 동사별로 쪼개면 도구 정의가 모델의 문맥을 더 먹는다.
- 옛 도구 이름은 목록에는 내지 않고, 부르면 동작하게 둔다. 이미 설정된 클라이언트가 깨지지 않는다 (확인 필요: 목록에 없는 도구를 부를 수 있는지).
- MCP의 `guide`는 새 이름으로 안내한다. 사용자가 `stickypane guide`와 MCP `guide`에서 같은 글을 읽는다(`internal/usecase/harness`).

**플러그인(`plugin/`).**
- `commands/*.md`(`ask`, `say`, `status`, `track`, `setup`)의 본문이 부르는 명령을 새 형태로 바꾼다. 슬래시 이름은 바꾸지 않는다.
- `skills/*/SKILL.md`, `rules/`, `guide.md`의 예시, `scripts/session-start.sh`가 찍는 안내 줄, 두 `plugin.json`.
- `test/plugin_test.go`는 안내와 스킬이 "실제 있는 명령만 쓰는지" 검사한다. 새 형태가 그 목록에 들어가야 통과한다.

**세 표면이 어긋나지 않게 잡는 테스트.** 명령 표의 `<명사> <동사>`와 MCP의 `<도구>` + `action`이 하나씩 짝이 되는지 검사한다. 한쪽에 명령을 더하고 다른 쪽을 빼먹으면 실패한다.

**나누어 낸다.** 한 번에 하면 크다. CLI와 별칭, MCP, 플러그인 안내와 스킬 순서로 풀 리퀘스트를 따로 낸다. 가이드와 스킬의 전환은 마지막이고, 그 전과 후를 #145로 견준다.

## 대안

| | 내용 | 득 | 실 |
| --- | --- | --- | --- |
| 추가만 | 기존 순서(`todo plan add`)로 새 명령을 더한다 | 가장 안전 | 규칙이 계속 섞여 있다 |
| 새 영역만 명사-동사 | `page add`, `worktree list`만 | 새 기능은 깔끔 | 기존과 순서가 섞인다 |
| 전체 맞춤(이 제안) | 위 규칙, 옛 형태는 별칭 | 한 규칙 | 가이드, 스킬, 테스트, README를 고쳐야 함 |

사용자가 한 규칙을 원해서 셋째로 간다. 위험은 별칭으로 줄인다.
에이전트가 지금 명령을 자주 틀린다는 자료는 없다. 새 규칙이 틀림을 줄이는지는 #145의 실험으로 옛 형태와 새 형태를 견줘서 본다.

## 바뀌는 것

- `cmd/stickypane/commands.go`의 명령 표: 이름과 별칭, 도움말. 표와 도움말이 어긋나지 않게 잡아 주는 테스트가 이미 있다.
- 새 명령 `note`, `page`, `worktree`, `diff`, `chat`의 처리부. 안쪽은 지금의 `api` 호출을 그대로 쓴다.
- `stickypane help --json`은 지금 없다(`flag provided but not defined: -json`). 새로 만든다.
- `stickypane <명령> --help`는 지금 사용법 한 블록만 낸다. `stickypane help <명령>`과 같은 내용(설명, 인자, 예시, 관련 명령)을 주게 맞춘다. 사람도 에이전트도 `--help`를 가장 먼저 치기 때문이다.
- `help`(번역 대상): 한국어와 영어 글. `docs/usage.md`, README, README.ko.md.
- `plugin/guide.md`, `plugin/rules/`, `plugin/skills/*`: 새 형태로. 줄 수 한도 55줄.
- MCP 서버의 도구 이름이 명령 이름을 따르는지 (확인 필요).

## 하지 않을 것

- 옛 형태를 없애거나 경고를 띄우지 않는다. 이 이슈에서는 하지 않는다.
- `init`, `setup`, `config`, `keys`, `env`, `watch`, `wait`, `answers`, `guide`, `mcp`, `bridge`처럼 대상이 노트가 아닌 명령은 건드리지 않는다.
- 한 줄 결과 출력의 형식을 바꾸지 않는다.

## 열린 질문

- 이름이 동사와 같은 노트(`todo add add x`)를 어떻게 읽을지.
- `note`를 생략하고 `stickypane show plan`처럼 짧게도 받을지. 지금 에이전트가 가장 많이 쓰는 형태라 별칭으로 남긴다.
- `index`를 `note list --summary`로 합칠지 따로 둘지.
- MCP 도구 이름과 `help --json`의 형식.

## 테스트 계획

- 옛 형태와 새 형태가 같은 결과를 내는지 짝으로 검사한다(표의 모든 줄).
- 명령 표와 도움말이 어긋나지 않는지(기존 테스트가 새 명령도 잡는지).
- 가이드와 스킬이 실제 있는 명령만 쓰는지(`test/plugin_test.go`)와 55줄 한도.
- `help --json`이 안정된 형식인지(골든 파일).

## 에이전트 브리프

- **목표:** 위 규칙으로 명령을 맞추고, 옛 형태는 모두 그대로 동작한다.
- **먼저 읽을 것:** `cmd/stickypane/commands.go`, `notes.go`, `help.go`, `help_test.go`, `docs/usage.md`, `plugin/`의 안내와 스킬, `test/plugin_test.go`.
- **할 일:** 별칭 짝 테스트를 먼저 쓴다. 안쪽 호출은 건드리지 않고 이름과 순서만 바꾼다. 가이드와 스킬은 마지막에 바꾸고 줄 수를 센다.
- **하지 말 것:** 옛 형태 제거, 새 모듈, 한 줄 결과 형식 변경.
- **끝난 조건:** `go vet ./... && go test -race ./... && test -z "$(gofmt -l .)"`가 통과하고, 옛 형태 짝이 모두 같은 결과를 내고, #145의 사례로 새 형태를 측정할 수 있다.
