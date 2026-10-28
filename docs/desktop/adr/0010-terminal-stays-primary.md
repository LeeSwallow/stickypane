# ADR-0010. 터미널이 기본 화면으로 남고, ROADMAP의 "no desktop"을 고친다

| 항목 | 값 |
|---|---|
| 상태 | Proposed |
| 날짜 | 2026-10-28 |
| 관련 요구 | R-02, R-04 |

## 맥락

`ROADMAP.md`의 목표는 "show it in the terminal, simply, with no desktop"이다. stickypane 앱은 데스크톱 앱이다.

## 결정

터미널 board는 기본 화면이고 설치·ssh·어떤 터미널에서나 되는 성질을 잃지 않는다. 앱은 고르는 사람에게만 더해진다. ROADMAP의 목표 문장을 "show it in the terminal first; a desktop overlay for those who want more"의 뜻으로 고친다(5단계).

## 결과

- `view`의 기본값은 `terminal`이다.
- 앱 때문에 본체 설치가 무거워지지 않는다.
