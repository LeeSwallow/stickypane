# Overlay App Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `desktop/` 모듈에 macOS·Linux용 데스크톱 앱을 만든다: 홈(세션 목록), 프로젝트별 손그림 overlay 창, 모든 shape 렌더러와 상호작용, Motion, 그리고 `init`·실행 연결과 손그림 DMG까지.

**Architecture:** Wails v3 셸(Go)이 창·트레이·단축키·단일 인스턴스를 맡고, 프로젝트마다 `stickypane bridge` 자식 프로세스와 stdio JSON으로 대화한다. 홈은 `~/.stickypane/`의 사건 파일을 SQLite에 색인해 보여 준다. 프런트엔드는 Svelte 5로 bridge의 `Data`를 그린다.

**Tech Stack:** Wails v3(`github.com/wailsapp/wails/v3`, CLI `wails3`), `modernc.org/sqlite`, Svelte 5 + Vite + TypeScript, unified(remark-parse·remark-gfm·remark-math·remark-rehype·rehype-sanitize·rehype-katex), Shiki, Mermaid, KaTeX, rough.js, Vitest. 글꼴 Gaegu·Pretendard Variable·JetBrains Mono(OFL-1.1).

**Spec:** `docs/desktop/` 전체. 선행: `docs/superpowers/plans/2026-10-28-overlay-core.md`가 끝나 있어야 한다(bridge 프로토콜 1).

## Global Constraints

- 본체 모듈(`go.mod` 루트)에 아무 의존성도 더하지 않는다. 앱의 Go 의존성은 `desktop/go.mod`, JS는 `desktop/frontend/package.json`.
- 앱은 노트를 직접 읽거나 쓰지 않는다. 노트에 관한 모든 것은 bridge를 거친다. `~/.stickypane/`는 읽고, `stickypane.db`·`overlay.json`만 쓴다.
- 네트워크: listener 없음. 프런트엔드 CSP는 `default-src 'self'`, 외부 URL 금지. 글꼴·렌더러는 번들.
- 플랫폼: darwin/arm64·amd64, linux/amd64·arm64. Windows 빌드 대상 없음.
- 색은 bridge의 `palette`에서만 온다. 스타일은 `doodle`(기본) | `neat`.
- 문구는 한국어("~해요" 체)와 영어 두 벌(03 §3.8). 프로젝트 창은 bridge의 `settings.language.resolved`, 홈은 `overlay.json`의 `language`. 하드코딩된 화면 문구 금지 — 모두 `t(key)`.
- 움직임은 bridge 사건에만 반응, 반복 애니메이션 금지, `prefers-reduced-motion`이면 opacity만. 토큰: 120/200/320/620ms, `--ease`·`--slap`·`--exit`(04 §4.2).
- 라이선스: 새 Go·npm 의존성은 MIT·BSD·Apache-2.0·ISC·OFL만. `desktop/scripts/licenses.sh`가 검사한다.
- 매 Task 끝: 본체 게이트 + `cd overlay && go vet ./... && go test -race ./... && npm --prefix frontend run check && npm --prefix frontend test`.
- 커밋 규칙은 본체와 같다(`(#10)`, push 금지).

## Review Focus

1. **bridge가 죽거나 `stickypane`이 PATH에 없음** — 창이 하얗게 멈추지 않고, 무엇이 없는지와 설치 한 줄을 보여 주며, bridge가 죽으면 3번까지 다시 띄운다. (A2 `TestClientRestartsADeadBridge`, A4 `TestWindowShowsMissingStickypane`)
2. **창이 열린 사이 프로젝트 폴더가 지워짐** — 그 창은 "이 보드는 사라졌어요"로 바뀌고 홈 목록에서 빠진다. (A3 `TestPruneDropsAFolderRemovedWhileOpen`, A4)
3. **낙관적으로 그린 뒤 `stale` 응답** — 200ms 안에 최신 Data로 되돌리고 행동을 잃어버렸다고 알린다. (A9 `act rolls back on stale` 테스트)
4. **아주 큰 노트·많은 노트** — 노트 300개, 2MB 로그에서 `board.load`가 1초 안, 로그는 마지막 1000줄만. (A7 성능 테스트, core Task 2의 1000줄 제한)
5. **악의적인 html 노트** — 네트워크·부모·바인딩·저장소 접근이 모두 막힌다. (A8 격리 테스트)

