#!/bin/sh
# showcase.sh [dir]: build an English demo board that shows what stickypane
# does, in dir (a new temporary project by default), and print the folder.
# Open it with `cd <dir> && stickypane`; demo/screenshots.py photographs it.
# Every note is made the way an agent would make it: one-line commands,
# and a file written where a shape needs one.
set -eu
dir="${1:-$(mktemp -d)/acme-api}"
mkdir -p "$dir"
cd "$dir"
git init -q 2>/dev/null || true
export STICKYPANE_USER=sam
sp() { stickypane "$@" >/dev/null; }

stickypane init --no-agent-docs >/dev/null
rm -f .sticky/welcome.md
sp theme catppuccin-mocha
sp language en

# --- The first tab: the work, as the user and the agent share it ---------
printf -- '---\ntype: board\n---\n## To do\n\n## Doing\n\n## Review\n\n## Done\n' | stickypane write 10-work --open >/dev/null
for c in "rate limits on login @claude" "OAuth callback" "refresh tokens @codex" \
	"password reset email" "audit log @sam" "schema migration @claude"; do
	sp card 10-work add "$c" --to "To do"
done
sp card 10-work move "refresh" --to Doing
sp card 10-work move "rate limits" --to Doing
sp card 10-work move "audit" --to Review
sp card 10-work move "schema" --to Done
sp set 10-work title="Auth sprint" size=page

for i in "read the issue and the RFC" "add the token endpoint" "validate scopes" \
	"write table tests" "update the README" "open the pull request"; do
	sp todo 20-plan add "$i"
done
sp todo 20-plan check "read the issue"
sp todo 20-plan check "token endpoint"
sp todo 20-plan check "scopes"
sp set 20-plan title="Token endpoint" size=half

stickypane write 30-progress --type chart --open --title "Cards by column" <<'MD' >/dev/null
---
from: 10-work
size: half
---
MD

sp say 40-chat "the token endpoint is in; 3 of 6 steps done" --as claude
sp say 40-chat "nice. keep the old /login working until Friday" --as sam
sp say 40-chat "ok: both routes stay, the old one logs a deprecation" --as claude
sp set 40-chat title="Chat" size=half

# --- The second tab: a release, with a question for the user -------------
mkdir -p .sticky/20-release
cat >.sticky/20-release/deploy.md <<'MD'
---
type: form
title: Deploy v2.4.0?
size: half
---
All 128 tests pass. Where should it go?

## Target
- ( ) staging
  Try it there first.
- ( ) production

## Also
- [x] run migrations
- [ ] clear the CDN cache

## Note
>

[ Deploy ] [ Cancel ]
MD
cat >.sticky/20-release/build.log <<'MD'
$ go test ./...
ok   github.com/acme/api/auth      0.41s
ok   github.com/acme/api/store     0.38s
ok   github.com/acme/api/http      0.92s
ok   github.com/acme/api/billing   0.27s
$ go build -o bin/api ./cmd/api
[exit 0 · 7.15s]
MD
cat >.sticky/20-release/flow.md <<'MD'
---
title: Release flow
size: half
---
```mermaid
graph LR
  PR --> CI --> staging --> production
```
MD
python3 - .sticky/20-release/latency.md <<'PY'
import sys, math
lines = ["---", "type: chart", "title: p95 latency (ms), 30 days", "view: spark", "size: half", "---"]
for i in range(30):
    v = 190 - i * 2.4 + 14 * math.sin(i / 2.3) + (25 if i == 11 else 0)
    lines.append(f"day {i + 1}: {round(v)}")
open(sys.argv[1], "w").write("\n".join(lines) + "\n")
PY
for f in deploy build.log flow latency; do sp show "20-release/$f"; done

# --- The third tab: notes that point at each other ----------------------
mkdir -p .sticky/30-notes
cat >.sticky/30-notes/design.md <<'MD'
---
title: Design notes
size: half
---
Tokens live 15 minutes; refresh tokens 30 days. See [[10-work|the board]]
and [[20-plan|the plan]] for where it stands.

