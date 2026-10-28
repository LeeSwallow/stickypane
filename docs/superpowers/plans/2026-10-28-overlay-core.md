# Overlay Core (본체) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** stickypane 본체가 다른 화면에 board를 여는 데 필요한 모든 것 — `Kind.Data`·`Kind.Act`, `html` shape, `api.Apply`, `~/.stickypane` 레지스트리, `stickypane bridge` — 을 표준 라이브러리만으로 만든다.

**Architecture:** 각 shape가 `Data`(파일이 말하는 것)와 `Act`(화면의 행동 → 기존 Op)를 내놓고, `api.Apply`가 버전을 확인한 뒤 `store.Apply`로 쓴다. 새 front end `internal/bridge`(layer 50)가 이를 newline-delimited JSON으로 stdio에 연다. `internal/registry`가 `~/.stickypane/events/*.jsonl`과 `run/<pid>.json`을 best-effort로 쓴다.

**Tech Stack:** Go 1.27, 표준 라이브러리만. 새 모듈 없음.

**Spec:** `docs/overlay/` (특히 02 §2.4–2.7, 08 §8.4, ADR-0002·0004·0005·0006·0008·0013). 이슈 #10.

## Global Constraints

- 매 Task 끝: `go vet ./... && go test -race ./... && test -z "$(gofmt -l .)"` 통과 (AGENTS.md).
- 패키지는 낮은 레이어만 import한다. 새 패키지는 `internal/layers` 표에 rank와 `docs/architecture.md`에 한 줄.
- 노트 변경은 모두 `doc.Op`를 `store.Apply`로. 파일을 직접 쓰지 않는다.
- 시각은 `internal/when`으로만 쓴다.
- 새 Go 모듈을 추가하지 않는다.
- 커밋은 Conventional Commits, 하나에 한 변경, `(#10)` 참조, AI 도구 이름을 넣지 않는다. push하지 않는다.
- 레지스트리 쓰기 실패는 어떤 명령도 실패시키지 않는다.
- bridge는 네트워크 listener를 열지 않는다.
- 주석·이름은 주변 코드의 문체(완전한 영어 문장, 무엇을 왜)를 따른다.

## Review Focus

1. **동시에 바뀐 파일에 대한 행동** — 화면이 본 뒤 에이전트가 같은 노트를 바꿨으면 `api.Apply`는 `ErrStale`로 거절하고 파일을 건드리지 않는다. (Task 8 `TestApplyRefusesAStaleVersion`)
2. **여러 프로세스의 동시 레지스트리 쓰기** — 줄이 섞이거나 잘리지 않는다. (Task 10 `TestAppendFromManyProcessesKeepsWholeLines`)
3. **쓸 수 없는 홈 폴더** — `HOME`이 읽기 전용이거나 없을 때 모든 명령이 그대로 성공한다. (Task 11 `TestCommandsSucceedWhenTheHomeIsReadOnly`)
4. **bridge 입력이 깨진 경우** — 잘못된 JSON 줄, 모르는 method, 없는 노트에 `error` 응답을 보내고 계속 산다. (Task 13 `TestBridgeAnswersBadInputAndKeepsRunning`)
5. **위치로 가리킨 항목이 범위 밖** — `Act`가 오류를 돌려주고 Op를 만들지 않는다. (Task 3·4·5의 `...OutOfRange` 테스트)

---

### Task 1: The contract — `Action`, `Data`, `Act`, `Version`

**Files:**
- Modify: `internal/widget/widget.go` (Kind에 두 필드, 새 타입)
- Create: `internal/widget/version.go`
- Modify: `internal/kinds/contract_test.go`
- Test: `internal/widget/version_test.go`

**Interfaces:**
- Produces:
  - `type Action struct { Name string \`json:"name"\`; Args map[string]string \`json:"args,omitempty"\` }`
  - `Kind.Data func(d doc.Document) any` — 문서: "what a note of this kind says, as a value that encodes to JSON".
  - `Kind.Act func(d doc.Document, a Action) (doc.Op, error)`
  - `func Version(d doc.Document) string` — `sha256(d.Bytes())`의 앞 16 hex.
  - `var ErrUnknownAction = errors.New("unknown action")`

