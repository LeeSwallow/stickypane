# 사례 분류: 조사한 사례 120개는 어디에 들어가는가

2026-11-23. [shape-usecases.md](2026-11-23-shape-usecases.md)의 사례 120개를 하나씩 읽고, 가이드의 어느 주제에 실을지 정한다. 기준은 [이슈 #146](https://github.com/LeeSwallow/stickypane/issues/146)([F-guide-by-situation.md](2026-11-23-worktree-issues/F-guide-by-situation.md))의 열두 주제에 새 주제를 더한 것이다. 쓸모없는 사례는 버려도 된다는 기준이라, 약하거나 겹치거나 에이전트가 드물게 할 일이거나 계획에 없는 기능이 필요한 사례는 `exclude`로 뺐다.

읽는 법. 키는 `<모양>-<번호>`다. 번호가 겹치기 때문이다(`book-B1`은 책, `board-B1`은 보드, `worktree-B1`은 워크트리 상황. `checklist-C1`과 `chart-C1`도 다르다). 판정은 셋 중 하나다.

- `topic`: 그 주제의 중심이 되는 사례. 자주 있고 값이 크다.
- `example`: 그 주제의 `references/`에 예시로 들어가는 사례. 다른 조리법을 더한다.
- `exclude`: 가이드에 싣지 않는다. 확인할 수 있는 이유를 단다.

사례는 모두 120개다(note 10, book 10, html 10, checklist 10, board 10, chat 10, form 10, log 10, chart 10, script 10, api 10, worktree 10). 찾은 방법: `^#{3,4} [A-Z]+[0-9]+\.`와 `^## [A-Z]+[0-9]+\. \*\*`(log, chart, script는 이차 제목이다)로 건져 구역 머리말로 모양을 정했다.

판정별 개수: **topic 29 / example 44 / exclude 47** (합 120). 끝의 개수 확인이 이 숫자를 다시 센다.

명령은 이 저장소의 `stickypane help`와 `stickypane kinds`로 확인했다(`stickypane 0.3.0`). 보고서가 쓰는 명령은 모두 있다. 없는 것이 둘 있다. `stickypane todo plan`처럼 동사 없이 현황만 읽는 형태는 없다(`todo <name> add|check|uncheck`뿐이라 `checklist-C8`은 `stickypane cat plan`으로 읽는다). `submitted_at` 키도 없다(`wait`의 출력은 `at:`이라 `form-F8`에 적었다). `write <책>/<쪽>`이 책 안에 쓰는지는 `stickypane help write`가 `deploy/steps` 꼴을 보여 주지만 책에 되는지는 실행해 확인하지 못했다(사용자의 보드를 건드리지 않으려고 돌리지 않았다).

## 주제

`Use when` 줄은 영어 한 문장이다. 의존은 #141 워크트리 추적, #142 쪽 넘김, #143 diff 보기, #144 훅 한 줄, #145 효과 측정, #146 상황별 가이드, #147 데스크톱, #148 책에 쪽 더하기, #149 명령 문법. 모든 주제는 #146이 가이드를 다시 자른 뒤에야 가이드에 실리므로 그것은 적지 않는다.

| 주제 | Use when | 모양 | 의존 |
| --- | --- | --- | --- |
| `using-the-board` | Use when the user says "on the board", or something is better seen than read, and to pick a shape. | 전부 | none |
| `tracking-the-work` | Use when a task has more than a few steps, or the user asks "where are we". | checklist, board, log | none |
| `talking-while-working` | Use when the user may send instructions while you work, or the answer you need is a sentence. | chat | none |
| `drawing-a-diagram` | Use when you explain a structure, a flow, an order of calls or a schema. | note + mermaid | none |
| `planning-before-code` | Use when a change is not small and the user should read the plan before you write code. | note, checklist, form, book | none |
| `reporting-a-blocker` | Use when you are stuck, or the choice is between ways forward. | note, log, chat, checklist | none |
| `choosing-between-options` | Use when the user must choose and the options can be listed. | form, html | none |
| `approving-a-risky-action` | Use when an action is hard to undo: deploy, delete, push, migrate. | form, board, log, chat | none |
| `showing-numbers` | Use when you have numbers over time, a distribution or a progress rate. | chart, script | none |
| `handing-over-a-command` | Use when a command should run on the user's yes, not by you. | script, log | none |
| `checking-an-api` | Use when you changed or are learning an HTTP, WebSocket or gRPC API and want proof. | api (`.http`) | none |
| `reading-long-material` | Use when a report, a guide or a set of docs runs to many pages. | book, html | none (`#148` makes adding a page one step) |
| **`connecting-notes`** (new) | Use when one note should point to, show, count or react to another. | `[[links]]`, `![[embeds]]`, `from:`, `watch` | none |
| **`working-in-a-worktree`** (new) | Use when you work in a git worktree, or another agent in one is waiting for the user. | form, say, checklist | `#141`, `#144`; the overlap case also `#143` |
| **`resuming-and-handing-off`** (new) | Use when a session starts from what an earlier one left, or ends and someone else must go on. | checklist, log, board | none |
| **`finishing-the-task`** (new) | Use when you are done and the user will come back to see that it is. | checklist, log, note, form | none |

검토했지만 주제로 세우지 않은 둘.

- `adding-a-page-to-a-book`([#148](2026-11-23-worktree-issues/H-add-page-to-book.md))은 주제로 세우지 않는다. 이것만 중심에 둔 사례가 없다. 책 안에 쪽을 더하는 동작(`N`, `stickypane write <책>/<쪽>`)이어서, #148이 서면 `reading-long-material`에 한 줄로 들어간다. 그 전에는 `write <책>/<쪽>`(미확인)을 쓴다.
- 명령 문법([#149](2026-11-23-worktree-issues/I-cli-grammar.md))은 주제가 아니라 모든 주제의 명령 줄에 닿는 이름 바꿈이다(`todo plan add`가 `todo add plan`이 된다). 이것에만 기대는 사례는 없어서 아래 의존 목록에 가로 의존으로 적었다.

`using-the-board`에는 `topic` 사례가 없다. 중심은 #146의 "무엇을 보여 줄까, 어떤 모양" 표이고, 사례로는 예시 하나(`html-H4`)만 들어간다.

## 분류표

열: 키 | 이름 | 판정 | 주제 | 이유 | 의존. `none`은 오늘 된다는 뜻이다.

| key | name | verdict | topic | why | depends on |
| --- | --- | --- | --- | --- | --- |
| note-N1 | 구현 전 설계 한 장 | topic | planning-before-code | 채팅에서 밀리는 계획을 옆에 고정한다. Cursor plan mode가 같은 쓰임의 근거 | none |
| note-N2 | 아키텍처 그림이 있는 설명 | topic | drawing-a-diagram | mermaid가 그림으로 그려진다는 것이 보드의 고유한 이점 (`kinds note`) | none |
| note-N3 | ER 다이어그램으로 스키마 변경 | example | drawing-a-diagram | `erDiagram`이라는 다른 조리법. 되돌리기 어려운 변경 앞에서 | none |
| note-N4 | ADR 한 쪽 | example | choosing-between-options | 사용자가 고른 뒤에만 "Decided"를 쓴다는 규칙이 다른 점. 선택의 결과를 남기는 조리법 | none |
| note-N5 | PR 요약 초안 | example | finishing-the-task | "무엇을, 왜, 어떻게 확인, 손대지 않은 것" 네 칸. 끝낼 때 쓰는 조리법 | none |
| note-N6 | 막힘과 선택지 보고 | topic | reporting-a-blocker | 같은 실패 세 번째에 노트를 만든다. 긴 로그를 채팅에 붓지 않는다 | none |
| note-N7 | 사이퀀스 다이어그램으로 재현 경로 | example | drawing-a-diagram | `sequenceDiagram`이라는 다른 조리법. 가설과 확인을 표시한다 | none |
| note-N8 | 임베드 요약 허브 | topic | connecting-notes | `![[plan]]`이 임베드의 핵심 쓰임. 원본이 하나로 남는다 | none |
| note-N9 | 자기 리뷰 모음 | exclude | | 보드 기능을 쓰지 않는 평범한 노트고, 보고서가 스스로 "자기 리뷰는 맹점을 공유한다"고 적었다. 같은 내용을 `note-N5`의 "확신 낮은 곳"이 이미 한다 | none |
| note-N10 | 릴리스 노트 초안 | exclude | | `note-N5`와 같은 "초안을 사람이 고친다"이고, 보고서가 "근거는 내 추론", 문구 규칙과 충돌 가능이라 적었다. 릴리스는 사용자 소관(AGENTS.md) | none |
| book-B1 | 온보딩 가이드 | topic | reading-long-material | 쪽을 넘기며 한 권으로 읽는 가장 흔한 쓰임 (`shapes.md`) | none |
| book-B2 | 기존 docs를 복사 없이 책으로 | topic | reading-long-material | `stickypane link docs`는 실제 명령이고 `kinds book`의 예시다 | none |
| book-B3 | 장문 조사 보고서 | example | reading-long-material | 요약을 첫 쪽에 두는 쪽 구성이 다른 조리법 | none |
| book-B4 | 런북 | exclude | | 사용자가 따라 하는 절차는 `script-S1`, `script-S2`가 하고, 보고서가 "자동 생성 근거 못 찾음, 틀린 명령은 사고"라 적었다 | none |
| book-B5 | 인시던트 포스트모템 | example | reading-long-material | 외부 근거가 가장 많다(Datadog 등). 쪽마다 요약, 타임라인, 원인, 조치 | none |
| book-B6 | 모듈별 설명서 | exclude | | `book-B1`과 같은 "저장소의 지도"이고 구조가 바뀌면 낡는다(보고서의 한계). 쪽 이름만 다르다 | none |
| book-B7 | 마이그레이션 계획과 진행 | example | planning-before-code | 계획은 쪽, 진행은 `[[링크]]`된 체크리스트라는 짝이 다른 조리법 | none (쪽 더하기는 `#148`로 한 걸음) |
| book-B8 | 혼합 형식 책(note+html) | example | reading-long-material | 쪽마다 모양이 달라도 된다(`shapes.md`). 터미널에서는 html 쪽이 글로만 읽히니 핵심 숫자는 글로 | none |
| book-B9 | 테스트 실패 분류집 | exclude | | 분류 표 하나면 노트 한 장으로 충분해 책이 필요 없고, 근거는 "내 추론"이다 | none |
| book-B10 | 학습 노트 | exclude | | `book-B1`과 같은 모양이고 독자만 다르다. 사용자가 시킨 일이 아니고 근거는 "내 추론" | none |
| html-H1 | 비교표가 있는 선택지 보고서 | topic | choosing-between-options | Claude Code 팀의 HTML 출력 권고와 맞고 터미널에서도 글로 읽혀 약점이 작다 | none |
| html-H2 | 벤치마크·성능 리포트 | example | showing-numbers | 숫자가 여러 개인 한 장의 표라는 점이 chart와 다르다. 반복 횟수와 편차를 함께 적는다 | none |
| html-H3 | 구현 전 계획 리뷰 페이지 | exclude | | `note-N1`과 같은 목적이고 보고서가 "편집·커밋할 문서면 note"라고 정했다. html은 읽기 전용이라 그 자리에서 고칠 수 없다 | none |
| html-H4 | 커버리지·테스트 결과 대시보드 | example | using-the-board | 도구가 만든 html을 `stickypane show <파일>`로 그대로 올리는 조리법. 새 문서를 쓰지 않는다 | none |
| html-H5 | UI 변경 전후 | exclude | | 터미널은 이미지를 못 보이고(보고서의 한계) 근거가 없다. 미리보기는 `html-H4`와 같은 `show` | none |
| html-H6 | 인터랙티브 설명서 | exclude | | 앱이 스크립트를 실제로 실행하는지 확인하지 못했다고 보고서가 적었고, 외부 스크립트는 보안 위험이다 | none |
| html-H7 | 의존성 그래프 html | exclude | | 도구가 만든 html을 `show`하는 `html-H4`와 같은 조리법 | none |
| html-H8 | 로그 분석 요약 페이지 | exclude | | 로그에서 핵심 줄만 남기는 `log-L2`와 같은 일이고 근거는 "내 추론" | none |
| html-H9 | 사용자 대상 산출물 미리보기 | exclude | | `stickypane show landing.html` 한 줄이라 `html-H4`와 같은 조리법 | none |
| html-H10 | 비용·일정 리포트 | exclude | | 위로 전달할 문서는 보드의 일이 아니다. 보고서가 "저장소에 남는 문서는 Markdown이 낫다"고 반론을 적었다 | none |
| checklist-C1 | 작업 시작 전 계획 공개 | topic | tracking-the-work | `todo add`를 단계마다 쓰고 코드를 건드리기 전에 사용자가 본다 | none |
| checklist-C2 | 끝낸 뒤에만 체크 | topic | tracking-the-work | 체크하면 보드가 `✅ 시각`을 쓴다. 시작할 때가 아니라 끝났을 때 | none |
| checklist-C3 | 완료 기준 체크리스트 | topic | finishing-the-task | 끝내기 전 빠진 항목이 눈에 띈다. 채점 순환의 위험은 항목마다 명령 증거를 붙여 막는다 | none |
| checklist-C4 | 사용자가 체크하는 사람 몫 단계 | topic | connecting-notes | `watch --once --type item.ticked`로 사람 단계를 추측하지 않고 기다린다. 타임아웃 필요 | none |
| checklist-C5 | 계획 변경 흔적 | example | tracking-the-work | 항목을 지우지 않고 새 줄과 `say`로 알리는 조리법 | none |
| checklist-C6 | 병렬 서브에이전트의 합류 지점 | example | tracking-the-work | 합류 항목을 마지막에 두는 조리법. 보고서가 "내 추론" | none |
| checklist-C7 | 막힌 일 표시와 우회 | example | reporting-a-blocker | `BLOCKED:` 항목과 `say` 질문의 짝. 막대가 오해를 부를 수 있다는 한계를 밝힌다 | none |
| checklist-C8 | 재개용 체크포인트 | topic | resuming-and-handing-off | 파일이 대화 밖 기억이다. 보고서가 말한 `stickypane todo plan`(동사 없이 현황)은 없어서 `stickypane cat plan`으로 읽는다 | none |
| checklist-C9 | 사용자가 던진 할 일 큐 | example | talking-while-working | 사용자가 보드에서 줄을 더하면 단계 사이에 `watch --once`로 본다. `note.changed` 이벤트 이름은 `help watch`의 목록에 보이지 않아 확인이 필요하다 | none |
| checklist-C10 | 증명 가능한 마무리 정리 | topic | finishing-the-task | 모두 체크하고 `log --time`으로 한 줄. 돌아온 사용자가 10초에 안다 | none |
| board-B1 | 에이전트 작업 레인 | example | tracking-the-work | 카드 끝 `@codex-1`으로 누가 쥐었는지 보이는 board의 조리법 | none |
| board-B2 | 검토 칸이 승인 게이트 | example | approving-a-risky-action | 카드 이동을 승인으로 쓰는 조리법. 에이전트가 스스로 옮기는 것을 막지 못한다는 한계를 밝힌다 | none |
| board-B3 | 정체 카드 감지 | exclude | | 보드가 알아서 표시하는 것이라 에이전트가 할 일이 없다(`kinds board`) | none |
| board-B4 | 후보 안 비교 카드 | exclude | | 보고서가 "선택은 form이 더 명확"이라 적었고 `form-F5`가 같은 일을 한다 | none |
| board-B5 | 이슈 분류 대기열 | exclude | | 에이전트가 드물게 하고 근거는 "내 추론"이며, `card add --to`를 칸 이름만 바꿔 쓰는 `board-B1`의 조리법이다 | none |
| board-B6 | 사용자의 우선순위 재배열 | example | talking-while-working | 에이전트가 To do의 순서를 읽고 다음 카드를 고른다. 사용자가 조종하는 조리법 | none |
| board-B7 | 서브에이전트 인계 | example | resuming-and-handing-off | 카드를 칸 경계로 넘기고 재현 명령을 카드에 적는다 | none |
| board-B8 | 위험 작업 분리 칸 | exclude | | `board-B2`의 사람 게이트와 같고, 등급을 에이전트가 정해 낮출 수 있다고 보고서가 한계로 적었다 | none |
| board-B9 | 변경 가능한 로드맵 | exclude | | 장기 계획은 작업 관리자가 아니라는 VISION에 닿는다. 근거가 "내 추론" | none |
| board-B10 | 실패한 카드의 역방향 이동 | exclude | | `card move --to Doing` 한 줄은 `board-B1`에 이미 있다. 쓰임이 에이전트의 신뢰도 계측이라 행동 지침이 아니다 | none |
| chat-T1 | 방향 전환 알림 | example | talking-while-working | 계획이 바뀐 이유만 `say`로 말한다. "모든 단계는 말하지 말라"는 한계를 함께 적는다 | none |
| chat-T2 | 비차단 지시 수신 | topic | talking-while-working | `watch --once --note chat --type message.added --timeout 1s`로 단계 사이마다 확인 | none |
| chat-T3 | 한 문장 답이 필요한 질문 | topic | talking-while-working | 선택이면 form, 문장이면 chat. 시간 초과(코드 3)에서 추측하지 않는다 | none |
| chat-T4 | 장시간 작업 중 짧은 상태 알림 | topic | talking-while-working | 시작, 변경, 끝의 세 점만. 알림이 쌓여 기록이 된다 | none |
| chat-T5 | 에이전트 대 사용자 이견 | example | approving-a-risky-action | 사용자의 지시가 위험하면 `say`로 묻고 form으로 올린다 | none |
| chat-T6 | 모호한 요청의 되묻기 | exclude | | `chat-T3`과 같은 `say` + `watch`이고 질문의 내용만 다르다 | none |
| chat-T7 | 여러 에이전트의 서명 구분 | example | talking-while-working | `say --as <이름>`(`help say`)이라는 실제 옵션. 세션 내내 같은 이름 | none |
| chat-T8 | 부재 중 안전 경로 후 보고 | example | approving-a-risky-action | 안전 경로를 택한 뒤 `say`로 사후에 알리는 부분이 `form-F7`(로그 한 줄)과 다르다 | none |
| chat-T9 | 거절 이유 기록 | exclude | | 사용자 답을 읽는 방법은 `chat-T2`, `chat-T3`과 같고 "같은 제안을 되풀이하지 않는다"는 행동 규칙이지 명령이 아니다 | none |
| chat-T10 | 인계 메모 | exclude | | `log-L7`과 같은 일을 `say`로 한 것이다. 로그가 세션을 넘어 남는다 | none |
| form-F1 | 배포 목적지 선택 | topic | approving-a-risky-action | 되돌리기 어려운 행동 앞의 정확한 정지점. 위험한 쪽을 기본값으로 두지 않는다 | none |
| form-F2 | 계획 승인 게이트 | topic | planning-before-code | `[ Approve ] [ Revise ]`와 의견 줄. 요약을 짧게 | none |
| form-F3 | 파괴적 명령 사전 확인 | topic | approving-a-risky-action | 명령이 아니라 효과(무엇이 사라지나)를 본문에 보인다 | none |
| form-F4 | 여러 항목 일괄 승인 | example | approving-a-risky-action | 체크박스 여러 줄과 `[ Apply selected ]`. 승인 피로를 줄이는 조리법 | none |
| form-F5 | 모호한 요구의 분기 질문 | topic | choosing-between-options | 라디오와 옵션별 설명, `> Other` 입력 줄 | none |
| form-F6 | 비밀 값은 폼에 쓰지 않는다 | example | approving-a-risky-action | 폼 답이 평문 `.sticky/*.md`에 남으므로 "환경 변수에 넣었으면 [ Done ]" 버튼만. 안전 규칙 | none |
| form-F7 | 시간 초과 시 안전 경로 | topic | approving-a-risky-action | `wait`의 종료 코드 3은 no. 로그에 한 줄 남긴다 | none |
| form-F8 | 오래된 답 재사용 방지 | topic | approving-a-risky-action | `submitted`가 남은 폼은 옛 답을 돌려준다. 새 승인은 새 폼. 보고서의 계측이 말한 키는 `wait`의 출력에서는 `at:`이고 `submitted_at`은 없다 | none |
| form-F9 | 사전 입력 폼 | example | choosing-between-options | 시작 전에 선호를 한 번에 모으는 조리법. 질문은 다섯 개 이하 | none |
| form-F10 | 검토 판정(approve / request changes) | example | finishing-the-task | `![[diff]]`로 변경 요약을 품은 판정 폼. 이름이 `diff`인 노트는 에이전트가 직접 쓰면 된다 | none (`#143`의 diff 책이 생기면 그것을 임베드할 수 있다) |
| log-L1 | 작업 일지 | topic | tracking-the-work | `log worklog --time "..."`는 `tracking-progress` 스킬의 예시 그대로고 가장 흔하다 | none |
| log-L2 | 실패한 명령의 마지막 줄 | example | reporting-a-blocker | 전체 출력 대신 원인에 가까운 줄과 원문 경로만 남기는 조리법 | none |
| log-L3 | 결정 기록 | exclude | | `note-N4`와 같은 결정 기록이고, 사용자 몫의 결정을 로그에만 쓰는 월권 위험이 크다(보고서의 한계) | none |
| log-L4 | 막힘과 재시도 로그 | exclude | | `log-L1`과 같은 `log --time`이고 문구만 다르다. 루프 중이면 로그도 폭주한다고 보고서가 적었다 | none |
| log-L5 | 도구 호출 감사 로그 | example | approving-a-risky-action | 파괴적 동작 직전에 `log audit --time`으로 남기는 조리법. 스스로 쓰는 기록이라 강제는 hook의 몫(보고서) | none |
| log-L6 | 사용자 지시 수신 확인 | exclude | | 지시를 읽는 것은 `chat-T2`, `card.moved` 확인은 `board-B6`에 있고, 확인 줄은 중복 소음이다 | none |
| log-L7 | 세션 인수인계 메모 | topic | resuming-and-handing-off | 파일이라 세션을 넘어 남는다. 날짜와 브랜치를 적는다 | none |
| log-L8 | 하트비트 | exclude | | `log-L1`과 같은 명령이고 빈도만 다르다. 빈도 기준이 없어 소음이 된다(보고서) | none |
| log-L9 | 스크립트 결과 해석 줄 | example | handing-over-a-command | 사용자가 돌린 스크립트의 `.log`를 읽고 해석 줄을 덧붙인다. 읽기 전에 쓰면 거짓 | none |
| log-L10 | 평가 실행 결과 로그 | exclude | | 에이전트 쓰임이 아니라 평가 하네스의 일이다(`#145`). 보고서도 하네스가 쓴 줄과 섞지 말라고 했다 | none |
| chart-C1 | 토큰 사용량 막대 | exclude | | 에이전트는 정확한 토큰 수를 모른다(보고서). 추정치를 숫자로 보이면 거짓 정밀도다 | 에이전트가 읽을 수 있는 토큰 수 출처. 계획에 없다 |
| chart-C2 | 일별 토큰 히트맵 | exclude | | `chart-C1`과 같은 자료 문제이고, 날짜 라벨을 에이전트가 손으로 쓰면 시각 규칙과 부딪힐 수 있다고 보고서가 적었다 | 위와 같다 |
| chart-C3 | 세션 비용 추세 | exclude | | 구독제 비용은 청구액이 아니고(보고서), 에이전트가 비용을 알 수 없다 | 위와 같다 |
| chart-C4 | 테스트 통과/실패 막대 | topic | showing-numbers | `chart tests set pass 41`로 최신 숫자를 같은 자리에 둔다. `set`은 이전 값을 지운다 | none |
| chart-C5 | flaky 비율 추세 | example | showing-numbers | 실행마다 `chart add`로 쌓고 `view: spark`. 표본 수를 함께 적는 조리법 | none |
| chart-C6 | 벤치마크 값 추세 | exclude | | `chart-C5`와 같은 `chart add`+spark이고 벤치 실행은 `script-S4`가 한다 | none |
| chart-C7 | 체크리스트 진행률(`from:`) | topic | showing-numbers | 에이전트가 세지 않아 틀릴 수 없다. 실제 지원되는 기능 (`connecting-notes` 스킬) | none |
| chart-C8 | 작업 종류별 시간 분포 | exclude | | 범주를 에이전트가 임의로 붙이면 일관성이 없다고 보고서가 적었고, 근거가 "내 추론"이다 | none |
| chart-C9 | 도구 호출 성공/실패 막대 | exclude | | 자가 계수는 부정확하다고 보고서가 적었다. 대안인 hook 집계는 만들어져 있지 않다 | 도구 호출을 세는 hook. 계획에 없다 |
| chart-C10 | 보드 활동 히트맵 | exclude | | `from:`이 날짜별로 나눠 세는지 확인하지 못했다고 보고서가 적었다 | 날짜별 `from:` 집계. 계획에 없다 |
| script-S1 | 배포 런북 | topic | handing-over-a-command | "읽고 승인하고 결과 확인"이 한 화면. 실행은 사용자의 Run (`kinds script`) | none |
| script-S2 | 드라이런 후 실제 실행 | example | handing-over-a-command | 드라이런 `.log`를 먼저 보게 하는 두 파일 조리법 | none |
| script-S3 | 테스트 반복 실행기 | exclude | | 읽기 전용이라 에이전트가 직접 돌려도 안전하다. 승인을 요구하면 피로만 쌓인다고 보고서가 적었다 | none |
| script-S4 | 벤치 실행 + 차트 갱신 | example | showing-numbers | 스크립트가 `chart set <커밋> <ms>`를 쓰는 조리법. `chart-C6`을 대신한다 | none |
| script-S5 | 정리/삭제 스크립트 | example | handing-over-a-command | 삭제 경로를 고정하고 와일드카드와 `rm -rf $VAR`를 금지하는 안전 규칙 | none |
| script-S6 | 환경 점검 스크립트 | exclude | | 읽기 전용이라 승인 클릭이 마찰이다(보고서의 한계). 에이전트가 직접 돌리면 된다 | none |
| script-S7 | 릴리스 체크 스크립트 | example | handing-over-a-command | `precheck.sh`에 태그와 푸시를 넣지 않는다. 이 저장소의 "Before you finish"와 맞다 | none |
| script-S8 | 재현 스크립트 | example | handing-over-a-command | `repro.sh`를 사용자가 눌러 확인한다. 말이 아니라 재현 | none |
| script-S9 | 서버 기동 스크립트 | exclude | | 근거가 "내 추론"이고 `script-S1`과 같은 모양이다. 오래 사는 프로세스의 `.log`가 계속 커진다 | none |
| script-S10 | 폼 후 스크립트 | example | handing-over-a-command | 폼 제출을 `watch --once --type form.submitted`로 기다린 뒤 스크립트를 올린다. 이중 승인이라 큰 일에만 | none |
| api-A1 | 구현 직후 스모크 | topic | checking-an-api | `stickypane api api --all`의 `[201 Created · 41ms · … 2/2 ✔]`가 사용자가 재현할 수 있는 증거 | none |
| api-A2 | 계약 먼저, 구현 나중 | example | checking-an-api | 구현 전 `0/n`, 끝나면 `n/n ✔`. assert가 못 잡는 계약이 있다는 한계 | none |
| api-A3 | 인증 흐름 | example | checking-an-api | `@capture`와 `env:`로 토큰이 채팅에 남지 않는다. `capture file`이 디스크에 남기는지는 확인하지 못했다 | none |
| api-A4 | 실행되는 API 문서 | exclude | | `api-A1`, `api-A2`와 같은 `.http` 파일이고 의도만 다르다. 설명 칸이 없어 산문이 따로 필요하다(보고서) | none |
| api-A5 | 변경 전후 회귀 비교 | exclude | | 응답 전체를 비교하는 기능이 없고 assert뿐이라고 보고서가 적었다. `--env`만 바꾼 `api-A1` | none |
| api-A6 | WebSocket 구독 확인 | example | checking-an-api | `@websocket`/`@ws` 단계라는 다른 문법. timeout 기반이라 장기 연결은 못 본다 | none |
| api-A7 | gRPC 탐색에서 요청까지 | example | checking-an-api | `GRPC host:port`와 `@grpc` 문법. 서비스 목록 기능이 없어 이름을 알아야 한다 | none |
| api-A8 | 사용자가 누르는 위험한 요청 | example | checking-an-api | 위험한 요청은 쓰기만 하고 보내지 않는다. `stickypane api <file> <요청>`은 묻지 않고 보내 가드가 없다(이슈 #146 열린 질문 8) | none |
| api-A9 | 재현 요청을 증거로 고정 | exclude | | `script-S8`과 같은 "사용자가 누르는 재현 파일"이고 `.http` 버전이다. 로그에 개인 정보가 남을 수 있다 | none |
| api-A10 | 외부 API 학습 검증 | example | checking-an-api | 실제 응답을 보고 assert를 써 넣어 추측을 관찰로 바꾼다. 상태를 바꾸는 요청은 `api-A8`처럼 | none |
| worktree-B1 | 새 worktree에서 첫 명령 | topic | working-in-a-worktree | 에이전트가 자기 폴더를 한 줄로 안다(`worktrees/<이름>/`). 지금은 `.sticky/`가 gitignore라 새 worktree에 보드가 없다 | #141 (폴더와 보드 찾기), #144 (훅 한 줄) |
| worktree-B2 | 서브에이전트의 임시 worktree | exclude | | 에이전트가 할 일이 아니라 `#141`의 설계 결정(폴더는 첫 쓰기에서 만든다)이 이미 푼다 | none |
| worktree-B3 | 어느 에이전트가 사용자를 기다리는가 | example | working-in-a-worktree | 답 없는 form과 `say` 질문을 워크트리 폴더에 두면 보드가 `waiting`으로 센다. 에이전트는 평소처럼 form을 쓰면 된다 | #141 (`waiting` 계산) |
| worktree-B4 | 같은 파일을 두 worktree가 고친다 | example | working-in-a-worktree | `git diff --name-only <base>...HEAD`의 겹침 경고. 에이전트는 겹침을 보면 `say`로 알린다 | #143 1단계 (겹침 경고) |
| worktree-B5 | 병합 순서 | exclude | | 병합은 stickypane의 일이 아니다(설계 비목표). `#143` 1단계에 PR 한 줄이 있을 뿐 병합 순서는 어느 이슈에도 없다 | 계획에 없다 |
| worktree-B6 | 기준 branch가 다르다 | exclude | | `#143`의 기준(base) 규칙이 이미 다루고(`worktree.baseRef: head`) 에이전트가 할 일이 없다 | none |
| worktree-B7 | 지워졌다 돌아온 worktree | exclude | | 옛 체크리스트를 보관 표시하는 규칙은 "검토할 만하다"에 그치고 어느 이슈에도 없다 | 계획에 없다 |
| worktree-B8 | 에이전트가 쓰는 위치를 아는 법 | exclude | | `worktree-B1`의 답이 `#144`이고 `cwd`로 판단하라는 것은 구현 세부(R4)다. 에이전트 행동이 아니다 | none |
| worktree-B9 | 실행 중인지 죽었는지 | exclude | | `#141`이 `locked`로 `running`을 계산한다. 에이전트가 쓸 것이 없다(Claude Code에서만 확인) | none |
| worktree-B10 | 공유 런타임 상태, 의존성, 디스크 | exclude | | 보고서가 "stickypane의 일이 아니다"라고 적었다. 남는 것은 `docs/shapes.md`의 한 줄(환경 파일을 워크트리 폴더에) | none |

## 주제별로 싣는 사례

고딕은 `topic`이고 나머지는 `example`이다.

| 주제 | 사례 |
| --- | --- |
| using-the-board | html-H4 |
| tracking-the-work | **checklist-C1**, **checklist-C2**, **log-L1**, checklist-C5, checklist-C6, board-B1 |
| talking-while-working | **chat-T2**, **chat-T3**, **chat-T4**, chat-T1, chat-T7, checklist-C9, board-B6 |
| drawing-a-diagram | **note-N2**, note-N3, note-N7 |
| planning-before-code | **note-N1**, **form-F2**, book-B7 |
| reporting-a-blocker | **note-N6**, checklist-C7, log-L2 |
| choosing-between-options | **html-H1**, **form-F5**, form-F9, note-N4 |
| approving-a-risky-action | **form-F1**, **form-F3**, **form-F7**, **form-F8**, form-F4, form-F6, board-B2, chat-T5, chat-T8, log-L5 |
| showing-numbers | **chart-C4**, **chart-C7**, chart-C5, html-H2, script-S4 |
| handing-over-a-command | **script-S1**, script-S2, script-S5, script-S7, script-S8, script-S10, log-L9 |
| checking-an-api | **api-A1**, api-A2, api-A3, api-A6, api-A7, api-A8, api-A10 |
| reading-long-material | **book-B1**, **book-B2**, book-B3, book-B5, book-B8 |
| connecting-notes | **note-N8**, **checklist-C4** |
| working-in-a-worktree | **worktree-B1**, worktree-B3, worktree-B4 |
| resuming-and-handing-off | **checklist-C8**, **log-L7**, board-B7 |
| finishing-the-task | **checklist-C3**, **checklist-C10**, note-N5, form-F10 |

`exclude`한 47개는 위 표의 `exclude` 줄이 곧 목록이다.

## 아직 안 지어진 기능에 기대는 사례

가이드에는 기능이 서야 넣는다. 기능이 없어도 지금 가이드에 실을 수 있는 사례는 위 표에서 `none`인 것이다.

| 기능 | 사례 | 어떻게 기대는가 |
| --- | --- | --- |
| #141 워크트리 추적 | worktree-B1, worktree-B3 | 워크트리 폴더와 `waiting` 계산. 이것 없이는 새 worktree에 보드가 없다 |
| #144 훅 한 줄 | worktree-B1 | 에이전트가 자기 폴더를 세션 시작에 한 줄로 안다 |
| #143 diff 보기 | worktree-B4 | 겹침 경고(1단계). `form-F10`의 `![[diff]]`는 지금도 직접 쓴 노트로 되고, 생기면 그것을 임베드할 수 있다(선택) |
| #148 책에 쪽 더하기 | book-B1, book-B3, book-B5, book-B7, book-B8 (쪽을 더하는 동작만) | 막지는 않는다. 지금은 `stickypane write <책>/<쪽>`이 책에도 되는지 실행으로 확인해야 한다(미확인) |
| #149 명령 문법 | 모든 사례의 명령 줄 | `todo plan add`가 `todo add plan`으로 바뀐다. 사례를 막지 않지만 가이드에 실을 때 새 문법으로 쓴다 |
| #145 효과 측정 | (없음) | 사례가 기대지 않는다. 주제를 가이드에 넣은 효과를 재는 도구다. `log-L10`은 이쪽의 일이라 뺐다 |
| #142 쪽 넘김, #147 데스크톱 | (없음) | 기대는 사례가 없다. html은 앱이 이미 그린다 |

계획에 없는 기능이 필요해 뺀 사례(가이드에 못 넣는다, 나중에 기능이 생기면 다시 본다): chart-C1, chart-C2, chart-C3(에이전트가 읽을 수 있는 토큰·비용 수), chart-C9(도구 호출 hook), chart-C10(날짜별 `from:`), worktree-B5(병합 순서), worktree-B7(다시 열린 폴더의 보관 규칙).

## 개수 확인

아래 숫자는 스크립트가 이 파일의 표에서 센 값이다(검증 명령은 보고에 적었다).

| 판정 | 개수 |
| --- | --- |
| topic | 29 |
| example | 44 |
| exclude | 47 |
| 합 | 120 |

사례 120 = topic 29 + example 44 + exclude 47.