![[20-plan]]
MD
cat >.sticky/30-notes/tokens.md <<'MD'
---
type: chart
title: Tokens used
view: bar
size: half
---
input: 61,200
output: 18,400
cache read: 142,000
cache write: 9,800
MD
python3 - .sticky/30-notes/commits.md <<'PY'
import sys, datetime, random
random.seed(7)
today = datetime.date.today()
lines = ["---", "type: chart", "title: Commits", "view: heat", "size: page", "---"]
first = today - datetime.timedelta(days=77 + today.weekday())  # a Monday, 11 weeks back
for i in range((today - first).days + 1):
    d = first + datetime.timedelta(days=i)
    n = 0 if d.weekday() >= 5 and random.random() < 0.7 else random.choice([0, 1, 2, 3, 4, 6, 9])
    lines.append(f"{d.isoformat()}: {n}")
open(sys.argv[1], "w").write("\n".join(lines) + "\n")
PY
sp show 30-notes/design
sp show 30-notes/tokens
sp show 30-notes/commits

# --- The fourth tab: an API to try, REST, WebSocket and gRPC ------------
# The requests go to public echo servers; offline they say "not sent yet".
mkdir -p .sticky/40-api
cat >.sticky/40-api/rest-client.env.json <<'MD'
{
  "$shared": { "user": "sam" },
  "dev": { "rest": "https://httpbin.org", "socket": "wss://ws.postman-echo.com/raw", "grpc": "grpcb.in:9000" }
}
MD
cat >.sticky/40-api/api.http <<'MD'
# @env dev

### Create user
POST {{rest}}/anything
Content-Type: application/json

{"name": "{{user}}", "role": "admin"}
# @assert response.statusCode == 200
# @assert response.json("json.name") == "{{user}}"

### Live updates
# @websocket timeout=5s idle-timeout=1500ms
# @ws send {"type":"subscribe","channel":"builds"}
# @ws send-json {"type":"ping","user":"{{user}}"}
# @ws ping heartbeat
# @ws close 1000 done
# @assert response.received >= 2
GET {{socket}}

### Say hello
# @grpc hello.HelloService/SayHello
# @grpc-plaintext true
# @assert response.grpc.status == "OK"
GRPC {{grpc}}

{"greeting": "{{user}}"}
MD
stickypane api 40-api/api --all >/dev/null 2>&1 || true
sp show 40-api/api
sp set 40-api/api size=page

# A demo is photographed at once, so give the times some history: the
# board wrote them all this minute. The card left in Doing for two hours
# is the one the board marks as stalled.
today=$(date +%Y-%m-%d)
ago() { date -v-"$1"M +%H:%M 2>/dev/null || date -d "-$1 minutes" +%H:%M; }
fix() { f=".sticky/$1"; shift; python3 - "$f" "$@" <<'PY'
import sys
p, pairs = sys.argv[1], sys.argv[2:]
s = open(p).read()
for i in range(0, len(pairs), 2):
    old, new = pairs[i], pairs[i + 1]
    k = s.find(old)
    if k >= 0:
        e = s.index('\n', k)
        line = s[k:e]
        import re
        line = re.sub(r'@@\{\d\d:\d\d\}', '@@{' + new + '}', line)
        line = re.sub(r'(\u2705 \S+) \d\d:\d\d', r'\1 ' + new, line)
        line = re.sub(r'^(@\S+) \d\d:\d\d', r'\1 ' + new, line)
        s = s[:k] + line + s[e:]
open(p, 'w').write(s)
PY
}
fix 10-work.md "OAuth callback" "$(ago 14)" "password reset" "$(ago 11)" "refresh tokens" "$(ago 125)" \
	"rate limits" "$(ago 18)" "audit log" "$(ago 22)" "schema migration" "$(ago 75)"
fix 20-plan.md "read the issue" "$(ago 95)" "token endpoint" "$(ago 51)" "validate scopes" "$(ago 12)"
fix 40-chat.md "@claude" "$(ago 9)"
fix 40-chat.md "@sam" "$(ago 6)"
python3 - .sticky/40-chat.md "$(ago 2)" <<'PY'
import sys, re
p, t = sys.argv[1], sys.argv[2]
lines = open(p).read().split('\n')
idx = [i for i, l in enumerate(lines) if l.startswith('@claude ')]
if len(idx) > 1:
    lines[idx[-1]] = re.sub(r'^(@\S+) \d\d:\d\d', r'\1 ' + t, lines[idx[-1]])
open(p, 'w').write('\n'.join(lines))
PY

sp show 10-work
echo "$dir"
