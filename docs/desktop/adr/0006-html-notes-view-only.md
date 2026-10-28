# ADR-0006. HTML 노트는 보기 전용이고 샌드박스에서 그린다

| 항목 | 값 |
|---|---|
| 상태 | Proposed |
| 날짜 | 2026-10-28 |
| 관련 요구 | R-10, R-11 |

## 맥락

에이전트가 리포트·차트를 HTML로 남기면 터미널은 그릴 수 없지만 webview는 그릴 수 있다. HTML 안의 스크립트는 신뢰할 수 없다.

## 결정

새 shape `html`(확장자 `.html`)을 둔다. overlay는 `sandbox="allow-scripts"`(same-origin 없음) iframe에 CSP `default-src 'none'`을 주입해 그린다. HTML은 파일에 쓰지 않는다. 상호작용은 기존 `form` shape로 한다([사용자 결정] R-11). 터미널은 태그를 걷은 텍스트와 "overlay에서 열기"를 보여 준다.

## 결과

- HTML 노트는 VISION 3의 루프 밖이다. 답이 필요하면 form을 쓴다.
- 격리 검사 자료를 테스트로 둔다(06 §6.3).
