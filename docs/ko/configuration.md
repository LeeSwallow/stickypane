# 설정

설정할 것은 없습니다. 모든 설정에 기본값이 있습니다. 보드의 `S`로는 테마·언어·칸 수를
바꾸고, 시간 형식(`time`)·에디터(`editor`)·카드 멈춤 기준(`stale`)은
`stickypane config`로만 바꿉니다(`stickypane config stale 1h`).
`stickypane theme`과 `stickypane language`는 `stickypane config theme`,
`stickypane config language`의 줄임이고, 값 없이 쓰면 목록을 보여 줍니다.
테마 목록과 `sticky.json`의 키는 영문
[configuration.md](../configuration.md)에 있습니다.
[README로 돌아가기](../../README.ko.md)

## 배치는 sticky.json에

배치(열림·크기·색·고정·순서·이름·테마·언어)는 노트가 아니라 `.sticky/sticky.json`에
저장됩니다. 보드가 쓰는 파일이라 직접 고칠 일은 `ignore` 목록뿐입니다. 그래서
화면을 바꿔도 노트 파일은 다시 쓰이지 않고, 에이전트가 쓰는 노트와 부딪히지
않습니다. `view`(`terminal` `overlay` `both`)와 `style`(`doodle` `neat`)은 만들고
있는 데스크톱 앱용이며 터미널은 무시합니다.

## 키

터미널 보드의 키는 저마다 이름 있는 동작에 묶여 있고, 키를 바꿔도 이름은 그대로입니다.
`stickypane keys`가 동작마다 이름, 키, 하는 일(보드의 언어로)을 보여 주고, 바꾼 것 옆에는
`(keys.json)`을 붙입니다. 스크립트에는 `--json`.

```sh
stickypane keys                          # 동작과 키 전부
stickypane keys set note-archive X       # X로 보관, x는 더 이상 보관하지 않음
stickypane keys set jot N ctrl+n         # 키 여러 개; 힌트에는 첫 키가 나옴
stickypane keys set note-archive "d x"   # 연속 키: d 다음 x
stickypane keys set theme-next           # 키 없이: 그 동작은 키가 없음
stickypane keys reset note-archive       # 다시 x로
stickypane keys reset                    # 모든 키를 처음대로
```

바꾼 키는 `~/.stickypane`(또는 `$STICKYPANE_HOME`)의 `keys.json`에 동작마다 한 줄로
남습니다. 직접 고쳐도 됩니다.

```json
{
  "jot": ["N", "ctrl+n"],
  "note-archive": ["X"],
  "theme-next": []
}
```

- 파일에 없는 동작은 키가 그대로이고, `[]`은 키를 없앱니다.
- 키는 글자(`x`, `X`, `?`), 이름 있는 키(`enter`, `tab`, `space`, `esc`, `up`, `pgdown`,
  `f5`…), 또는 그 앞에 수식키(`ctrl+x`, `alt+enter`, `shift+tab`)입니다. `shift+x`는
  `X`로 씁니다. `ctrl+c`는 언제나 종료라 다른 동작에 줄 수 없습니다.
- 띄어 쓴 키는 차례로 누르는 연속 키입니다: `"g g"`, `"d x"`. 기다리는 키(`d…`)와 이어
  누를 수 있는 키가 아래 줄에 나옵니다. `esc`를 누르거나 어디에도 이어지지 않는 키를 누르면
  취소되고, 그 키도 함께 버려집니다. 숫자로 시작하거나(숫자는 횟수) `esc` 뒤로 이어지는 연속
  키는 쓸 수 없습니다. 도움말에는 `"g g"`가 `gg`로 나옵니다.
- 키 하나가 그 자체로 동작이면서 더 긴 키의 앞부분이면(`g`와 `g g`) 보드는 `g` 뒤에
  600ms를 기다렸다가 `g`를 실행합니다. 앞부분이기만 한 키는 다음 키를 끝까지 기다립니다.
