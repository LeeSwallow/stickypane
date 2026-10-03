# API 노트: REST, WebSocket, gRPC (실험)

`.http` 노트를 resterm·Bruno 같은 API 도구로 넓힌다. 한 파일 안에서 REST,
WebSocket, gRPC 요청을 나란히 쓰고, 보드에서 보내고, 결과를 같은 로그로 본다.
에이전트는 같은 파일을 명령줄과 MCP로 보낸다.

## 원칙

- **파일이 정본이다.** 문법은 resterm을 따른다. 같은 파일이 resterm에서도 돈다.
  stickypane만 읽는 것은 `@pre`, `@post`처럼 이름으로 드러낸다.
- **설정이 없다.** 환경은 이미 읽는 `rest-client.env.json`, `resterm.env.json`,
  `.env`, 프로세스 환경 변수 그대로다.
- **의존성을 늘리지 않는다.** WebSocket(RFC 6455)과 gRPC(HTTP/2 + protobuf
  wire)는 표준 라이브러리 위에 직접 쓴다. Go 1.24부터 표준 `net/http`가 암호화
  없는 HTTP/2(h2c)를 지원한다.
- **보내는 것은 사람의 몫이다.** 보드에서는 enter 뒤 확인을 받는다. 명령줄과
  MCP는 에이전트가 직접 보낸다(지금 REST와 같다).

## 문법

```http
@base = http://localhost:8080

### Log in                         ← REST: 지금 그대로
POST {{base}}/login
Content-Type: application/json

{"user": "{{user}}"}
# @assert response.statusCode == 200
# @capture file token {{response.json.token}}

### Chat                           ← WebSocket: ws:// 또는 @websocket
# @websocket timeout=5s idle-timeout=2s subprotocols=chat.v2
# @ws send {"type":"hello"}
# @ws send-json {"type":"join","room":"general"}
# @ws wait 500ms
# @ws ping heartbeat
# @ws close 1000 bye
# @assert response.received >= 1
GET ws://localhost:8080/chat
Authorization: Bearer {{token}}

### Get user                       ← gRPC: GRPC 줄과 @grpc
# @grpc users.v1.Users/Get
# @grpc-plaintext true
# @grpc-metadata x-trace-id: demo-1
# @grpc-descriptor users.protoset   ← 없으면 서버 리플렉션, 그것도 없으면 필드 번호
# @assert response.grpc.status == "OK"
# @assert response.json("name") == "min"
GRPC localhost:9090

{"id": 7}
```

WebSocket 단계: `send`, `send-json`, `send-base64`, `ping`, `pong`, `wait`,
`close`. 마지막 단계 뒤 `idle-timeout` 동안 들어오는 메시지를 더 받는다.

gRPC: 단항(unary)과 서버 스트리밍. 본문은 protobuf JSON이고, 스트리밍
응답은 JSON 배열이다. 스키마는 `@grpc-descriptor`(protoset 파일), 서버
리플렉션(`grpc.reflection.v1`, `v1alpha`) 순으로 찾는다. 둘 다 없으면 필드
번호를 키로 쓰는 JSON(`{"1": 7}`)으로 보내고 받는다(`protoc --decode_raw`처럼).

## 요청과 응답의 흐름

```mermaid
graph LR
  file[.http 파일] --> prep[환경·변수·@pre]
  prep --> rest[HTTP/1.1·2]
  prep --> ws[WebSocket]
  prep --> grpc[gRPC h2/h2c]
  rest --> res[Result]
  ws --> res
  grpc --> res
  res --> checks[@assert·@capture·@post]
  checks --> log[.log 파일]
  log --> board[보드 패널]
  log --> agent[명령줄·MCP]
```

준비와 검사는 프로토콜과 상관없이 하나다. 프로토콜마다 다른 것은 보내는
부분과, 응답을 `Result`에 담는 방식뿐이다.

| | REST | WebSocket | gRPC |
| --- | --- | --- | --- |
| 상태 | HTTP 상태 | `101 Switching Protocols` | gRPC 코드를 HTTP 상태에 대응 (`200 OK`, `404 NOT_FOUND`) |
| `response.json` | 본문 | 받은 메시지의 배열 | 응답 메시지 (스트리밍이면 배열) |
| 더 있는 값 | | `response.received`, `response.sent` | `response.grpc.status`, `response.grpc.message`, `response.trailer.X` |
| 로그 본문 | 들여쓴 JSON | 주고받은 순서의 기록 | 메서드, 메타데이터, 메시지 JSON, 트레일러 |

## 화면

요청 목록은 프로토콜을 왼쪽 배지로 구분한다. REST는 지금처럼 메서드 색이다.

