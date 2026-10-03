package httpfile

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/LeeSwallow/stickypane/internal/doc"
)

// maxBody is how much of a response is kept.
const maxBody = 4 << 20

// Runner sends the requests of a file.
type Runner struct {
	Root  string            // the project folder: hooks run here and .env is read here
	Here  string            // the .http file's folder: files it names are found here
	Dirs  []string          // the folders environment files are looked for in, in order
	Env   string            // the environment chosen; empty takes the file's, then dev
	Saved map[string]string // what earlier responses captured; Send adds to it
	// Client sends the request; nil is a client with a 30 second timeout.
	Client *http.Client
}

// Check is the outcome of one "@assert".
type Check struct {
	Expr string
	OK   bool
	Got  string // what the response held, when the check failed
}

// Result is what sending one request came to.
type Result struct {
	Request  Request
	Protocol string // http, websocket or grpc
	Env      string // the environment used, if any
	Curl     string // the request as a curl command, to send it again by hand
	Status   int
	Text     string // "200 OK"
	Header   http.Header
	Body     []byte
	Took     time.Duration
	Checks   []Check
	Captured []string // the names captured
	Hooks    []string // what the hooks printed
	Frames   []Frame     // a WebSocket session, in order
	GRPC     *GRPCResult // a gRPC call's status and trailers
	Err      error       // why it was not sent or a hook failed
}

// Counts says how many messages a WebSocket session sent and received.
func (r Result) Counts() (sent, received int) {
	for _, f := range r.Frames {
		if f.Kind != "text" && f.Kind != "binary" {
			continue
		}
		if f.Out {
			sent++
		} else {
			received++
		}
	}
	return sent, received
}

// Passed counts the checks that held.
func (r Result) Passed() int {
	n := 0
	for _, c := range r.Checks {
		if c.OK {
			n++
		}
	}
	return n
}

// Failed reports whether the request failed: not sent, an error status, a
// hook that failed or a check that did not hold.
func (r Result) Failed() bool {
	return r.Err != nil || r.Status >= 400 || r.Passed() < len(r.Checks)
}

// osEnv reads a variable from the process, and else from the project's .env.
func (rn *Runner) osEnv() func(string) (string, bool) {
	dot, _ := DotEnv(filepath.Join(rn.Root, ".env"))
	return func(name string) (string, bool) {
		if v, ok := os.LookupEnv(name); ok {
			return v, true
		}
		v, ok := dot[name]
		return v, ok
	}
}

