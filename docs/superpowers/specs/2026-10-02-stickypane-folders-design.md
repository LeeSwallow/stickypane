# stickypane 구조 확장: 고정 창, 폴더, 여러 파일 종류, 설정 파일

2026-10-02. 사용자가 실제 화면을 써 보며 준 요구를 묶은 문서다. 앞선 문서(`2026-10-01-stickypane-open-notes-design.md`)의 화면 구조를 일부 뒤집는다.

## 1. 요구 (사용자 말 그대로)

1. "화면에 꽉찬 노트들을 원해. 드래그해서 내리는 건 막아 놓고, 노트는 fixed이지만 노트 내부를 드래그해서 확인할 수 있는 구조"
2. "노트 내부에도 페이지(파일로 나눠짐), 스크롤이 있는 느낌"
3. "폴더 기반으로 마크다운, 로그, chart 등등을 읽어오고 불러올 수 있도록. 관련 설정은 json이든 뭐든에 저장하고 불러올 수 있으면 공유도 가능하고 readme 같은 거 읽기 좋을 듯. `.sticky` 같은 걸로 (shell 스크립트, 차트, 마크다운, 로그 등등 저장하고 쉽게 불러오는 구조, 로그는 실시간으로 읽어서 시각화)"
4. "수량 chart 증가나 todo list 관리 같은 건 LLM에게 일임하지 말고 api나 skill로 제공하면 모든 파일을 안 읽어도 되지 않으려나"
5. "nvim처럼 수정 가능하게, 페이지네이션 가능하게"

## 2. 결정

### 2-1. 화면: 고정 창 (구현됨)

- 열린 노트는 화면을 빈틈없이 나눠 갖는 창이다. 화면 전체는 스크롤하지 않는다.
- 가로: `size`대로 줄을 만들고(`page` 한 줄, `half` 둘이 한 줄, `card` 셋 이상), 줄마다 폭을 끝까지 늘린다.
- 세로: 줄들이 높이를 나눈다. 짧은 노트는 필요한 만큼만, 긴 노트들이 나머지를 똑같이 나눈다. 남는 높이도 똑같이 나눠 화면이 항상 꽉 찬다.
- 긴 노트는 창 안에서 스크롤한다. 오른쪽 테두리에 스크롤 막대가 나온다. 포커스한 노트는 `j` `k` `g` `G`와 페이지 키로, 마우스는 포인터 아래 노트를 휠로 스크롤한다.
- 한 창에 최소 10줄을 준다. 그보다 줄어들 노트는 다음 화면으로 넘어가고 `[` `]`로 넘긴다. 제목 줄 아래에 `2/3`이 나온다.
- 로그는 끝을 따라간다. 위로 스크롤하면 멈추고, 다시 끝으로 내리면 따라간다.

앞선 문서의 "열린 노트는 자르지 않는다"(6절)는 이 결정으로 대체한다.

### 2-2. 폴더 = 페이지가 있는 노트 (책)

- 노트 폴더 안의 하위 폴더 하나가 창 하나다. 그 안의 파일 하나가 페이지 하나다.
- 창 테두리에 `폴더 이름 · 페이지 제목`과 `2/5`가 나온다. `,` `.`(또는 `<` `>`)로 페이지를 넘긴다. 페이지마다 스크롤 위치를 따로 기억한다.
- 폴더는 한 단계만 본다. `archive/`와 `.`으로 시작하는 폴더는 보지 않는다.
- 보드 조작(체크, 카드 이동, 편집, 삭제)은 지금 보이는 페이지의 파일에 적용한다. 열기·크기·색·고정은 폴더(창)에 적용한다.
- 명령과 MCP에서는 `docs/guide`처럼 폴더를 붙여 페이지를 가리킨다.

"보드 전체를 여러 화면(대시보드)으로 나누는 폴더"는 만들지 않았다. 열린 노트가 넘치면 화면이 자동으로 나뉘고(2-1), 주제별로 묶고 싶으면 폴더로 묶어 창 하나로 만들면 된다. 필요해지면 그때 더한다.

### 2-3. 파일 종류

확장자가 모양을 정한다. 마크다운만 머리말을 읽는다.