---

### Task A1: Scaffold the module, the build and the checks

**Files:** Create `desktop/go.mod`, `desktop/main.go`, `desktop/Taskfile.yml`(wails3 기본), `desktop/frontend/`(Svelte 5 + Vite + TS 템플릿), `desktop/frontend/src/assets/fonts/*`(Gaegu 400·700, PretendardVariable.woff2, JetBrainsMono 400·700) + `LICENSE-*.txt`, `desktop/scripts/licenses.sh`, `desktop/build/appicon.png`(사용자가 준 아이콘), `.github/workflows/desktop.yml`(macos-latest·ubuntu-latest: Go 테스트, 프런트 검사·테스트, `wails3 build`)

- [ ] Step 1: `wails3 init -n "Stickypane Overlay" -t svelte-ts` 결과를 `desktop/`로 옮기고, 빈 창이 뜨는지 `wails3 dev`로 확인(사람이 본다).
- [ ] Step 2: `licenses.sh`가 `go-licenses`가 아닌 `go list -m -json all`과 `npm ls --json`·각 패키지 `license` 필드로 허용 목록을 검사하고, 어긋나면 이름을 출력하고 1로 끝난다. 테스트: 허용 안 되는 가짜 패키지를 넣은 fixture에서 1.
- [ ] Step 3: CI 워크플로가 두 OS에서 통과(로컬은 `act` 없이 macOS 단계만 확인).
- [ ] Step 4: Commit `build(overlay): scaffold the desktop app with its fonts and checks (#10)`

### Task A2: The bridge client

**Files:** Create `desktop/internal/bridge/{client.go,types.go}`, `client_test.go`, `testdata/fakebridge/main.go`

**Interfaces:**
- Produces: `func Start(ctx context.Context, stickypane, project string) (*Client, error)`(`hello`로 protocol==1 확인, 아니면 `ErrProtocol`), `(*Client).Call(ctx, method string, params, result any) error`, `(*Client).Notifications() <-chan Notification`, `(*Client).Close() error`, `func Find() (string, error)`(PATH → `/opt/homebrew/bin` → `/usr/local/bin` → `~/go/bin`; GUI 앱은 셸 PATH를 못 받기 때문). 타입은 core Task 13의 JSON 그대로.
- [ ] Step 1: 실패 테스트 — `TestClientHelloChecksTheProtocol`, `TestClientCallsAndReceivesNotifications`, `TestClientRestartsADeadBridge`(fake가 1번 죽음 → 재시작 → 같은 Client로 다시 호출 성공; 3번 넘게 죽으면 `ErrBridgeDown`), `TestFindLooksBeyondTheGUIPath`.
- [ ] Step 2–4: FAIL → 구현 → PASS. Step 5: Commit `feat(overlay): talk to stickypane bridge (#10)`

### Task A3: The `~/.stickypane` index

**Files:** Create `desktop/internal/index/{index.go,schema.sql,ingest.go,prune.go,query.go}`, `index_test.go`

