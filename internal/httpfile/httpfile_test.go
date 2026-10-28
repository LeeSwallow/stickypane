package httpfile

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestSend(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/login":
			b, _ := io.ReadAll(r.Body)
			if string(b) != `{"user": "min"}` {
				w.WriteHeader(400)
				return
			}
			io.WriteString(w, `{"token":"t-1","user":"min"}`)
		case "/v1/me":
			if r.Header.Get("Authorization") != "Bearer t-1" {
				w.WriteHeader(401)
				io.WriteString(w, `{"error":"no token"}`)
				return
			}
			io.WriteString(w, `{"tags":["a","b"]}`)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "rest-client.env.json"), []byte(`{"dev":{"host":"`+srv.URL+`"},"prod":{"host":"http://prod"}}`), 0o644)
	rn := &Runner{Root: dir, Dirs: []string{dir}}
	f := Parse(sample)

	login := rn.Send(context.Background(), f, 0)
	if login.Failed() || login.Env != "dev" || rn.Saved["token"] != "t-1" {
		t.Fatalf("login: %+v\n%s", login, Log(login))
	}
	me := rn.Send(context.Background(), f, 1)
	if me.Failed() {
		t.Fatalf("me: %s", Log(me))
	}
	if len(me.Hooks) != 1 || !strings.Contains(me.Hooks[0], `"status":200`) {
		t.Errorf("the post hook should read the response: %q", me.Hooks)
	}
	log := Log(me)
	if !strings.HasPrefix(log, "$ curl -sS '"+srv.URL+"/v1/me' -H 'Authorization: Bearer ••••'") {
		t.Errorf("log should start with the curl line:\n%s", log)
	}
	if !strings.Contains(log, "✔ body.tags contains \"a\"") || !EndLine.MatchString(strings.TrimSpace(log[strings.LastIndex(strings.TrimSpace(log), "\n")+1:])) {
		t.Errorf("log:\n%s", log)
	}

	// Without the token the check fails and says what came back.
	rn.Saved = nil
	me = rn.Send(context.Background(), f, 1)
	if me.Err == nil || !strings.Contains(me.Err.Error(), "token") {
		t.Errorf("a missing variable should stop the request: %v", me.Err)
	}
}

func TestChecks(t *testing.T) {
	r := &response{status: 404, body: []byte(`{"a":{"b":[1,{"c":"x"}]},"n":3}`), header: http.Header{"X-Id": {"7"}}}
	for expr, want := range map[string]bool{
		"status == 404":                    true,
		"response.statusCode != 200":       true,
		"body.a.b[1].c == \"x\"":           true,
		"body.a.b[1].c == x":               true,
		"body.n >= 3":                      true,
		"body.n < 3":                       false,
		"header.X-Id == 7":                 true,
		"body.a.b contains 1":              true,
		"body.missing exists":              false,
		"body.a exists":                    true,
		"time < 100000":                    true,
		`response.json("a.b[1].c") == "x"`: true,
		`response.json("$.n") == 3`:        true,
		`"7" in response.header("X-Id")`:   true,
		`response.text() contains "c"`:     true,
		`body.a.b[1].c == "x == y"`:        false,
	} {
		if got, seen := r.check(expr); got != want {
			t.Errorf("%s: %v, got %q", expr, got, seen)
		}
	}
}

func TestFill(t *testing.T) {
	t.Setenv("HTTPFILE_TEST", "secret")
	out, missing := Fill("{{a}} {{ $processEnv HTTPFILE_TEST }} {{b}}", Scope{{"a": "1"}}, func(n string) (string, bool) { return os.LookupEnv(n) })
	if out != "1 secret {{b}}" || len(missing) != 1 || missing[0] != "b" {
		t.Errorf("%q %v", out, missing)
	}
}
