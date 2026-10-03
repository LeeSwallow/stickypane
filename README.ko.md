# stickypane

**코딩 에이전트가 쓰고 당신이 읽는, 터미널 속 보드.**

Claude Code, Codex 같은 AI 코딩 에이전트를 위한 칸반, 체크리스트, 차트, 폼,
실시간 로그. 평범한 Markdown 파일과 CLI, MCP 서버, Go로 만든 터미널 UI(TUI)로
이루어져 있고 tmux, WezTerm 등 어느 터미널에서든 에이전트 옆에 띄웁니다.

[English](README.md) · 한국어

> 이 문서는 `README.md`의 번역이며 요약본입니다(커밋 `8e8b745` 기준이고, '설치',
> '시작', '기여' 절과 폴더 표는 그 뒤의 변경까지 반영했습니다). 영문이 원본이고 더
> 자세합니다. 두 문서가 다르면 영문이 맞습니다.

```
 ▦ Auth work   ☑ Login API   ≣ build   ▤ Tokens   ◉ Deploy now?   ✎ notes          1/1
╔ ▦ Auth work ═══════════════════════════════════════════════════════ 4 cards ╗
║ To do (1)               Doing (1)               Done (2)                       ║
║ ▎ payments              › login API             ▎ schema                       ║
╚═════════════════════════════════════════════════════════════════════════════════╝
╭ ☑ Login API ─────────────── 2/3 ╮╭ ≣ build ──────────────────────── 212 lines ╮
│   ☑ Add endpoint                ││ ok   internal/store      0.41s             █
│   ☐ Write tests                 ││ [exit 0 · 7.15s]                           █
╰─────────────────────────────────╯╰────────────────────────────────────────────╯
```

옆 pane에서 일하는 에이전트는 글을 많이 쏟아내고, 정작 중요한 것은 그 글과
함께 흘러 올라가 버립니다. 계획이 어디까지 왔는지, 무엇이 끝났는지, 당신이 뭘
결정해 주길 바라는지. 그건 채팅에 둘 게 아니라 보드에 둘 것입니다.

stickypane이 그 보드입니다. 평범한 파일이 든 폴더(`.sticky/`) 하나를 터미널
창에 패널로 보여 줍니다. 에이전트가 파일을 쓰면 나타나고, 항목을 체크하면
체크되고, 질문을 하면 버튼 달린 폼이 뜹니다. 당신이 버튼을 누르면 에이전트가
답을 읽어 갑니다.