**Interfaces:**
- Produces: `func Open(home string) (*Index, error)`(WAL, 스키마 08 §8.4.3, `meta.schema_version=1`, 깨졌으면 `stickypane.db.broken-<시각>`으로 옮기고 다시 만듦), `(*Index).Ingest() error`, `(*Index).Prune(now time.Time) (removed []Project, err error)`, `(*Index).Restore(id int64) error`, `(*Index).Sessions(now time.Time) (Groups, error)` — `Groups{Active []Row; Days []Day{Date string; Rows []Row}; Removed []Project}`, `Row{Project; Agents []string; Live []Session; LastAt time.Time}`.
- 규칙은 08 §8.4.4 표 그대로. "활성" = 살아 있는 run 파일(pid 생존 + 시작 시각 일치) 또는 `.sticky/` mtime이 30분 안.
- [ ] Step 1: 실패 테스트(표의 행마다 하나) — `TestIngestReadsOnlyNewLines`, `TestIngestWaitsForAPartialLastLine`, `TestIngestSkipsBrokenLines`, `TestPruneMarksAMissingFolderAndRestoresItWhenBack`, `TestPruneDropsAFolderRemovedWhileOpen`, `TestPruneDeletesRowsAfter30Days`, `TestDeadPidsEndTheirSessions`, `TestRebuildFromEventsWhenTheDBIsCorrupt`, `TestDeletedEventFileForgetsItsOffset`, `TestOldEventFilesAreRemovedAfter90Days`, `TestSessionsGroupsByLocalDay`.
- [ ] Step 2–4: FAIL → 구현 → PASS. Step 5: Commit `feat(overlay): index the boards on this machine by day (#10)`

### Task A4: Windows, tray, hotkey, single instance

**Files:** Create `desktop/internal/shell/{app.go,windows.go,tray.go,state.go}`, `shell_test.go`; Modify `desktop/main.go`

**Interfaces:**
- Produces: 홈 창 1개(860×620), 프로젝트 창은 경로당 1개(프레임 없음, 840×화면 70%, 위치·크기·핀은 `~/.stickypane/overlay.json`), 트레이(활성 수, 최근 5개, 홈, 종료), 전역 단축키 기본 `Option+Space`(설정 가능, 등록 실패 시 홈에 한 줄 안내), 핀 = `SetAlwaysOnTop`, 핀 꺼짐이면 `HideOnFocusLost`, 단일 인스턴스로 두 번째 실행의 `--project=<path>`를 받아 창을 연다.
- Wails 바인딩 서비스 `ProjectService`(`Load`, `Act`, `View`, `Create`, `Run`, `Open`, `SetSetting`)와 `HomeService`(`Ensure`, `Sessions`, `Restore`, `PickFolder`, `InitProject`, `StickypaneStatus`)를 프런트에 연다. 알림은 Wails 이벤트 `board.changed:<window id>`.
- [ ] Step 1: 실패 테스트(셸 로직을 Wails 없이 시험할 수 있게 `windows.go`는 인터페이스 뒤에) — `TestOpenProjectReusesItsWindow`, `TestWindowStateRoundTrips`, `TestWindowShowsMissingStickypane`(Find 실패 → 창 상태 `missing`), `TestSecondLaunchOpensTheProject`.
- [ ] Step 2–4: FAIL → 구현 → PASS. Step 5: 이 Mac에서 `wails3 dev`로 단축키·핀·트레이를 사람이 확인하고 결과를 07 실행 기록에.
- [ ] Step 6: Commit `feat(overlay): home and board windows, the tray and the hotkey (#10)`

### Task A5: The look — tokens, styles, doodle parts

**Files:** Create `desktop/frontend/src/lib/theme/{tokens.ts,palette.ts}`, `src/lib/doodle/{Paper.svelte,Sheet.svelte,Tape.svelte,RoughBox.svelte,Crayon.svelte,Check.svelte,Radio.svelte,Underline.svelte,Mark.svelte,Circle.svelte,Arrow.svelte,Logo.svelte}`, `src/lib/doodle/rough.ts`, 테스트 `*.test.ts`

