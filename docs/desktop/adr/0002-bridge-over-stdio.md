# ADR-0002. overlay 앱은 `stickypane bridge`와 stdio JSON으로 대화한다

| 항목 | 값 |
|---|---|
| 상태 | Proposed |
| 날짜 | 2026-10-28 |
| 관련 요구 | R-04 |

## 맥락

Go의 `internal/` 패키지는 다른 모듈에서 import할 수 없다. overlay를 같은 모듈에 두면 앱 셸의 의존성이 본체 `go.mod`에 들어온다(반목표 X5). HTTP 서버는 VISION 4와 충돌한다. MCP 서버는 에이전트용 텍스트 도구라 구조화된 데이터와 변화 알림이 없다.

## 결정

본체에 새 front end `internal/surface/bridge`(layer 50)와 `stickypane bridge` 명령을 둔다. overlay 앱이 이를 자식 프로세스로 띄우고 newline-delimited JSON으로 요청(`board.load`, `note.act`, `note.create`, `note.open`, `settings.*`)과 알림(`board.changed`)을 주고받는다. 프로토콜에는 버전이 있다.

## 결과

- 네트워크 listener가 없다.
- 본체가 진실의 원천으로 남고, 앱 셸을 바꿔도 본체는 바뀌지 않는다.
- 앱은 `stickypane` 바이너리가 PATH에 있어야 한다. 프로토콜 버전이 맞지 않으면 업데이트를 안내한다.
- 스크립트 실행을 `internal/surface/app`에서 낮은 레이어로 옮기는 리팩터가 필요하다.
