# ADR-0011. 글꼴은 OFL 글꼴을 앱에 번들한다

| 항목 | 값 |
|---|---|
| 상태 | Proposed |
| 날짜 | 2026-10-28 |
| 관련 요구 | R-18, R-23 |

## 맥락

손글씨·본문·코드에 글꼴이 필요하다. overlay는 CSP로 외부 요청을 막으므로 웹 글꼴 CDN을 쓸 수 없다. 사용자는 오픈소스로 쓸 수 있는 글꼴을 원한다(R-23).

## 결정

Gaegu(손글씨), Pretendard Variable(본문), JetBrains Mono(코드), 필요하면 D2Coding(한글 고정폭). 모두 OFL-1.1이며 앱에 번들한다.

## 결과

- 라이선스 사본을 앱과 `THIRD_PARTY_NOTICES.md`에 넣는다.
- 글꼴을 수정하면 예약된 이름을 쓰지 않는다(JetBrains Mono는 상표).
