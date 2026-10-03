# stickypane

**코딩 에이전트가 쓰고 당신이 읽는, 터미널 속 보드.**

[English](README.md) · 한국어

> 이 문서는 `README.md`의 번역이며 요약본입니다(커밋 `8e8b745` 기준). 영문이
> 원본이고 더 자세합니다. 두 문서가 다르면 영문이 맞습니다.

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

```sh
brew install --cask LeeSwallow/tap/stickypane
# 또는
go install github.com/LeeSwallow/stickypane/cmd/stickypane@latest
```

macOS와 Linux를 지원합니다.

## 시작

```sh
cd your-project
stickypane init          # .sticky/ 를 만들고 에이전트에게 알립니다
stickypane               # 보드를 엽니다
stickypane language ko   # 화면을 한국어로
```

`init`은 `AGENTS.md`나 `CLAUDE.md`에 짧은 안내문을 넣습니다. `init --skill`은
대신 Claude Code 스킬로 설치합니다. 플러그인으로 쓰려면:

```sh
claude plugin marketplace add LeeSwallow/stickypane
claude plugin install board@stickypane
```

에이전트 옆에 띄우기: `tmux split-window -h stickypane`, `tmux display-popup -E stickypane`,
`wezterm cli split-pane --right -- stickypane`.

## 폴더가 곧 보드

`.sticky/`의 파일 하나가 노트 하나입니다. 확장자가 종류를 정합니다.

| 파일 | 무엇 |
| --- | --- |
| `*.md` | 노트. 머리말 `type`으로 `board`(칸반), `checklist`, `chart`, `form`, `log` |
| `*.log` `*.txt` `*.out` | 로그. 그대로 보여 주고 자라는 대로 따라감 |
| `*.sh` | 스크립트. 실행 버튼이 달림 (매번 y/n 확인 후 프로젝트 폴더에서 실행) |
| 폴더 | 페이지가 있는 노트. 파일 하나가 페이지 하나, `,` `.`으로 넘김 |

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
`docs/superpowers/specs/`(설계 문서, 한국어). 기여 방법은
[CONTRIBUTING.md](CONTRIBUTING.md), 번역은 [docs/translating.md](docs/translating.md).

## 라이선스

MIT. 쓰인 팔레트와 라이브러리는 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)에.
