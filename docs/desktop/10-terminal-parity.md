# 10. 터미널 기능 대응표

> 상태: Draft · 2026-10-28 · 관련 요구: R-34, R-35
>
> 터미널 보드에서 사용자가 할 수 있는 모든 일을 앱에서 어떻게 하는지 정한다. 원칙 O1(앱이 할 수 있는 일은 터미널도 할 수 있다)의 반대 방향, **터미널에서 되는 일은 앱에서도 된다**를 확인하는 표다. 출처: `internal/surface/app/*`, `internal/usecase/prefs`, `cmd/stickypane`의 조사(2026-10-28).

**판정**: ✅ 같은 기능 · ↗ 더 낫게(그래픽이라 가능) · ↪ 다른 길로(이유 있음) · ✗ 하지 않음(이유 있음) · 미구현 아직 하지 않음(할 일). **bridge** 열은 필요한 bridge 메시지(없으면 이미 있는 것으로 충분).

## 10.1 화면 배치

| 터미널 | 키 | 앱 | 판정 | bridge |
|---|---|---|---|---|
| 탭 = 최상위 폴더, 번호로 전환 | `1`–`9`, `(` `)`, 클릭 | 손글씨 탭 줄. 클릭·`⌘1`–`⌘9`. 탭 제목은 폴더의 `sticky.json` title 또는 숫자 접두사를 뺀 이름 | ✅ | `board.load` |
| 지금 탭을 `sticky.json` `tab`에 저장, 에이전트의 `show`가 탭을 바꿈 | — | [설계자 판단] 앱의 탭 선택은 **창 상태**로 두고 `tab`을 쓰지 않는다(터미널의 화면을 빼앗지 않으려고). 단 `tab` 값이 바뀌면(에이전트가 `show`) 앱도 그 탭으로 간다 | ↪ | `board.changed.settings.tab` |
| 노트 이름 줄(열림·닫힘·포커스·`•` 갱신 표시, 넘치면 스크롤) | — | 탭 아래 "메모 꽂이": 닫힌 노트는 접힌 메모지 칩, 열린 노트는 그리드에. 바뀐 노트는 제목 옆 손그림 점 ● | ✅ | `modTime`, 창의 seen 맵 |
| 열기·닫기 | `o`, `enter`, 칩 클릭 | 칩 클릭으로 열기, 노트 머리의 `×`로 닫기 | ✅ | `note.arrange {open}` |
| 너비 = 화면의 비율 `1/4` `1/3` `1/2` `2/3` `3/4` `1` (옛 이름 card/half/page도 읽음) | `+` `=` / `-`, 오른쪽 테두리 끌기 | 노트 머리의 너비 그림(채운 막대 + ½)과 좁게·넓게 버튼, 오른쪽 가장자리 끌기(가까운 비율에 붙음), `-` `=`. 12열 그리드: `1/4`=3열 … `1`=12열, 720px 미만은 1열 | ✅ | `note.arrange {size}` |
| 고정 높이 rows | `shift+↑` `shift+↓` (`shift+pgup` `shift+pgdn` 다섯 줄), `0`, 아래 테두리 끌기, 더블클릭은 내용만큼 | 아래 가장자리 끌기(↗), 더블클릭은 내용만큼 | ↗ | `note.arrange {rows}` |
| 고정(pin) | `p` | 노트 머리의 압정 버튼. 고정 노트가 먼저 | ✅ | `note.arrange {pin}` |
| 색 순환 | `c` | 노트 머리의 색 점을 누르면 여섯 색 팔레트 | ↗ | `note.arrange {color}` |
| 제목 바꾸기 | `R` | 제목을 더블클릭해 그 자리에서 고친다 | ↗ | `note.arrange {title}` (터미널과 같은 규칙: .md는 front matter, 나머지는 sticky.json) |
| 순서 바꾸기 | `{` `}` | 노트를 끌어 놓기(↗), `{` `}` 키도 | ↗ | `tab.order {tab, order[]}` |
| 자동 타일링, 화면 나눔 `[` `]` | `[` `]` | 12열 그리드에 비율대로, 한 줄을 꽉 채워(터미널의 타일링과 같은 규칙) 놓고 창은 세로로 스크롤한다. 화면 나눔은 필요 없다 | ↪ | — |
| 노트 안 스크롤, log는 끝 따라가기 | `j k g G` 등, 휠 | 노트 안 스크롤(휠·트랙패드), log 끝 따라가기, 위로 올리면 멈춤 | ✅ | — |
| 책(폴더 안 폴더)과 페이지 넘기기 | `,` `.` `<` `>` | `‹ 2 / 5 ›`, `[` `]`, 스와이프, 페이지 이름 목록 ([03 §3.7](03-design-system.md#37-노트-크기--크게-보기--페이지)) | ✅ | `board.load` 노트의 `pages` |
| 연결된 파일(심볼릭 링크) | `show <path>`, `link` | 그대로 보이고, 노트 머리에 🔗와 실제 경로 툴팁 | ✅ | `board.load` 노트의 `linked`, `target` |
| 숨김: `.trash`, `archive/`, `ignore` | — | 같은 규칙(bridge가 이미 걸러 준다) | ✅ | — |
| 읽을 수 없는 노트 | 자동 | 노트 안에 "이 노트를 보여 줄 수 없어요: …", 편집 동작은 막힘 | ✅ | 노트의 `error` |

## 10.2 이동 · 크게 보기 · 도움말 · 찾기

| 터미널 | 키 | 앱 | 판정 | bridge |
|---|---|---|---|---|
| 다음·이전 노트 | `tab` / `shift+tab` | 같은 키, 포커스 링은 accent 손그림 테두리 | ✅ | — |
| 크게 보기(zoom) | `enter`, `z`, 더블클릭 | 같음. 노트가 창을 채운다 | ✅ | — |
| 도움말 | `?` | 메모지 한 장에 키 목록(언어에 맞춰) | ✅ | — (문구는 앱 i18n) |
| 찾기·목록(index) | `i`, `/` | `⌘K` 또는 `/`: 모든 탭의 노트를 찾는 팔레트. 줄마다 아이콘·제목·`2/5`·`← N` 백링크·한 줄 요지·수정 시각. 단어 모두 맞아야 함 | ✅ | `board.index`(= `api.Index`) |
| 링크 패널 | `f` | 노트 머리의 `← N` 링크 수를 누르면 이 노트와 링크로 이어진 노트 목록 | ✅ | `board.index`의 링크 정보 |
| `[[링크]]`, `![[끼워 넣기]]` | 표시만 | 링크는 누르면 그 노트로 이동(↗), 끼워 넣기는 그 노트의 렌더러로 한 단계까지 | ↗ | 노트 `data`의 링크 대상, 끼워 넣을 노트의 `data` |
| 새로고침 | `r` | watch가 안 되면 홈에 "파일 변화를 볼 수 없어요 · 새로고침" | ✅ | `board.load` |
| 끝내기 | `q` | 창 닫기·숨기기(`Esc`), 앱 끝내기는 트레이 | ↪ | — |

## 10.3 노트 만들기 · 관리

| 터미널 | 키 | 앱 | 판정 | bridge |
|---|---|---|---|---|
| 끄적이기 | `N`, `n` | [사용자 결정] 앱에는 두지 않는다(R-36). 빠른 메모도 "+ 새 노트"에서 모양 note를 골라 만든다. 터미널의 `N`은 그대로 | ✗ | — |
| 모양 골라 추가(카탈로그) | `a` | **"+ 새 노트"**(제목줄 버튼, `a`). 왼쪽 모양 목록, 오른쪽에 설명·제목·탭과 그 모양의 만들기 옵션(`kinds.list`의 `options`로 그린 칸: 한 줄·여러 줄·고르기·숫자·켜고 끄기), 아래에 실제 렌더러로 그린 예시. 앱은 값만 보내고 본문은 모양의 `Kind.Compose`가 쓴다. 모양이 거절한 값은 그 칸 아래에 bridge의 말 그대로 | ✅ | `kinds.list`(이름·아이콘·라벨·blurb·example·data·options) + `note.create {tab, kind, title, options}` |
| 보관 | `x` | 노트 머리 메뉴 "보관" | ✅ | `note.archive` |
| 지우기(휴지통) | `D` + 확인 | 메뉴 "지우기" + 확인 메모지 | ✅ | `note.trash` |
| 되돌리기(마지막 x/D 하나) | `u` | 지운 직후 하단 토스트 "되돌리기"(↗: 토스트가 사라지기 전까지), `⌘Z` | ✅ | `note.restore` |
| 폴더로 옮기기 | `m` | 노트를 탭으로 끌어 놓기(↗), 메뉴 "옮기기…"(탭·책·새 폴더) | ↗ | `note.move {note, to}` |
| 한 줄 입력 | 아래 줄 프롬프트 | 그 자리 입력칸(제목, 카드, 항목, 필드) | ✅ | 각 `Act` |
| 스크립트 실행 + 확인 | `enter`/`space`, 클릭 | ▶ 실행 버튼 + 확인 메모지 "<프로젝트>에서 X를 실행할까요?". 로그는 스크립트 노트 아래 | ✅ | `note.run` |
| api 요청 보내기 + 확인 | `enter`/`space`, 클릭 | 요청 목록에서 고르고 ▶ 보내기 + 확인. 결과·검사 표시 | ✅ | `note.send {note, request}` |
| api 환경(`# @env`) 바꾸기 | `e` | 노트 머리의 환경 선택 | 미구현 | `note.act` (api `Act: env`) — 아직 없음: api에 `Act`가 없다 |
| 여기서 편집(vi 같은 편집기) | `e` | [설계자 판단] 하지 않는다(N1). 대신 `E`/"편집기로 열기" | ✗ | — |
| `$EDITOR`로 편집 | `E` | GUI 편집기 또는 OS 기본 앱 | ↪ | `note.open` |
| 빈 화면 | — | "아직 노트가 없어요. + 만들기로 만들거나 에이전트에게 붙여 달라고 해 보세요." / "열린 노트가 없어요." | ✅ | — |

## 10.4 shape별

| shape | 터미널 | 앱 | bridge |
|---|---|---|---|
| note | Markdown 일부, mermaid ASCII, 링크·끼워 넣기 | Markdown 전부(unified), Shiki, Mermaid SVG, KaTeX, 링크 이동 | `data.markdown` |
| board | `h l j k`, `H L` 열 이동, `J K` 순서, `n` 새 카드, ⚠ 정체 | 끌어 놓기로 이동·순서(↗), 같은 키, 열 머리의 `+`, 정체 동그라미 | `move`, `add` |
| checklist | `space` 체크, `n` 새 항목, 진행 막대, 완료는 시각과 함께 아래로 | 클릭 체크, 아래 `+ 항목`, 진행 막대, 완료 묶음 | `toggle`, `add` |
| form | 고르기·체크·필드·버튼, 제출 뒤 바꾸면 제출 해제 | 손그림 라디오·체크·입력칸·버튼, 같은 규칙 | `choose`, `fill`, `submit` |
| chart | bar · spark · heat, `from:` 파생 | 같은 세 가지를 SVG로(손그림/단정) | `data` (파생은 bridge가 `Derive`) |
| log | 끝 따라가기, "마지막으로 늘어난 시각" | 같음, 새 줄이 아래에서 밀려 들어옴 | `data.lines` |
| chat | `n` 말하기, `@user HH:MM` | 아래 입력칸, 이름은 터미널과 같은 규칙(`$STICKYPANE_USER` → git → 로그인) | `say` (who는 bridge가 같은 규칙으로 채움) |
| script | ▶ Run, 로그 | 같음 | `note.run` |
| api (.http) | 요청 고르기·보내기·환경 | 같음(환경 바꾸기는 미구현). 고른 요청의 응답은 같은 이름의 .log에서 | `data.requests`·`data.env`, `note.send`; `Act: env`는 미구현 |
| html | 텍스트만 | 샌드박스에서 실제로 그림(HTML·CSS, 페이지의 스크립트는 돌지 않음 — 02 §2.7) | `data.html` |

## 10.5 설정

| 터미널 설정(`S`, `stickypane config`) | 값 | 앱 |
|---|---|---|
| theme | auto + 10개 | 같은 목록. auto는 OS의 밝음·어두움을 따른다. `T`로 순환도 |
| language | auto · en · ko | 같음 ([03 §3.9](03-design-system.md#39-언어)) |
| time | auto · ko · en-US · en-GB · ja · de · fr · iso | 같음. 시각·날짜 표시를 이 설정으로 |
| editor | auto + 찾은 편집기 | 같은 목록 중 GUI 편집기만 고를 수 있음(터미널 편집기는 회색, 이유 툴팁) |
| stale | 15m · 30m · 1h · 4h · 1d · off | 같음. 정체 동그라미의 기준 |
| (앱만) style | doodle · neat | 앱 설정에만 |
| (앱만) view | terminal · overlay · both | 앱 설정과 `init` |
| `ignore` | 손으로만 | 손으로만(설정 화면에 넣지 않는다) |

앱의 설정 창(`⌘,`, ⋯ → 설정…)은 탭 넷이다: **이 보드 · 이 컴퓨터 · 터미널과 에이전트 · 정보**. 탭은 화살표·Home·End로 옮기고, Tab으로 그 탭의 칸에 들어간다. 탭 안이 창보다 길면 그 탭만 스크롤하고 아래 끝이 흐려져 더 있다는 걸 보인다. "이 보드"(테마·스타일·보드 언어)는 아래처럼 `sticky.json`에, "이 컴퓨터"(앱 언어, 모드 시스템·다크·라이트, 움직임 시스템·줄이기·끄기, 단축키)는 `~/.stickypane/overlay.json`에 쓴다. "터미널 명령 설치"·"에이전트 연결"은 #22에서 채운다.

설정은 모두 루트 `sticky.json`에 쓰여 터미널과 앱이 같은 값을 본다. bridge: `settings.get`(`time`·`editor`·`stale`도 읽는다) / `settings.set {theme|language|style|view}`. **미구현**: `settings.set`의 `time`·`editor`·`stale`(지금은 터미널 `S`나 `stickypane config`로만 바꾼다).

## 10.6 마우스 · 알림

| 터미널 | 앱 |
|---|---|
| 클릭(탭, 칩, 위젯), 더블클릭 크게 보기, 휠, 확인의 Yes 클릭. 끌기 없음 | 전부 + **끌기**(카드 이동, 노트 순서, 노트를 다른 탭으로, 높이) |
| 상태 줄 한 줄(다음 키까지) | 하단 토스트(손그림 메모지), 3초 뒤 사라짐, 되돌리기 같은 버튼이 있으면 6초 |
| `•` 바뀐 노트 | 제목 옆 손그림 점, 핀 모드에서는 바뀐 순간 하이라이트([04](04-motion.md)) |

## 10.7 bridge에 더할 것 (core Task 13b)

| 메시지 | 하는 일 | 재사용 |
|---|---|---|
| `note.arrange {note, open?, size?, rows?, pin?, color?, title?}` | `note.view`를 대체한다. 터미널과 같은 저장 규칙 | `arrange.Change` (`stickypane set`과 같은 길) |
| `tab.order {tab, order[]}` | 노트 순서 | 터미널 `{` `}`이 쓰는 함수 |
| `note.move {note, to}` | 폴더·책·최상위로 | `api.Move` |
| `note.archive` / `note.trash` / `note.restore {note}` | 보관·휴지통·되돌리기 | `api.Archive` / `api.Remove` / `api.Restore` |
| `note.send {note, request}` | api 요청 보내기 | 터미널의 send 경로를 runner처럼 낮은 레이어로 |
| `kinds.list` | 카탈로그 | `widget.Registry` |
| `board.index` | 찾기·링크 패널 | `api.Index` |
| `board.load` 보강 | 노트의 `linked`, `target`, `error`, `color`, `title`; settings의 `tab`, `time`, `editor`, `stale` | `store.Load`, `prefs` |
| 변경 뒤 `Derive` | 파생 차트를 터미널처럼 갱신 | `api.Derive` |
| chat `say`의 `who` | 비어 있으면 터미널과 같은 사용자 이름 규칙 | 터미널 chat 경로 |
