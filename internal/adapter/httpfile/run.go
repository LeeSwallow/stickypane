package httpfile

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/LeeSwallow/stickypane/internal/core/doc"
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
	Captured []string    // the names captured
	Hooks    []string    // what the hooks printed
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
		filled, _ := Fill(a, scope, osEnv) // "== {{user}}" compares with the user
		ok, seen := got.check(filled)
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
