# ADR-0018. 앱은 도메인 · 포트 · 유스케이스 · 어댑터 · 화면 층으로 나누고 테스트로 지킨다

| 항목 | 값 |
|---|---|
| 상태 | Accepted |
| 날짜 | 2026-10-29 |
| 관련 요구 | R-47, R-48, #24, #32 |

## 맥락

앱의 색인(`desktop/internal/index`)은 손으로 쓴 SQL, 파일 읽기, `ps` 실행, 규칙이 한 패키지에 섞여 있었다. 레지스트리 형식을 따로 베꼈고, `sticky.json`을 직접 읽었고, `~/.stickypane`의 파일을 지웠다. 그래서 본체와 앱이 같은 파일을 두 곳에서 쓰는 일이 생겼다(#32). 사용자는 SQL을 ORM으로 옮기고(R-47), 쓰는 쪽과 저장소를 비즈니스 규칙에서 떼어 층으로 나누라고 했다(R-48).

## 결정

앱 모듈은 안에서 밖으로 다섯 층이다([02 §2.6.1](../02-system-architecture.md#261-앱-모듈의-레이어)).

1. **도메인**: 값과 규칙만 둔다(`index/model`, `prefs`). 파일, DB, 프로세스, 시계, Wails를 import하지 않는다.
2. **포트**: 유스케이스가 바깥에 바라는 것을 인터페이스로 둔다(`index/ports`). 저장소들과 작업 단위(`Store.Write`, 시작할 때 쓰기 잠금을 잡는 트랜잭션), `Records`, `Boards`, `Clock`, `ProcessChecker`.
3. **유스케이스**: 흐름과 트랜잭션 경계를 맡는다(`index/service`). 포트로만 일한다.
4. **어댑터**: 포트를 구현한다. GORM + `glebarez/sqlite`(`gormstore`), `~/.stickypane` 읽기(`registryfs`), 프로젝트 폴더(`boardfs`), `ps`(`procs`), bridge 클라이언트.
5. **조립과 화면**: `index.Open`이 부품을 잇고 셸과 `main.go`가 창에 낸다.

`desktop/internal/layers`의 테스트가 두 가지를 확인한다. 하나는 층이 아래로만 import하는지이고, 다른 하나는 도메인·포트·유스케이스가 `os`·`io`·`net`·`database/`·GORM·드라이버·Wails를 import하지 않는지다. 새 패키지는 그 표에 층을 받아야 들어온다.

레지스트리는 앱 모듈이 본체의 `internal/adapter/registry`를 import할 수 없으므로 `registryfs`와 `model`의 구조체로 읽는다. 대신 본체가 쓰고 고정하는 골든 파일(`internal/adapter/registry/testdata/home`)을 앱의 테스트가 상대 경로로 그대로 읽는다. 형식이 바뀌면 본체 테스트가 골든을 다시 쓰게 하고, 앱 테스트가 깨진다.

## 결과

- `registry.Read`를 공개 패키지로 꺼내 `replace ../`로 의존하는 길은 택하지 않았다. 앱 모듈이 본체 트리와 모듈 그래프에 묶이고, `Read`는 모든 기록을 매번 통째로 읽는다. 색인은 파일마다 바이트 오프셋을 트랜잭션 안에서 읽어 한 줄을 한 번만 반영해야 하는데, `Read`로는 그 일을 할 수 없다. bridge 메시지 타입을 베껴 두는 방식(ADR-0002)과도 같은 길이다.
- 앱은 `~/.stickypane`의 파일을 지우지 않는다. 죽은 `run/` 파일은 세션 종료로만 반영하고, 파일은 본체의 `registry.Prune`이 지운다.
- 앱은 `sticky.json`을 읽지 않는다. 제목은 폴더 이름으로 시작하고, 셸이 bridge의 `board.load` 답에서 `title`을 받아 `Retitle`한다.
- 유스케이스는 DB 없이 가짜 포트로 시험할 수 있다. SQLite의 동작(동시 열기, 깨진 파일, 동시 ingest)은 조립된 `index` 테스트가 계속 지킨다.
- DB 파일과 스키마는 첫 버전과 같다. GORM 마이그레이터가 만드는 DDL도 같은 열·키·외래 키이고, 첫 버전이 만든 DB를 그대로 연다(`gormstore` 테스트).
- 본체 쪽에도 같은 원칙을 적용한다(RA13). 손대는 곳부터 옮기고, 본체의 `internal/layers`가 같은 검사를 맡는다.