- [ ] **Step 1: Write the failing test** — `version_test.go`: `TestVersionChangesWithContent` — 같은 문서 두 번은 같은 값, 본문 한 글자 다르면 다른 값, 길이 16.
- [ ] **Step 2: Run** `go test ./internal/widget/` — Expected: FAIL (Version undefined)
- [ ] **Step 3: Implement** 타입·필드·`Version`. 모든 kind가 `Data`를 갖췄는지 보는 contract 검사는 Task 7에서 더한다(그때 모든 shape가 갖춘다).
- [ ] **Step 4: Run** `go test ./internal/...` — Expected: PASS
- [ ] **Step 5: Commit** `feat(widget): a shape can say what it holds and turn a screen's action into an op (#10)`

### Task 2: `Data` for the shapes that are only read

**Files:**
- Modify: `internal/widget/note/note.go`, `internal/widget/chart/model.go`, `internal/widget/logview/*.go`, `internal/widget/chat/model.go`, `internal/widget/script/*.go`, `internal/widget/api/*.go`, 각 `kind.go`
- Test: 각 패키지 `data_test.go`

**Interfaces:**
- Produces (JSON 필드 이름 고정 — 앱이 이 이름을 쓴다):
  - note: `{"markdown": string}`
  - chart: `{"view": "bar"|"spark"|"heat", "prose": []string, "points": [{"label","text","value": number|null}]}`
  - logview: `{"lines": []string}` (마지막 1000줄)
  - chat: `{"messages": [{"day","who","clock","text"}]}`
  - script: `{"source": string, "lang": "sh"|"ps1"}`
  - api: `{"source": string}` (요청 파일 원문. 구조화는 나중)

- [ ] **Step 1: Write failing tests** — 각 kind의 `Example`로 `TestDataOfTheExample`: `json.Marshal(Kind.Data(d))`가 기대 JSON 문자열과 같다(테스트 안에 리터럴로).
- [ ] **Step 2: Run** `go test ./internal/widget/...` — Expected: FAIL
- [ ] **Step 3: Implement** — 기존 `read()`/parse 결과를 export용 struct(태그 포함)로 옮긴다. 새 파싱 로직을 쓰지 않는다.
- [ ] **Step 4: Run** — Expected: PASS
- [ ] **Step 5: Commit** `feat(widget): read-only shapes say what they hold (#10)`

### Task 3: checklist `Data` and `Act`

**Files:** Modify `internal/widget/checklist/model.go` (Data), `ops.go` (Act), `kind.go`; Test `internal/widget/checklist/data_test.go`

**Interfaces:**
- Produces: Data `{"items":[{"text","checked","done"}],"prose":[]string,"done":int,"total":int}` (`done`은 RFC 3339 또는 생략). Act: `toggle {i}` → 기존 `Toggle`, `add {text}` → `AddItem`.

- [ ] **Step 1: Failing tests** — `TestActToggleTicksTheItem`(i=1 토글 후 `Data.items[1].checked==true`), `TestActToggleOutOfRange`(i=99 → error, Op nil), `TestActAddAppends`.
- [ ] **Step 2: Run** `go test ./internal/widget/checklist/` — FAIL
- [ ] **Step 3: Implement** — `Act`는 Args를 `strconv.Atoi`로 읽고 범위를 검사한 뒤 기존 Op를 돌려준다. 완료 시각은 쓰지 않는다(`Tend`가 쓴다).
- [ ] **Step 4: Run** — PASS
- [ ] **Step 5: Commit** `feat(checklist): open the list to other screens (#10)`

### Task 4: board `Data` and `Act`

**Files:** Modify `internal/widget/board/model.go`, `ops.go`, `kind.go`; Test `internal/widget/board/data_test.go`

**Interfaces:**
- Produces: Data `{"desc":[]string,"columns":[{"title","cards":[{"text","who":[]string,"moved","detail":[]string,"stalled":bool}]}]}` — `stalled`은 기존 view가 쓰는 판정(마지막 열 아님 + 30분)을 `widget.Now()`로 계산하는 함수를 view와 공유한다. Act: `move {col,i,to,at}` → `MoveCard`, `add {col,text}` → `AddCard`.

