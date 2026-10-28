# 설정

설정할 것은 없습니다. 모든 설정에 기본값이 있고, 보드에서 `S`를 누르거나
`stickypane config`로 바꿉니다. 테마 목록과 `sticky.json`의 키는 영문
[configuration.md](../configuration.md)에 있습니다.
[README로 돌아가기](../../README.ko.md)

## 배치는 sticky.json에

배치(열림·크기·색·고정·순서·이름·테마·언어)는 노트가 아니라 `.sticky/sticky.json`에
저장됩니다. 보드가 쓰는 파일이라 직접 고칠 일은 `ignore` 목록뿐입니다. 그래서
화면을 바꿔도 노트 파일은 다시 쓰이지 않고, 에이전트가 쓰는 노트와 부딪히지
않습니다. `view`(`terminal` `overlay` `both`)와 `style`(`doodle` `neat`)은 만들고
있는 데스크톱 앱용이며 터미널은 무시합니다.

## 이 컴퓨터에 남는 기록

`~/.stickypane`(또는 `$STICKYPANE_HOME`) 폴더가 **이미 있을 때만** 씁니다. 폴더는
stickypane이 만들지 않고 데스크톱 앱이 만들므로, 터미널만 쓰면 홈에 아무것도
생기지 않습니다. 있으면 보드 목록을 보여 주려고 두 가지를 적습니다.

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
