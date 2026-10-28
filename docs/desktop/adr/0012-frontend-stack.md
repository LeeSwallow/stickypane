# ADR-0012. 프런트엔드는 Svelte와 검증된 렌더러로 만든다

| 항목 | 값 |
|---|---|
| 상태 | Proposed |
| 날짜 | 2026-10-28 |
| 관련 요구 | R-05 |

## 맥락

사용자는 "best js 모듈"로 Markdown과 시각화를 제대로 그리길 원한다(R-05).

## 결정

Svelte 5 + Vite + TypeScript. Markdown은 unified(remark-gfm, remark-math, rehype-sanitize), 코드는 Shiki, 다이어그램은 Mermaid, 수식은 KaTeX, 손그림은 rough.js.

## 결과

- 번들은 앱 안에 들어가고 네트워크를 쓰지 않는다.
- npm 의존성 라이선스를 검사하는 스크립트가 필요하다.
- React + Motion이 차선으로 남는다([05 A8](../05-alternatives.md#a8-프런트엔드-렌더러)).