- [ ] **Step 1: Failing tests** — `TestActMoveCardToAnotherColumn`(Example에서 Doing의 0번을 Done의 0번 자리로 → Data에서 확인), `TestActMoveOutOfRange`, `TestDataMarksAStalledCard`(`widget.Now`를 고정).
- [ ] **Step 2: Run** — FAIL
- [ ] **Step 3: Implement** — 정체 판정을 `stalled(c card, last bool, now time.Time) bool`로 꺼내 view와 Data가 함께 쓴다.
- [ ] **Step 4: Run** — PASS
- [ ] **Step 5: Commit** `feat(board): open the board to other screens (#10)`

### Task 5: form `Data` and `Act`

**Files:** Modify `internal/widget/form/model.go`, `ops.go`, `kind.go`; Test `internal/widget/form/data_test.go`

**Interfaces:**
- Produces: Data `{"lines":[{"kind":"text"|"choice"|"check"|"field"|"button","text","group","checked","value"}],"submitted":string,"at":string}` — 기존 `line`·`control` 구조에서 그대로. Act: `choose {line}` → `Choose`, `fill {line,text}` → `Fill`, `submit {label}` → `Submit{Label, At: when.Format(widget.Now())}`.

- [ ] **Step 1: Failing tests** — `TestActChooseThenSubmit`(Answers가 `api.Answers`와 같은 결과), `TestActChooseOutOfRange`, `TestActOnASubmittedFormStillWorks`(기존 `unsubmit` 규칙을 따른다).
- [ ] **Step 2–4:** FAIL → 구현 → PASS
- [ ] **Step 5: Commit** `feat(form): open forms to other screens (#10)`

### Task 6: chat `Act`

**Files:** Modify `internal/widget/chat/ops.go`, `kind.go`; Test `internal/widget/chat/data_test.go`

- [ ] **Step 1: Failing test** — `TestActSayAppendsAMessageFromTheUser`: `say {text}` → 메시지가 붙고 `who`는 `Args["who"]`(앱은 `"you"`를 보낸다), 빈 text는 오류.
- [ ] **Step 2–4:** FAIL → 기존 `api.Say`가 쓰는 Op를 재사용 → PASS
- [ ] **Step 5: Commit** `feat(chat): a screen can post a message (#10)`

### Task 7: the `html` shape

**Files:**
- Create: `internal/widget/html/{kind.go,format.go,model.go,ops.go,view.go,events.go}`, `html_test.go`
- Modify: `internal/kinds/kinds.go` (한 줄), `internal/layers/layers_test.go` (`"internal/widget/html": 22`), `docs/architecture.md` 표, `internal/kinds/contract_test.go` (Task 1의 추가분)

**Interfaces:**
- Produces: `Kind{Name:"html", Exts:[".html",".htm"], New:".html", Icon:"⌗", Label:"HTML (view only)"}`; Data `{"html": string}`; Act 없음; Events 없음(빈 함수 아님 — 필드 nil).
- view: 태그·`<script>`·`<style>` 내용을 걷어낸 텍스트를 감싸서 그리고, 마지막 줄 `open it in the overlay to see it drawn`(i18n 키 추가, ko 번역 포함).

- [ ] **Step 1: Failing tests** — `TestHTMLViewShowsTextWithoutTags`(`<h1>Bench</h1><script>x()</script><p>a &amp; b</p>` → `Bench`, `a & b`, `x()` 없음), `TestHTMLDataIsTheSource`, 그리고 `internal/kinds/contract_test.go`에 추가: 모든 kind는 `Data != nil`이고 `Example`로 만든 문서의 `Data`가 `json.Marshal`에 성공하며 `null`이 아니다. `Act`가 있는 kind는 `Act(d, Action{Name: "no-such"})`가 `errors.Is(err, widget.ErrUnknownAction)`.
- [ ] **Step 2: Run** `go test ./internal/...` — FAIL
- [ ] **Step 3: Implement** — 태그 제거는 `golang.org/x/net/html` 없이 표준 라이브러리 `html.UnescapeString` + 작은 상태 기계(태그 안/밖, script·style 블록 건너뛰기).
- [ ] **Step 4: Run** 전체 게이트 — PASS
- [ ] **Step 5: Commit** `feat: an html note, drawn as text in the terminal (#10)`

