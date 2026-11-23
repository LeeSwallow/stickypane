# Shape use cases for agents

Research, 2026-11-23. Four reports, each written from the real commands (`stickypane kinds`, `help`) and the web. A claim marked "내 추론" is the researcher's inference, not a source. Unverified items are marked in place. One source did not open when checked (docs.langchain.com/oss/python/human-in-the-loop returned 404); sixteen sampled links, fourteen opened.


---

## note / book / html 에이전트 유스케이스 30선

## 0. 근거와 전제

확인한 사실(저장소에서 직접 봄)
- `stickypane kinds note`: Markdown 아무거나, 한 줄도 노트. ```` ```mermaid ```` 블록(graph, flowchart, sequence, erDiagram)은 그림으로 그려진다. 만들기: `echo "..." | stickypane write deploy-notes --open`.
- `stickypane kinds html`: `.html`/`.htm`. 터미널은 태그·스크립트·스타일 없이 글만, 데스크톱 앱은 페이지를 그대로 그린다. 화면에서 바꿀 수 없다(읽기 전용). 만들기: `stickypane show report.html`.
- book (docs/shapes.md, `kinds book`): 탭 안의 폴더 하나가 한 노트처럼 스크롤되고 파일마다 이름 달린 줄 아래에 놓인다. `,` `.` 로 쪽을 넘긴다. 예: `.sticky/deploy/guide/01-intro.md`, `02-usage.md`. `stickypane link <folder>` 는 프로젝트 폴더를 복사 없이 책으로 보여 준다.
- 스킬(connecting-notes): `[[링크]]`, 한 줄짜리 `![[plan]]` 은 그 노트를 자기 모양대로 읽기 전용으로 그린다. `stickypane show README.md` 는 프로젝트의 아무 파일을 보드에 올린다.
- 확인하지 못한 것: book 폴더 안에 `stickypane write` 로 쪽을 직접 쓰는 경로 문법. 아래 book 사례의 "만드는 법"은 폴더 구조(실제)와 `link`(실제)만 쓰고, 나머지는 "내 추론"으로 표시했다.

웹 조사에서 얻은 큰 흐름
- Claude Code 팀은 계획·리뷰·보고서를 Markdown 대신 HTML로 받는 방식을 권한다: https://claude.com/blog/using-claude-code-the-unreasonable-effectiveness-of-html
- 반론: 저장소에 남고 커밋 단위로 리뷰할 문서는 Markdown이 낫고, HTML은 비교·도식·상호작용 검토용. 같은 맥락 정리: https://docs.plannotator.ai/learn/code-context/ai-agent-html-vs-markdown 및 https://dev.to/mixture-of-experts/how-to-align-coding-agents-with-your-plans-better-than-markdown-without-burning-tokens-27ai (코딩 에이전트 TUI는 아직 Markdown 우선, 터미널은 HTML을 못 그린다는 언급 포함).
- Cursor plan mode: 계획이 열리는 파일이고 사용자가 고친 뒤 Build, 작업 공간에서는 `.cursor/plans/` 버전 관리 파일: https://cursor.com/docs/agent/plan-mode
- 다이어그램은 코드(Mermaid)로 두어야 에이전트가 읽고 고친다: https://itnext.io/coding-agents-cant-see-your-architecture-diagrams-fix-that-e47ea2cfa763 , https://erdembircan.github.io/blog/mermaid-flowcharts-agentic-development
- 산출물 생성과 독립 리뷰 에이전트 분리, 사람 승인: https://awslabs.github.io/aidlc-workflows/reference/diagrams/ 
- AI 포스트모템은 "초안, 사람이 고친다": https://www.datadoghq.com/blog/engineering/llms-for-postmortems.md , https://www.make.com/workflows/draft-incident-postmortems-from-chat-history
- ADR, runbook 자동 생성은 검색에서 직접 근거를 못 찾았다. 해당 사례는 "내 추론".

공통 계측 원칙(각 사례에서 반복하지 않는다): 보드 파일이 존재하는가, 형식이 맞는가(mermaid 블록이 파싱되는가), 내용이 코드·로그의 실제 사실과 일치하는가(식별자·파일 경로가 저장소에 있는가), 사용자가 채팅으로 되묻지 않고 다음 행동을 했는가.

---

## 1. note (Markdown + mermaid + [[링크]] + ![[임베드]])

### N1. 구현 전 설계 한 장
- 상황: 사용자가 큰 기능을 시키면 에이전트가 코딩 전에 접근안을 쓴다.
- 보드에서의 모양: `cat plan.md | stickypane write plan-auth --open` ; 본문은 "## 결정 / ## 대안 / ## 위험" 과 흐름도 mermaid 블록.
- 왜 보드: 채팅은 스크롤로 밀려나고 다음 지시에 묻힌다. 보드는 구현 내내 옆에 고정된다.
- 얻는 것: 코딩 전에 방향을 고칠 기회. 잘못된 구현 한 판을 아낀다.
- 근거: https://cursor.com/docs/agent/plan-mode
- 위험·한계: 사용자가 안 읽으면 장식. 승인 장치가 없으면 계획이 구속력이 없다.
- 계측: 구현 커밋 이전에 plan 노트가 있고, 노트의 파일 경로가 저장소에 실제로 있다.

### N2. 아키텍처 그림이 있는 설명
- 상황: "이 모듈 어떻게 돌아가?" 에 답할 때.
- 모양: 노트 안에 ```` ```mermaid ```` flowchart 한 블록과 3줄 설명.
- 왜 보드: 채팅의 mermaid 는 코드 덩어리로 보인다. 보드는 그림으로 그린다(kinds note 에 명시).
- 얻는 것: 읽는 데 걸리는 시간이 줄고 틀린 곳이 눈에 띈다.
- 근거: https://itnext.io/coding-agents-cant-see-your-architecture-diagrams-fix-that-e47ea2cfa763
- 위험·한계: 지원 종류가 graph, flowchart, sequence, erDiagram 뿐. 그 밖의 문법(C4, gantt 등)은 안 그려질 수 있다(내 추론).
- 계측: mermaid 블록 문법 검사 통과, 그림 속 노드 이름이 코드의 실제 패키지·함수명과 대조된다.

### N3. ER 다이어그램으로 스키마 변경 설명
- 상황: 마이그레이션 직전에 테이블 관계를 보인다.
- 모양: ```` ```mermaid ```` `erDiagram` 블록 + "바뀐 컬럼" 목록.
- 왜 보드: 되돌리기 어려운 변경이라 사용자가 눈으로 확인할 자리가 필요.
- 얻는 것: 마이그레이션 실행 전 승인.
- 근거: 내 추론(Mermaid erDiagram 지원은 kinds note 에서 확인).
- 위험·한계: 큰 스키마는 그림이 빽빽해진다.
- 계측: 노트의 테이블·컬럼이 실제 마이그레이션 파일과 일치.

### N4. ADR(결정 기록) 한 쪽
- 상황: 에이전트가 두 길 중 하나를 고르고 이유를 남긴다.
- 모양: "Decided: 토큰은 세션 쿠키에 둔다. / 이유 / 버린 안" (help note 의 예시 형식과 같은 결이다).
- 왜 보드: 결정이 채팅에 묻히면 나중에 같은 논쟁을 다시 한다.
- 얻는 것: 결정 이력이 파일로 남고 `[[링크]]` 로 계획과 이어진다.
- 근거: 내 추론(ADR 자동 생성 직접 근거는 못 찾음).
- 위험·한계: 사용자가 결정에 동의하지 않았는데 "Decided" 로 쓰는 월권.
- 계측: 결정 노트에 대안이 최소 1개, 이유가 있고, 사용자 승인 이벤트 뒤에만 "Decided" 표기.

### N5. 코드 변경 PR 요약 초안
- 상황: 작업을 마치고 PR 설명 쓰기 전.
- 모양: "무엇을 / 왜 / 어떻게 확인했나 / 손대지 않은 것" 4칸 노트, 테스트 결과 한 줄.
- 왜 보드: 사용자가 PR 올리기 전에 한 번 고쳐 본다.
- 얻는 것: PR 설명 품질 상승, 리뷰어 질문 감소.
- 근거: 내 추론.
- 위험·한계: 요약이 실제 diff 와 어긋나면 자신감만 키운다.
- 계측: 요약에 나온 파일명이 diff 에 모두 있고, diff 의 주요 파일이 요약에 빠지지 않았다.

### N6. 막힘과 선택지 보고
- 상황: 테스트가 계속 깨져 방향을 정해야 할 때.
- 모양: 노트에 "시도한 것 3개 / 실패 원인 / 다음 후보 A, B". 선택은 form 스킬(asking-the-user)로 물어도 된다.
- 왜 보드: 긴 로그를 채팅에 붓지 않고, 사용자는 요약만 본다.
- 얻는 것: 에이전트가 헛돌며 시간·토큰을 쓰는 것을 일찍 끊는다.
- 근거: 내 추론.
- 위험·한계: 요약이 낙관적이면 문제를 가린다.
- 계측: 같은 실패 3회째에 노트가 만들어졌는가, 노트에 실패 명령과 종료 코드가 들어 있는가.

### N7. 사이퀀스 다이어그램으로 버그 재현 경로
- 상황: 비동기 버그의 호출 순서를 사용자와 맞춘다.
- 모양: ```` ```mermaid ```` `sequenceDiagram` + "의심 지점" 한 줄.
- 왜 보드: 시간 순서는 글보다 그림이 빠르다.
- 얻는 것: 원인 가설을 같이 검증.
- 근거: https://erdembircan.github.io/blog/mermaid-flowcharts-agentic-development (흐름도를 컨텍스트에 두는 이점)
- 위험·한계: 그림이 가설일 뿐 사실이 아닌데 사실처럼 보일 수 있다.
- 계측: 그림에 가설/확인 표시가 있고 확인 항목은 로그 줄이 인용된다.

### N8. 임베드로 만든 요약 허브
- 상황: 여러 노트(계획, 체크리스트, 로그)를 한 화면에 모은다.
- 모양: 노트 한 줄 `![[plan]]` (connecting-notes 스킬에 있는 문법), 위에 `[[결정]]` 링크.
- 왜 보드: 채팅에는 "다른 메시지 보여 주기"가 없다.
- 얻는 것: 한 곳에서 현황 파악, 원본은 하나만 유지.
- 근거: 저장소 스킬(connecting-notes). 효용은 내 추론.
- 위험·한계: 임베드는 읽기 전용 표시다. 임베드 대상이 지워지면 빈칸.
- 계측: 임베드 대상 노트가 실제로 존재, 허브에 같은 내용이 복붙되지 않았다.

### N9. 리뷰 의견 모음(자기 리뷰)
- 상황: 구현 후 스스로 diff 를 점검해 찜찜한 곳을 적는다.
- 모양: "확신 낮음" 항목마다 `파일:줄` 과 이유.
- 왜 보드: 사용자가 우선순위를 매길 수 있다.
- 얻는 것: 리뷰 시간을 위험한 곳에 쓴다.
- 근거: https://awslabs.github.io/aidlc-workflows/reference/diagrams/ (생성자와 별도 리뷰 단계)
- 위험·한계: 같은 에이전트의 자기 리뷰는 맹점을 공유한다.
- 계측: 지적한 `파일:줄` 이 존재하고, 심은 버그를 몇 개 짚었는가(심은 버그 평가).

