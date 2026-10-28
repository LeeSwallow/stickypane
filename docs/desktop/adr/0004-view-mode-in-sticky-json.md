# ADR-0004. 보기 방식은 `stickypane init`이 묻고 `sticky.json`에 둔다

| 항목 | 값 |
|---|---|
| 상태 | Proposed |
| 날짜 | 2026-10-28 |
| 관련 요구 | R-06, R-12, R-13 |

## 맥락

보기 방식(터미널 / overlay / 둘 다)은 프로젝트마다 다를 수 있다(R-12). VISION 4는 손으로 쓰는 설정 파일을 금한다.

## 결정

`sticky.json` 루트에 `"view": "terminal" | "overlay" | "both"`를 둔다. 없으면 `terminal`. `stickypane init --view <값> --style <값>` 플래그로 정하고, 터미널(TTY)에서 사람이 직접 `init`을 실행할 때만 묻는다. 에이전트·앱·자동 init은 묻지 않는다(멈추면 안 된다). 앱의 빈 상태에서 폴더를 고르면 `--view overlay`를 넘긴다. `stickypane`(인자 없음)은 이 값에 따라 TUI나 앱을 연다.

## 결과

- 혼자 쓰는 레포에서는 커밋되어도 문제없다(R-13).
- 앱이 설치되지 않은 곳에서는 터미널로 대신 열고 한 줄로 안내한다.
- 에이전트 가이드의 "`sticky.json`은 건드리지 않는다"가 그대로 적용된다.