| 확장자 | 모양 | 동작 |
|---|---|---|
| `.md` | 머리말의 `type` (없으면 일반 노트) | 지금과 같다 |
| `.log` `.txt` `.out` | 로그 | 파일을 그대로 보여 주고 끝을 따라간다. 1MB가 넘으면 끝부분만 읽는다 |
| `.sh` | 스크립트 | 내용을 보여 주고 `[ Run ]` 버튼을 단다. 누르면 확인을 받고 프로젝트 폴더에서 실행한다. 출력은 옆의 `<이름>.log`에 실시간으로 쌓이고 그 로그가 창으로 열린다 |

- 다른 확장자는 보지 않는다.
- 프로젝트의 다른 파일(README 등)은 링크로 붙인다: `stickypane link README.md`가 노트 폴더에 심볼릭 링크를 만든다. 링크된 파일은 보드가 고치더라도 원래 자리에서 고친다(이미 구현된 규칙).

### 2-4. 배치는 `sticky.json`, 내용은 파일

- 열림·크기·높이·색·고정은 노트 폴더의 `sticky.json`에 노트 이름별로 저장한다. 보드에서 `o` `+` `-` `c` `p`를 누르면 이 파일만 바뀐다.
- 이유: 로그·스크립트·링크된 README에는 머리말을 쓸 수 없다. 그리고 보드가 배치를 바꿀 때마다 에이전트의 파일을 고쳐 쓰는 일이 없어진다. 이 파일을 커밋하면 배치를 공유할 수 있다.
- 마크다운 머리말의 `open` `size` `rows` `color` `pin`은 "처음 배치"로 계속 읽는다. `sticky.json`에 값이 있으면 그것이 이긴다.
- 에이전트가 배치를 바꾸려면 `stickypane set <이름> open=true`를 쓴다. 이 명령은 배치 키를 `sticky.json`에 쓰고, 나머지 키(`title`, `type`, `view`)는 머리말에 쓴다.

```json
{
  "notes": {
    "00-linear.md": { "open": true, "size": "page", "color": "blue", "pin": true },
    "docs": { "open": true, "size": "half" },
    "build.log": { "open": true, "rows": 12 }
  }
}
```

### 2-5. 폴더 이름

새 프로젝트는 `.sticky/`를 만든다. `.stickypane/`도 계속 읽는다(둘 다 있으면 `.sticky/`).

### 2-6. 작은 수정 명령 (구현됨)

`todo`, `card`, `chart`, `log`, `set`. 에이전트가 노트를 읽지도, 통째로 다시 쓰지도 않고 한 가지만 바꾼다. 없는 노트는 만든다. MCP 도구도 같다. `init --skill`은 안내문을 Claude Code 스킬로 설치한다.

### 2-7. 내장 편집기 (구현됨)

`e`로 노트 파일을 vi 방식으로 고친다. `E`는 `$EDITOR`.

### 2-8. 삭제·이동 (구현됨, 오픈소스 조사 기반)

sampler, wtf, zellij, lazygit, yazi, taskwarrior-tui, kanban-md, redthread, tuiboard, Backlog.md, sidecar, 그리고 resterm과 Bruno의 소스를 읽고 공통 패턴을 따랐다.

- 삭제는 지우지 않는다. `D`는 확인 뒤 `.sticky/.trash/`로, `x`는 `archive/`로 옮기고 `u`가 마지막 것을 되돌린다 (yazi의 trash, redthread·taskwarrior-tui의 undo).
- `{` `}`로 이웃과 자리를 바꾼다 (yazi 탭, redthread 보드). 순서는 `sticky.json`의 `order`에 쓰고, 없는 노트는 이름순으로 뒤에 선다 (Bruno의 `seq`: 없는 항목도 자리를 가진다).
- `m`은 옮길 폴더 목록을 띄우고 Enter로 확정, Esc로 취소한다 (kanban-md `m`, Backlog.md 이동 모드).
- 명령줄은 화면과 같은 저장소 함수를 쓴다: `rm` `restore` `archive` `mv` `link` (kanban-md의 CLI 미러링). 어느 것도 다른 노트를 덮어쓰거나 `.sticky/` 밖으로 나가지 않는다 (Bruno의 `validatePathIsInsideCollection`, 충돌 시 접미사).
- 표시 이름은 파일 이름과 다르다 (Bruno의 name/filename 분리): 폴더·로그·스크립트·링크된 파일의 이름은 `sticky.json`의 `title`에 두고 `R`로 바꾼다. 파일은 그대로다.
- `sticky.json`은 경로로 키를 잡고, `version`을 갖고, 임시 파일에 쓴 뒤 바꿔치기하며, 손으로 쓰는 `ignore` 목록을 둔다 (Bruno `bruno.json`의 `ignore`, sampler의 title 매칭이 깨지는 문제를 피함).
- 편집 중 파일이 바뀌면: 고친 게 없으면 조용히 다시 읽고, 있으면 버퍼를 지키고 알린다 (resterm).
- 휴지통은 `.sticky/.gitignore`로 git에서 뺀다 (resterm·Bruno의 init이 `.gitignore`를 쓰는 것을 따름).