// Send sends the request i of f: it runs the pre hooks, fills in the
// variables, sends, checks and captures, and runs the post hooks.
func (rn *Runner) Send(ctx context.Context, f File, i int) Result {
	r := f.Requests[i]
	res := Result{Request: r, Protocol: r.Protocol()}
	if rn.Saved == nil {
		rn.Saved = map[string]string{}
	}
	osEnv := rn.osEnv()

	envs, err := LoadEnvs(rn.Dirs...)
	if err != nil {
		res.Err = err
		return res
	}
	want := rn.Env
	if want == "" {
		want = f.Env
	}
	res.Env = envs.Pick(want)
	if res.Env != "" && envs[res.Env] == nil {
		res.Err = fmt.Errorf("no environment %q in %s", res.Env, strings.Join(EnvFiles[:3], ", "))
		return res
	}
	envVars := map[string]string{}
	for k, v := range envs["$shared"] {
		envVars[k] = v
	}
	for k, v := range envs[res.Env] {
		envVars[k] = v
	}
	fileVars, reqVars, hookVars := map[string]string{}, map[string]string{}, map[string]string{}
	scope := Scope{hookVars, reqVars, rn.Saved, fileVars, envVars}
	for _, v := range f.Vars {
		fileVars[v.Name] = value(v.Value, scope, osEnv)
	}
	for _, v := range r.Vars {
		reqVars[v.Name] = value(v.Value, scope, osEnv)
	}

	for _, cmd := range r.Pre {
		cmd, _ = Fill(cmd, scope, osEnv)
		out, err := rn.hook(ctx, cmd, nil)
		for _, l := range doc.SplitLines(out) {
			if k, v, ok := strings.Cut(l, "="); ok && isName(k) {
				hookVars[k] = v
				continue
			}
			res.Hooks = append(res.Hooks, l)
		}
		if err != nil {
			res.Err = fmt.Errorf("@pre %s: %w", cmd, err)
			return res
		}
	}

	var missing []string
	fill := func(s string) string {
		out, m := Fill(s, scope, osEnv)
		missing = append(missing, m...)
		return out
	}
	url := fill(r.URL)
	header := make([][2]string, len(r.Header))
	for j, h := range r.Header {
		header[j] = [2]string{h[0], fill(h[1])}
	}
	body := fill(r.Body)
	var steps []Step
	var g GRPC
	switch res.Protocol {
	case "websocket":
		if r.WS != nil {
			for _, st := range r.WS.Steps {
				steps = append(steps, Step{Op: st.Op, Arg: fill(st.Arg)})
			}
			res.Curl = Websocat(url, masked(header), r.WS.Subprotocols)
		} else {
			res.Curl = Websocat(url, masked(header), nil)
		}
	case "grpc":
		g = GRPC{Reflection: true}
		if r.GRPC != nil {
			g = *r.GRPC
		}
		g.Method, g.Authority = fill(g.Method), fill(g.Authority)
		md := g.Metadata
		g.Metadata = nil
		for _, kv := range md {
			g.Metadata = append(g.Metadata, [2]string{kv[0], fill(kv[1])})
		}
		shown := g
		shown.Metadata = masked(g.Metadata)
		res.Curl = Grpcurl(url, body, masked(header), shown)
	default:
		res.Curl = Curl(r.Method, url, masked(header), body)
	}
	if len(missing) > 0 {
		res.Err = fmt.Errorf("no value for %s", strings.Join(unique(missing), ", "))
		return res
	}
	switch res.Protocol {
	case "websocket":
		rn.websocket(ctx, &res, url, header, steps)
	case "grpc":
		rn.grpc(ctx, &res, url, body, header, g)
	default:
		rn.http(ctx, &res, url, header, body)
	}
	if res.Err != nil {
		return res
	}
	rn.after(ctx, &res, scope, osEnv)
	return res
}

// http sends a plain HTTP request.
func (rn *Runner) http(ctx context.Context, res *Result, url string, header [][2]string, body string) {
	r := res.Request
	req, err := http.NewRequestWithContext(ctx, r.Method, url, strings.NewReader(body))
	if err != nil {
		res.Err = err
		return
	}
	for _, h := range header {
		if strings.EqualFold(h[0], "Host") {
			req.Host = h[1]
			continue
		}
		req.Header.Add(h[0], h[1])
	}
	client := rn.Client
	if client == nil {
		timeout := 30 * time.Second
		if r.Timeout > 0 {
			timeout = r.Timeout
		}
		client = &http.Client{Timeout: timeout}
	}
	started := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		res.Err = err
		res.Took = time.Since(started)
		return
	}
	res.Body, err = io.ReadAll(io.LimitReader(resp.Body, maxBody))
	resp.Body.Close()
	res.Took = time.Since(started)
	res.Status, res.Text, res.Header = resp.StatusCode, resp.Status, resp.Header
	if err != nil {
		res.Err = err
	}
}

// after checks and captures what came back and runs the post hooks, the
// same for every protocol.
func (rn *Runner) after(ctx context.Context, res *Result, scope Scope, osEnv func(string) (string, bool)) {
	r := res.Request
	got := response{status: res.Status, header: res.Header, body: res.Body, took: res.Took, grpc: res.GRPC}
	got.sent, got.received = res.Counts()
	for _, a := range r.Asserts {
		ok, seen := got.check(a)
		res.Checks = append(res.Checks, Check{Expr: a, OK: ok, Got: seen})
	}
	for _, c := range r.Captures {
		if v, ok := got.lookup(c.Expr); ok {
			rn.Saved[c.Name] = text(v)
			res.Captured = append(res.Captured, c.Name)
		}
	}
	if len(r.Post) > 0 {
		in := got.json()
		for _, cmd := range r.Post {
			cmd, _ = Fill(cmd, scope, osEnv)
			out, err := rn.hook(ctx, cmd, in)
			res.Hooks = append(res.Hooks, doc.SplitLines(out)...)
			if err != nil {
				res.Err = fmt.Errorf("@post %s: %w", cmd, err)
				break
			}
		}
	}
}