**Interfaces:**
- 격자 노트 바탕과 모드(시스템·다크·라이트, 03 §3.4): `paperGrid` CSS, `pickByMode(themeName, mode, osDark)`로 짝 테마 선택, OS 변경 구독.
- Produces: `applyTheme(palette: Palette, dark: boolean)`(03 §3.4 파생 규칙 그대로: wash 35%/78%, 외곽선 55%/Text, paper 6%/50%, dot 12%/16%), `agentColor(name: string, notes: string[]): string`(본체 노트 색 해시와 같은 함수 — 본체 `color` 해시 구현을 TS로 옮기고 같은 입력에 같은 결과인지 golden으로), `seedOf(id: string): number`, 스타일 `doodle|neat`에서 `roughness` 0 처리.
- [ ] Step 1: 실패 테스트 — `palette derivations match 03 §3.4`, `agentColor matches the Go hash for 10 names`, `contrast is at least 4.5:1 for every theme × card colour`(어긋나면 wash를 더 밝게 하는 규칙을 고친다), `neat style draws straight`.
- [ ] Step 2–4: FAIL → 구현(목업 `overlay-doodle-v3-themes.html`의 `crayon`·`roughBox`가 출발점) → PASS. Step 5: Commit `feat(overlay): the doodle and neat looks in every colour theme (#10)`

### Task A5b: Korean and English

**Files:** Create `desktop/frontend/src/i18n/{index.ts,ko.json,en.json}`, `src/settings/Language.svelte`; Modify `desktop/internal/shell/state.go`(`overlay.json`의 `language`), 테스트 `i18n.test.ts`, `desktop/internal/shell/state_test.go`

**Interfaces:**
- Produces: `t(key: string, values?: Record<string, unknown>): string`(영어 기준, 빠진 키는 영어), `setLanguage(code: 'ko'|'en')`(즉시 다시 그림, Svelte store), `relativeTime(d: Date, now: Date, lang)`, `dayLabel(date: string, today: Date, lang)`(Intl). 설정 UI는 한국어 / English / 시스템.
- [ ] Step 1: 실패 테스트 — `every ko key exists in en`(키 집합이 같다), `missing key falls back to English`, `day labels say 오늘/어제 and Today/Yesterday`, `relative time 2분 전 / 2m ago`, `changing language in a project window calls settings.set language`, Go `TestHomeLanguageDefaultsToAuto`.
- [ ] Step 2–4: FAIL → 구현 → PASS. 이후 모든 Task의 문구는 두 파일에 함께 넣는다(검사 테스트가 키 집합 차이를 잡는다).
- [ ] Step 5: Commit `feat(overlay): Korean and English, shared with the terminal's language (#10)`

### Task A6: Home — the list of boards

**Files:** Create `desktop/frontend/src/home/{Home.svelte,Empty.svelte,Sessions.svelte,SessionRow.svelte}`, `src/i18n/{ko.json,en.json}`

**Interfaces:**
- 랜딩·설명 화면 없음(R-32). 앱 시작 시 `HomeService.Ensure()`가 `~/.stickypane/`을 만들고 바로 세션 목록.
- 세션 목록은 `HomeService.Sessions()` 그룹 순서(지금 움직이는 중 → 오늘 → 어제 → 날짜), 줄 구성은 08 §8.3, 하단 "사라진 프로젝트 n개 정리함 · 실행 취소".
- 비었으면 `Empty.svelte` 한 장: 메모지 + "아직 붙은 보드가 없어요" + [폴더 고르기](`PickFolder` → `.sticky/` 없으면 `InitProject(path)` = `stickypane init --view overlay`). `StickypaneStatus().found == false`면 설치 한 줄과 복사 버튼.
- [ ] Step 1: 실패 테스트(Vitest + @testing-library/svelte) — `home opens straight to the list`, `empty state offers one button`, `empty state shows the install line when stickypane is missing`, `sessions are grouped active, today, yesterday, by date`, `undo restores a removed project`.
- [ ] Step 2–4: FAIL → 구현 → PASS. Step 5: Commit `feat(overlay): the list of boards on this machine (#10)`

### Task A7: The board window and the first renderers

**Files:** Create `desktop/frontend/src/board/{Board.svelte,TitleBar.svelte,Glance.svelte,Tabs.svelte,Grid.svelte,NoteFrame.svelte}`, `src/shapes/{Note.svelte,markdown.ts,Checklist.svelte,Kanban.svelte,Card.svelte}`

