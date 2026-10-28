# ADR-0003. 앱 셸은 Wails로, 별도 모듈 `desktop/`에 만든다

| 항목 | 값 |
|---|---|
| 상태 | Proposed |
| 날짜 | 2026-10-28 |
| 관련 요구 | R-03, R-07 |

## 맥락

사용자는 브라우저가 아닌 overlay 앱을 원한다(R-07). 후보는 Electron, Tauri, Wails, 네이티브다([05 A2](../05-alternatives.md#a2-앱-셸)).

## 결정

Wails v3([ADR-0015](0015-wails-v3.md), Go + 시스템 webview)로 만들고, 레포 안 `desktop/` 디렉터리에 별도 `go.mod`를 둔다. 프런트엔드는 `desktop/frontend/`.

## 결과

- 레포 언어가 Go로 유지된다.
- 웹 프런트엔드 빌드 체인(npm)이 `desktop/` 안에만 들어온다.
- 본체 바이너리와 `go.mod`는 바뀌지 않는다.
