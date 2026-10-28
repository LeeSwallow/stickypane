package syntax

import "testing"

const sample = `@base = {{host}}/v1
# @env dev

### Log in
# @pre echo user=min
POST {{base}}/login
Content-Type: application/json

{"user": "{{user}}"}
# @capture file token {{response.body.token}}
# @assert status == 200
# @assert body.user == "min"

### Me
GET {{base}}/me
Authorization: Bearer {{token}}
# @assert status == 200
# @assert header.Content-Type contains json
# @assert body.tags contains "a"
# @post cat
`

func TestParse(t *testing.T) {
	t.Parallel()
	f := Parse(sample)
	if len(f.Requests) != 2 {
		t.Fatalf("requests: %d", len(f.Requests))
	}
	if f.Env != "dev" || len(f.Vars) != 1 || f.Vars[0] != (Var{"base", "{{host}}/v1"}) {
		t.Errorf("file: env %q vars %v", f.Env, f.Vars)
	}
	login := f.Requests[0]
	if login.Name != "Log in" || login.Method != "POST" || login.URL != "{{base}}/login" {
		t.Errorf("login: %+v", login)
	}
	if login.Body != `{"user": "{{user}}"}` {
		t.Errorf("body: %q", login.Body)
	}
	if len(login.Pre) != 1 || len(login.Asserts) != 2 || len(login.Captures) != 1 || login.Captures[0].Name != "token" {
		t.Errorf("directives: %+v", login)
	}
	me := f.Requests[1]
	if me.Method != "GET" || len(me.Header) != 1 || me.Header[0][0] != "Authorization" || len(me.Post) != 1 {
		t.Errorf("me: %+v", me)
	}
	if me.Start <= login.Start || me.Line <= me.Start {
		t.Errorf("lines: %d %d %d", login.Start, me.Start, me.Line)
	}
}

func TestParseBareURLAndFirstRequestWithoutSeparator(t *testing.T) {
	t.Parallel()
	f := Parse("https://example.com/a\n\n###\n/b\n")
	if len(f.Requests) != 2 || f.Requests[0].Method != "GET" || f.Requests[1].URL != "/b" {
		t.Fatalf("%+v", f.Requests)
	}
}