- 키 앞의 숫자는 횟수입니다: `3J`는 노트를 세 줄 아래로 옮깁니다. 옮기기, 크기, 포커스, 쪽,
  화면, `(` `)`가 횟수를 받고(최대 99), 다른 동작은 한 번만 합니다. 숫자 하나만 누르면 같은
  시간을 기다린 뒤 그 탭으로 가고, `0`만 누르면 `0`의 동작, 숫자 뒤의 `0`은 숫자입니다(`10J`).
- 두벌식 한글 입력이 켜져 있어도 보드는 자모를 그 자리의 영문 키로 읽습니다: `ㅓ`는 `j`,
  `ㅌ`은 `x`, `ㅃ`은 `Q`. 입력 칸, 편집기, 검색은 친 그대로 받습니다. 파일에는 어느 쪽으로
  써도 됩니다.
- 한 키는 한 동작에만 묶입니다. 두 키를 맞바꾸려면 파일에서 두 동작을 함께 고치거나,
  키를 내줄 쪽을 먼저 바꾸세요.
- 보드가 쓸 수 없는 것(없는 동작, 읽을 수 없는 키, 두 동작이 원하는 키, JSON이 아닌 파일)은
  처음 키를 그대로 씁니다. 보드는 그래도 열리고, 무엇을 빼고 썼는지 아래 줄에 한 줄로
  알립니다. 이유는 `stickypane keys`가 알려 줍니다.
- 도움말(`?`), 아래 줄, 키를 말하는 메시지(`u 키로 되돌립니다`)도 파일을 따릅니다.
- 열린 노트의 키가 먼저입니다. 열린 칸반은 `h` `j` `k` `l`과 `H` `J` `K` `L`을, 체크리스트는
  `space`와 `n`을 보드의 동작보다 먼저 받습니다. 그런 키를 동작에 주면 그 노트에 포커스가
  있을 때만 빼고 동작합니다. 횟수는 노트의 키를 가로채지 않지만(칸반에서 `3j`는 `j`),
  이미 기다리는 연속 키는 가로챕니다.
- `~/.stickypane/keys.json`이 있다고 기록이 남지는 않습니다.

## 이 컴퓨터에 남는 기록

`~/.stickypane`(또는 `$STICKYPANE_HOME`)에 데스크톱 앱의 설정 파일 `overlay.json`이
**있을 때만** 씁니다. 폴더도 파일도 stickypane이 만들지 않고, 파일은 앱이 처음 켜질 때
씁니다. 그래서 터미널만 쓰면 다른 일로 폴더가 있어도 아무것도 기록되지 않습니다.
파일이 있으면 보드 목록을 보여 주려고 두 가지를 적습니다.

- `events/YYYY-MM-DD.jsonl`: 한 줄에 사건 하나, `{v: 1, at, type, …}`. 시각, 종류(`session.start`,
  `session.end`, `project.init`, `project.touch`), 프로젝트 경로, 세션이면 프로세스
  번호와 종류(`tui` `mcp` `watch` `bridge`), `project.touch`면 명령 이름(프로젝트·명령당
  1분에 한 번).
- `run/<pid>.json`: 세션이 살아 있는 동안 프로세스 번호, 프로젝트 경로, 종류, 시작
  시각. 세션이 끝나면 지워집니다. `run/touch-*` 파일은 1분에 한 번만 적기 위한 빈
  표시 파일입니다.

터미널이나 데스크톱 앱에서 보드를 열 때 stickypane이 필요 없어진 기록을 지웁니다.
프로세스가 죽은 세션의 `run/` 파일, 90일이 지난 사건, 하루 지난 표시 파일입니다.
이 파일들을 지우는 프로그램은 stickypane 하나뿐입니다.

노트 내용, 명령 인자, 파일 내용은 적지 않습니다. 밖으로 나가는 것도 없습니다. 폴더를
지우면 멈춥니다.
