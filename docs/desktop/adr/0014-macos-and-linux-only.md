# ADR-0014. 앱은 macOS와 Linux만 지원한다

| 항목 | 값 |
|---|---|
| 상태 | Proposed |
| 날짜 | 2026-10-28 |
| 관련 요구 | R-30 |

## 맥락

Wails v3는 Windows도 지원하지만 사용자는 지원 영역을 줄이고 이 Mac에서 먼저 시험하길 원한다(R-30).

## 결정

앱은 macOS(arm64, amd64)와 Linux(amd64, arm64)용만 만든다. 본체 터미널 board는 지금처럼 Windows도 지원한다. Linux는 CI 빌드·테스트까지 하고 실기 확인은 [미검증]으로 둔다.

## 결과

- `view=overlay`인 프로젝트를 Windows에서 열면 터미널로 열고 한 줄로 안내한다.
- Wayland의 전역 단축키는 보장하지 않는다. 트레이와 `stickypane`으로 연다.