**Interfaces:**
- 배치: 03 §3.6(2열, page=2칸, 720px 미만 1열, 순서는 bridge가 준 순서). Glance 줄은 board 노트들의 카드 `who`에서 에이전트별 지금 카드(Doing 우선)·경과·정체.
- `markdown.ts`: `render(md: string, style): Promise<string>` — remark-parse → remark-gfm → remark-math → remark-rehype → rehype-sanitize(기본 스키마 + `class`는 코드·수식만) → rehype-katex → Shiki(번들 테마 2개: 밝음·어두움, 언어는 지연 로드) → HTML. ```` ```mermaid ```` 블록은 Mermaid로 SVG(테마 변수는 palette에서, `securityLevel: 'strict'`).
- 노트 크기·높이·크게 보기·책 페이지: 03 §3.7 그대로. 크기 버튼/`-` `=` → `ProjectService.View`(bridge `note.view`), 아래 가장자리 끌기 → `rows`, 더블클릭/`Enter` → 크게 보기(창 상태), 책은 `‹ n / m ›`와 `[` `]`.
- [ ] Step 1: 실패 테스트 — `size button sends note.view and the grid reflows`, `zoom fills the window and Esc returns`, `book pages flip with [ and ]`, `markdown renders GFM table, task list, footnote`, `markdown strips script and on* attributes`, `mermaid block becomes svg`, `math renders`, `board draws columns and stalled ring`, `glance picks the Doing card per agent`, `300 notes render under 1s`(jsdom 근사, 실패 시 가상 스크롤).
- [ ] Step 2–4: FAIL → 구현 → PASS. Step 5: 이 Mac에서 실제 프로젝트(`stickypane` 레포 자신)를 열어 확인. Step 6: Commit `feat(overlay): draw notes, checklists and kanban boards (#10)`

### Task A7b: Free canvas and torn-off notes

**Spec:** 03 §3.6.1, ADR-0017.

**Files:** Create `desktop/frontend/src/board/{Canvas.svelte,place.ts,LayoutToggle.svelte}`, `src/torn/TornNote.svelte`; Modify `src/board/Board.svelte`, `desktop/internal/shell/{state.go,windows.go,services.go}` (places and torn windows in overlay.json; `ProjectService.Places/SetPlace/Tear/Untear`), tests.

**Interfaces:**
- `overlay.json` `projects.<path>`: `layout: "canvas"|"grid"`, `places.<note>: {x,y,w,h,tilt,z}`, `torn.<note>: {screenX,screenY,w,h,pinned}`; saved through the existing serialized/fsynced save.
- Canvas: drag (pointer events) moves a note; release saves; ⌘+wheel zooms 50–200%; new notes get the grid slot then keep it; z = last touched on top; reuses the Note component from A7 unchanged.
- Tear-off: "떼어내기" in the note header (or drag out of the canvas) opens a frameless small window `?window=note&path=<p>&note=<n>` that renders the same Note component with the project's bridge client (shared, not a second bridge); its own pin; closing returns it to the canvas; window positions restored on launch.
- Note removed → its place/torn entries pruned on the next load; `note.move` → entries renamed.
- [ ] Step 1: failing tests — `canvas drag saves a place`, `a new note starts at its grid slot`, `layout toggle keeps both`, Go `TestPlacesRoundTripPerProject`, `TestTornWindowsReopenOnLaunch`, `TestPlacesPrunedWhenANoteDisappears`, `TestPlacesFollowARename`.
- [ ] Step 2–4: FAIL → implement → PASS. Step 5: launch and capture canvas + a torn note over another app. Step 6: Commit `feat(overlay): stick notes anywhere, on the canvas or on the screen (#10)`

### Task A8: The rest of the shapes and the html sandbox

**Files:** Create `src/shapes/{Form.svelte,Chart.svelte,Log.svelte,Script.svelte,Chat.svelte,Api.svelte,Html.svelte,htmlFrame.ts}`, `desktop/frontend/test/html-isolation/*.html`

**Interfaces:**
- `htmlFrame.ts`: `frameDoc(html: string): string` — `<meta http-equiv="Content-Security-Policy" content="default-src 'none'; img-src data:; style-src 'unsafe-inline'; script-src 'unsafe-inline'; font-src data:">`를 `<head>` 맨 앞에 넣는다. iframe은 `sandbox="allow-scripts"`, `srcdoc`. 부모는 `message`에서 `{type:"height", value:number}`만 받는다.
- [ ] Step 1: 실패 테스트 — 각 shape 렌더 테스트, `frameDoc puts the CSP first even without a head`, 격리 자료 6개(fetch, 외부 img, `parent.document`, `top.location`, Wails 바인딩 호출, `localStorage`)는 A13의 실제 webview 검사로 확인하고 결과를 기록.
- [ ] Step 2–4: FAIL → 구현 → PASS. Step 5: Commit `feat(overlay): forms, charts, logs, scripts, chat and sandboxed html (#10)`

### Task A9: Interactions

**Files:** Create `src/board/{act.ts,drag.ts,Create.svelte,ConfirmRun.svelte}`; Modify shape 컴포넌트

**Interfaces:**
- `act(note, version, action)`: 즉시 로컬 Data를 바꿔 그리고 `ProjectService.Act`; `stale`이면 최신 Data로 되돌리고 토스트 "그사이 바뀌어서 다시 그렸어요". 행동 이름·인자는 core Task 3–6 그대로.
- 카드 드래그(포인터 이벤트, 키보드 대안 `H`/`L`), "+ 생성"(`A`) = 손그림 카탈로그(`kinds.list`: 모양 목록 + 설명 + 실제 렌더러로 그린 예시) → 제목 입력 → `Create({tab, kind, title})`. 빠른 끄적이기는 없다(R-36). 긴 편집은 `Open`.
- 스크립트·api 실행은 확인 창(문구는 터미널의 `tr.ConfirmRun`과 같은 뜻) 뒤 `Run`.
- [ ] Step 1: 실패 테스트 — `tick is optimistic`, `act rolls back on stale`, `drag across columns sends move with indices`, `create picks a shape and a title, then the note drops in`, `run asks first`.
- [ ] Step 2–4: FAIL → 구현 → PASS. Step 5: Commit `feat(overlay): tick, move, answer, run and create from the overlay (#10)`

### Task A10: Motion

**Files:** Create `src/motion/{tokens.ts,choreo.ts,flip.ts,pen.ts,intro.ts}`; Modify Kanban·Checklist·Grid·창 열기/닫기

**Interfaces:**
- `choreograph(events: BoardEvent[], reduce: boolean): Plan` — 04 §4.3 표의 사건→움직임, §4.4 묶음 규칙(≤3 각자 60ms 간격, 그 이상 이동만 30ms 간격 최대 6개, 전체 1.2초 상한, 새 알림이 오면 진행 중인 것을 끝 상태로), 창이 숨겨져 있으면 재생하지 않고 ● 표시만.
- 시작 인트로 "꾸물꾸물"(04 §4.3.1): `intro.ts`의 `playIntro(kind: 'launch'|'reopen', reduce, style)`; 클릭·키로 건너뛰기.
- [ ] Step 1: 실패 테스트 — `launch intro plays once per process`, `reopen plays only the notes wriggle`, `a key skips the intro to its end state`, `reduced motion skips the intro`, `card.moved plays lift-carry-slap`, `more than three events are batched and capped at 1.2s`, `a new notification finishes the running plan`, `reduced motion is opacity only`, `hidden window marks instead of playing`, `user's own action gets no highlight`.
- [ ] Step 2–4: FAIL → 구현 → PASS. Step 5: Commit `feat(overlay): motion that shows what changed (#10)`

### Task A11: Choosing and opening the overlay (본체)

**Files:** Modify `internal/initcmd/initcmd.go`(플래그 `--view`, `--style`; stdin이 TTY이고 사람이 `init`을 직접 실행할 때만 묻는다 — 에이전트·앱·자동 init은 묻지 않는다), `internal/env/overlay.go`(앱 찾기: macOS `/Applications/Stickypane Overlay.app` → `open -a ... --args --project=`, Linux PATH의 `stickypane-overlay`), `cmd/stickypane`(인자 없는 실행이 `ViewMode`에 따라), 테스트

- [ ] Step 1: 실패 테스트 — `TestInitAsksOnlyOnATerminal`(TTY 아님 → 묻지 않고 기본값), `TestInitViewFlag`, `TestNoArgsOpensTheOverlayWhenChosen`(실행할 명령을 기록하는 가짜 env), `TestNoArgsFallsBackToTheTerminalWithAHint`, `TestWindowsAlwaysUsesTheTerminal`.
- [ ] Step 2–4: FAIL → 구현 → PASS. Step 5: Commit `feat: init asks how to view the board, and stickypane opens the overlay (#10)`

### Task A12: Packaging — doodle DMG, Linux archive, release

**Files:** Create `desktop/build/dmg/background.html`, `scripts/desktop-dmg.sh`, `desktop/build/linux/stickypane-overlay.desktop`; Modify `.goreleaser.yaml`(또는 별도 `desktop/.goreleaser.yaml`), `THIRD_PARTY_NOTICES.md`

- [ ] Step 1: `background.html`을 09 §9.2 그대로(660×420, 밝은 종이, 테이프 제목 "stickypane overlay", "끌어다 붙이면 끝!", 빨강 크레용 화살표 (170,210)→(490,210), 노란 메모지 Gatekeeper 안내) 만들고 헤드리스 Chrome으로 `background.png`·`background@2x.png` → `tiffutil -cathidpicheck`로 `background.tiff`.
- [ ] Step 2: `desktop-dmg.sh <app> <out.dmg>` — `hdiutil create -srcfolder` 임시 UDRW → 마운트 → `.background/` 복사 → AppleScript로 창 크기·아이콘 크기 112·위치·배경 → `bless`·볼륨 아이콘 → `hdiutil convert -format UDZO`. ad-hoc `codesign -s - --deep`.
- [ ] Step 3: Linux tar.gz(바이너리 + `.desktop` + 아이콘 + 라이선스), cask `stickypane-overlay`.
- [ ] Step 4: 이 Mac에서 DMG를 만들어 마운트하고 Finder 창 스크린샷을 `docs/desktop/assets/dmg.png`로 남긴다(사람이 모양 확인).
- [ ] Step 5: Commit `build(overlay): a hand-drawn DMG and a Linux archive (#10)`

### Task A13: End-to-end on this Mac, and the words

**Files:** `README.md`, `README.ko.md`(overlay 절: 설치, 보기 방식, 테마, `~/.stickypane`), `ROADMAP.md`(목표 문장, ADR-0010), `CHANGELOG.md`, `docs/desktop/*`(구현에서 정한 것 반영, 07에 "실행 기록" 절), `docs/desktop/assets/`(스크린샷)

- [ ] Step 1: 01 §1.7 장면 A·B·C를 이 Mac에서 재현하고 스크린샷·GIF를 남긴다. HTML 격리 자료 6개를 실제 webview에서 열어 모두 막히는지 기록.
- [ ] Step 2: 08 §8.4.4 표의 상황을 손으로 재현(폴더 지우기·되살리기, `kill -9`, db 삭제, 이벤트 파일 잘라내기) — 각 결과를 07 실행 기록에.
- [ ] Step 3: 문서 링크 검사, Mermaid 렌더 확인, 에이전트 가이드 55줄 이하, 전체 게이트.
- [ ] Step 4: Commit `docs: the overlay in the README, the roadmap and the design record (#10)`