### N10. 릴리스 노트·변경 요약 초안
- 상황: 릴리스 전 지난 커밋을 사용자 눈으로 정리.
- 모양: Conventional Commits 별 묶음 목록 노트.
- 왜 보드: 사람이 말투를 고쳐 바로 붙인다.
- 얻는 것: 문서 작업 시간 절감.
- 근거: 내 추론(초안은 사람이 고친다는 원칙은 https://www.datadoghq.com/blog/engineering/llms-for-postmortems.md 와 같은 결).
- 위험·한계: 없는 기능을 쓰는 환각. 사용자 노출 문구는 짧고 담백해야 한다는 프로젝트 규칙과 충돌 가능.
- 계측: 항목마다 커밋 해시가 대응하고, 커밋에 없는 항목이 0.

---

## 2. book (폴더 하나 = 쪽이 여러 개인 한 권)

공통: 폴더 구조는 `.sticky/<탭>/<책>/01-intro.md, 02-usage.md` (shapes.md). 쪽을 쓰는 구체 명령 문법은 확인하지 못했으므로 "내 추론".

### B1. 온보딩 가이드
- 상황: 낯선 저장소를 처음 맡은 에이전트(또는 새 사람)가 지도를 만든다.
- 모양: `onboarding/01-구조.md, 02-빌드.md, 03-테스트.md, 04-함정.md`
- 왜 보드: 쪽 넘김(`,` `.`)으로 한 권처럼 읽는다. 채팅의 긴 한 덩어리보다 찾기 쉽다.
- 얻는 것: 저장소 맥락을 한 번에 공유.
- 근거: 저장소 shapes.md(책 동작). 사례는 내 추론.
- 위험·한계: 코드가 바뀌면 낡는다.
- 계측: 쪽에 나온 명령이 실제로 돌아가는가(빌드·테스트 명령 실행 검증).

### B2. 기존 docs 폴더를 복사 없이 책으로
- 상황: 저장소 `docs/` 를 설명하며 같이 읽는다.
- 모양: `stickypane link docs` (kinds book 의 예시 그대로).
- 왜 보드: 사본이 없어 원본이 바뀌면 같이 바뀐다.
- 얻는 것: 문서 이중 관리 없음.
- 근거: 저장소 `kinds book`.
- 위험·한계: 폴더 안 파일이 쪽 이름으로 보이므로 이름 정리가 필요.
- 계측: `link` 후 보드에 책이 뜨고 쪽 수가 폴더 파일 수와 같다.

### B3. 장문 조사 보고서(쪽별 주제)
- 상황: 라이브러리 비교, 마이그레이션 조사 등.
- 모양: `research/01-요약.md, 02-후보비교.md, 03-벤치마크.md, 04-권고.md`
- 왜 보드: 요약부터 읽고 필요한 쪽만 판다.
- 얻는 것: 긴 조사의 결론을 먼저 얻는다.
- 근거: https://dev.to/mixture-of-experts/how-to-align-coding-agents-with-your-plans-better-than-markdown-without-burning-tokens-27ai (긴 산출물의 구조화), 사례는 내 추론.
- 위험·한계: 쪽이 많아지면 아무도 안 읽는다.
- 계측: 01 이 한 화면에 들어가고, 인용 URL 이 실제 열린다.

### B4. 런북(운영 절차서)
- 상황: 배포·롤백 절차를 에이전트가 정리.
- 모양: `runbook/01-사전점검.md, 02-배포.md, 03-롤백.md, 04-확인.md`
- 왜 보드: 순서가 있는 절차는 쪽 순서가 곧 순서다.
- 얻는 것: 사용자가 따라 하며 확인.
- 근거: 내 추론(런북 자동 생성 근거 못 찾음).
- 위험·한계: 틀린 명령이 있으면 사고로 이어진다. 파괴적 명령은 실행 금지 표시 필요.
- 계측: 각 명령이 dry-run 으로 검증됐다는 표시가 있다.

### B5. 인시던트 포스트모템 한 권
- 상황: 장애가 끝난 직후.
- 모양: `incident-0412/01-요약.md, 02-타임라인.md, 03-원인.md, 04-조치.md`
- 왜 보드: 채팅 이력에서 모은 뒤 사람이 고친다.
- 얻는 것: 초안 작성 시간 절감.
- 근거: https://www.datadoghq.com/blog/engineering/llms-for-postmortems.md , https://www.make.com/workflows/draft-incident-postmortems-from-chat-history
- 위험·한계: 원인 단정 위험. 비난 없는 문체 유지 필요.
- 계측: 타임라인의 시각이 로그 시각과 일치(시각은 보드가 쓴다는 저장소 규칙과 별개로 본문 시각은 로그에서 인용).

### B6. 모듈별 설명서(코드 읽기 지도)
- 상황: 큰 코드를 파일 단위로 설명.
- 모양: `tour/01-core.md, 02-adapter.md, 03-usecase.md, 04-surface.md`
- 왜 보드: 층별로 쪽을 넘기며 구조를 이해한다.
- 얻는 것: 코드 리뷰 전에 머릿속 지도.
- 근거: 내 추론.
- 위험·한계: 디렉터리 구조가 바뀌면 낡는다.
- 계측: 설명한 패키지 경로가 전부 존재.

### B7. 마이그레이션 계획과 진행
- 상황: 여러 단계 이행(버전 업, 모듈 이동).
- 모양: `migrate/01-목표.md, 02-단계.md, 03-위험.md, 04-되돌리기.md`
- 왜 보드: 단계는 쪽, 진행은 별도 체크리스트 노트로 `[[링크]]`.
- 얻는 것: 되돌리기 계획을 시작 전에 본다.
- 근거: 내 추론.
- 위험·한계: 계획과 진행 두 곳이 어긋난다.
- 계측: 04-되돌리기가 있고 단계 수가 체크리스트 항목 수와 같다.

### B8. 혼합 형식 책(note + html)
- 상황: 설명(Markdown) 사이에 시각 비교 페이지(html)를 끼운다. shapes 설명에 쪽의 모양이 서로 다를 수 있다고 있음.
- 모양: `perf/01-요약.md, 02-차트.html, 03-원시표.md`
- 왜 보드: 한 권에서 글과 그림을 함께 본다.
- 얻는 것: 요약은 터미널에서, 차트는 앱에서 본다.
- 근거: 저장소 shapes(책의 쪽은 모양이 달라도 됨).
- 위험·한계: 터미널에서는 html 이 글만 보인다.
- 계측: html 쪽에도 글로 된 핵심 숫자가 있어 터미널에서 읽힌다.

### B9. 테스트 실패 분류집
- 상황: CI 실패 수십 건을 원인별로 묶는다.
- 모양: `triage/01-개요.md, 02-환경문제.md, 03-진짜버그.md, 04-플레이키.md`
- 왜 보드: 분류별로 쪽이 나뉘어 스캔이 쉽다.
- 얻는 것: 무엇부터 고칠지 결정.
- 근거: 내 추론.
- 위험·한계: 잘못 분류하면 진짜 버그가 플레이키로 묻힌다.
- 계측: 분류 정답이 있는 CI 로그 묶음으로 정확도 측정.

### B10. 학습 노트(사용자가 모르는 영역 입문서)
- 상황: 사용자가 처음 쓰는 도구를 에이전트가 쪽 단위 강의로 만든다.
- 모양: `learn/01-개념.md, 02-첫예제.md, 03-자주하는실수.md`
- 왜 보드: 채팅은 되묻다 흐트러지고, 책은 자기 속도로 읽는다.
- 얻는 것: 에이전트 작업과 병행해 사용자가 배운다.
- 근거: 내 추론.
- 위험·한계: 틀린 설명이 고착된다. 최신 문서 확인 필요.
- 계측: 예제 코드가 실제로 실행되고 공식 문서와 대조된다.

---

## 3. html (읽기 전용 웹 페이지; 터미널은 글, 앱은 렌더)

공통 주의: 터미널은 글만 보이므로 핵심은 글로도 읽혀야 한다. 스크립트·스타일은 터미널에서 제거된다. 외부 스크립트는 보안 위험이 있다는 지적: https://windowsforum.com/windows-news.4/claude-code-html-reports-improve-reviews-but-add-security-risks.441262/

### H1. 비교표가 있는 선택지 보고서
- 상황: 라이브러리 A/B/C 중 고르기.
- 모양: `stickypane show compare.html` ; 표와 색 강조.
- 왜 보드: 표·색은 Markdown 보다 풍부하다.
- 얻는 것: 한눈 비교.
- 근거: https://claude.com/blog/using-claude-code-the-unreasonable-effectiveness-of-html
- 위험·한계: 토큰이 더 들고(한 2차 출처는 약 5배라 주장, 검증 안 됨), 터미널은 표가 글로 풀린다.
- 계측: 표 셀의 사실(버전, 라이선스)이 공식 출처와 일치.

### H2. 벤치마크·성능 리포트
- 상황: 최적화 전후 수치 보고. help html 예: "Run took 4.2s, down from 6.1s."
- 모양: `stickypane show report.html`
- 왜 보드: 차트와 표가 한 페이지.
- 얻는 것: 개선 폭이 직관적.
- 근거: 저장소 kinds html 예시 형식. 효용은 내 추론.
- 위험·한계: 차트 숫자 조작·오독. 단일 실행 수치는 소음.
- 계측: 숫자가 실제 벤치 출력과 같고, 반복 횟수와 편차가 적혀 있다.

### H3. 구현 전 계획 리뷰 페이지(전후 구조 비교)
- 상황: 되돌리기 힘든 변경 전.
- 모양: 변경 파일 목록, 전후 구조, 위험, 열린 질문이 있는 html.
- 왜 보드: 사용자가 승인하기 전에 한 번 훑는다. 읽은 뒤 버리는 일회성 문서.
- 얻는 것: 승인 품질.
- 근거: https://dev.to/mixture-of-experts/how-to-align-coding-agents-with-your-plans-better-than-markdown-without-burning-tokens-27ai (검색 요약에서 확인)
- 위험·한계: 읽기 전용이라 그 자리에서 고칠 수 없다. 에이전트 선호 결론으로 유도할 위험(plannotator 지적).
- 계측: 페이지의 변경 파일 목록이 실제 diff 와 같다.

### H4. 커버리지·테스트 결과 대시보드
- 상황: 테스트 돌린 뒤 실패 위치를 시각화.
- 모양: 도구가 만든 html 리포트를 그대로 `stickypane show`.
- 왜 보드: 도구가 이미 만드는 html 을 변환 없이 보인다.
- 얻는 것: 에이전트가 새 문서를 쓰지 않고 기존 리포트를 연다.
- 근거: 내 추론(`show` 는 프로젝트 파일을 올린다는 저장소 스킬 근거).
- 위험·한계: 외부 리소스를 부르는 리포트는 앱에서 깨지거나 보안 문제.
- 계측: 올라온 파일이 실제 최신 실행 결과(수정 시각)인가.

### H5. 시각적 UI 변경 전후
- 상황: 화면 수정 후 결과를 보여 준다.
- 모양: 전/후 스냅샷을 나란히 둔 html.
- 왜 보드: 이미지를 채팅에 첨부하는 것보다 한 페이지에서 비교.
- 얻는 것: 디자인 확인 빨라짐.
- 근거: 내 추론.
- 위험·한계: 터미널에서는 이미지가 안 보인다.
- 계측: 전후 이미지가 실제 캡처이고 설명 글이 있다.

### H6. 인터랙티브 설명서(슬라이더·토글)
- 상황: 알고리즘·요금 계산을 직접 만져 보게.
- 모양: 스크립트가 든 html.
- 왜 보드: 앱은 페이지를 그리므로 만져 볼 수 있다(kinds html: 앱이 페이지를 그린다). 단, 스크립트가 앱에서 실제로 실행되는지는 확인하지 못했다.
- 얻는 것: 이해가 빠르다.
- 근거: https://claude.com/blog/using-claude-code-the-unreasonable-effectiveness-of-html
- 위험·한계: 터미널 사용자에게는 무의미. 스크립트 실행 가능 여부 불확실, 보안.
- 계측: 앱에서 열어 상호작용 동작 확인, 터미널 글 대체 문구 존재.

### H7. 코드 구조도(의존성 그래프 렌더)
- 상황: 도구가 만든 의존성 그래프 html 을 보인다.
- 모양: `stickypane show deps.html`
- 왜 보드: 큰 그래프는 mermaid 로 그리기엔 크다.
- 얻는 것: 순환·과밀 모듈 발견.
- 근거: 내 추론.
- 위험·한계: 큰 그래프는 읽기 어렵다.
- 계측: 그래프의 간선이 실제 import 와 일치(샘플 검사).

### H8. 로그 분석 요약 페이지
- 상황: 수천 줄 로그를 시간대·오류 종류별 표로.
- 모양: 요약 표 + 대표 줄.
- 왜 보드: 채팅에 로그를 붓지 않는다.
- 얻는 것: 원인 후보를 빨리 좁힌다.
- 근거: 내 추론(AI 포스트모템 요약과 같은 결: https://www.datadoghq.com/blog/engineering/llms-for-postmortems.md).
- 위험·한계: 요약에서 빠진 줄이 진짜 원인일 수 있다.
- 계측: 집계 개수가 grep 결과와 같다.

### H9. 사용자 대상 산출물 미리보기(랜딩, 메일 템플릿)
- 상황: 에이전트가 만든 html 결과물 자체가 제품일 때.
- 모양: `stickypane show landing.html`
- 왜 보드: 브라우저를 따로 열 필요 없이 옆에서 본다.
- 얻는 것: 즉시 피드백.
- 근거: 내 추론.
- 위험·한계: 읽기 전용이라 수정 요청은 채팅. 반응형·실제 브라우저 차이.
- 계측: 앱 렌더와 실제 브라우저 렌더 비교.

### H10. 비용·일정 리포트(경영진용 한 장)
- 상황: 사용자가 보고서를 위로 전달해야 할 때.
- 모양: 스타일 입힌 html 한 장.
- 왜 보드: 완성된 형태를 바로 보고, 그대로 파일로 전달.
- 얻는 것: 전달 가능한 산출물.
- 근거: https://claude.com/blog/using-claude-code-the-unreasonable-effectiveness-of-html ; 반론: 저장소에 남는 문서는 Markdown 이 낫다(plannotator 링크 위).
- 위험·한계: 버전 관리·diff 리뷰가 어렵다.
- 계측: 숫자가 원 데이터와 일치하고 날짜·기준이 명시.

---

## 4. 가장 강한 사례와 겹침

가장 강한 3개
- note: N1(구현 전 설계, 채팅에서 밀려나는 문제를 보드가 정면으로 해결, Cursor plan mode 가 같은 사용 방식을 증명), N2(mermaid 가 그림으로 그려지는 것이 보드의 고유 이점), N6(막힘 보고: 사용자가 보드만 봐도 개입 시점을 안다).
- book: B2(`stickypane link docs`, 복사 없이 즉시 되는 실제 명령), B1(온보딩, 쪽 넘김이 정보 구조와 맞음), B5(포스트모템: 외부 근거가 가장 많다).
- html: H1(비교), H2(벤치마크 숫자 보고), H3(구현 전 계획 리뷰). 모두 Claude Code 팀의 "HTML 출력" 권고와 맞고, 터미널에서 글로도 읽혀 약점이 작다.

겹침
- N1(설계)과 H3(계획 리뷰 페이지)는 같은 목적. 선택 기준: 편집·커밋될 문서면 note, 일회성 승인용 비교면 html.
- N4(ADR)와 B7(마이그레이션 책)은 결정 기록이 겹친다. 결정 하나는 note, 여러 쪽 묶음은 book.
- B8 은 book 안에서 note 와 html 이 만나는 지점. B3(조사 보고서)와 H1(비교)도 겹친다.
- N5/N10 과 B5 는 모두 "초안을 사람이 고친다" 패턴.
- 구조적 한계 하나: html 은 읽기 전용이고 터미널에서는 글만 보이므로, 핵심 결론은 항상 note 로 한 줄 남기는 편이 안전하다(내 추론).

---

## checklist · board · chat · form: 에이전트 유스케이스 40선

확인한 사실(저장소에서 `stickypane kinds`/`help`와 스킬 4종을 실행·열람):
- checklist: `- [ ]`/`- [x]` 줄, 진행 막대. `stickypane todo <name> add|check|uncheck <item>`. 체크하면 줄 끝에 `✅ 시각`을 보드가 쓴다.
- board: `## 칸`, `- 카드`, 카드 끝 `@codex-1`. `stickypane card <name> add|move <card> --to <칸>`. 30분 안 움직인 카드는 마지막 칸 밖이면 표시된다.
- chat: `@이름 HH:MM 내용`. `stickypane say <name> [--as W] <text>`. 답은 `stickypane watch --once --note chat --type message.added --timeout 10m --json`.
- form: `- ( )` 하나, `- [ ]` 여럿, `> ` 입력, `[ 버튼 ]`. 버튼을 누르면 front matter에 `submitted: 버튼`. `stickypane wait <name> --timeout 10m [--json]`, `stickypane answers <name>`. 종료 코드 3은 시간 초과. 이미 submitted인 폼은 즉시 옛 답을 돌려준다(스킬에 명시).
- 그 밖에 스킬에서 본 것: `stickypane show/hide <name>`, `stickypane log <name> --time "..."`, `stickypane chart`, `[[링크]]`, `![[임베드]]`, 차트 `from: a, b`, `stickypane watch --exec` (`STICKY_*` 환경 변수), 이벤트 `item.ticked card.moved form.submitted message.added`.

## 외부 근거 요약 (URL은 각 항목에 반복)
- LangGraph/LangChain: `interrupt()`로 멈추고 approve / edit / reject(+피드백)로 재개. 여러 호출의 결정은 위치가 아니라 ID로 짝지어야 한다. https://docs.langchain.com/oss/python/human-in-the-loop (열리지 않음: 404)
- MCP elicitation: 서버가 작업 도중 사용자에게 구조화 입력을 요청. accept / decline / cancel 세 응답, 평평한 스키마. 민감한 값은 form이 아닌 URL 모드로. https://modelcontextprotocol.io/specification/2025-11-25/client/elicitation
- 승인 피로: 프롬프트가 많고 효과 없이 명령만 보여주면 "항상 허용"·YOLO 모드로 간다. https://arxiv.org/pdf/2604.13536 , https://workos.com/blog/approval-fatigue-agent-governance , https://arxiv.org/pdf/2610.06047
- Anthropic 자율성 연구: 숙련 사용자는 자동 승인을 늘리면서도 더 자주 끼어든다. 복잡한 작업에서 에이전트의 자체 질문이 늘어난다. https://www.anthropic.com/research/measuring-agent-autonomy , https://anthropic.com/research/trustworthy-agents
- 칸반: 사람 검토 게이트를 가진 TUI https://raw.githubusercontent.com/kagan-sh/kagan-legacy/main/README.md ; 대화형 보드 https://github.com/saltbo/agent-kanban ; 의존 카드 자동 시작 https://github.com/cline/kanban ; 논문 https://arxiv.org/pdf/2611.04100 ; https://hermes-agent.nousresearch.com/docs/user-guide/features/kanban
- 계획 모드: https://code.claude.com/docs/en/permission-modes , https://cursor.com/blog/dynamic-planning-agents , https://flaviocopes.com/ai-glossary/plan-mode/
- AG-UI: 사람 개입 협업을 내세운 이벤트 프로토콜. https://github.com/ag-ui-protocol/ag-ui (interrupt 이벤트 이름은 튜토리얼 수준이라 확인 못 함)

주의: 아래 "근거"의 "내 추론"은 출처 없이 내가 도출한 것이다. 계측은 `.sticky/` 파일 mtime·`✅` 시각·`submitted`·`watch` 이벤트 순서를 로그와 대조하는 방식이다.

---

## 1. checklist (`stickypane todo`)

### C1. 작업 시작 전 계획 공개
- 상황: 5단계 이상 리팩터링을 시작하기 직전, 사용자가 다른 일을 하는 중.
- 모양: `stickypane todo refactor add "find every caller"` 를 단계마다 반복, 첫 줄이 곧 계획.
- 왜 보드: 채팅은 스크롤로 밀리고 계획은 계속 보고 있어야 한다.
- 얻는 것: 코드를 건드리기 전에 계획의 틀린 부분을 잡는다.
- 근거: Anthropic이 "계획을 먼저 보여주고 사용자가 검토"하는 설계를 기술 https://anthropic.com/research/trustworthy-agents
- 위험·한계: 계획이 형식이 되어 아무도 안 읽을 수 있다. 읽혔다는 신호가 없다.
- 계측: 첫 파일 편집 시각이 첫 `todo add` 시각보다 늦은가.

### C2. 끝낸 뒤에만 체크
- 상황: 긴 작업 중 사용자가 "어디까지 됐어?"를 묻고 싶어함.
- 모양: `stickypane todo refactor check "move"` 후 `refactor.md: 2/3` 출력을 보고. 줄에 `✅ 시각`이 남는다.
- 왜 보드: 진행 막대와 시각이 자동으로 쓰이는 기록이다.
- 얻는 것: 말이 아니라 시각이 붙은 증거로 진행을 믿는다.
- 근거: tracking-progress 스킬 "끝났을 때 체크, 시작할 때 아님".
- 위험·한계: 거짓 체크. 체크가 곧 검증은 아니다.
- 계측: 체크 시각 직전에 해당 테스트/명령이 성공했는가(로그 대조).

### C3. 완료 기준(Definition of Done) 체크리스트
- 상황: PR 올리기 전. 테스트, 린트, 문서, 변경 로그.
- 모양: `stickypane todo ship add "go vet passes"` 등 4줄.
- 왜 보드: 빠진 항목이 눈에 띈다. 채팅 요약은 빠진 것을 숨긴다.
- 얻는 것: "다 했습니다"의 근거를 한 화면에서 확인.
- 근거: 내 추론.
- 위험·한계: 에이전트가 자기가 쓴 기준에 맞춰 채점하는 순환.
- 계측: 체크된 항목마다 대응 명령이 실제 실행됐는가.

### C4. 사용자가 직접 체크하는 항목(사람 몫 단계)
- 상황: 에이전트가 못 하는 일(키 발급, 수동 QA)이 섞인 작업.
- 모양: 보드에서 사용자가 `space`로 체크, 에이전트는 `stickypane watch --once --note ship --type item.ticked`.
- 왜 보드: 체크 한 번이 "내가 했다"는 신호가 되고, 이벤트로 에이전트가 이어간다.
- 얻는 것: 에이전트가 사람 단계를 추측하지 않고 기다린다.
- 근거: connecting-notes 스킬의 `item.ticked` 이벤트.
- 위험·한계: 사용자가 체크를 잊으면 영영 대기. 타임아웃 필요.
- 계측: 사람 몫 항목이 체크되기 전에 그 뒤 단계를 시작하지 않았는가.

### C5. 계획 변경 흔적
- 상황: 중간에 범위가 바뀜(파일 3개 추가).
- 모양: `stickypane todo plan add "handle the legacy folder"`, 필요 없어진 건 `uncheck`가 아니라 새 줄과 `say`로 알림.
- 왜 보드: 항목이 늘어난 사실이 막대 분모에 바로 드러난다.
- 얻는 것: 범위 증가(scope creep)를 숨길 수 없다.
- 근거: 내 추론. (Cursor가 계획이 구현 중 적응한다고 설명 https://cursor.com/blog/dynamic-planning-agents)
- 위험·한계: 항목을 조용히 지우는 것은 CLI로 막지 못한다.
- 계측: 시작 때보다 항목 수가 늘면 같은 시점에 chat/log 설명이 있는가.

### C6. 병렬 서브에이전트의 합류 지점
- 상황: 서브에이전트 3개가 각자 일하고, 끝나면 하나씩 체크.
- 모양: 항목 3개 + `stickypane todo merge add "merge results"`; 마지막은 앞 3개가 체크된 뒤에만.
- 왜 보드: 한 곳에서 누가 안 끝났는지 본다.
- 얻는 것: 병렬 작업의 막힌 곳을 즉시 파악.
- 근거: 내 추론. 유사: 의존 작업 자동 시작 https://github.com/cline/kanban
- 위험·한계: 서브에이전트가 같은 파일을 동시에 쓰면 충돌. 항목 단위로 `todo`가 쓰는 것이므로 CLI 사용이 전제.
- 계측: merge 항목 체크가 앞 3개 체크보다 늦은가.

### C7. 막힌 일 표시와 우회
- 상황: 테스트 하나가 계속 실패해 단계가 멈춤.
- 모양: `stickypane todo plan add "BLOCKED: flaky test in auth"` 뒤 `stickypane say chat "auth test is flaky, skip or fix?"`
- 왜 보드: 막힘이 다음 항목 위에 고정된다.
- 얻는 것: 에이전트가 조용히 우회하다 길을 잃는 것을 막는다.
- 근거: Anthropic: 복잡한 작업에서 에이전트의 자체 질문이 늘어난다 https://anthropic.com/research/trustworthy-agents
- 위험·한계: BLOCKED 표기가 체크 표시 규칙과 섞여 막대가 오해를 부른다.
- 계측: 같은 실패가 3회 이상 되풀이되기 전에 BLOCKED 항목이 생겼는가.

### C8. 재개용 체크포인트
- 상황: 컨텍스트가 압축되거나 새 세션이 같은 작업을 이어받음.
- 모양: 에이전트는 시작할 때 `stickypane todo plan`(현황)을 읽고 첫 미체크 항목부터.
- 왜 보드: 파일이 대화 밖에 남는 기억이다.
- 얻는 것: 중단 뒤에도 중복 작업·누락이 없다.
- 근거: 보드가 대화보다 오래 사는 조정 상태라는 주장 https://arxiv.org/pdf/2611.04100
- 위험·한계: 파일이 현실과 어긋나면 틀린 기억을 믿는다.
- 계측: 재개 후 이미 체크된 항목을 다시 하지 않았는가.

### C9. 사용자가 던진 할 일 큐
- 상황: 사용자가 작업 중에 "그것도 해줘"를 보드에서 `n`으로 항목 추가.
- 모양: 사용자의 새 줄이 생김. 에이전트는 사이사이 `watch --once --note plan --type note.changed --timeout 1s`로 확인.
- 왜 보드: 지시를 채팅에 끊어 보내면 현재 작업을 방해하고, 줄로 쌓이면 순서를 정할 수 있다.
- 얻는 것: 에이전트를 멈추지 않고 방향을 보탠다.
- 근거: 내 추론. (`n new item` 키는 kinds checklist)
- 위험·한계: 에이전트가 새 줄을 못 보면 무시됨.
- 계측: 사용자가 추가한 줄이 N분 안에 에이전트의 `check`로 이어졌거나 사유가 말해졌는가.

### C10. 증명 가능한 마무리 정리
- 상황: 작업 종료, 사용자가 자리로 돌아옴.
- 모양: 모두 체크 + `stickypane log worklog --time "refactor done: 14 files, tests green"`.
- 왜 보드: 채팅의 긴 요약 대신 시각이 붙은 기록.
- 얻는 것: 돌아와서 10초 안에 상태를 파악.
- 근거: tracking-progress 스킬(마무리에 체크와 로그).
- 위험·한계: 끝났다고 선언하면서 일부는 미체크로 남기는 불일치.
- 계측: 종료 메시지 시점에 미체크 항목이 0인가, 아니면 그 이유가 언급됐는가.

---

## 2. board (`stickypane card`)

### B1. 에이전트 작업 레인
- 상황: 여러 에이전트(`@codex-1`, `@claude-2`)가 한 저장소에서 일함.
- 모양: `stickypane card work add "login API" --to Doing` 후 카드 끝에 `@codex-1`.
- 왜 보드: 누가 무엇을 쥐었는지는 공간 배치로 읽는다.
- 얻는 것: 중복 작업과 충돌을 눈으로 피한다.
- 근거: https://github.com/saltbo/agent-kanban , https://hermes-agent.nousresearch.com/docs/user-guide/features/kanban
- 위험·한계: 담당 표기를 안 갱신하면 거짓 정보.
- 계측: 같은 파일을 두 카드가 동시에 Doing에서 건드린 일이 있는가.

### B2. 검토(Review) 칸이 곧 승인 게이트
- 상황: 구현을 끝낸 카드가 사람 검토 없이는 Done으로 갈 수 없다는 규칙.
- 모양: 칸 `To do | Doing | Review | Done`. `stickypane card work move login --to Review` 후 사용자가 `L`로 Done.
- 왜 보드: 승인이 카드 이동이라는 한 번의 동작이고, 상태가 남는다.
- 얻는 것: 승인 질문이 폭주하지 않고 카드 단위로 모인다. 사용자가 편한 때에 처리.
- 근거: 사람 검토를 구조로 강제하는 TUI https://raw.githubusercontent.com/kagan-sh/kagan-legacy/main/README.md
- 위험·한계: 에이전트가 스스로 Done으로 옮길 수 있다(CLI상 막히지 않음).
- 계측: Done 이동의 행위자가 사용자인가, Review 이후 사용자 이동 이벤트(`card.moved`)가 있었는가.

### B3. 정체 카드 감지
- 상황: 에이전트가 한 카드에서 오래 멈춤.
- 모양: 보드가 30분 넘게 안 움직인 카드를 마지막 칸 밖에서 표시한다(kinds board).
- 왜 보드: 채팅은 침묵을 보여주지 않지만 보드는 시간이 보인다.
- 얻는 것: 에이전트가 멈췄는지 일하는 중인지 짐작이 아니라 표시로 안다.
- 근거: 내 추론.
- 위험·한계: 긴 빌드는 정상인데 정체로 오인.
- 계측: 표시가 붙은 카드에 에이전트가 10분 안에 `say`로 사정을 말했는가.

### B4. 후보 안 비교: 카드 하나가 한 선택지
- 상황: 구현 방법 3개를 병행 시도.
- 모양: 칸 `Options | Trying | Chosen`, 카드 3장. 사용자가 고른 것을 `Chosen`으로.
- 왜 보드: 안들을 나란히 놓고 이동으로 선택한다.
- 얻는 것: 에이전트가 하나를 몰래 고르지 않는다.
- 근거: 내 추론.
- 위험·한계: 선택을 form이 더 명확히 받는 경우가 있다(구조화된 이유).
- 계측: Chosen 칸에 카드가 생기기 전에 최종 구현을 병합하지 않았는가.

### B5. 이슈 분류(triage) 대기열
- 상황: 에이전트가 이슈 20개를 읽고 Backlog를 분류.
- 모양: `stickypane card triage add "fix typo" --to "Easy"`, `--to "Needs human"`.
- 왜 보드: 분류 결과를 칸 수와 모양으로 한눈에 비교.
- 얻는 것: 사용자는 "Needs human" 칸만 본다.
- 근거: 내 추론. 유사: https://github.com/cline/kanban
- 위험·한계: 잘못된 분류가 칸 안에 조용히 묻힘.
- 계측: 사용자가 Needs human에서 Easy로 옮긴 비율(오분류율).

### B6. 사용자의 우선순위 재배열
- 상황: 에이전트가 쌓은 To do를 사용자가 `J K`로 순서를 바꿈.
- 모양: To do 안 카드 순서가 곧 우선순위.
- 왜 보드: 순서를 말로 적는 것보다 위아래가 빠르다.
- 얻는 것: 다음에 할 일에 대한 조종권.
- 근거: 키 `J K reorder` (kinds board). 효과는 내 추론.
- 위험·한계: 에이전트가 순서를 안 읽고 임의 순서로 진행.
- 계측: 다음에 Doing으로 가는 카드가 To do 맨 위인가.

### B7. 서브에이전트 인계(hand-off)
- 상황: 구현 에이전트가 테스트 에이전트에게 넘김.
- 모양: `stickypane card work move login --to Testing` 및 카드 아래 들여쓴 세부 줄.
- 왜 보드: 인계 지점이 칸 경계로 드러난다.
- 얻는 것: 책임이 어디 있는지 안다.
- 근거: https://hermes-agent.nousresearch.com/docs/user-guide/features/kanban
- 위험·한계: 세부가 부족한 인계(컨텍스트 손실).
- 계측: 인계 카드에 재현 명령/경로가 들어 있는가.

### B8. 위험 작업 분리 칸
- 상황: 마이그레이션, 삭제 같은 되돌리기 어려운 작업.
- 모양: 칸 `Safe | Needs approval`. 후자는 에이전트가 건드리지 않음.
- 왜 보드: 위험 등급이 위치로 보인다. 승인 요청이 한곳에 모이면 일괄 처리 가능.
- 얻는 것: 승인 피로를 줄이는 등급제(저위험은 자동, 고위험만 대기).
- 근거: 티어드 승인 권고 https://workos.com/blog/approval-fatigue-agent-governance
- 위험·한계: 등급 기준을 에이전트가 정하면 낮춰 매길 수 있다.
- 계측: Needs approval 카드가 사용자 이동 없이 실행되지 않았는가.

### B9. 변경 가능한 로드맵: 칸 = 단계
- 상황: 장기 프로젝트(주 단위)에서 사용자가 일정을 훑음.
- 모양: `Now | Next | Later`. 에이전트가 카드를 `Now`에서 `Done`으로.
- 왜 보드: 큰 그림과 현재 위치를 함께 본다.
- 얻는 것: 방향이 틀어지면 카드를 직접 이동.
- 근거: 내 추론.
- 위험·한계: 카드가 너무 많으면 읽히지 않음.
- 계측: Now 칸 카드 수가 상한(예: 3)을 넘지 않았는가.

### B10. 실패한 카드의 역방향 이동
- 상황: Done이라 했던 카드의 테스트가 나중에 깨짐.
- 모양: `stickypane card work move login --to Doing` 후 `say`로 이유.
- 왜 보드: 되돌린 사실이 기록이다. 신뢰도 보정에 쓰인다.
- 얻는 것: 에이전트의 "끝났다"를 얼마나 믿을지 근거가 쌓인다.
- 근거: Anthropic: 신뢰는 경험으로 쌓이고 숙련 사용자는 개입도 더 한다 https://www.anthropic.com/research/measuring-agent-autonomy
- 위험·한계: 되돌림을 에이전트가 숨기고 새 카드를 만드는 경우.
- 계측: 되돌림 이후 같은 카드 제목으로 재진입했는가, 중복 카드가 생겼는가.

---

## 3. chat (`stickypane say`)

### T1. 방향 전환 알림
- 상황: 발견한 사실이 계획을 바꿈(레거시 폴더 발견).
- 모양: `stickypane say chat "found a legacy folder; plan changes"`.
- 왜 보드: 진행 막대와 달리 이유를 말할 곳이 필요하다. 사용자는 보드를 보고 있다.
- 얻는 것: 놀라지 않는다.
- 근거: talking-in-chat 스킬(바뀐 계획을 말하라).
- 위험·한계: 모든 걸 말하면 소음. 스킬은 "모든 단계는 말하지 말라"고 한다.
- 계측: 계획 변경 직후 N초 안에 `say`가 있었는가.

### T2. 비차단 지시 수신
- 상황: 사용자가 작업 중 "legacy는 건너뛰어"를 보드에서 침.
- 모양: `stickypane watch --once --note chat --type message.added --timeout 1s` 로 단계 사이마다 확인.
- 왜 보드: 채팅 창 대신 보드에서 말한다. 에이전트가 차단되지 않는다.
- 얻는 것: 실행 중 조종.
- 근거: talking-in-chat 스킬. 사람이 끼어드는 빈도가 숙련도와 함께 늘어남 https://www.anthropic.com/research/measuring-agent-autonomy
- 위험·한계: 폴링 사이 사이의 지연. 확인을 건너뛰는 에이전트.
- 계측: 사용자 메시지 시각 이후 첫 도구 호출이 지시를 반영했는가.

### T3. 한 문장 답이 필요한 질문
- 상황: 선택지가 아닌 서술 답("어떤 이름이 좋아?").
- 모양: `say chat "name for the new module?"` 후 `watch --once ... --timeout 10m --json`.
- 왜 보드: 선택지 없는 답은 form보다 chat.
- 얻는 것: 가벼운 왕복.
- 근거: talking-in-chat 스킬("문장이면 chat, 선택이면 form").
- 위험·한계: 타임아웃(코드 3)에서 추측하는 것.
- 계측: 코드 3일 때 안전 경로를 택했는가.

### T4. 장시간 작업 중 짧은 상태 알림
- 상황: 40분짜리 마이그레이션.
- 모양: `say chat "migration: 14 files, started"`, 끝에 `say chat "done, 41/41 pass"`.
- 왜 보드: 알림이 쌓여 기록이 된다.
- 얻는 것: 시작, 변경, 끝의 세 점만으로 믿음.
- 근거: talking-in-chat 스킬.
- 위험·한계: 상태 알림이 진행 막대의 중복.
- 계측: 메시지 수가 단계 수보다 현저히 적은가.

### T5. 에이전트 대 사용자 이견
- 상황: 사용자의 지시가 위험해 보임.
- 모양: `say chat "that deletes the cache dir; confirm?"` 후 확인은 form으로 승격.
- 왜 보드: 이견을 같은 자리에서 남긴다.
- 얻는 것: 에이전트가 맹종하지도 몰래 어기지도 않는다.
- 근거: 내 추론.
- 위험·한계: 에이전트의 과잉 반문.
- 계측: 파괴적 지시 뒤 확인 메시지가 먼저 오는가.

### T6. 모호한 요청의 되묻기
- 상황: "빠르게"가 무슨 뜻인지 불명확.
- 모양: `say chat "faster = latency or build time?"`.
- 왜 보드: 잘못 달리기 전에 한 줄로 잡는다.
- 얻는 것: 오작업 방지.
- 근거: 복잡한 작업에서 에이전트의 질문이 늘어난다 https://anthropic.com/research/trustworthy-agents
- 위험·한계: 지나친 질문은 피로. "한 번만" 규칙(asking-the-user).
- 계측: 질문이 1회로 끝났는가, 같은 질문을 반복했는가.

### T7. 여러 에이전트의 서명 구분
- 상황: 두 에이전트가 한 chat 노트를 씀.
- 모양: `stickypane say chat --as codex-1 "tests pass"`.
- 왜 보드: 발화자가 줄마다 박힌다.
- 얻는 것: 누구 말인지 구분.
- 근거: `--as` 옵션(help say).
- 위험·한계: 이름을 바꿔 쓰는 에이전트.
- 계측: `--as` 이름이 세션 내내 일정한가.

### T8. 사용자 부재 중 쌓이는 질의 응답(비동기)
- 상황: 사용자가 자리를 비운 사이 에이전트가 안전 경로로 진행.
- 모양: `say chat "no reply; I took the safe option (skip migration)"`.
- 왜 보드: 돌아온 사용자가 읽고 뒤집을 수 있다.
- 얻는 것: 에이전트가 막히지 않고 사용자는 사후 개입.
- 근거: asking-the-user 스킬(시간 초과 시 안전 경로).
- 위험·한계: 안전 경로가 안전하지 않을 수 있다.
- 계측: 시간 초과 후 한 행동이 사전 정의한 "안전 경로"인가.

### T9. 피드백 루프: 사용자의 거절 이유
- 상황: 에이전트의 제안을 사용자가 짧게 거절.
- 모양: 사용자가 `n`으로 "no, keep the old API". 에이전트가 반영 후 `say`로 재확인.
- 왜 보드: 거절 이유가 기록으로 남아 다음 작업에 쓰인다.
- 얻는 것: LangGraph의 reject-with-feedback에 해당하는 흐름.
- 근거: https://docs.langchain.com/oss/python/human-in-the-loop (열리지 않음: 404)
- 위험·한계: 에이전트가 이유를 무시하고 같은 제안 반복.
- 계측: 거절 뒤 같은 제안이 재등장하는 횟수.

### T10. 인계 메모
- 상황: 세션 종료 직전에 다음 세션을 위해 남기는 요약.
- 모양: `say chat "next: wire the retry; tests in x_test.go flaky"`.
- 왜 보드: 파일이 다음 세션의 입력이다.
- 얻는 것: 재개 비용 절감.
- 근거: 내 추론.
- 위험·한계: 길어지면 노이즈. 한 줄 원칙.
- 계측: 새 세션이 첫 행동 전에 이 메시지를 읽었는가.

---

## 4. form (`stickypane wait` / `answers`)

### F1. 배포 목적지 선택
- 상황: 빌드가 초록, 어디로 갈지는 사용자 결정.
- 모양: asking-the-user 스킬의 form: `## Target`, `- ( ) staging`, `- ( ) production`, `[ Deploy ] [ Cancel ]` 후 `stickypane wait deploy --timeout 10m`.
- 왜 보드: 버튼 한 번이 명시적 승인이고, 답이 파일에 남는다.
- 얻는 것: 되돌리기 어려운 행동 앞의 정확한 정지점.
- 근거: https://docs.langchain.com/oss/python/human-in-the-loop (열리지 않음: 404) (approve/reject), 스킬.
- 위험·한계: 기본 선택이 `production`이면 반사 승인. 위험한 쪽을 기본으로 두지 않는다.
- 계측: 배포 명령이 `submitted` 이후에만 실행됐는가.

### F2. 계획 승인 게이트
- 상황: 큰 변경의 계획을 보여주고 진행 여부를 묻는다.
- 모양: 본문에 계획 요약, `[ Approve ] [ Revise ]`, 입력 `> ` 줄에 수정 의견.
- 왜 보드: 채팅의 "진행할까요?"는 놓치기 쉽지만 폼은 눈에 띄고 거부도 같은 동선.
- 얻는 것: Claude Code plan mode의 승인과 같은 효과를 에이전트 종류와 무관하게.
- 근거: https://code.claude.com/docs/en/permission-modes , https://flaviocopes.com/ai-glossary/plan-mode/
- 위험·한계: 계획이 길면 읽지 않고 승인. 요약 길이를 제한.
- 계측: `submitted: Revise`이면 수정 의견을 반영한 새 계획이 나왔는가.

### F3. 파괴적 명령 사전 확인
- 상황: `rm`, `git push --force`, DB 삭제.
- 모양: 대상과 영향 목록을 본문에, `[ Delete ] [ Cancel ]`.
- 왜 보드: 효과(무엇이 사라지나)를 보여주는 승인이 명령 문자열보다 낫다.
- 얻는 것: 프롬프트가 명령만 보여주는 문제를 줄인다.
- 근거: 프롬프트가 효과 대신 명령을 보여준다는 지적 https://arxiv.org/pdf/2604.13536
- 위험·한계: 효과 목록이 틀리면 승인이 거짓 정보 위에 선다.
- 계측: 폼의 목록이 실제 삭제 대상과 일치하는가(사후 diff).

### F4. 여러 항목 일괄 승인
- 상황: 의존성 업그레이드 8개를 하나씩 묻지 않고.
- 모양: `- [ ] bump lodash` 형태의 체크박스 8줄과 `[ Apply selected ]`.
- 왜 보드: 8번의 개별 프롬프트 대신 한 화면의 한 결정. 승인 피로 완화.
- 얻는 것: 선택적 승인.
- 근거: https://workos.com/blog/approval-fatigue-agent-governance (요청량 대 승인 시간 지표)
- 위험·한계: 전체 체크 후 누르는 습관. 일괄 승인은 개별 검토를 대체하지 않는다.
- 계측: 체크 안 한 항목을 적용하지 않았는가.

### F5. 모호한 요구의 분기 질문
- 상황: 요구가 둘로 읽힘(캐시 무효화 방식).
- 모양: `## Strategy`에 `- ( )` 2개, 각 옵션 아래 들여쓴 설명, `> ` 줄에 단서.
- 왜 보드: 선택지마다 근거 한 줄을 붙여 비교.
- 얻는 것: 추측 구현 방지.
- 근거: MCP elicitation이 구조화된 스키마 입력을 요청하는 것과 같은 형태 https://modelcontextprotocol.io/specification/2025-11-25/client/elicitation
- 위험·한계: 선택지가 에이전트의 프레이밍에 갇힌다. `> Other` 입력 줄을 둔다.
- 계측: 선택지 외 답변이 쓰였을 때도 반영했는가.

### F6. 비밀 값은 폼에 쓰지 않는다
- 상황: 에이전트가 API 키를 필요로 함.
- 모양: 폼에 키를 받지 않고, "환경 변수에 넣었으면 [ Done ]" 버튼만.
- 왜 보드: 폼 답은 평문 파일 `.sticky/*.md`에 남는다. 민감한 입력은 쓰면 안 된다는 규칙이 MCP에도 있다.
- 얻는 것: 키가 저장소에 새지 않는다.
- 근거: MCP는 form 모드에서 민감한 값을 받지 말라고 한다 https://modelcontextprotocol.io/specification/2025-11-25/client/elicitation
- 위험·한계: 사용자가 습관적으로 입력란에 붙여넣을 수 있다.
- 계측: `.sticky/`에서 키 패턴이 grep되지 않는가.

### F7. 시간 초과 시 안전 경로
- 상황: 사용자가 자리 비움.
- 모양: `stickypane wait deploy --timeout 10m` 이 코드 3 → 폼은 두고 배포하지 않음.
- 왜 보드: 코드 3을 명시적 신호로 쓴다.
- 얻는 것: 침묵을 동의로 해석하지 않는다.
- 근거: asking-the-user 스킬("시간 초과면 안전 경로, 절대 추측 금지").
- 위험·한계: 안전 경로가 정의되지 않은 작업.
- 계측: 코드 3 뒤 위험 행동이 없었는가.

### F8. 오래된 답 재사용 방지
- 상황: 같은 폼 파일을 다시 열어 질문.
- 모양: `submitted` 키가 남아 있으면 `wait`가 즉시 옛 답으로 돌아온다(스킬 명시).
- 왜 보드: 이 함정은 형식의 약점이므로 새 이름으로 폼을 만들거나 `hide` 후 새로 쓴다.
- 얻는 것: 옛 승인으로 새 위험 행동을 하지 않는다.
- 근거: asking-the-user 스킬. LangGraph도 결정과 호출을 ID로 짝지으라고 경고 https://docs.langchain.com/oss/python/human-in-the-loop (열리지 않음: 404)
- 위험·한계: 에이전트가 새 이름을 쓰는 것을 잊는다.
- 계측: 승인 폼의 `at` 시각이 질문 생성 시각보다 늦은가.

### F9. 사전 입력 폼: 작업 시작 전 선호 수집
- 상황: 새 작업 시작 때 테스트 방식, 범위, 브랜치 규칙.
- 모양: 라디오 + 체크박스 + 입력 줄, `[ Start ]`. 이후 `stickypane answers deploy --json`.
- 왜 보드: 한 번에 모아 받고 질문을 줄인다.
- 얻는 것: 중간에 끊기는 질문 감소.
- 근거: 숙련 사용자는 사전에 자율성을 주고 필요할 때 개입 https://www.anthropic.com/research/measuring-agent-autonomy
- 위험·한계: 질문이 너무 많으면 포기. 5개 이하로.
- 계측: 시작 이후 같은 사안을 다시 묻지 않았는가.

### F10. 검토 결과 판정(approve / request changes)
- 상황: 에이전트가 PR 초안을 만들고 사용자가 판정.
- 모양: 본문에 변경 요약과 `![[diff]]`, `[ Approve ] [ Request changes ]`, `> ` 이유.
- 왜 보드: 판정과 이유가 한 파일에 묶인다.
- 얻는 것: LangGraph의 approve / edit / reject 세 갈래를 본다.
- 근거: https://docs.langchain.com/oss/python/human-in-the-loop (열리지 않음: 404)
- 위험·한계: diff를 안 읽고 승인.
- 계측: Request changes 뒤의 커밋이 이유를 다루는가.

---

## 각 shape의 가장 강한 3개
- checklist: C4(사람 몫 단계를 `item.ticked`로 기다림), C1(코드 전에 계획 공개), C8(재개 체크포인트).
- board: B2(Review 칸이 승인 게이트), B8(위험 칸 분리로 승인 피로 완화), B3(정체 카드 표시).
- chat: T2(비차단 지시 수신), T8(부재 중 안전 경로 후 사후 보고), T9(거절 이유 기록).
- form: F1(되돌리기 어려운 행동 앞 정지점), F7(시간 초과는 동의가 아니다), F8(옛 답 재사용 방지).

## 선택·승인이 다른 shape와 엮이는 패턴

1. form이 diff/로그를 품는다: 폼 본문에 `![[run]]`(로그 노트 임베드)나 변경 요약을 넣어 효과를 보고 승인. 명령만 보여주는 프롬프트의 약점 보완. https://arxiv.org/pdf/2604.13536
2. form 제출이 checklist를 닫는다: `stickypane watch --note deploy --type form.submitted --exec '...'` 로 `todo ship check "deploy approved"` 같이 체크를 연결(connecting-notes의 `--exec`). 체크의 근거가 사용자 버튼이 된다.
3. board의 Review 칸 + chat 이유: 카드가 Review에 오면 `say`로 한 줄 요약, 사용자가 카드를 옮기거나 chat에 이유를 남긴다. Kagan식 사람 게이트. https://raw.githubusercontent.com/kagan-sh/kagan-legacy/main/README.md
4. chat에서 form으로 승격: 대화 중 선택지가 3개 이하로 좁혀지면 `say`로 알리고 form을 연다(스킬의 분업: 선택이면 form, 문장이면 chat).
5. 승인 진행 차트: `from: plan, work` 차트가 checklist 완료율과 board 칸별 카드 수를 세고, 승인 대기 폼 수를 함께 보여준다(connecting-notes: 차트는 폼의 답도 센다). 승인이 얼마나 쌓였는지로 승인 피로를 관측. https://workos.com/blog/approval-fatigue-agent-governance
6. 카드 이동이 에이전트를 깨운다: `watch --type card.moved --exec '[ "$STICKY_TO" = Done ] && ...'` 로 사람의 이동을 다음 작업의 트리거로. 반대로 Needs approval 칸의 카드가 Doing으로 옮겨질 때만 위험 작업 시작.
7. 체크리스트 항목에서 form으로 링크: `- [ ] confirm target [[deploy]]` 처럼 `[[링크]]`로 항목이 질문 폼을 가리키고, 폼이 제출될 때까지 항목은 미체크.
8. 시간 초과 흔적을 log로: `wait`가 코드 3이면 `stickypane log worklog --time "no answer, took safe path"` 로 사후 검토 가능.
9. 거절 이유의 재사용: form의 `> ` 이유 입력을 chat에 `say`로 한 줄 옮겨 다음 제안이 그것을 지키는지 확인.

## 계측 모음(eval 시나리오 후보)
- 대기 준수: 위험 행동 시각이 `submitted` 시각 뒤인가(F1, F3, B8).
- 코드 3 처리: 시간 초과 후 안전 경로를 택하는가(F7, T8).
- 옛 답 재사용: 승인 폼 `at`이 요청 이후인가(F8).
- 반영 지연: 사용자 메시지·카드 이동 후 에이전트 행동이 바뀌는 데 걸린 시간(T2, B6).
- 질문 밀도: 한 작업당 form 수와 평균 응답 시간. 시간이 줄고 수가 늘면 승인 피로의 징후(근거: WorkOS 지표 제안).
- 체크 진실성: 체크 시각 앞에 대응 증거 명령이 있는가(C2, C3).

---

## log / chart / script 셰이프: 에이전트 유스케이스 30선

확인한 사실 (stickypane kinds, help, docs/shapes.md, 두 SKILL.md 기준)

- log: `stickypane log <name> [--time] <text>`. 한 줄이 한 항목, 끝을 따라가고 위로 스크롤하면 멈춤. `.log` `.txt` `.out` 파일도 로그. 이벤트 `log.appended`.
- chart: `stickypane chart <name> set|add <label> <number>`. `label: number` 줄이 값, 나머지 줄은 위에 놓이는 글. `view: bar|spark|heat`(heat는 라벨이 날짜일 때). 음수 가능. 앞에서 본 `from:`은 다른 노트(체크리스트, 보드, 폼, 로그 줄 수)를 세어 값을 써 줌. 이벤트 `chart.changed`.
- script: `.sh`/`.ps1`. enter, Run 클릭, 더블클릭이면 매번 사용자에게 먼저 묻고(`Run deploy.sh in myproject?`) 프로젝트 폴더에서 실행. 출력은 같은 이름의 `.log`가 스크립트 아래에 열림. 보드 키 `enter` run. `stickypane show deploy.sh`로 보임.
- 반응: `stickypane watch [--note N] [--type T] [--json] [--exec CMD] [--once] [--timeout D]`, 환경변수 `STICKY_EVENT/NOTE/TYPE/ITEM/FROM/TO`. 이벤트 note.created, note.changed, item.ticked, card.moved, form.submitted, log.appended, chart.changed, message.added.
- 시각은 보드가 `--time`으로 쓴다(AGENTS.md: 에이전트가 손으로 쓰지 않음).
- 이 문서의 "보드에서의 모양"은 위에서 본 명령과 파일 형식만 쓴다. 스크립트 본문은 예시일 뿐 실행하지 않았다.

공통 근거 URL (아래 사례에서 번호로 부름)

- [S1] OTel GenAI 에이전트 스팬: https://opentelemetry.io/docs/specs/semconv/gen-ai/gen-ai-agent-spans/index.md (operation: create_agent, invoke_agent, invoke_workflow, plan, execute_tool. 상태 Development)
- [S2] OTel GenAI 메트릭: https://raw.githubusercontent.com/open-telemetry/semantic-conventions-genai/main/docs/gen-ai/gen-ai-metrics.md (`gen_ai.client.token.usage`, `gen_ai.client.operation.duration`)
- [S3] Claude Code 관측(SDK 문서): https://code.claude.com/docs/en/agent-sdk/observability (모델 요청과 도구 실행 스팬, 토큰/비용 카운터)
- [S4] AWS, 코딩 에이전트 인사이트: https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/coding-agents-insights.html (토큰, 턴당 지연, 도구 호출, 승인)
- [S5] Splunk, 도구 호출/계획 평가: https://www.splunk.com/en_us/blog/artificial-intelligence/agent-evaluation-tool-calling-planning.html (tool-call precision, argument accuracy)
- [S6] pass@k 설명(Arize): https://arize.com/?p=33399 와 CLEAR 논문 https://arxiv.org/pdf/2511.14136v1 (pass@k vs pass^k, 성공 1건당 비용)
- [S7] Trunk 불안정 테스트 대시보드: https://docs.trunk.io/flaky-tests/overview/dashboard (Healthy/Flaky/Broken)
- [S8] Metabase flaky 지표: https://www.metabase.com/metrics/flaky-test-rate (같은 커밋에서 통과와 실패가 공존해야 엄밀한 flaky, 낭비 시간 순 정렬)
- [S9] Cypress flaky 감지: https://www-production.cypress.io/blog/introducing-flaky-test-detection-alerts (심각도, 추세, 알림)
- [S10] Bencher: https://raw.githubusercontent.com/bencherdev/bencher/main/README.md ; Go 성능 모니터링: https://go.googlesource.com/wiki/+show/HEAD/PerformanceMonitoring.md (커밋마다 기록, 기준선과 비교, 잡음)
- [S11] cbdr: https://github.com/asayers/cbdr (두 커밋의 실행을 번갈아 돌려 잡음 분산)
- [S12] HITL 승인(LangChain): https://docs.langchain.com/oss/python/concepts/human-in-the-loop ; TrueFoundry https://www.truefoundry.com/docs/agent-platform/agent-harness/human-in-the-loop (상태 변경, 되돌릴 수 없는 일, 운영 영향이 있는 도구에 승인. 거절은 에이전트에게 돌려줌)
- [S13] n8n HITL 도구: https://docs.n8n.io/build/integrate-ai/ai-examples/human-in-the-loop-for-tools (삭제, 외부 발신, 결제 예시)
- [S14] 기여 히트맵 구현: https://towardsdatascience.com/create-githubs-style-contributions-plot-for-your-time-series-data-79df84ec93da (7행 x 주 열, 칸 하나가 하루)
- [S15] Claude Code 비용 추적(Bindplane): https://bindplane.com/blog/claude-code-opentelemetry-per-session-cost-and-token-tracking (세션별 토큰/비용; 내용은 상세 미확인)

---

## 1. log 셰이프 (10)

## L1. **작업 일지 (한 줄씩, 시각 포함)**
- 상황: 에이전트가 20분 넘게 일하고 사용자는 다른 창을 본다. 단계가 바뀔 때마다.
- 모양: `stickypane log worklog --time "tests passed, 3 flaky left"` 한 줄이 `14:02 tests passed...`로 쌓임.
- 왜 보드: 채팅은 새 답이 오면 위로 밀리고 알림이 쌓인다. 로그는 제자리에서 끝만 따라간다.
- 얻는 것: 눈 한 번으로 "지금 어디"와 "무엇이 끝났나".
- 근거: tracking-progress SKILL.md의 예시 그대로. 관측 면에서는 [S1]의 execute_tool 스팬을 사람이 읽는 줄로 줄인 것(내 추론).
- 위험·한계: 줄이 너무 잦으면 소음. 한 줄에 무엇을 했는지와 결과가 같이 있어야 함.
- 계측: 작업 종료 시 `.sticky/worklog`의 줄 수 >= 1, 시각이 단조 증가, 마지막 줄이 결과(성공/실패)를 말하는가.

## L2. **실패한 명령의 마지막 20줄**
- 상황: 빌드나 테스트가 실패했을 때, 에이전트가 원인에 가까운 줄만 남긴다.
- 모양: 파이프로 `.log` 파일에 덧붙이거나 `stickypane log build "FAIL internal/store: want 3 got 2"`.
- 왜 보드: 전체 출력은 채팅을 덮는다. 보드 로그는 사용자가 원할 때 스크롤.
- 얻는 것: 에이전트의 요약과 원문 근거가 한 곳에.
- 근거: 내 추론. 빌드/테스트 로그를 요약해 남기는 관행 일반.
- 위험·한계: 요약이 틀리면 사용자가 속는다. 원문 파일 경로를 같이 적어야 함.
- 계측: 로그의 오류 줄이 실제 실패 출력에 부분 문자열로 존재하는가(환각 검사).

## L3. **결정 기록 (왜 이렇게 했나)**
- 상황: 에이전트가 두 길 중 하나를 고를 때.
- 모양: `stickypane log decisions --time "kept sqlite: no new module, README says stdlib first"`.
- 왜 보드: 채팅의 결정은 파묻힌다. 로그는 나중에 위로 읽는 기록.
- 얻는 것: 나중에 "그때 왜"에 답이 있음.
- 근거: 내 추론(결정 로그/ADR 관행).
- 위험·한계: 사용자가 정해야 할 일을 에이전트가 혼자 정하고 적어 두기만 할 수 있다. 그건 `asking-the-user` 영역.
- 계측: 결정 줄이 "무엇을, 왜"를 둘 다 담는가. 사용자 몫의 결정이 로그에만 있고 폼 질문이 없으면 감점.

## L4. **막힘과 재시도 로그**
- 상황: 같은 오류에 두 번 이상 부딪힐 때.
- 모양: `stickypane log worklog --time "retry 2/3: port busy, picked 8081"`.
- 왜 보드: 사용자가 루프를 일찍 알아채고 개입한다.
- 얻는 것: 에이전트가 헛돌고 있는지가 보임.
- 근거: 에이전트 평가에서 단계 수와 반복이 효율 지표로 쓰임 [S5][S6]. 보드 적용은 내 추론.
- 위험·한계: 에이전트가 루프 중이면 로그도 스스로 폭주할 수 있다.
- 계측: 같은 문구의 연속 줄 수 상한. 재시도 횟수 대비 최종 성공 여부.

## L5. **도구 호출 감사 로그**
- 상황: 파일 삭제, 설치, 네트워크 호출처럼 영향이 큰 동작을 할 때.
- 모양: `stickypane log audit --time "rm -r build/ (generated)"`.
- 왜 보드: 되돌릴 수 없는 일의 기록은 사용자 눈앞에 있어야 한다.
- 얻는 것: 사후 추적과 신뢰.
- 근거: 상태 변경 도구에 승인과 기록을 두는 HITL 가이드 [S12][S13].
- 위험·한계: 에이전트가 적지 않으면 아무도 모른다(자가 보고). 강제는 hook이 해야 함.
- 계측: 평가 샌드박스에서 실제 파괴적 명령을 심고, 로그에 대응 줄이 있는 비율.

## L6. **사용자 지시 수신 확인 로그**
- 상황: 사용자가 `watch`로 전달된 이벤트(폼 제출, 카드 이동)를 보냈을 때.
- 모양: `stickypane log worklog --time "got: moved login to Done, running tests"`.
- 왜 보드: 입력이 보드에서 왔으니 확인도 보드에서.
- 얻는 것: 입력이 누락되지 않았다는 확신.
- 근거: connecting-notes SKILL.md의 이벤트 목록. 확인 줄은 내 추론.
- 위험·한계: 이벤트가 많으면 중복.
- 계측: 평가에서 이벤트를 주입하고 N초 안에 대응 로그 줄이 생기는지.

## L7. **세션 인수인계 메모**
- 상황: 컨텍스트가 길어져 새 세션이 이어받을 때.
- 모양: 끝에 `stickypane log handoff "next: fix store_test; branch fix/x; do not touch docs/"`.
- 왜 보드: 파일이라 세션이 바뀌어도 남는다. 채팅은 사라진다.
- 얻는 것: 다음 에이전트(또는 같은 에이전트)가 빈손으로 시작하지 않는다.
- 근거: 내 추론(보드가 폴더의 일반 파일이라는 사실에서).
- 위험·한계: 오래된 메모가 틀린 정보를 남김. 날짜(--time)와 브랜치를 같이 적어야 함.
- 계측: 새 세션에게 로그만 주고 이어서 하게 한 뒤 재질문 횟수를 센다.

## L8. **원격/긴 작업의 하트비트**
- 상황: 마이그레이션이나 대용량 다운로드처럼 몇 분 걸리는 일.
- 모양: `stickypane log job --time "batch 12/40 done"`.
- 왜 보드: 멈췄는지 느린지를 사용자가 판단하게.
- 얻는 것: 불안 감소. 마지막 줄 시각으로 정체를 알아봄.
- 근거: 내 추론. 지연을 지표로 보는 점은 [S2]의 operation.duration과 같은 발상.
- 위험·한계: 하트비트 빈도를 정하지 않으면 로그가 부풀어 오른다.
- 계측: 줄 간격 분포와 최대 공백이 N분을 넘지 않는가.

## L9. **스크립트 실행 결과 해석 줄**
- 상황: 사용자가 Run으로 돌린 스크립트의 `.log`가 생겼을 때, 에이전트가 읽고 결론을 덧붙임(별도 로그에).
- 모양: `stickypane log deploy-notes --time "deploy.log: 2 warnings, 0 errors"`.
- 왜 보드: 원문 `.log`와 해석을 나란히 놓는다.
- 얻는 것: 긴 출력을 읽지 않고도 결과를 안다.
- 근거: shapes.md(스크립트 출력은 같은 이름의 .log). 해석은 내 추론.
- 위험·한계: 원문을 읽지 않고 쓰면 환각. 실행 전에 줄을 쓰면 거짓.
- 계측: 해석 줄의 수치(경고 수 등)가 `.log` 원문의 실제 수와 일치하는가.

## L10. **평가 실행 결과 로그 (스킬 평가 자체)**
- 상황: stickypane 스킬을 평가하는 하네스가 케이스별 결과를 보드에 쓴다.
- 모양: `stickypane log eval --time "case tracking-3: PASS (4 cmds, 0 retries)"`.
- 왜 보드: 평가 진행을 사람이 실시간으로 본다.
- 얻는 것: 실패한 케이스를 즉시 열어 본다.
- 근거: [S5][S6] 지표를 케이스 단위 줄로 기록. 형식은 내 추론.
- 위험·한계: 하네스가 쓴 줄이라 에이전트 자가 보고와 섞이면 안 됨.
- 계측: 로그 줄 수 == 실행한 케이스 수.

---

## 2. chart 셰이프 (10)

## C1. **토큰 사용량 막대 (입력/출력/캐시)**
- 상황: 긴 작업 중 비용이 걱정될 때.
- 모양: `stickypane chart tokens add input 1200`, `stickypane chart tokens add output 300`, 값은 `input: 1,200` 형태로 막대.
- 왜 보드: 숫자가 늘어나는 걸 한눈에. 채팅 숫자는 흘러간다.
- 얻는 것: 비용 감각.
- 근거: [S2] token.usage는 input/output으로 나눔. [S3][S4] 코딩 에이전트의 토큰 카운터. 에이전트가 자기 토큰을 아는지는 환경에 달림.
- 위험·한계: 에이전트는 정확한 토큰을 모를 수 있다. 추정이면 "추정"이라고 title에 써야 함.
- 계측: 보드 값과 SDK/OTel이 보고한 실제 값의 오차율.

## C2. **일별 토큰 히트맵**
- 상황: 몇 주에 걸친 사용 패턴.
- 모양: `view: heat`, 줄 `2026-09-29: 8,900`. 날짜별 `stickypane chart tokens add 2026-09-29 8900`.
- 왜 보드: 달력 칸으로 급증한 날이 보인다.
- 얻는 것: "그 날 뭐 했지"를 찾는다.
- 근거: shapes.md의 heat 예시. 7행 x 주 열 달력 관행 [S14].
- 위험·한계: 날짜 키를 에이전트가 직접 쓰면 AGENTS.md의 시각 규칙(시각은 보드가 쓴다)과 어긋날 수 있음. 날짜 라벨은 값 이름이라 허용인지 확인 필요.
- 계측: 라벨이 YYYY-MM-DD 형식이고 한 날에 한 줄인가.

## C3. **세션 비용 추세 (spark)**
- 상황: 작업당 비용이 오르는지 보고 싶을 때.
- 모양: `view: spark`, 줄마다 `task1: 0.42`, `task2: 0.61`.
- 왜 보드: 한 줄 추세는 채팅에 쓸 수 없는 모양.
- 얻는 것: 비용 회귀 감지.
- 근거: 성공 1건당 비용을 함께 보라는 제안 [S6]. 비용 수집은 [S3][S15].
- 위험·한계: 구독제는 API 환산 비용이 실제 청구가 아님(검색 결과 중 Dash0 설명).
- 계측: 값이 숫자이고 음수/NaN이 없고 라벨이 유일한가.

## C4. **테스트 통과/실패 막대**
- 상황: 테스트 스위트를 돌린 직후.
- 모양: `stickypane chart tests set pass 41`, `stickypane chart tests set fail 0`.
- 왜 보드: 숫자 두 개를 계속 같은 자리에 둠.
- 얻는 것: 최신 상태.
- 근거: 내 추론. 테스트 대시보드 일반 [S7].
- 위험·한계: set을 쓰면 이전 값이 사라진다(추이는 안 남음). 추이는 C5처럼.
- 계측: set 후 값이 실제 `go test` 출력의 수와 같은가.

## C5. **실행마다 하나씩 쌓는 flaky 비율 추세**
- 상황: 테스트를 여러 번 돌려 불안정한 테스트를 찾는 중.
- 모양: `view: spark`; `stickypane chart flake add run1 2`.
- 왜 보드: 추세선이 "나아지고 있나"를 말해 준다.
- 얻는 것: 고쳤는지의 증거.
- 근거: flake rate의 롤링 추적과 Healthy/Flaky/Broken 분류 [S7][S9]. 같은 커밋 통과/실패 공존이 엄밀한 정의 [S8].
- 위험·한계: 몇 번 돌려서는 통계가 약하다. 표본 수를 같이 적어야 함.
- 계측: 표본 수가 함께 기록되는가. "고쳤다"는 주장에 N회 연속 통과 증거가 있는가.

## C6. **벤치마크 값 추세 (커밋마다)**
- 상황: 성능이 중요한 코드를 바꾼 뒤.
- 모양: `view: spark`; `stickypane chart bench set abc123 412`(ms).
- 왜 보드: 커밋별 한 점이 줄로 이어짐.
- 얻는 것: 회귀를 눈으로 본다.
- 근거: 커밋마다 기록하고 기준선과 비교 [S10]. 잡음 [S10][S11].
- 위험·한계: 한 번 잰 값은 잡음. 번갈아 측정하는 방법 [S11]이 낫다.
- 계측: 기준선 대비 값을 함께 기록하고, 측정 횟수가 1이 아닌가.

## C7. **체크리스트 진행률 (from:)**
- 상황: 작업 단계가 체크리스트로 있을 때.
- 모양: 노트 front matter `from: plan, work`(connecting-notes SKILL.md)로 보드가 완료/미완료를 센다.
- 왜 보드: 에이전트가 숫자를 직접 세지 않아도 틀리지 않는다.
- 얻는 것: 전체 진행의 한 줄 요약.
- 근거: connecting-notes SKILL.md의 "from: before a watch".
- 위험·한계: 원본 노트가 정확해야 의미 있음.
- 계측: 에이전트가 숫자를 수기로 쓰지 않고 `from:`을 쓰는가(파일 front matter 검사).

## C8. **작업 종류별 시간/단계 분포**
- 상황: 하루 끝에 무엇에 시간을 썼는지.
- 모양: `stickypane chart effort add review 3`, `add debug 5`(막대).
- 왜 보드: 비율이 바로 보임.
- 얻는 것: 반복되는 낭비를 알아봄(낭비 시간 순 정렬 관행 [S8]).
- 근거: 내 추론.
- 위험·한계: 범주를 에이전트가 임의로 붙이면 일관성 없음. 고정 라벨 목록 필요.
- 계측: 라벨이 사전에 정한 집합 안에 있는 비율.

## C9. **도구 호출 성공/실패 막대**
- 상황: 에이전트가 자기 도구 호출을 센다(스스로 또는 hook이).
- 모양: `stickypane chart tools add ok 1`, `add error 1`.
- 왜 보드: 오류 비율이 나빠지는 순간을 본다.
- 얻는 것: 환경 문제나 잘못된 접근의 조기 발견.
- 근거: tool-call precision과 argument accuracy [S5]. 에이전트 도구 호출/승인 수 [S4].
- 위험·한계: 자가 계수는 부정확. hook이나 watch로 집계하는 편이 낫다(connecting-notes의 watch --exec).
- 계측: 보드 수치와 실제 로그의 도구 호출 수 비교.

## C10. **보드 활동 히트맵 (로그 줄 수에서 계산)**
- 상황: 한 달 동안 에이전트가 보드에 얼마나 썼는지.
- 모양: `from: worklog`처럼 로그 줄 수를 세는 차트(connecting-notes: "lines"를 센다). 날짜별 달력은 라벨이 날짜일 때.
- 왜 보드: 활동이 몰린 날과 빈 날이 달력에 보인다.
- 얻는 것: 일하는 리듬.
- 근거: 기여 그래프 관행 [S14]. `from:`이 로그 줄 수를 센다는 점은 connecting-notes SKILL.md.
- 위험·한계: `from:`이 날짜별로 나누어 세는지는 확인하지 못함. 날짜별로 안 되면 C2처럼 직접 쓴다.
- 계측: 집계 수치가 원본 로그 줄 수와 같은가.

---

## 3. script 셰이프 (10)

공통 안전 원칙: 스크립트는 파일이라 에이전트가 쓸 수 있지만 실행은 사용자의 "예" 이후에만. 승인 대상은 상태 변경, 되돌릴 수 없는 일, 운영 영향 [S12][S13]. 거절은 에이전트가 알 수 있어야 한다(S12는 거절을 에이전트에게 돌려주라고 권함. stickypane에서 어떻게 알려지는지는 미확인: 출력 `.log` 유무로 추정).

## S1. **배포 런북 (deploy.sh)**
- 상황: 빌드가 초록이고 사용자가 배포를 결정할 때.
- 모양: `deploy.sh`에 `#!/bin/sh` + 단계 명령. `stickypane show deploy.sh`. 출력은 `deploy.log`.
- 왜 보드: 사용자가 내용을 읽고, 바로 아래에서 결과를 본다.
- 얻는 것: "읽고 승인하고 결과 확인"이 한 화면.
- 근거: shapes.md(스크립트는 매번 묻는다). 승인 패턴 [S12].
- 위험·한계: 승인한 사람이 스크립트를 읽지 않으면 형식적. 스크립트는 짧고 한 가지 일만.
- 계측: 에이전트가 직접 배포 명령을 실행하지 않고 스크립트로 제안했는가(트랜스크립트 검사).

## S2. **마이그레이션 드라이런 후 실제 실행**
- 상황: DB나 파일 이동처럼 되돌리기 어려운 일.
- 모양: `migrate-dry.sh`와 `migrate.sh` 두 파일. 드라이런 `.log`를 먼저 본다.
- 왜 보드: 두 단계를 나란히 놓고 순서를 사용자가 지킨다.
- 얻는 것: 실행 전에 영향을 안다.
- 근거: 되돌릴 수 없는 일에 승인 [S13]. 드라이런 순서는 내 추론.
- 위험·한계: 드라이런이 실제와 다르면 안심이 거짓.
- 계측: 비파괴 드라이런이 먼저 존재하는가. 실제 스크립트가 드라이런 결과를 전제하는가.

## S3. **테스트 반복 실행기 (flaky 찾기)**
- 상황: 불안정한 테스트를 의심할 때.
- 모양: `flake.sh`: `go test -count=10 ./...` 한 줄. 결과는 `flake.log`.
- 왜 보드: 오래 걸리는 일을 사용자 시점에 돌린다.
- 얻는 것: 에이전트 대신 사용자가 시간을 정함.
- 근거: flaky는 반복 실행으로 판별 [S8]. 
- 위험·한계: 읽기 전용이지만 자원을 쓴다. 에이전트 직접 실행도 안전한 경우가 많아 굳이 스크립트가 필요한가 판단해야 함.
- 계측: 안전한 읽기 명령을 쓸데없이 스크립트로 올리지 않았는가(과잉 승인 요구는 피로를 만든다).

## S4. **벤치마크 실행 + 차트 갱신**
- 상황: 성능 변경을 검증할 때.
- 모양: `bench.sh`가 벤치 실행 후 `stickypane chart bench set <commit> <ms>`로 쓴다.
- 왜 보드: 실행 결과가 차트 C6로 이어진다.
- 얻는 것: 측정과 기록이 한 번에.
- 근거: [S10][S11].
- 위험·한계: 노이즈. 사용자가 다른 일을 하는 중에 돌리면 값이 흔들림.
- 계측: 스크립트가 실제로 `stickypane chart` 명령을 쓰는가(파일 검사).

## S5. **정리/삭제 스크립트 (빌드 산출물)**
- 상황: 용량 정리, 캐시 삭제.
- 모양: `clean.sh`에 삭제 대상 경로를 명시적으로. 와일드카드와 `rm -rf $VAR` 금지.
- 왜 보드: 삭제는 사용자가 목록을 보고 승인.
- 얻는 것: 에이전트가 혼자 지우지 않는다.
- 근거: 삭제 데이터는 승인 대상 [S13].
- 위험·한계: 변수가 비면 엉뚱한 곳이 지워진다.
- 계측: 스크립트에 `rm`이 있을 때 경로가 고정되어 있고 프로젝트 안인가.

## S6. **환경 점검 스크립트 (읽기 전용)**
- 상황: 세션 시작 시 도구 버전과 설정을 확인.
- 모양: `doctor.sh`: `go version; git status`. 출력 `doctor.log`.
- 왜 보드: 결과가 로그로 남아 나중에도 본다.
- 얻는 것: 재현 가능한 점검.
- 근거: 내 추론.
- 위험·한계: 읽기 전용이면 승인 클릭이 마찰. 사용자에게 가치가 있는 경우만.
- 계측: 점검 스크립트에 쓰기 명령이 없는가.

## S7. **릴리스 체크 스크립트**
- 상황: 태그 전에 vet, test, gofmt를 모두 돌린다(이 저장소의 "Before you finish" 명령).
- 모양: `precheck.sh`: `go vet ./... && go test -race ./... && test -z "$(gofmt -l .)"`.
- 왜 보드: 태그와 푸시는 사용자 소관(AGENTS.md: 푸시/태그 전에 묻는다). 점검은 사용자 눈앞에서.
- 얻는 것: 릴리스 직전의 결과가 한 파일에.
- 근거: AGENTS.md.
- 위험·한계: 점검 통과가 안전을 보장하지는 않는다.
- 계측: 에이전트가 태그/푸시 명령을 스크립트에 넣지 않았는가.

## S8. **재현 스크립트 (버그 보고)**
- 상황: 에이전트가 버그를 찾아 재현 방법을 사용자에게 보여줄 때.
- 모양: `repro.sh`가 실패하는 명령을 그대로. 출력 `repro.log`.
- 왜 보드: 사용자가 직접 눌러 확인한다.
- 얻는 것: 말이 아니라 재현.
- 근거: 내 추론.
- 위험·한계: 환경 의존이면 사용자 쪽에서 다른 결과.
- 계측: repro.sh가 실제로 실패를 재현하는가(평가 환경에서 에이전트가 쓴 스크립트를 실행해 종료 코드 확인. 평가 하네스가 하는 것이지 이 조사에서 한 것이 아님).

## S9. **서버/개발 환경 기동 스크립트**
- 상황: 앱 확인을 위해 서버를 띄운다.
- 모양: `serve.sh`: 백그라운드 프로세스를 띄울 때는 종료 방법도 로그에.
- 왜 보드: 포트를 쓰는 프로세스는 사용자가 알고 승인.
- 얻는 것: 잔여 프로세스를 줄임.
- 근거: 내 추론.
- 위험·한계: 오래 사는 프로세스는 출력 `.log`가 계속 커진다.
- 계측: 기동 스크립트에 종료 안내가 있는가.

## S10. **조건부 후속 작업 (watch와 함께)**
- 상황: 폼에서 사용자가 "Deploy"를 누르면 스크립트를 제안.
- 모양: `stickypane watch --note deploy --type form.submitted --once`로 기다린 뒤, 에이전트가 `deploy.sh`를 보드에 올림. 실행은 여전히 사용자의 Run.
- 왜 보드: 폼 승인과 실행 승인이 분리된다(질문, 실행의 두 겹).
- 얻는 것: 사용자의 의도와 실행 사이에 한 번 더 확인.
- 근거: connecting-notes SKILL.md의 watch. HITL 단계 분리는 [S12].
- 위험·한계: 이중 승인 피로. 위험이 큰 일에만.
- 계측: 폼 제출 전에 스크립트를 올리지 않았는가. 폼 거절 시 스크립트가 없는가.

---

## 4. 계측 설계

## 4.1 기록할 지표 (출처 포함)

| 지표 | 정의 | 출처 |
| --- | --- | --- |
| 과제 성공률 pass@1 | 한 번에 성공한 비율 | [S6] |
| pass^k (k회 모두 성공) | 일관성. 스킬이 안정적인지 | [S6] (p^k 가정, 독립 시행) |
| 성공 1건당 비용 | 총 비용 / 성공 건수 | [S6] |
| 토큰 (input/output/cache 구분) | 스킬 문서 길이의 비용 | [S2][S3] |
| 지연: 총 시간, 첫 의미 있는 동작까지 | 보드 갱신이 늦지 않은가 | [S2][S6] |
| 명령 수 / 재시도 수 | 스킬이 단계를 줄이는가 | [S5][S6] |
| 도구 호출 precision, argument accuracy | 실제 존재하는 명령과 올바른 인수를 썼는가(예: `chart tokens add input 1200` 형식) | [S5] |
| 승인 필요 동작에서의 준수율 | 스크립트 없이 위험 명령을 직접 실행하지 않았나 | [S12][S13] (적용은 내 추론) |
| 소음 비율 | 쓸데없는 로그 줄/알림 비율 | 내 추론 |
| 환각 검사 | 로그/차트 수치가 실제 출력과 일치 | 내 추론 (L2, L9, C4에서 사용) |

스팬/트레이스 형식은 [S1]의 operation(plan, execute_tool, invoke_agent)을 참고로 케이스를 "작업, 도구 실행" 두 층으로 기록. 단 GenAI 규약은 Development 상태라 속성명이 바뀔 수 있다. 각 평가 실행에 모델 이름, 스킬 버전, 시간 제한을 같이 적는 것은 [S6] 요약의 권고.

## 4.2 판정 방식 (스킬별 정답 검사)

1. 형식 검사(결정적): 결과 파일을 `stickypane kinds log|chart|script`와 대조. front matter `type`, `label: number` 줄, `.sh`의 shebang 등. 존재하지 않는 명령(`stickypane help`에 없는 것)을 쓰면 실패(tool-call precision).
2. 값 검사(결정적): 차트와 로그의 수치를 실제 출력에서 다시 계산해 비교.
3. 행동 검사(트랜스크립트): 승인이 필요한 동작을 스크립트로 올렸는가, 시각을 손으로 쓰지 않았는가(`--time` 사용), 사용자 몫의 결정을 질문했는가.
4. 안정성: 같은 케이스를 k번 돌려 pass^k. 한 번만 성공하는 스킬은 약한 스킬.
5. 모델 채점은 마지막: 줄이 "한 줄에 한 사실"이고 `AI 말투` 없이 짧은가(memory의 문구 기준)는 사람이 샘플 검토.

## 4.3 평가 결과를 보드에 보이는 법 (stickypane 자체로)

- 케이스별 줄: `stickypane log eval --time "case logging-2: PASS 4 cmds"`. 실패는 FAIL과 이유 한 줄.
- 스킬별 성공률 막대: `stickypane chart eval-pass set tracking-progress 9` (10회 중 9). 같은 노트에 `eval-total` 대신 상단 글로 "10 runs each"를 적는다.
- 버전별 추세: `view: spark`로 `v0.4.0: 8`, `v0.4.1: 9`. 스킬을 고칠 때마다 한 점.
- 일별 히트맵: 평가를 매일 돌린다면 `view: heat`에 `2026-11-23: 27`(통과한 케이스 수). 저조한 날이 보인다.
- 비용: `stickypane chart eval-tokens add input <n>`과 `output`.
- `from:`: 케이스 체크리스트 노트를 하나 만들어 두고 `from:` 차트로 완료율을 자동 집계.
- 평가 하네스가 쓴 값과 에이전트의 자가 보고를 같은 노트에 섞지 않는다.
- 한계: `chart set`은 이전 값을 지운다. 이력이 필요하면 버전마다 라벨을 새로 쓰거나 log에 남긴다.

---

## 5. 가장 강한 3개씩

## log
1. **L9 스크립트 결과 해석 줄**: 원문 `.log`와 해석이 나란히, 수치로 환각 검사가 쉬움.
2. **L5 도구 호출 감사 로그**: 승인 가이드 [S12][S13]와 직결, 계측도 명확(파괴 명령을 심고 대응 줄 확인).
3. **L7 인수인계 메모**: 파일이라는 보드의 본질(세션을 넘어 남음)을 가장 잘 씀. 재질문 횟수로 측정.

## chart
1. **C7 from: 진행률**: 에이전트가 세지 않아 틀릴 수 없음. 실제 지원되는 기능.
2. **C5 flaky 추세**: [S7][S8][S9]에 근거가 두터움. 표본 수 기록을 규칙으로.
3. **C1 토큰 막대**: [S2][S3][S4]로 업계 지표와 맞고 사용자가 가장 먼저 궁금해함. 다만 추정치임을 밝혀야 함.

## script
1. **S1 배포 런북**: 사용자 승인이라는 stickypane의 핵심 약속을 가장 정직하게 보여 줌. [S12]
2. **S7 릴리스 체크**: 이 저장소의 실제 점검 명령과 일치, 태그/푸시는 사용자 몫이라는 규칙과 맞음.
3. **S10 폼 후 스크립트**: 의도 승인과 실행 승인을 나눠 이중 안전. 과용하면 피로이므로 위험한 일에만.

---

## 6. 조사의 한계

- OTel GenAI 규약은 Development 상태이고, 메트릭 이름은 이차 자료에서 확인했으므로 정확한 이름은 스펙에서 다시 볼 것.
- 임계값(예: flaky 5%)은 서드파티 문서의 값이라 기준으로 삼지 않았다.
- stickypane의 날짜별 `from:` 집계, 스크립트 거절이 에이전트에게 전달되는 방식, `stickypane write`로 스크립트를 만드는 방법은 이번에 확인하지 못했다. 해당 사례에는 "미확인"을 달았다.
- 소개한 모든 스크립트는 실행하지 않았고 보드와 저장소는 수정하지 않았다.

---

## api 셰이프 · worktree · 선행 사례 조사

작성일 2026-11-23. 근거는 두 갈래다. (1) 이 저장소에서 직접 실행하고 읽은 것: `stickypane kinds api`, `stickypane help api`, docs/shapes.md, VISION.md, worktree-design.md, worktree-issues-draft.md. (2) 오늘 가져온 웹 페이지. 웹 문서가 확인해 주지 않은 것은 "내 추론"으로 표시했다. 웹 요약은 작은 모델이 줄인 것이라 세부 수치는 원문 재확인이 필요하다.

## 0. 확인한 사실 (이 아래 모든 명령과 형식은 여기서 나온다)

- 만들기: `stickypane api api.http "Log in"`. 파일은 `### 이름`으로 요청을 나누고 `@base = https://httpbin.org`, `{{base}}` 변수를 쓴다.
- 검사와 캡처: `# @assert response.statusCode == 200`, `# @assert response.json("json.user") == "min"`, `# @capture file token {{response.json.token}}`.
- WebSocket은 `# @websocket ...`과 `# @ws send/send-json/wait/close` 단계, gRPC는 `GRPC host:port`와 `# @grpc package.Service/Method`, `# @grpc-plaintext true`, `# @grpc-descriptor file.protoset`(없으면 서버 reflection, 그것도 없으면 필드 번호).
- 환경은 `rest-client.env.json` 또는 `resterm.env.json`, `# @env dev`, 값은 `env:NAME`, `{{$processEnv NAME}}`, 프로젝트 `.env`.
- stickypane만 읽는 것은 `# @pre <command>`, `# @post <command>`. resterm과 REST Client에서는 이 줄이 무시되거나 의미가 없을 것이다(내 추론).
- 보드에서 키: `j k` 요청 선택, `enter` 보내기(사용자에게 먼저 묻는다), `e` 환경 바꾸기.
- 에이전트용 CLI: `stickypane api <name> [request] [--env E] [--all]`. 이 형태는 묻지 않고 보낸다. 결과는 같은 이름의 `.log`에 쌓이고 마지막 줄이 `[200 OK · 38ms · Log in · 2/2 ✔]` 꼴이다. 요청은 curl로 적히고 자격 증명은 가려진다. `stickypane watch --note api.log`로 전송을 들을 수 있다. MCP 도구 `api`도 같다.
- 다른 명령(docs/AGENTS.md): `stickypane show`, `todo`, `card`, `chart`, `log`, `say`, `write`, `wait`, `set`.
- 설계 문서의 전제: `.sticky/`는 gitignore 대상이라 새 worktree에는 보드가 없다. 레지스트리는 `~/.stickypane`.
- VISION의 판단 기준: 폴더가 곧 제품, 에이전트는 새로 배울 것이 없음, 네트워크를 듣지 않음, 화면은 조용함, 작업 관리자나 채팅 클라이언트가 아님.

---

## A. `api` 셰이프의 에이전트 사용 사례 10개

### A1. 구현 직후 스모크
- 상황: 에이전트가 엔드포인트를 만들고 서버를 띄웠다. 동작한다는 말만 하면 사용자는 믿을 근거가 없다.
- 보드에서의 모양: `api.http`에 요청 2~3개와 `# @assert response.statusCode == 201`. 에이전트가 `stickypane api api --all`로 보내면 `api.log`에 요청(curl), 본문, 검사, 끝줄 `[201 Created · 41ms · Create user · 2/2 ✔]`.
- 왜 보드인가: 채팅의 "잘 됩니다"는 주장이고, 로그의 ✔ 줄은 사용자가 같은 파일로 다시 보낼 수 있는 증거다.
- 사용자가 얻는 것: `enter` 한 번으로 직접 재현. 에이전트의 말을 믿지 않아도 된다.
- 근거: 내 추론. `--all`의 존재와 로그 형식은 `stickypane help api`에서 확인.
- 위험·한계: 서버가 에이전트의 환경에서만 떠 있으면 사용자 재현이 실패한다. assert가 느슨하면 거짓 ✔.
- 계측: `api.log` 끝줄의 `n/m ✔` 비율. `stickypane watch --note api.log` 이벤트 수.

### A2. 계약 먼저, 구현 나중 (red/green)
- 상황: 에이전트가 API 변경을 시작하기 전에 기대 응답을 assert로 먼저 쓴다.
- 보드에서의 모양: `# @assert response.json("user.id") exists`, `# @assert "json" in response.header("Content-Type")`를 가진 요청들. 처음에는 `0/2`, 구현이 끝나면 `2/2 ✔`.
- 왜 보드인가: 진행이 숫자 하나로 접힌다. 체크리스트를 따로 갱신하지 않아도 로그가 곧 진행 상황이다.
- 사용자가 얻는 것: 에이전트가 만들려는 계약을 구현 전에 검토하고 고칠 수 있다.
- 근거: 내 추론. assert 연산자(`== != < > <= >= in contains exists`)는 `kinds api`에서 확인.
- 위험·한계: assert가 표현 못 하는 계약(스키마 전체, 필드 부재)은 못 잡는다. 스키마 검증은 Schemathesis 같은 도구의 영역이다(https://schemathesis.readthedocs.io/en/stable/).
- 계측: 구현 전 `0/n`을 쓴 세션 비율, 요청별 첫 ✔까지 전송 횟수.

### A3. 인증 흐름 한 번 풀어 두기
- 상황: 로그인 -> 토큰 -> 보호된 API. 에이전트가 매번 curl로 토큰을 복사하며 헤맨다.
- 보드에서의 모양: 첫 요청에 `# @capture file token {{response.json.token}}`, 둘째 요청 헤더에 `{{token}}`. 비밀번호는 `env:API_PASSWORD`나 `.env`. 로그에서 헤더 자격 증명은 가려진다.
- 왜 보드인가: 흐름이 요청 두 개의 순서로 눈에 보이고, 토큰이 채팅 기록에 남지 않는다.
- 사용자가 얻는 것: 인증이 왜 실패하는지(401인지, 캡처가 비었는지)를 로그에서 바로 본다.
- 근거: 캡처 문법은 `kinds api`. 비슷한 요청 변수 방식은 REST Client(https://github.com/Huachao/vscode-restclient)와 JetBrains(https://www.jetbrains.com/help/idea/http-client-in-product-code-editor.html)에도 있다.
- 위험·한계: `capture file`이 토큰을 디스크에 남기는지, 남긴다면 어디인지는 이번에 확인하지 못했다. 가려지는 범위가 헤더와 메타데이터라고만 문서에 있다. 본문의 비밀은 보이게 남을 수 있다(내 추론).
- 계측: 401/403 후 같은 흐름의 재시도 횟수.

### A4. 실행되는 API 문서
- 상황: 에이전트가 "API 문서를 써 줘" 요청을 받는다. 마크다운 표는 낡는다.
- 보드에서의 모양: 요청 하나당 `###` 섹션 하나, 이름과 예시 본문과 assert. 문서 자체가 `.http`다. 사용자는 `j k enter`로 문서를 읽으며 실제로 보낸다.
- 왜 보드인가: 설명이 아니라 시도가 가능한 문서다. 에이전트가 파일 하나만 쓰면 된다(VISION 2).
- 사용자가 얻는 것: 문서와 서버의 불일치를 읽는 동안 발견한다.
- 근거: JetBrains와 resterm이 `.http`를 버전 관리되는 요청 문서로 쓴다(https://github.com/unkn0wn-root/resterm). Bruno도 요청을 평문 파일로 저장한다(https://docs.usebruno.com/introduction/what-is-bruno).
- 위험·한계: 응답 필드의 의미 설명은 여전히 산문이 필요하다. `.http`에는 설명 칸이 `###` 이름과 `#` 주석뿐이다.
- 계측: 문서 요청 중 `✔`가 최근에 나온 비율.

### A5. 변경 전후 회귀 비교
- 상황: 리팩터링이 응답을 바꾸지 않았는지 확인한다.
- 보드에서의 모양: 같은 `api.http`를 `--env before`, `--env after`(`resterm.env.json`의 두 환경)로 보낸다. 같은 assert가 양쪽에서 통과해야 한다.
- 왜 보드인가: 두 실행의 끝줄이 나란히 남는다. 채팅은 첫 실행을 밀어낸다.
- 사용자가 얻는 것: 어떤 요청이 깨졌는지 이름으로 안다.
- 근거: 내 추론. `--env`는 `help api`에서 확인.
- 위험·한계: 응답 전체를 비교하는 기능은 없다. 확인한 것은 assert뿐이다. 타임스탬프나 id가 섞인 응답은 assert 작성이 번거롭다.
- 계측: 리팩터링 PR당 `--all` 실행 수와 ✔ 비율.

### A6. WebSocket 구독 확인
- 상황: 실시간 이벤트 기능을 추가했다. 연결과 메시지 수를 확인해야 한다.
- 보드에서의 모양: 문서의 예처럼 `# @websocket timeout=5s idle-timeout=1s`, `# @ws send {"type":"subscribe",...}`, `# @ws wait 500ms`, `# @ws close 1000 done`, `# @assert response.received >= 1`. 로그는 대화록이고 끝줄은 `[101 Switching Protocols · 1.1s · Live updates · 2↑ 2↓ · 1/1 ✔]`.
- 왜 보드인가: 오간 메시지의 시간순 대화록을 사용자가 읽는다. 터미널 세션에서 휘발되는 것이 파일로 남는다.
- 사용자가 얻는 것: 에이전트 없이도 같은 세션을 다시 돌린다.
- 근거: `kinds api`와 docs/shapes.md.
- 위험·한계: 장기 연결이나 양방향 장기 스트림은 timeout 기반이라 못 본다. gRPC도 unary와 server-streaming만.
- 계측: `received` 값이 0인 실행 비율.

### A7. gRPC 탐색에서 요청까지
- 상황: 에이전트가 낯선 gRPC 서비스에 붙는다.
- 보드에서의 모양: `GRPC localhost:50051`, `# @grpc users.v1.Users/Get`, `# @grpc-plaintext true`, JSON 본문. descriptor가 없으면 reflection. 결과는 `response.grpc.status == "OK"`나 `NOT_FOUND`.
- 왜 보드인가: 탐색 결과가 재사용 가능한 요청 파일로 굳는다. grpcurl 한 줄은 쉘 기록에서 사라진다.
- 사용자가 얻는 것: 서비스가 실제로 열려 있는지(reflection 응답 여부)를 본다.
- 근거: grpcurl이 reflection을 기본으로 쓰고 protoset이 없으면 proto 소스가 필요하다(https://github.com/fullstorydev/grpcurl). stickypane의 필드 번호 대체는 `kinds api`.
- 위험·한계: `list`나 `describe`에 해당하는 서비스 목록 기능이 `help api`에서 보이지 않았다. 서비스와 메서드 이름은 에이전트가 알아야 한다(내 추론).
- 계측: 필드 번호 폴백으로 쓴 요청 수.

### A8. 사용자가 누르는 위험한 요청
- 상황: DELETE나 결제 같은 요청은 에이전트가 직접 보내면 안 된다.
- 보드에서의 모양: 에이전트는 요청만 `.http`에 써 두고 보내지 않는다. 사용자가 `enter`를 누르면 확인 후에 전송된다.
- 왜 보드인가: 승인이 채팅 한 줄이 아니라 눌러야 하는 버튼이다(VISION 3).
- 사용자가 얻는 것: 요청 본문과 환경(`e`)을 보고 보낸다.
- 근거: `enter`가 묻고 보낸다는 `kinds api`. 반대로 `stickypane api <file> <request>`는 묻지 않고 보낸다.
- 위험·한계: 에이전트가 CLI로 직접 보낼 수 있어 규칙이 강제되지 않는다. 가드가 있다면 가이드 한 줄뿐이다(내 추론). 실제 막을 방법은 확인하지 못했다.
- 계측: 에이전트의 CLI 전송 중 `--env prod` 비율. 로그의 환경 이름으로 센다.

### A9. 재현 요청을 증거로 고정
- 상황: 버그 신고. 에이전트가 요청으로 재현했고, 다른 에이전트나 사람이 이어받아야 한다.
- 보드에서의 모양: `repro.http` 하나와 로그. 로그에 curl과 응답과 실패한 assert가 남는다.
- 왜 보드인가: 보고서의 "재현 방법" 문단이 파일 하나로 대체된다.
- 사용자가 얻는 것: 이어받는 쪽이 같은 요청을 한 번의 `enter`로 돌린다.
- 근거: 내 추론. Schemathesis가 발견마다 최소 curl이나 grpcurl 명령을 내놓는 것과 같은 발상(https://schemathesis.readthedocs.io/en/stable/).
- 위험·한계: `.sticky/`가 gitignore라 다른 컴퓨터로 가지 않는다(worktree-design.md). 로그의 응답 본문에 개인 정보가 들어갈 수 있다.
- 계측: 신고별 `repro.http` 존재 여부, 이어받은 세션의 첫 실행이 ✔로 끝났는지.

### A10. 외부 API 학습 검증
- 상황: 에이전트가 처음 보는 타사 API(샌드박스 키)를 붙인다. 기억에서 꺼낸 응답 모양은 틀릴 수 있다.
- 보드에서의 모양: 첫 요청의 실제 응답을 보고 `# @assert response.json("data.id") exists`를 써 넣는다. 키는 `env:THIRD_PARTY_KEY`.
- 왜 보드인가: 에이전트가 실제 응답을 봤다는 흔적이 남는다. 사용자가 추측과 관찰을 구분한다.
- 사용자가 얻는 것: 에이전트의 가정이 관찰로 바뀌는 순간.
- 근거: 내 추론.
- 위험·한계: 레이트 리밋과 비용. 외부 서비스의 상태를 바꾸는 요청은 A8과 같은 문제.
- 계측: 연동 작업에서 첫 요청까지의 시간(로그에는 시간이 쌓이는지 확인하지 못했다: 확인 필요).

### A 전체에 대한 비교 메모
- 같은 파일이 resterm과 VS Code REST Client에서 돌아간다는 것이 이 셰이프의 약속이다. 그러나 REST Client는 assertion 기능이 문서에 없다(https://github.com/Huachao/vscode-restclient). `# @assert`는 resterm 쪽 문법이다. JetBrains는 `client.test`와 `client.assert`로 응답 핸들러 스크립트를 쓴다. 문법이 다르다.
- httpyac은 CI에서 `.http`를 테스트로 돌리는 CLI가 있다고 소개한다(https://httpyac.github.io/guide/). 여기서 확인한 것은 소개 문구뿐이고 assert 문법 페이지는 찾지 못했다(404).
- resterm은 `resterm run`으로 JSON이나 JUnit 출력을 내고 `@poll`, `@retry`, 워크플로를 가진다. stickypane의 `api`는 그중 일부(assert, capture, ws, grpc)만 한다.

---

## B. worktree: 여러 에이전트가 일할 때 중요한 10가지 상황

설계 문서 평가 기호: [맞음] 문서가 이미 다룬다, [빠짐] 다루지 않는다, [틀릴 수 있음] 현실과 어긋날 수 있다.

### B1. 에이전트가 worktree를 새로 만들고 첫 명령을 내린다
- 보드가 보일 것: 폴더 `worktrees/<이름>/`, 오버뷰에 branch와 state, 첫 할일.
- 설계 대조: [맞음] `.sticky/` gitignore라서 새 worktree에 보드가 없다는 진단과, 메인의 보드를 쓰는 방침.
- [틀릴 수 있음] Claude Code의 기본 worktree 위치는 `.claude/worktrees/<name>/`이고 브랜치 이름은 `worktree-<name>`이다(https://code.claude.com/docs/en/worktrees.md). 설계의 폴더 이름 규칙(`/`를 `-`로)을 그대로 적용하면 폴더가 `worktree-feature-auth`가 된다. 접두어를 떼는 규칙은 없다.

### B2. Claude Code의 서브에이전트가 임시 worktree를 만든다
- 보드가 보일 것: 사용자가 이름도 모르는 짧은 수명의 worktree가 폴더로 불어난다.
- 설계 대조: [빠짐]. 같은 문서에 따르면 서브에이전트마다 임시 worktree가 생기고, 변경이 없으면 끝나자마자 자동 삭제된다. 설계는 "git에 없는 worktree는 closed 처리하고 폴더를 남긴다"라서, 하루에 폴더 수십 개가 closed로 쌓일 수 있다(내 추론). 제안: 변경 없이 사라진 worktree의 폴더는 첫 노트 외에 쓴 것이 없으면 만들지 않거나 지운다. 단 "닫힌 폴더는 보드가 지우지 않는다"는 설계 원칙과 충돌하니 사용자가 정할 일.

### B3. 어느 에이전트가 사용자를 기다리는가
- 보드가 보일 것: worktree별 한 줄: 일하는 중 / 입력 기다림 / 끝남.
- 설계 대조: [빠짐]. 설계의 state는 `open|closed`뿐이다. 사용자가 가장 먼저 묻는 것은 "누가 나를 기다리나"다. Claude Code의 agent view는 Working, Needs input, Idle, Completed, Failed, Stopped와 PR 상태를 한 화면에 모은다(https://code.claude.com/docs/en/agent-view.md). Orca도 "Needs you" 구역과 알림을 둔다(https://github.com/stablyai/orca). stickypane가 에이전트 내부 상태를 알 수는 없다. 대신 에이전트가 `stickypane say`나 `asking-the-user`의 form으로 남긴 미답변 질문을 worktree 폴더에서 세는 쪽이 VISION에 맞는다(내 추론).

### B4. 같은 파일을 두 worktree가 고친다
- 보드가 보일 것: 두 worktree의 변경 파일 교집합 경고.
- 설계 대조: [빠짐]. 설계는 diff 보기를 별도 기능(이슈 C)으로 미뤘다. 여러 글이 말하듯 충돌은 사라지는 것이 아니라 병합 시점으로 밀린다(검색 결과 요약: https://dev.to/jsmanifest/parallel-ai-coding-with-git-worktrees-run-multiple-agents-without-conflicts-11na 등, 블로그 수준). 파일 목록의 교집합은 `git diff --name-only <base>...HEAD`로 계산할 수 있어 diff 본문 없이 가능하다(내 추론). 이슈 C보다 앞서 가치가 있다.

### B5. 병합 순서
- 보드가 보일 것: 먼저 병합할 branch, 기준 branch 대비 뒤처진 정도, PR 링크와 체크.
- 설계 대조: [빠짐]. 설계의 비목표는 "worktree 만들기, 옮기기, 지우기"이고 병합은 말하지 않는다. Conductor는 diff, 체크 탭, PR 열기, 병합, 보관을 흐름으로 묶는다(https://www.conductor.build/docs/). Claude Code는 PR 라벨과 체크 색을 보여준다. stickypane가 병합을 하지 않더라도 에이전트가 오버뷰에 PR 번호를 적는 줄 하나는 필요하다.

### B6. 기준 branch가 다르다
- 보드가 보일 것: 각 worktree의 base.
- 설계 대조: [빠짐]. Claude Code는 기본이 원격 기본 branch에서 따는 `"fresh"`이고 `worktree.baseRef: "head"`로 현재 HEAD에서 딸 수도 있다. 같은 문서. 이슈 C의 `diff` 노트가 base 기본값을 `main`으로 가정하는 것은 `head` 모드에서 틀릴 수 있다.

### B7. worktree가 지워졌다가 같은 이름으로 돌아온다
- 보드가 보일 것: 같은 폴더가 다시 `open`.
- 설계 대조: [맞음] "돌아온 branch는 닫힌 폴더를 다시 연다". Claude Code도 같은 이름을 다시 쓰면 기존 worktree를 연다. 단, 같은 문서에 따르면 병합돼 원격 branch가 삭제된 worktree는 기본 branch로 리셋되며 열릴 수 있다. 폴더의 옛 할일이 새 작업에 남아 거짓말을 한다(내 추론). 재오픈 시 옛 체크리스트를 보관 표시하는 규칙을 검토할 만하다.

### B8. 에이전트가 쓰는 위치를 어떻게 아는가
- 보드가 보일 것: 세션 시작 시 한 줄 `worktree: feat-chart-ui -> worktrees/feat-chart-ui/`.
- 설계 대조: [맞음] 세션 훅 한 줄을 제안한 것은 가이드 55줄 제약에 맞다. [틀릴 수 있음] 같은 문서: worktree로 들어간 뒤에도 `${CLAUDE_PROJECT_DIR}`는 시작한 프로젝트 루트에 머물고 훅 입력 JSON의 `cwd`가 worktree를 가리킨다. 훅이 `CLAUDE_PROJECT_DIR`로 위치를 판단하면 항상 메인으로 읽는다. `cwd`를 써야 한다. 데스크톱 앱 세션은 설정, 훅, 스킬을 메인 체크아웃 루트에서 읽는다.

### B9. 실행 중인지 죽었는지
- 보드가 보일 것: worktree가 지금 에이전트에 잠겨 있는지.
- 설계 대조: [빠짐]. 설계는 `locked`를 열린 것으로만 본다. Claude Code는 에이전트가 도는 동안 worktree에 `git worktree lock`을 걸고 끝나면 푼다. 죽은 세션의 잠금은 다음 청소에서 푼다(같은 문서). 그렇다면 `git worktree list --porcelain`의 `locked`는 "에이전트가 도는 중"의 거의 공짜 신호다. 다만 모든 도구(Orca, Conductor 등)가 같은 일을 하는지는 확인하지 못했다. Claude Code 한정이다.

### B10. 공유 런타임 상태, 의존성, 디스크
- 보드가 보일 것: 포트, 데이터베이스, 설치 상태, 크기.
- 설계 대조: [빠짐]. worktree는 파일만 가른다. 블로그들은 의존성(node_modules 등)이 worktree마다 필요해 디스크가 불어나고, DB와 포트는 공유되어 경쟁한다고 반복한다(https://dev.to/jsmanifest/parallel-ai-coding-with-git-worktrees-run-multiple-agents-without-conflicts-11na 외, 블로그 수준). Claude Squad 이슈에도 worktree 설정 훅(의존성, env, 포트 격리) 요청이 있다(https://github.com/smtg-ai/claude-squad/issues #260, 제목만 확인). 이는 stickypane의 일이 아니다. 하지만 `api` 노트와 만나는 곳이 있다: 두 worktree가 같은 `api.http`를 쓰면 `@base`의 포트가 충돌한다. 환경 파일을 worktree 폴더 안에 두는 것이 낫다(A5와 연결, 내 추론).

### B 요약: 설계 문서에 대한 판정
- 맞는 것: 메인 보드 공유, git이 진실, 닫힌 폴더 보존, 락과 원자적 쓰기, 가이드 한 줄.
- 틀릴 수 있는 것: 폴더 이름(도구별 접두어), 훅 위치 판단(`cwd`), 서브에이전트 임시 worktree의 폴더 폭증.
- 빠진 것: 대기 상태, PR과 체크, 병합 순서, base branch, 파일 교집합 경고, 잠금 신호.

---

## C. 선행 사례 표

| 도구 | 에이전트 옆에 보이는 것 | stickypane의 다른 점 | 출처 |
| --- | --- | --- | --- |
| Orca | worktree마다 터미널 분할, 내장 브라우저, diff 리뷰와 줄 단위 메모를 에이전트에 보냄, PR과 체크 연결, 알림, 모바일 앱, `orca worktree create` CLI | 앱이 에이전트를 실행하고 감싼다. stickypane는 에이전트를 실행하지 않고 폴더만 비춘다. 에이전트가 쓴 노트가 중심 | https://github.com/stablyai/orca |
| Conductor | 작업마다 격리 workspace, 터미널, diff 뷰어, 체크 탭, 스크립트, 사용량 제한. Claude Code, Codex, Cursor, OpenCode 병렬 | Mac 앱이고 흐름이 PR 병합과 보관까지 이어진다. stickypane는 터미널 우선이며 병합을 하지 않는다 | https://www.conductor.build/docs/ |
| Claude Squad | tmux 세션 목록, 세션마다 미리보기와 diff 탭, 작업마다 worktree와 branch, 백그라운드 자동 수락 | TUI가 에이전트 프로세스를 소유한다. 이슈에는 세션 목록에 저장소 이름, 컴팩트 모드, worktree 설정 훅 요청이 있다. stickypane는 프로세스를 소유하지 않는다 | https://github.com/smtg-ai/claude-squad , https://github.com/smtg-ai/claude-squad/issues |
| Claude Code agent view | 백그라운드 세션을 상태(Working, Needs input, Idle, Completed, Failed, Stopped)로 묶은 목록, 한 줄 요약, 나이, PR 라벨과 체크 색 | 에이전트 내부 상태를 안다는 점에서 stickypane보다 정확하다. 대신 Claude Code에만 해당. stickypane는 에이전트가 무엇이든 파일로 | https://code.claude.com/docs/en/agent-view.md |
| Cursor cloud agents(구 background agents) | 대화, 코드 변경, 산출물(스크린샷, 영상, 로그), 환경, 원격 데스크톱 접수, 팀원이 URL로 읽기 전용 열람 | 클라우드 VM에서 돈다. stickypane는 로컬 폴더이고 네트워크를 듣지 않는다(VISION 4) | https://cursor.com/docs/background-agent |
| Codex cloud | 작업마다 별도 작업 공간, 변경 파일과 체크 결과 검토, 후속 요청, 커밋과 PR. 환경은 설치 스크립트, 비밀, 네트워크 허용 목록 | 클라우드 작업 단위. 로그 뷰는 페이지에서 확인하지 못했다 | https://learn.chatgpt.com/docs/cloud |
| Devin | 작업 공간에 Shell, 내장 IDE, 브라우저. 진행 단계를 눌러 열기, 사용자가 인수 가능 | 대화형 에이전트 UI 자체다. stickypane는 대화 클라이언트가 아니다(VISION) | https://docs.devin.ai/get-started/devin-intro |
| Warp | 세로 탭에 branch, worktree, PR 메타데이터. 입력 필요 시 알림. 에이전트 변경에 코멘트를 달아 한 번에 되돌려 보냄 | 터미널 앱 자체의 기능. stickypane는 어떤 터미널에서도 | https://www.warp.dev/agents |
| Zed agent panel | 변경 파일과 줄 수, Review Changes 멀티버퍼에서 hunk 수락과 거부, 스레드 사이드바, 에이전트가 만지는 파일을 따라가기 | 에디터 안에 있다. 변경 검토가 중심 | https://zed.dev/docs/ai/agent-panel |
| VS Code Copilot agents | Agents 창에서 프로젝트 간 세션 관리, 세션 폴더나 worktree에 편집 저장, 클라우드 세션(실험) | 문서는 worktree가 보안 경계가 아니라고 명시한다. 세션별 표시 내용은 가져온 페이지에서 확인하지 못했다 | https://code.visualstudio.com/docs/copilot/agents/overview |
| Vibe Kanban | 칸반 이슈에서 workspace를 만들면 branch, 터미널, 개발 서버, 인라인 diff 리뷰(코멘트를 에이전트에 보냄), 브라우저 미리보기, PR | 칸반이 자체 데이터 모델의 작업 관리다. README 첫머리에 "sunsetting"이라고 적혀 있다. stickypane는 작업 관리자가 되지 않는다(VISION) | https://github.com/BloopAI/vibe-kanban |
| JetBrains HTTP Client | `###` 요청, 환경 파일, 응답 핸들러 스크립트 `client.test`와 `client.assert`, gRPC unary와 서버 스트리밍, WebSocket, GraphQL, 응답을 `.idea/httpRequests/`에 저장, CLI 러너 | IDE 도구. stickypane는 같은 파일을 에이전트 옆 창으로 가져오고 assert를 resterm 문법으로 씀. 따라서 JetBrains 파일과 100% 호환은 아니다(내 추론) | https://www.jetbrains.com/help/idea/http-client-in-product-code-editor.html |
| VS Code REST Client | `.http`, 변수 4종, 환경, 인증 여러 종류, 요청 기록 50개. assertion 설명 없음 | `@assert`와 gRPC, WebSocket을 지원 | https://github.com/Huachao/vscode-restclient |
| resterm | 터미널 API 클라이언트, `.http`, `@assert`, `@capture`, `@poll`, `@retry`, 워크플로, `@profile`, OpenAPI 가져오기, JSON과 JUnit 출력, gRPC, WebSocket, SSE | stickypane의 `api`가 문법을 빌려 온 원본. stickypane는 반응형 보드의 한 노트로 둔다 | https://github.com/unkn0wn-root/resterm |
| Bruno, httpyac | Bruno는 요청을 평문 파일로 저장하고 CLI와 CI 실행(문서가 말하는 형식은 YAML). httpyac은 REST, GraphQL, gRPC, CLI, 인증 방식 다수 | 에이전트가 쓴 요청을 사람이 같은 자리에서 누르는 작은 공간이 목적 | https://docs.usebruno.com/introduction/what-is-bruno , https://httpyac.github.io/guide/ |

한계: Zed, Warp, VS Code, Codex의 세부는 작은 모델이 줄인 요약이라 화면 구성을 단정하지 않았다. 에이전트 캔버스나 아티팩트 계열은 이번에 깊이 보지 못했다(조사 누락).

---

## D. 설계 문서가 바꿔야 할 가장 중요한 5가지

1. **worktree의 상태를 `open|closed` 둘로 두지 말 것.** 사용자가 보드를 보는 이유는 "누가 나를 기다리나"다(Claude Code agent view, Orca가 모두 이 질문을 중심에 둔다). 에이전트의 내부 상태는 모르니, 오버뷰에 에이전트가 직접 쓰는 줄(`working / waiting / done`)과 미답변 form 개수를 두는 쪽이 VISION(에이전트에게 새로 가르치지 않음)과 맞는지를 먼저 정하자. 근거: https://code.claude.com/docs/en/agent-view.md
2. **Claude Code 실제 동작에 맞춰 이름과 수명을 고칠 것.** 기본 경로 `.claude/worktrees/<name>`, branch `worktree-<name>`, 서브에이전트의 임시 worktree, 깨끗하면 자동 삭제, 실행 중 `git worktree lock`. 접두어 처리, 임시 worktree에 폴더를 만들지 않는 규칙, `locked`를 "에이전트 실행 중" 신호로 쓰는 안을 문서에 넣어야 한다. 근거: https://code.claude.com/docs/en/worktrees.md
3. **세션 훅은 `CLAUDE_PROJECT_DIR`가 아니라 훅 입력의 `cwd`로 worktree를 판단할 것.** 이슈 D에 한 줄 추가. 안 그러면 Claude Code 안의 worktree 세션이 항상 메인으로 읽혀 폴더 안내가 틀린다. 같은 문서의 "Hook paths don't follow the worktree".
4. **diff 본문보다 먼저 "변경 파일 교집합"과 PR 링크 줄이 가치가 크다.** 이슈 C를 쪼개 첫 단계를 `git diff --name-only <base>...HEAD`와 worktree 간 겹침 경고, 오버뷰의 PR 번호 한 줄로 낮추자. 병합 충돌은 사라지지 않고 병합 시점으로 밀린다는 글이 반복된다(블로그 수준). base가 `main`이라는 가정도 `worktree.baseRef: "head"`에서 깨진다.
5. **`api` 셰이프를 worktree 설계와 묶을 것.** 두 worktree가 한 `api.http`를 공유하면 `@base` 포트가 부딪친다. 환경 파일(`resterm.env.json`)을 worktree 폴더에 두고 `# @env <worktree>`로 가르는 관행을 가이드가 아닌 `docs/shapes.md`에 한 줄 적자. 그리고 `stickypane api <file> <request>`가 묻지 않고 보낸다는 점(A8)은 위험한 요청이 있는 파일에서 에이전트 가이드 한 줄이 필요한지 정해야 한다. 로그가 `capture file`과 본문의 비밀을 어디까지 가리는지는 아직 확인하지 못했으니, 설계 전에 코드로 확인할 것.

## 이번 조사에서 확인하지 못한 것
- `capture file`의 디스크 위치와 보존 범위.
- `.worktreeinclude`가 gitignore된 `.sticky/`를 복사할 수 있는지(패턴이 맞으면 복사될 것이라는 추론뿐. 같은 문서가 gitignore된 파일 중 패턴에 맞는 것만 복사한다고 했다). 그러면 B1의 "새 worktree에는 보드가 없다"는 전제가 사용자 설정에 따라 달라진다.
- Orca와 Conductor가 worktree를 놓는 위치와 branch 이름 규칙.
- 에이전트 캔버스와 아티팩트 계열 도구.