### Task 8: `api.Apply`

**Files:** Create `internal/api/apply.go`, Test `internal/api/apply_test.go`

**Interfaces:**
- Produces: `func (a *API) Apply(name, version string, act widget.Action) (string, error)` — 새 버전을 돌려준다. `var ErrStale = errors.New("the note changed since it was read")`.
- 구현: `store.Apply`에 넘기는 Op 안에서 `widget.Version(d) != version`이면 `ErrStale`; 아니면 `Kind.Act` → Op 적용 → `a.tend`. 끝나면 `a.Derive()`(기존 `change`와 같은 이유).

- [ ] **Step 1: Failing tests** — `TestApplyTicksAndStampsTheTime`(완료 시각이 `when` 형식으로 들어감), `TestApplyRefusesAStaleVersion`(파일 바이트가 그대로인지 확인), `TestApplyOnAKindWithoutAct`, `TestApplyOnAMissingNote`(→ `doc.ErrConflict`).
- [ ] **Step 2–4:** FAIL → 구현 → PASS
- [ ] **Step 5: Commit** `feat(api): apply a screen's action to a note it read (#10)`

### Task 9: move the script runner below the screens

**Files:**
- Create: `internal/runner/runner.go`, `runner_test.go` (layer 12: store 위, widget 아래)
- Modify: `internal/app/run.go`, `internal/layers/layers_test.go`, `docs/architecture.md`

**Interfaces:**
- Produces: `func Start(st *store.Store, name string, now func() time.Time) (*Run, error)`; `func (r *Run) Wait() (code int, took time.Duration, err error)`; `func LogOf(name string) string`. 로그 머리줄·열기 동작은 지금 `app.run`과 같다(열기는 호출자 몫으로 남긴다).

- [ ] **Step 1:** `TestStartWritesTheLogAndExitCode`(`.sh` 스크립트 `echo hi; exit 3` → 로그에 `hi`, code 3). 기존 `internal/app` 스크립트 테스트는 그대로 통과해야 한다.
- [ ] **Step 2–4:** FAIL → 옮기기(동작 변화 없음) → 전체 게이트 PASS
- [ ] **Step 5: Commit** `refactor: the script runner moves below the screens, for the bridge (#10)`

### Task 10: `internal/registry`

**Files:** Create `internal/registry/{registry.go,live.go}`, `registry_test.go`; Modify layers 표(rank 5), `docs/architecture.md`

**Interfaces:**
- Produces:
  - `func Home() string` — `STICKYPANE_HOME`, 없으면 `~/.stickypane`.
  - `type Event struct { V int; At, Type, Project, Kind, Cmd string; PID int }` (JSON 태그는 08 §8.4.2 그대로, 빈 값 생략)
  - `func Enabled() bool` — `Home()` 폴더가 있을 때만 true. 없으면 아래 함수는 모두 아무것도 하지 않는다(폴더는 overlay 앱이 만든다).
  - `func Append(e Event)` — 오류를 돌려주지 않는다. `events/<로컬 날짜>.jsonl`에 `O_APPEND|O_CREATE`, 한 번의 `Write`로 한 줄. 4096바이트 넘으면 버린다.
  - `func Live(project, kind string) (end func())` — `session.start` 기록 + `run/<pid>.json`(`{"pid","project","kind","started"}`) 생성, `end()`가 `session.end` 기록 + 파일 삭제.
  - `func Touch(project, cmd string)` — `run/touch-<sha1(project)[:12]>`의 mtime이 1분 안이면 건너뛴다.

- [ ] **Step 1: Failing tests** — `TestNothingIsWrittenWithoutTheHomeFolder`(`STICKYPANE_HOME`이 없는 경로 → 아무 파일도 생기지 않음), `TestAppendWritesADatedLine`(`STICKYPANE_HOME=t.TempDir()`), `TestAppendFromManyProcessesKeepsWholeLines`(테스트 바이너리를 `-test.run=TestHelperAppend`로 20개 띄워 각 200줄 → 4000줄 모두 JSON 유효), `TestAppendIgnoresAnUnwritableHome`(디렉터리 0500), `TestLiveCreatesAndRemovesTheRunFile`, `TestTouchIsThrottled`.
- [ ] **Step 2–4:** FAIL → 구현 → PASS
- [ ] **Step 5: Commit** `feat: record on this machine which boards are used, by day (#10)`

