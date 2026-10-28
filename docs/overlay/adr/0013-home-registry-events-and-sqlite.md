# ADR-0013. 세션 레지스트리는 `~/.stickypane/`의 날짜별 사건 파일과 앱의 SQLite 색인이다

| 항목 | 값 |
|---|---|
| 상태 | Proposed |
| 날짜 | 2026-10-28 |
| 관련 요구 | R-27, R-28, R-29 |

## 맥락

앱 홈은 이 컴퓨터의 stickypane 프로젝트와 활성 세션을 보여 줘야 한다(R-27). 프로젝트가 지워지거나 옮겨지고, 프로세스가 죽는 일이 생긴다(R-29). 사용자는 홈 폴더의 `.sticky` 같은 폴더를 원했고, JSON 대신 SQLite를 고려하라고 했다(R-28). 본체는 표준 라이브러리 먼저 규칙과 VISION 1(상태는 파일)을 지킨다.

## 결정

`~/.stickypane/`(`STICKYPANE_HOME`으로 바꿀 수 있음)이 **있을 때만** 본체가 날짜별 `events/YYYY-MM-DD.jsonl`을 덧붙이고 오래 사는 프로세스마다 `run/<pid>.json`을 둔다. overlay 앱이 이를 읽어 `stickypane.db`(SQLite, WAL, `modernc.org/sqlite`)에 색인한다. 시작할 때와 홈을 열 때 사라진 프로젝트·죽은 프로세스를 정리한다([08 §8.4](../08-app-home-and-sessions.md#84-레지스트리--sticky)).

## 결과

- 이름이 `~/.sticky`가 아닌 이유: `store.Find`가 위로 올라가며 찾는 board 폴더와 겹친다.
- 폴더는 overlay 앱이 처음 실행될 때 만든다. 터미널만 쓰는 사람에게는 아무것도 기록되지 않는다.
- 본체에 새 모듈이 들어오지 않는다. SQLite는 overlay 모듈에만 있다. (#24부터 GORM과 순수 Go 드라이버 `glebarez/sqlite`로 쓴다. 파일과 스키마는 같다.)
- 기록 파일을 쓰고 지우는 것은 본체뿐이다(#32). 앱은 읽기만 한다.
- 진실의 원천은 파일이다. `stickypane.db`는 지워도 `events/`에서 다시 만든다.
- 레지스트리 쓰기는 best-effort라 홈 폴더에 쓸 수 없어도 명령이 실패하지 않는다.
- 프로젝트 경로가 홈 폴더 파일에 남는다. README에 무엇을 기록하는지 적는다.