// secretHeaders are the headers whose values the log does not show: the
// log is on the board and the agent reads it.
var secretHeaders = []string{"authorization", "proxy-authorization", "cookie", "token", "secret", "key", "password", "session"}

// masked hides the values of headers that carry credentials.
func masked(header [][2]string) [][2]string {
	out := make([][2]string, len(header))
	for i, h := range header {
		out[i] = h
		name := strings.ToLower(h[0])
		for _, s := range secretHeaders {
			if strings.Contains(name, s) {
				out[i][1] = hide(h[1])
				break
			}
		}
	}
	return out
}

// hide keeps the scheme of a credential ("Bearer") and hides the rest.
func hide(v string) string {
	if scheme, _, ok := strings.Cut(v, " "); ok && len(scheme) <= 10 {
		return scheme + " ••••"
	}
	return "••••"
}

// hook runs a command of the file in the project folder, with stdin, and
// returns what it printed.
func (rn *Runner) hook(ctx context.Context, command string, stdin []byte) (string, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}
	cmd.Dir = rn.Root
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		err = fmt.Errorf("exit %d", exit.ExitCode())
	}
	return string(out), err
}

func isName(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if !(c == '_' || c == '.' || c == '-' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

func unique(s []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, x := range s {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}

// response is what checks and captures look into.
type response struct {
	status int
	header http.Header
	body   []byte
	took   time.Duration
	parsed any
	isJSON bool
	tried  bool
	// What a WebSocket session or a gRPC call adds.
	sent, received int
	grpc           *GRPCResult
}

// json is what a post hook reads on its standard input.
func (r *response) json() []byte {
	h := map[string]string{}
	for k := range r.header {
		h[k] = r.header.Get(k)
	}
	var body any = string(r.body)
	if v, ok := r.data(); ok {
		body = v
	}
	out, _ := json.Marshal(map[string]any{"status": r.status, "headers": h, "body": body, "time": r.took.Milliseconds()})
	return out
}

func (r *response) data() (any, bool) {
	if !r.tried {
		r.tried = true
		r.isJSON = json.Unmarshal(r.body, &r.parsed) == nil
	}
	return r.parsed, r.isJSON
}

// lookup finds what an expression names: status, time (in milliseconds),
// body, body.a.b[0], header.Name, with or without "response." before, as
// resterm and JetBrains write them ("response.statusCode", "response.json.id").
func (r *response) lookup(expr string) (any, bool) {
	e := strings.TrimSpace(expr)
	e = strings.TrimSuffix(strings.TrimPrefix(e, "{{"), "}}")
	e = strings.TrimSpace(e)
	e = strings.TrimPrefix(e, "response.")
	// resterm's calls: json("a.b[0]"), header("Name"), text().
	if name, arg, ok := call(e); ok {
		switch name {
		case "json":
			e = strings.TrimSuffix("body."+strings.TrimPrefix(strings.TrimPrefix(arg, "$"), "."), ".")
		case "header":
			e = "header." + arg
		case "text":
			return string(r.body), true
		}
	}
	head, rest, _ := strings.Cut(e, ".")
	if i := strings.Index(head, "["); i > 0 {
		head, rest = head[:i], strings.TrimPrefix(e[i:], ".")
	}
	switch head {
	case "status", "statusCode", "code":
		return float64(r.status), true
	case "received", "sent":
		if head == "received" {
			return float64(r.received), true
		}
		return float64(r.sent), true
	case "grpc":
		if r.grpc == nil {
			return nil, false
		}
		switch rest {
		case "status":
			return r.grpc.CodeName, true
		case "code":
			return float64(r.grpc.Code), true
		case "message":
			return r.grpc.Message, true
		}
		return nil, false
	case "trailer", "trailers":
		if r.grpc == nil || rest == "" {
			return nil, false
		}
		v := r.grpc.Trailer.Values(rest)
		if len(v) == 0 {
			return nil, false
		}
		return strings.Join(v, ", "), true
	case "time", "duration":
		return float64(r.took.Milliseconds()), true
	case "header", "headers":
		if rest == "" {
			return nil, false
		}
		v := r.header.Values(rest)
		if len(v) == 0 {
			return nil, false
		}
		return strings.Join(v, ", "), true
	case "body", "json":
		if rest == "" {
			if v, ok := r.data(); ok {
				return v, true
			}
			return string(r.body), true
		}
		v, ok := r.data()
		if !ok {
			return nil, false
		}
		return walk(v, rest)
	}
	return nil, false
}

// call reads `json("a.b")` into "json" and "a.b".
func call(e string) (name, arg string, ok bool) {
	open := strings.Index(e, "(")
	if open < 0 || !strings.HasSuffix(e, ")") {
		return "", "", false
	}
	arg = strings.TrimSpace(e[open+1 : len(e)-1])
	if len(arg) >= 2 && (arg[0] == '"' || arg[0] == '\'') && arg[len(arg)-1] == arg[0] {
		arg = arg[1 : len(arg)-1]
	}
	return e[:open], arg, true
}

// walk follows a path such as "items[0].id" into decoded JSON.
func walk(v any, path string) (any, bool) {
	for path != "" {
		var key string
		switch {
		case strings.HasPrefix(path, "["):
			end := strings.Index(path, "]")
			if end < 0 {
				return nil, false
			}
			n, err := strconv.Atoi(path[1:end])
			list, ok := v.([]any)
			if err != nil || !ok || n < 0 || n >= len(list) {
				return nil, false
			}
			v, path = list[n], strings.TrimPrefix(path[end+1:], ".")
			continue
		default:
			end := strings.IndexAny(path, ".[")
			if end < 0 {
				key, path = path, ""
			} else {
				key, path = path[:end], strings.TrimPrefix(path[end:], ".")
			}
		}
		m, ok := v.(map[string]any)
		if !ok {
			return nil, false
		}
		if v, ok = m[key]; !ok {
			return nil, false
		}
	}
	return v, true
}

// operators, longest first so that ">=" is not read as ">".
var operators = []string{" contains ", " in ", " exists", "==", "!=", ">=", "<=", ">", "<"}

// check evaluates an assertion such as `status == 200`,
// `body.user == "min"`, `time < 500` or `header.Content-Type contains json`.
// It returns whether it held and, when not, what was there.
func (r *response) check(expr string) (bool, string) {
	for _, op := range operators {
		i := outsideQuotes(expr, op)
		if i < 0 {
			continue
		}
		left, right := strings.TrimSpace(expr[:i]), strings.TrimSpace(expr[i+len(op):])
		if op == " in " { // resterm: "json" in response.header("Content-Type")
			left, right, op = right, left, " contains "
		}
		got, ok := r.lookup(left)
		op = strings.TrimSpace(op)
		if op == "exists" {
			return ok, "missing"
		}
		if !ok {
			return false, "missing"
		}
		want := literal(right)
		var held bool
		switch op {
		case "==":
			held = equal(got, want)
		case "!=":
			held = !equal(got, want)
		case "contains":
			held = contains(got, want)
		default:
			a, aok := got.(float64)
			b, bok := want.(float64)
			if aok && bok {
				held = map[string]bool{">": a > b, "<": a < b, ">=": a >= b, "<=": a <= b}[op]
			}
		}
		return held, short(text(got))
	}
	// A bare expression holds when it names something truthy.
	got, ok := r.lookup(expr)
	return ok && got != nil && got != false && got != "", short(text(got))
}

// outsideQuotes finds op in expr where it is not inside a quoted string.
func outsideQuotes(expr, op string) int {
	var quote byte
	for i := 0; i < len(expr); i++ {
		switch c := expr[i]; {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case strings.HasPrefix(expr[i:], op):
			return i
		}
	}
	return -1
}

// literal reads the right side of a check: JSON, or else a bare word.
func literal(s string) any {
	var v any
	if err := json.Unmarshal([]byte(s), &v); err == nil {
		return v
	}
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		return s[1 : len(s)-1]
	}
	return s
}

func equal(a, b any) bool {
	if x, ok := a.(float64); ok {
		if y, ok := b.(float64); ok {
			return x == y
		}
	}
	return text(a) == text(b)
}

func contains(a, b any) bool {
	if list, ok := a.([]any); ok {
		for _, x := range list {
			if equal(x, b) {
				return true
			}
		}
		return false
	}
	return strings.Contains(text(a), text(b))
}

// text writes a JSON value as a variable holds it: strings as they are.
func text(v any) string {
	switch v := v.(type) {
	case string:
		return v
	case nil:
		return "null"
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func short(s string) string {
	if r := []rune(s); len(r) > 60 {
		return string(r[:59]) + "…"
	}
	return s
}