### Task 11: wire the registry into the commands

**Files:** Modify `cmd/stickypane/*.go`(init, TUI 시작, `mcp`, `watch`, `bridge`, 노트를 바꾸는 명령), Test `cmd/stickypane/registry_test.go`

- [ ] **Step 1: Failing tests** — `TestInitRecordsTheProject`(`project.init` 한 줄), `TestEditTouchesTheProject`, `TestCommandsSucceedWhenTheHomeIsReadOnly`(`STICKYPANE_HOME`을 읽기 전용 폴더로 두고 `init`·`todo`·`card` 성공).
- [ ] **Step 2–4:** FAIL → 명령 표의 공통 경로(프로젝트를 찾은 직후) 한 곳에서 `Touch`, 오래 사는 명령은 `defer registry.Live(...)()` → PASS
- [ ] **Step 5: Commit** `feat: commands note their project in ~/.stickypane (#10)`

### Task 12: `view` and `style` settings

**Files:** Modify `internal/store/view.go`, Test `internal/store/view_test.go`

**Interfaces:**
- Produces: `func (s *Store) ViewMode() string`(`"terminal"` 기본), `SetViewMode(string) error`(허용: terminal/overlay/both, 기본값이면 키 삭제), `func (s *Store) Style() string`(`"doodle"` 기본), `SetStyle(string) error`(doodle/neat). 기존 `SetView`(노트별 보기)와 이름이 겹치지 않게 `ViewMode`.

- [ ] **Step 1:** `TestViewModeDefaultsToTerminalAndLeavesNoTrace`, `TestSetViewModeRejectsUnknownValues`, 같은 두 개를 Style에.
- [ ] **Step 2–4:** FAIL → `SetRoot` 위에 구현 → PASS
- [ ] **Step 5: Commit** `feat(store): a board says how it is viewed and in which style (#10)`

### Task 13: `internal/bridge` and `stickypane bridge`

**Files:**
- Create: `internal/bridge/{bridge.go,protocol.go,load.go}`, `bridge_test.go`
- Modify: `cmd/stickypane/commands.go`(표에 `bridge`, usage 한 줄), `internal/layers/layers_test.go`(50), `docs/architecture.md`

**Interfaces:**
- Produces: `func Serve(ctx context.Context, a *api.API, reg widget.Registry, in io.Reader, out io.Writer) error`; `const Protocol = 1`.
- 메시지(한 줄 JSON):
  - 요청 `{"id":n,"method":m,"params":{...}}`, 응답 `{"id":n,"result":...}` 또는 `{"id":n,"error":{"code":"stale"|"bad_request"|"not_found"|"failed","message":...}}`, 알림 `{"method":"board.changed","params":{...}}`.
  - `hello` → `{"protocol":1,"version":"<stickypane 버전>","project":"<root>"}`
  - `board.load` → `{"title","tabs":[{"name","title","notes":[{"name","kind","icon","size","rows","color","pinned","version","summary","data","modTime","pages":[같은 형식, 책일 때만]}]}],"settings":{"theme","style","view","language":{"choice":"auto|ko|en","resolved":"ko|en"}},"palette":{...internal/theme 역할 + notes[6]}}`
  - `note.act {note,version,action}` → `{"version"}`
  - `note.create {tab,kind,title,body}` → `{"note"}` (`api.Write`)
  - `note.run {note}` → `{"code","took"}` (runner; 사용자 확인은 앱이 이미 받았다)
  - `note.view {note, size?, rows?, open?, pin?}` → `store.SetView`(값 검사: size는 card|half|page|"", rows ≥ 0) → `{}`. 테스트 `TestBridgeViewChangesTheNotesSize`(sticky.json에 기록)
  - `note.open {note}` → GUI 편집기로 연다: `$VISUAL`의 프로그램 이름이 `internal/env`의 GUI 편집기 목록(`code`, `cursor`, `zed`, `subl`, `mate`, `gedit`, `kate`)에 있으면 그것, 아니면 macOS `open -t <path>`, Linux `xdg-open <path>`. 기다리지 않는다. 테스트 `TestOpenPrefersAGUIEditorAndFallsBackToTheSystem`(실행 명령을 기록하는 가짜)
  - `settings.set {theme|style|view|language}` → `{}` (language는 `auto|ko|en`, `stickypane language`와 같은 저장 경로)
  - 알림 `board.changed {"notes":[바뀐 노트 board.load 형식],"removed":[name],"events":[{"note",...widget.Event}]}` — `store.Watch` 신호마다 이전 버전 맵과 비교.