따르지 않은 것: Bruno의 "공유 상태(구조)와 개인 상태(열린 탭)를 다른 파일에" 분리. `sticky.json` 하나에 열림까지 두었다. 혼자 쓰는 동안은 파일 하나가 단순하고, 팀이 커밋해서 쓰다가 충돌이 잦아지면 그때 `open`만 로컬 파일로 뺀다.

## 3. 구현 순서

1. 고정 창 배치 (끝남)
2. 작은 수정 명령, 스킬 (끝남)
3. 내장 편집기 (끝남)
4. 파일 종류와 폴더(책) (끝남)
5. `sticky.json` (끝남)
6. 스크립트 실행 (끝남)
7. `.sticky/` 이름, `link` 명령 (끝남)
8. 삭제·이동·되돌리기 (끝남, 2-8절)

## 4. 하지 않은 것

- 폴더를 대시보드(화면 묶음)로 쓰는 것. 2-2절 참고.
- `.csv`를 차트나 표로 그리는 것. 표 위젯과 함께 한다.
- 스크립트 출력에 입력을 보내는 것, 실행 중 중단.
- 편집기의 횟수 접두(`3dd`)와 비주얼 모드.

## 5. 디자인 시스템: 테마 (구현됨)

resterm, lazygit, zellij, yazi, k9s, crush와 Neovim 테마(catppuccin, tokyonight, gruvbox, nord, dracula)의 소스를 읽고 정했다.

- 화면의 모든 색은 **역할** 열두어 개로 그린다: `text`, `muted`, `accent`, `select`, `good`, `warn`, `bad`, `info`, 노트 색 여섯. 위젯은 역할 이름의 스타일만 쓰고, 테마가 바뀌면 전부 따라온다 (`internal/theme`).
- 관례를 따른 결정: 포커스는 테두리 **색**으로 보인다(모양만 바꾸지 않는다. lazygit `activeBorderColor`, zellij `frame_selected`, resterm `PaneBorderFocus`); 포커스 없는 창은 중립 회색 테두리에 제목만 노트 색; 선택 줄은 반전이 아니라 배경 색(catppuccin `Visual`, lazygit `selectedLineBgColor`); 키 힌트는 accent(lazygit `optionsTextColor`, k9s `menu.keyColor`); 밝은 배경에서는 `faint` 속성을 쓰지 않고 muted 색을 쓴다(resterm `theme_runtime.go`).
- 테마는 배경을 칠하지 않는다. 터미널 배경을 그대로 두고 그에 맞는 전경색만 고른다. `auto`는 터미널이 보고한 배경에 따라 기본 dark/light를 고른다(lazygit의 `darkTheme`/`lightTheme` 방식).
- 내장 테마 10개: stickypane dark/light, catppuccin mocha/latte, tokyonight night/day, gruvbox dark/light, nord, dracula. 값은 각 테마 소스 그대로이고 README에 출처와 라이선스를 적었다.
- 선택은 `sticky.json`의 `theme`에 저장한다. 보드에서 `T`, 명령줄에서 `stickypane theme <이름>`.
- 하지 않은 것: 사용자 정의 테마 파일(resterm의 TOML 테마). 역할이 열두 개뿐이라 내장 테마로 충분할 때까지 미룬다.
