# 09. 배포와 설치 화면

> 상태: Draft · 2026-10-28 · 관련 요구: R-30, R-31

## 9.1 산출물

| 플랫폼 | 산출물 | 만드는 법 |
|---|---|---|
| macOS (arm64, amd64) | `Stickypane-Overlay-<버전>-<arch>.dmg` 안에 `Stickypane Overlay.app` | `wails3` 빌드 → `scripts/overlay-dmg.sh`(hdiutil + 배경 + 아이콘 배치) |
| macOS | Homebrew cask `stickypane-overlay` | 같은 DMG를 가리킨다 |
| Linux (amd64, arm64) | `stickypane-overlay-<버전>-linux-<arch>.tar.gz` (바이너리 + `.desktop` + 아이콘) | `wails3` 빌드. WebKitGTK 4.1이 필요하다 |

본체 `stickypane`은 지금처럼 goreleaser로 배포한다. overlay는 같은 태그에 산출물을 더한다. [설계자 판단] overlay는 Windows용을 만들지 않는다(R-30).

## 9.2 DMG 설치 화면

[사용자 결정] DMG 설치 화면도 doodle 양식이 드러나게 꾸민다(R-31).

```text
┌──────────────────────── Stickypane Overlay ─────────────────────────┐
│   (점 모눈 종이, 테이프 붙은 손글씨 제목)                                │
│        stickypane overlay                                             │
│        "끌어다 붙이면 끝!"                                              │
│                                                                       │
│     ┌─────────┐      ~~~~~~~~~~~~~~~~~~~~~~>      ┌─────────┐          │
│     │ [앱 아이콘]│    (손으로 그린 크레용 화살표)    │Applications│        │
│     └─────────┘                                   └─────────┘          │
│   Stickypane Overlay                               Applications        │
│                                                                       │
│   ┌ 노란 메모지 (살짝 기울어짐) ──────────────────────┐                  │
│   │ 처음 열 때 "확인되지 않은 개발자"가 뜨면            │                  │
│   │ 앱을 우클릭 → 열기 를 눌러 주세요                   │                  │
│   └────────────────────────────────────────────────┘                  │
└───────────────────────────────────────────────────────────────────────┘
```

| 요소 | 규칙 |
|---|---|
| 창 크기 | 660×420 pt, 배경 이미지는 @1x 660×420과 @2x 1320×840을 `tiff`로 합친다 |
| 배경 | 밝은 종이(`stickypane-light`의 `--paper`) + 점 모눈. Finder 아이콘 이름 글자가 검은색이라 밝은 배경을 쓴다 |
| 제목 | Gaegu Bold, 테이프 한 조각 |
| 화살표 | rough.js로 그린 크레용 화살표(아이콘의 빨강), 앱 아이콘에서 Applications로 |
| 아이콘 위치 | 앱 (170, 210), Applications 바로가기 (490, 210), 아이콘 크기 112 |
| 메모지 | 서명·공증이 없는 동안 Gatekeeper 안내. 서명하게 되면 "끌어다 놓으면 끝"으로 바꾼다 |
| 볼륨 아이콘 | 앱 아이콘과 같은 `.icns` |

배경 이미지는 손으로 그리지 않고 **코드로 만든다**: `overlay/build/dmg/background.html`(rough.js + Gaegu, 번들된 자원만)을 헤드리스 Chrome으로 두 배율 PNG로 찍는다. 그래서 테마·문구를 바꿔도 다시 만들 수 있다.

## 9.3 서명과 공증

- 지금 본체 바이너리도 공증하지 않는다(README의 cask가 quarantine을 지운다). overlay 앱도 처음에는 **ad-hoc 서명**(`codesign -s -`)만 한다.
- Developer ID와 공증은 [미결정]. 하게 되면 DMG의 메모지 문구를 바꾸고 cask의 quarantine 단계를 뺀다.

## 9.4 첫 실행 연결

1. 사용자가 DMG에서 앱을 Applications로 끌어 놓고 연다.
2. 앱은 [08 §8.2](08-app-home-and-sessions.md#82-첫-실행--앱-안의-랜딩)의 랜딩을 보인다.
3. 02 카드가 `stickypane`을 찾지 못하면 `brew install --cask LeeSwallow/tap/stickypane`을 보이고, 찾으면 버전과 bridge 프로토콜 버전을 확인한다.

## 9.5 검증

| 항목 | 방법 |
|---|---|
| DMG 레이아웃 | 만든 DMG를 마운트해 Finder 창을 스크린샷으로 남긴다(이 Mac에서) |
| 앱 실행 | Applications에 복사 후 실행, 랜딩이 뜨는지 |
| Linux | CI(ubuntu)에서 빌드와 Go 테스트. 실기 실행은 [미검증] |