- [ ] **Step 1: Failing tests** — `io.Pipe`로 `Serve`를 띄운다: `TestBridgeLoadsTheBoard`, `TestBridgeActChangesTheFileAndAnswersTheNewVersion`, `TestBridgeReportsAnAgentsMoveAsAnEvent`(다른 경로로 `api.Card` 호출 → 2초 안에 `card.moved`), `TestBridgeAnswersBadInputAndKeepsRunning`, `TestBridgeStaleActIsRefused`.
- [ ] **Step 2–4:** FAIL → 구현(쓰기는 mutex로 한 줄씩) → PASS
- [ ] **Step 5:** `cmd/stickypane`에 `bridge` 명령 + `registry.Live(root, "bridge")`, CLI 테스트 `TestBridgeCommandSaysHello`.
- [ ] **Step 6: Commit** `feat: stickypane bridge opens the board to a desktop screen (#10)`

### Task 13b: the bridge arranges and manages notes as the terminal does

**Spec:** `docs/overlay/10-terminal-parity.md` §10.7 (the table is the requirement).

**Files:** Modify `internal/bridge/*`, `internal/bridge/bridge_test.go`; move the api-request sender used by `internal/app/send.go` below the screens if the bridge cannot import it (same pattern as Task 9's runner); `docs/overlay/02-system-architecture.md` §2.4.3 with the final message list.

**Interfaces:**
- Produces: `note.arrange {note, open?, size?, rows?, pin?, color?, title?}` (replaces `note.view`, same storage rules as `stickypane set` via `arrange.Change`), `tab.order {tab, order[]}`, `note.move {note, to}`, `note.archive {note}`, `note.trash {note}`, `note.restore {note}`, `note.send {note, request}`, `kinds.list` (each kind's name, icon, label, blurb, example and its example's `data`), `board.index` (= `api.Index` with links), `board.load` additions (`linked`, `target`, `error`, `color`, `title` per note; `tab`, `time`, `editor`, `stale` in settings), `api.Derive` after every change, chat `say` fills an empty `who` with the terminal's user-name rule.

- [ ] **Step 1: Failing tests** — one per message: `TestBridgeArrangeWritesLikeSet`, `TestBridgeOrderMatchesTheTerminalSwap`, `TestBridgeMoveIntoABook`, `TestBridgeTrashThenRestore`, `TestBridgeSendRunsTheRequest` (against `rpc/rpctest`-style local test server or the existing httpfile test server), `TestBridgeKindsListsEveryShapeWithItsExampleData`, `TestBridgeIndexHasBacklinks`, `TestBridgeDerivesChartsAfterAChange`, `TestBridgeSayUsesTheTerminalName`.
- [ ] **Step 2–4:** FAIL → implement by calling the same functions the terminal and CLI call → PASS (full gate).
- [ ] **Step 5: Commit** `feat(bridge): arrange, move, trash, restore, send, catalog and index (#10)`

### Task 14: words

**Files:** `docs/architecture.md`(새 패키지 줄 확인), `README.md`·`README.ko.md`(명령 목록에 `bridge` 한 줄, `~/.stickypane`에 무엇을 기록하는지 짧은 절), `internal/initcmd/guide.md`(`.html` 노트 한 줄, 55줄 이하), `CHANGELOG.md`(Unreleased).

- [ ] **Step 1:** 고친 뒤 스킬·가이드 테스트 포함 전체 게이트 실행 — PASS
- [ ] **Step 2: Commit** `docs: the bridge, html notes and what stickypane records in ~/.stickypane (#10)`
