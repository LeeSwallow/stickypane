# ADR-0005. shape는 `Data`와 `Act`로 다른 화면에 자신을 연다

| 항목 | 값 |
|---|---|
| 상태 | Proposed |
| 날짜 | 2026-10-28 |
| 관련 요구 | R-04 |

## 맥락

각 shape의 모델은 unexported이고, 화면 계약(`widget.Widget`)은 셀과 키 입력을 전제로 한다. 다른 화면은 구조를 꺼낼 수도, 클릭을 Op로 바꿀 수도 없다.

## 결정

`widget.Kind`에 `Data func(doc.Document) any`와 `Act func(doc.Document, Action) (doc.Op, error)`를 더한다. 행동은 위치로 가리키고 노트 버전을 함께 보낸다. `api.Apply`가 버전 확인 → `Act` → `Tend` → `store.Apply`를 한다.

## 결과

- 새 쓰기 로직이 없다. `Act`는 `ops.go`의 기존 Op를 고를 뿐이다.
- `Data`는 파일에서 읽을 수 있는 것만 담는다.
- 새 shape는 `Data`를 갖춰야 한다. contract 테스트가 확인한다.