> **상태: 설계 단계.** 1.0 전이고 보드가 어떻게 동작해야 하는지 아직 정하는
> 중입니다. 파일 이름, 명령 이름, `sticky.json`이 v0.1.0 전까지 바뀔 수
> 있습니다. 지금 가장 도움이 되는 건 에이전트와 하루 써 보고 "보드가 이렇게
> 해 주길 바랐다"를 알려 주시는 것입니다:
> [디자인 피드백 이슈](https://github.com/LeeSwallow/stickypane/issues/new?template=design_feedback.yml)나
> [Discussions](https://github.com/LeeSwallow/stickypane/discussions).

## 설치

프로그램 하나이고 따로 필요한 것이 없습니다. macOS, Linux, Windows에서, 유니코드와
색을 보여 주는 터미널이면 어디서나 돕니다. 특별한 글꼴도 필요 없습니다.

```sh
brew install --cask LeeSwallow/tap/stickypane                      # Homebrew
go install github.com/LeeSwallow/stickypane/cmd/stickypane@latest  # Go 1.26 이상
```

[릴리스](https://github.com/LeeSwallow/stickypane/releases)에는 시스템마다 압축
파일이 있습니다(`darwin`, `linux`, `windows` × `amd64`, `arm64`). 풀어서 나온
`stickypane`을 PATH에 있는 폴더에 두면 됩니다. Windows는 zip을 풀어 PATH에
추가합니다. 명령 예시와 체크섬 확인, macOS Gatekeeper 처리는 영문 README의
[Install](README.md#install)에 있습니다.

```sh
stickypane version   # 설치 확인
stickypane env       # 찾은 시스템, 셸, 창 도구, 언어, 편집기
```

지우려면 플러그인을 넣었을 때 `stickypane setup --undo`를 먼저 하고, 설치한
방법대로 지웁니다. 노트는 각 프로젝트의 `.sticky/`에 평범한 파일로 남습니다.

## 시작

**1. 에이전트 옆에 보드를 띄웁니다.** 프로젝트 폴더에서 창을 나눠 `stickypane`을
실행합니다. `stickypane env`가 지금 터미널에 맞는 명령을 알려 줍니다.

| 터미널 | 명령 |
| --- | --- |
| tmux | `tmux split-window -h stickypane` |
| WezTerm | `wezterm cli split-pane --right -- stickypane` |
| Zellij | `zellij run --direction right -- stickypane` |
| Windows Terminal | `wt -w 0 split-pane -V stickypane` |

**2. 처음 실행하면 프로젝트를 준비합니다.** git 저장소 맨 위에 `.sticky/`를 만들고
에이전트에게 보드를 알린 뒤 엽니다. 설정할 것은 없습니다. 모든 설정에 기본값이
있고, 보드에서 `S`로 바꿉니다. 화면을 한국어로 하려면 `stickypane language ko`.

**3. 에이전트에게 한 번 알려 줍니다.** 셋 중 하나입니다.

| 방법 | 명령 | 대상 |
| --- | --- | --- |
| 플러그인 | `stickypane setup` | Claude Code, Codex: 스킬 다섯 개, 슬래시 명령, 세션 훅 |
| 안내문 | 2단계가 `CLAUDE.md`나 `AGENTS.md`에 넣습니다 | 그 파일을 읽는 모든 에이전트 |
| MCP | `stickypane setup --mcp` | MCP를 쓰는 모든 클라이언트 |

**4. 시키면 됩니다.** "이 리팩터 체크리스트를 보드에 띄워 줘", "배포 전에 보드에서
물어봐". 에이전트가 파일을 쓰면 보드에 나타나고, 보드에서 한 일은 같은 파일에
남습니다.

플러그인은 계층으로 되어 있습니다. 입구 스킬 `using-the-board`가 일에 맞는 스킬을
알려 줍니다: 진행 추적(`tracking-progress`), 묻기(`asking-the-user`), 채팅
(`talking-in-chat`), 노트 연결(`connecting-notes`).

## 폴더가 곧 보드

`.sticky/`의 파일 하나가 노트 하나입니다. 확장자가 종류를 정합니다.

| 파일 | 무엇 |
| --- | --- |
| `*.md` | 노트. 머리말 `type`으로 `board`(칸반), `checklist`, `chart`, `form`, `log` |
| `*.log` `*.txt` `*.out` | 로그. 그대로 보여 주고 자라는 대로 따라감 |
| `*.sh` | 스크립트. 실행 버튼이 달림 (매번 y/n 확인 후 프로젝트 폴더에서 실행) |
| 폴더 | 탭. 화면 하나를 따로 가지며 그 안의 파일이 노트 |
| 탭 안의 폴더 | 페이지가 있는 노트. 파일 하나가 페이지 하나, `,` `.`으로 넘김 |

마크다운 노트 안의 ```mermaid 블록은 그림으로 그려집니다(순서도, 시퀀스, ER).
`stickypane show README.md`처럼 프로젝트의 어떤 파일이든 복사 없이 보드에
올릴 수 있습니다(심볼릭 링크).

배치(열림·크기·색·고정·순서·이름·테마·언어)는 노트가 아니라 `.sticky/sticky.json`에
저장됩니다. 보드가 쓰는 파일이라 직접 고칠 일은 `ignore` 목록뿐입니다.

## 키

| 노트 | | 열린 칸반·체크리스트 | |
| --- | --- | --- | --- |
| `tab` `shift+tab` | 다음, 이전 노트 | `h` `l` | 칸 바꾸기 |
| `enter` | 열기, 그다음 크게 | `j` `k` | 카드·항목 바꾸기 |
| `o` | 열기 / 닫기 | `H` `L` | 카드를 옆 칸으로 |
| `+` `-` | 키우기 / 줄이기 | `J` `K` | 카드 순서 |
| `{` `}` | 앞으로 / 뒤로 옮기기 | `space` | 체크 |
| `m` | 폴더로 옮기기 | `n` | 새 카드·항목 |
| `e` `E` | 여기서 편집(vi 방식), `nvim`로 | **폼** | |
| `x` `D` `u` | 보관, 삭제(휴지통), 되돌리기 | `enter` `space` | 고르기, 입력, 누르기 |
| `T` | 다음 테마 | **크게 본 노트** | |
| `[` `]` | 이전 / 다음 화면 | `esc` | 뒤로 |
| `?` `q` | 도움말, 종료 | | |

마우스: 제목 클릭(열기·포커스·닫기), 노트 클릭(포커스, 항목 체크), 더블 클릭(크게),
휠(그 노트 스크롤). tmux에서는 `set -g mouse on`이 필요합니다.

## 에이전트용 명령

```sh
stickypane show plan                 # 띄우기 (경로를 주면 링크해서 띄움)
stickypane todo plan add "테스트 작성"      # plan.md: 0/1
stickypane todo plan check 테스트           # plan.md: 1/1
stickypane card work move 로그인 --to Done  # work.md: 2 cards
stickypane chart tokens add input 800       # tokens.md: input = 2,000
stickypane log worklog --time "테스트 통과"
stickypane set plan open=true size=half
stickypane mv plan docs/   /   rm plan   /   restore plan
stickypane wait deploy --timeout 10m        # 폼의 버튼이 눌릴 때까지
```

없는 노트는 만들어지고, 항목·카드·칸은 글자 일부나 `#2` 같은 순번으로 가리킬 수
있습니다. `stickypane mcp`는 같은 명령을 MCP 도구로 내놓습니다.

## 설계

[VISION.md](VISION.md)(원칙), [ROADMAP.md](ROADMAP.md)(계획과 열린 질문),
`docs/superpowers/specs/`(설계 문서, 한국어), [docs/architecture.md](docs/architecture.md)(코드 구조).

## 기여와 문의

지금은 코드보다 "보드가 이렇게 해 주길 바랐다"는 피드백이 더 도움이 됩니다.
아이디어가 이슈, 제안, 풀 리퀘스트, 릴리스가 되는 과정과 이슈를 맡는 방법, 개발
환경은 [CONTRIBUTING.md](CONTRIBUTING.md)에 있습니다. 시작하기 좋은 이슈에는
`good first issue`나 `accepted` 라벨이 붙습니다. 코딩 에이전트는
[AGENTS.md](AGENTS.md)를 먼저 읽습니다. 질문은
[Discussions](https://github.com/LeeSwallow/stickypane/discussions), 버그는
[이슈](https://github.com/LeeSwallow/stickypane/issues/new/choose), 보안 문제는
[SECURITY.md](SECURITY.md)대로 비공개로 알려 주세요. 번역은
[docs/translating.md](docs/translating.md).

## 라이선스

[MIT](LICENSE). 쓰인 라이브러리는 모두 MIT나 BSD 라이선스이고, 테마 팔레트와 함께
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)에 있습니다. 릴리스 압축 파일마다
그 라이선스 원문이 들어 있습니다.
