package httpfile

// What came back, and the small language of @assert and @capture that looks
// into it: a status, a header, a path into the JSON body, an operator.

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

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
