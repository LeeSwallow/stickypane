# ADR-0015. Wails는 v3를 쓴다

| 항목 | 값 |
|---|---|
| 상태 | Proposed |
| 날짜 | 2026-10-28 |
| 관련 요구 | R-26, R-27 |

## 맥락

홈 창과 프로젝트별 창이 동시에 필요하고(다중 창), 트레이·전역 단축키·always-on-top·포커스를 잃으면 숨기기·단일 인스턴스가 필요하다. Wails v2는 창이 하나뿐이다.

## 결정

Wails v3(`github.com/wailsapp/wails/v3`, CLI `wails3`)를 쓴다. v3의 `app.Window.New`, `GlobalShortcut.Register`, `SystemTray`, `AlwaysOnTop`, `HideOnFocusLost`를 쓴다. 구현을 시작할 때 고정하는 정확한 버전을 07의 실행 기록에 남긴다.

## 결과

- v3가 베타라면 API가 바뀔 수 있다. 셸 코드는 `desktop/` 안에 가두고, 본체와는 bridge 프로토콜로만 이어져 있어 영향이 앱에 그친다.
- [ADR-0003](0003-wails-app-shell.md)의 "Wails"는 v3를 뜻한다.
