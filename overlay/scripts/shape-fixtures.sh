#!/bin/sh
# shape-fixtures.sh makes the samples the shapes' tests draw from: a
# project in a temporary folder with a note of every shape, written with
# the real `stickypane` of this repository, then read back through
# `stickypane bridge` as the app reads it (board.load). HOME and
# STICKYPANE_HOME point into the temporary folder, so nothing of the
# person's own is read or written.
#
#   sh overlay/scripts/shape-fixtures.sh
#
# writes overlay/frontend/src/shapes/fixtures/board.json.
set -eu

root=$(cd "$(dirname "$0")/../.." && pwd)
out="$root/overlay/frontend/src/shapes/fixtures/board.json"
tmp=$(mktemp -d)
srv=''
trap 'test -n "$srv" && { kill "$srv"; wait "$srv"; } 2>/dev/null; rm -rf "$tmp"' EXIT

mkdir -p "$tmp/home" "$tmp/sphome" "$tmp/proj" "$tmp/www"
(cd "$root" && go build -o "$tmp/stickypane" ./cmd/stickypane)
sp="$tmp/stickypane"
export HOME="$tmp/home" STICKYPANE_HOME="$tmp/sphome" STICKYPANE_USER=min

# a server on this machine for the .http note to send to
echo '{"ok": true, "user": "min"}' >"$tmp/www/hello.json"
port=$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1])')
python3 -m http.server "$port" --bind 127.0.0.1 --directory "$tmp/www" >/dev/null 2>&1 &
srv=$!

cd "$tmp/proj"
"$sp" init --no-agent-docs >/dev/null

printf '%s\n' '# Release notes' '' 'Ship **0.2** after the [[plan]] is done.' '' '```go' 'fmt.Println("hi")' '```' |
	"$sp" write readme --title "Release notes" >/dev/null
printf '%s\n' '- [x] write tests' '- [ ] tag the release' '- [ ] tell the team' |
	"$sp" write plan --type checklist --title Plan >/dev/null
"$sp" write work --type board --opt columns=Todo --opt columns=Doing --opt columns=Done >/dev/null
"$sp" card work add "login API" --to Doing >/dev/null
"$sp" card work add "docs" --to Todo >/dev/null

printf '%s\n' '## Where to deploy?' '- (x) staging' '- ( ) production' '  the one people use' '' '## Checks' '- [x] tests' '- [ ] lint' '' '## Note' '> after lunch' '' '[ Deploy ] [ Cancel ]' |
	"$sp" write deploy --type form --title Deploy >/dev/null
printf '%s\n' '## Ready?' '- (x) yes' '- ( ) no' '' '[ Go ]' |
	"$sp" write ready --type form --title Ready >/dev/null

printf '%s\n' 'Tokens by package' 'app: 61' 'board: 31' 'checklist: 21' 'store: 16' |
	"$sp" write tokens --type chart --title Tokens >/dev/null
"$sp" write trend --type chart --title Trend --opt view=spark \
	--opt values='mon: 3' --opt values='tue: 5' --opt values='wed: 4' --opt values='thu: 9' --opt values='fri: 7' >/dev/null
"$sp" write commits --type chart --title Commits --opt view=heat \
	--opt values='2026-10-01: 2' --opt values='2026-10-02: 0' --opt values='2026-10-05: 5' --opt values='2026-10-06: 1' \
	--opt values='2026-10-08: 8' --opt values='2026-10-14: 3' --opt values='2026-10-20: 4' --opt values='2026-10-27: 6' >/dev/null

"$sp" log worklog --time "tests passed" >/dev/null
"$sp" log worklog --time "started on review feedback" >/dev/null
"$sp" log worklog "pushed the fix" >/dev/null

"$sp" say chat "can you rerun the tests?" --as min >/dev/null
"$sp" say chat "41/41 pass" --as claude >/dev/null

printf '%s\n' '#!/bin/sh' 'echo building' 'echo done' | "$sp" write build.sh >/dev/null
printf '%s\n' "@base = http://127.0.0.1:$port" '' '### Hello' 'GET {{base}}/hello.json' '# @assert response.statusCode == 200' '' '### Missing' 'GET {{base}}/nothing' '# @assert response.statusCode == 200' |
	"$sp" write api.http >/dev/null
printf '%s\n' '<h1>Benchmark</h1>' '<p>Run took <b>4.2s</b>, down from 6.1s.</p>' '<script>document.body.dataset.ran = "yes"</script>' |
	"$sp" write report.html >/dev/null

# the app's calls: run the script, send both requests, submit one form,
# then load the board
python3 - "$sp" "$out" <<'EOF'
import json, subprocess, sys
sp, out = sys.argv[1], sys.argv[2]
p = subprocess.Popen([sp, "bridge"], stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True)
n = 0
def call(method, params=None):
    global n
    n += 1
    p.stdin.write(json.dumps({"id": n, "method": method, "params": params or {}}) + "\n")
    p.stdin.flush()
    while True:
        m = json.loads(p.stdout.readline())
        if m.get("id") == n:
            if "error" in m:
                raise SystemExit(f"{method}: {m['error']}")
            return m["result"]
def note(name):
    for t in call("board.load")["tabs"]:
        for x in t["notes"]:
            if x["name"] == name:
                return x
call("hello")
call("note.run", {"note": "build.sh"})
call("note.send", {"note": "api.http", "request": 1})
call("note.send", {"note": "api.http", "request": 2})
call("note.act", {"note": "ready.md", "version": note("ready.md")["version"], "action": {"name": "submit", "args": {"label": "Go"}}})
board = call("board.load")
p.stdin.close()
p.wait()
with open(out, "w") as f:
    json.dump(board, f, indent=1, ensure_ascii=False)
    f.write("\n")
EOF
echo "wrote $out"