```
╭ ⇄ api ────────────────────────────────────── 3 requests ╮
│ environment dev                                          │
│                                                          │
│ › POST   Log in       {{base}}/login            1 ✓      │
│   WS     Chat         ws://localhost:8080/chat  5 steps  │
│   GRPC   Get user     users.v1.Users/Get        2 ✓      │
│ ─ log · 2m · 101 Switching Protocols · 1.2s · Chat · 3↑ 4↓│
│ → 14:02:01.120  {"type":"hello"}                         │
│ ← 14:02:01.134  {"type":"welcome","id":42}               │
│ → 14:02:01.140  {"type":"join","room":"general"}         │
│ ← 14:02:01.161  {"type":"joined","members":3}            │
│ → ping heartbeat                                         │
│ ← pong heartbeat                                         │
│ ✔ response.received >= 1                                 │
╰──────────────────────────────────────────────────────────╯
```

- **WebSocket 기록**: `→` 보낸 것, `←` 받은 것. 시각은 연결 뒤 경과가 아니라
  벽시계(밀리초)로 쓴다. 채팅 노트와 같은 읽기 방향이다. 제어 프레임(ping,
  pong, close)은 흐리게 그린다.
- **gRPC**: 첫 줄에 `grpc users.v1.Users/Get · plaintext · reflection`,
  메타데이터와 트레일러는 흐리게, 메시지는 들여쓴 JSON. 오류는
  `404 NOT_FOUND · user 7 not found`처럼 끝줄에 빨갛게.
- **목록의 오른쪽**: 검사 수(`2 ✓`), 훅(`⚙`), WebSocket 단계 수.
- **요청별 결과**: `.log` 파일은 요청 제목마다 `### 제목` 구간을 두고, 다시
  보내면 그 구간만 바꾼다. 보드는 커서가 있는 요청의 구간을 보여 주고, 아직
  보내지 않은 요청이면 "아직 보내지 않았습니다"라고 한다.
- 패널 아래 로그 구분선의 끝줄 요약(`101 … · 3↑ 4↓`)은 지금 REST의
  `[200 OK · 38ms · …]`와 같은 자리, 같은 색 규칙이다.

## 환경 변수

- `{{name}}`은 요청 줄, 헤더, 본문, `@ws` 단계, `@grpc-metadata`, gRPC 대상
  주소 어디에나 쓴다.
- 값의 출처는 지금 순서 그대로다: `@pre` 출력, 요청 변수, 캡처, 파일 변수,
  환경 파일(`$shared` 다음 선택한 환경), `env:NAME`과 `{{$processEnv NAME}}`.
- 보드의 `e`로 환경을 바꾸면 파일 첫머리의 `# @env`가 바뀐다.
- 비밀은 로그에 남기지 않는다. 지금 헤더를 가리는 규칙을 gRPC 메타데이터와
  WebSocket 핸드셰이크 헤더에도 똑같이 쓴다.

## 에이전트가 다루는 법

| 할 일 | 명령줄 | MCP |
| --- | --- | --- |
| 요청 목록 | `stickypane api api.http` (`#1 WS Chat`) | `api` 도구, `name`만 |
| 하나 보내기 | `stickypane api api.http Chat --env dev` | `api` 도구, `request` |
| 전부 보내기 | `--all` | `all: true` |
| 결과 기다리기 | `stickypane watch --once --note api.log` | `wait_event` |
| 쓰기 | 파일을 쓴다. 형식은 `stickypane kinds api` | `write_note` |

결과는 로그 그대로다. 실패하면 종료 코드 1과 실패 수를 말한다.

## 테스트

전부 `go test`로, 네트워크 없이 돈다.

- **WebSocket**: 프레임 인코딩(마스킹, 126/127 길이, 조각), 핸드셰이크 키
  검증. `httptest` 서버가 연결을 넘겨받아(hijack) 메아리 서버가 된다.
- **gRPC**: protobuf wire 인코딩·디코딩, descriptor 읽기, JSON 대응.
  `http.Server`의 h2c로 테스트 서버를 띄워 단항, 서버 스트리밍, 오류 상태,
  리플렉션을 시험한다. 테스트용 descriptor는 우리 인코더로 만든다.
- **전체 흐름**: 한 `.http` 파일의 REST, WS, gRPC를 `stickypane api --all`로
  보내고 로그를 확인한다.

## 범위 밖 (다음에)

클라이언트 스트리밍과 양방향 gRPC, SSE, GraphQL, `.proto` 파일 직접 읽기,
Bruno `.bru` 파일 가져오기. 문법 자리는 resterm에 맞춰 비워 둔다.
