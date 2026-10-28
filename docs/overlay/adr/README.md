# ADR 목록

> 양식: Michael Nygard, [*Documenting Architecture Decisions*](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions) — 상태 · 맥락 · 결정 · 결과

| # | 결정 | 상태 |
|---|---|---|
| [0001](0001-second-screen-of-the-same-folder.md) | overlay는 같은 폴더의 두 번째 화면이다 | Proposed |
| [0002](0002-bridge-over-stdio.md) | overlay 앱은 `stickypane bridge`와 stdio JSON으로 대화한다 | Proposed |
| [0003](0003-wails-app-shell.md) | 앱 셸은 Wails로, 별도 모듈 `overlay/`에 만든다 | Proposed |
| [0004](0004-view-mode-in-sticky-json.md) | 보기 방식은 `stickypane init`이 묻고 `sticky.json`에 둔다 | Proposed |
| [0005](0005-kind-data-and-act.md) | shape는 `Data`와 `Act`로 다른 화면에 자신을 연다 | Proposed |
| [0006](0006-html-notes-view-only.md) | HTML 노트는 보기 전용이고 샌드박스에서 그린다 | Proposed |
| [0007](0007-hotkey-toggle-and-pin.md) | 창은 전역 단축키로 토글하고, 핀으로 고정한다 | Proposed |
| [0008](0008-two-theme-axes.md) | 테마는 색과 스타일 두 축이고, 기본 스타일은 손그림이다 | Proposed |
| [0009](0009-motion-explains-change.md) | 움직임은 변화를 설명할 때만 쓴다 | Proposed |
| [0010](0010-terminal-stays-primary.md) | 터미널이 기본 화면으로 남고, ROADMAP의 "no desktop"을 고친다 | Proposed |
| [0011](0011-bundled-open-fonts.md) | 글꼴은 OFL 글꼴을 앱에 번들한다 | Proposed |
| [0012](0012-frontend-stack.md) | 프런트엔드는 Svelte와 검증된 렌더러로 만든다 | Proposed |
| [0013](0013-home-registry-events-and-sqlite.md) | 세션 레지스트리는 `~/.stickypane/`의 날짜별 사건 파일과 앱의 SQLite 색인이다 | Proposed |
| [0014](0014-macos-and-linux-only.md) | overlay 앱은 macOS와 Linux만 지원한다 | Proposed |
| [0015](0015-wails-v3.md) | Wails는 v3를 쓴다 | Proposed |
| [0016](0016-dmg-doodle-installer.md) | macOS 설치는 손그림 배경의 DMG로 한다 | Proposed |
| [0017](0017-free-canvas-and-torn-notes.md) | overlay는 자유 캔버스와 떼어낸 노트를 두고, 위치는 이 컴퓨터에만 둔다 | Proposed |
| [0018](0018-app-layers.md) | 앱은 도메인 · 포트 · 유스케이스 · 어댑터 · 화면 층으로 나누고 테스트로 지킨다 | Accepted |
