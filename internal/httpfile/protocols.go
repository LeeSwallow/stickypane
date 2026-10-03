package httpfile

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/LeeSwallow/stickypane/internal/rpc"
	"github.com/LeeSwallow/stickypane/internal/wsock"
)

// Frame is one line of a WebSocket session's transcript: a message sent or
// received, or a pause the script took.
type Frame struct {
	Out  bool   // sent, not received
	Kind string // text, binary, ping, pong, close, wait
	Data string // the text, a binary message's size, a wait's length
	Code int    // a close's status code
	At   time.Time
}

// GRPCResult is what a gRPC call adds to a Result.
type GRPCResult struct {
	Method    string
	Code      int
	CodeName  string
	Message   string
	Schema    string // where the descriptors came from
	Streaming bool
	Trailer   http.Header
}

// Defaults for a WebSocket session that does not say.
const (
	wsTimeout = 10 * time.Second // the handshake
	wsIdle    = time.Second      // listening after the last step
	wsLimit   = 60 * time.Second // the whole session
)

// websocket runs a WebSocket request: the handshake with the request's
// headers, then each step, writing down every message both ways. The
// received messages are the body that checks look into, as a JSON array.
func (rn *Runner) websocket(ctx context.Context, res *Result, url string, header [][2]string, steps []Step) {
	ws := res.Request.WS
	if ws == nil {
		ws = &WebSocket{}
	}
	timeout, idle := ws.Timeout, ws.Idle
	if timeout <= 0 {
		timeout = wsTimeout
	}
	if idle <= 0 {
		idle = wsIdle
	}
	h := http.Header{}
	for _, kv := range header {
		h.Add(kv[0], kv[1])
	}
	limit := wsLimit
	if res.Request.Timeout > 0 {
		limit = res.Request.Timeout
	}
	ctx, cancel := context.WithTimeout(ctx, limit)
	defer cancel()
	dialCtx, dialCancel := context.WithTimeout(ctx, timeout)
	started := time.Now()
	c, err := wsock.Dial(dialCtx, url, h, ws.Subprotocols)
	dialCancel()
	if err != nil {
		res.Err, res.Took = err, time.Since(started)
		return
	}
	defer c.End()
	res.Status, res.Text, res.Header = c.Status, "101 Switching Protocols", c.Header

	var received []any
	keep := func(m wsock.Message) {
		f := Frame{Kind: m.Kind.String(), Data: string(m.Data), Code: m.Code, At: m.At}
		if m.Kind == wsock.Binary {
			f.Data = fmt.Sprintf("%d bytes", len(m.Data))
		}
		res.Frames = append(res.Frames, f)
		if m.Kind == wsock.Text || m.Kind == wsock.Binary {
			var v any
			if json.Unmarshal(m.Data, &v) != nil {
				v = string(m.Data)
			}
			received = append(received, v)
		}
	}
	closed := false
	// listen keeps what comes in for d, or until the server closes.
	listen := func(d time.Duration) {
		end := time.Now().Add(d)
		for !closed {
			left := time.Until(end)
			if left <= 0 || ctx.Err() != nil {
				return
			}
			m, err := c.Read(left)
			if err != nil {
				if !wsock.IsTimeout(err) {
					closed = true
				}
				return
			}
			keep(m)
			if m.Kind == wsock.Close {
				closed = true
			}
		}
	}
	listen(50 * time.Millisecond) // what the server says on its own when it opens
	for _, s := range steps {
		if closed {
			break
		}
		out := Frame{Out: true, Kind: s.Op, Data: s.Arg, At: time.Now()}
		var err error
		switch s.Op {
		case "send", "send-json":
			out.Kind = "text"
			if s.Op == "send-json" && !json.Valid([]byte(s.Arg)) {
				err = fmt.Errorf("@ws send-json: %s is not JSON", s.Arg)
				break
			}
			err = c.Send(wsock.Text, []byte(s.Arg))
		case "send-base64":
			var b []byte
			if b, err = base64.StdEncoding.DecodeString(s.Arg); err == nil {
				out.Kind, out.Data = "binary", fmt.Sprintf("%d bytes", len(b))
				err = c.Send(wsock.Binary, b)
			}
		case "send-file":
			var b []byte
			if b, err = os.ReadFile(rn.path(s.Arg)); err == nil {
				out.Kind, out.Data = "binary", fmt.Sprintf("%d bytes from %s", len(b), s.Arg)
				err = c.Send(wsock.Binary, b)
			}
		case "ping":
			err = c.Send(wsock.Ping, []byte(s.Arg))
		case "pong":
			err = c.Send(wsock.Pong, []byte(s.Arg))
		case "wait":
			var d time.Duration
			if d, err = time.ParseDuration(s.Arg); err == nil {
				res.Frames = append(res.Frames, out)
				listen(d)
				continue
			}
		case "close":
			code, reason := 1000, s.Arg
			if first, rest, _ := strings.Cut(s.Arg, " "); first != "" {
				if n, err := strconv.Atoi(first); err == nil {
					code, reason = n, strings.TrimSpace(rest)
				}
			}
			out.Code, out.Data = code, reason
			err = c.Close(code, reason)
		default:
			err = fmt.Errorf("@ws %s: the steps are send, send-json, send-base64, send-file, ping, pong, wait and close", s.Op)
		}
		if err != nil {
			res.Err = err
			break
		}
		res.Frames = append(res.Frames, out)
		listen(10 * time.Millisecond)
	}
	if res.Err == nil {
		listen(idle)
	}
	res.Took = time.Since(started)
	res.Body, _ = json.Marshal(received)
	if received == nil {
		res.Body = []byte("[]")
	}
}

// grpc makes a gRPC call. A status other than OK is a response, mapped to
// the HTTP status that means the same, so it ends and fails like one.
func (rn *Runner) grpc(ctx context.Context, res *Result, target, body string, header [][2]string, g GRPC) {
	call := rpc.Call{
		Target:     target,
		Method:     g.Method,
		Plaintext:  g.Plaintext,
		Authority:  g.Authority,
		Metadata:   append(append([][2]string{}, header...), g.Metadata...),
		Body:       []byte(body),
		Reflection: g.Reflection,
		Timeout:    res.Request.Timeout,
	}
	if g.Method == "" {
		res.Err = errors.New("a GRPC request needs # @grpc package.Service/Method")
		return
	}
	if g.Descriptor != "" {
		b, err := os.ReadFile(rn.path(g.Descriptor))
		if err != nil {
			res.Err = fmt.Errorf("@grpc-descriptor: %w", err)
			return
		}
		call.Descriptor = b
	}
	if call.Timeout == 0 {
		call.Timeout = 30 * time.Second
	}
	out, err := rpc.Invoke(ctx, call)
	res.Took, res.Header = out.Took, out.Header
	res.GRPC = &GRPCResult{Method: g.Method, Code: out.Code, CodeName: out.CodeName, Message: out.Message,
		Schema: out.Schema, Streaming: out.Streaming, Trailer: out.Trailer}
	if err != nil {
		res.Err = err
		return
	}
	res.Status = rpc.HTTPStatus(out.Code)
	res.Text = strconv.Itoa(res.Status) + " " + out.CodeName
	switch {
	case out.Streaming:
		res.Body = append([]byte("["), joinJSON(out.Messages)...)
		res.Body = append(res.Body, ']')
	case len(out.Messages) == 1:
		res.Body = out.Messages[0]
	}
}

func joinJSON(msgs [][]byte) []byte {
	var out []byte
	for i, m := range msgs {
		if i > 0 {
			out = append(out, ',')
		}
		out = append(out, m...)
	}
	return out
}

// path is a file the .http file names, relative to its folder.
func (rn *Runner) path(name string) string {
	if filepath.IsAbs(name) {
		return name
	}
	base := rn.Here
	if base == "" {
		base = rn.Root
	}
	return filepath.Join(base, name)
}

// Websocat writes a WebSocket request as a websocat command, the curl of
// WebSockets, so it can be opened again by hand.
func Websocat(url string, header [][2]string, protocols []string) string {
	parts := []string{"websocat"}
	for _, h := range header {
		parts = append(parts, "-H", quote(h[0]+": "+h[1]))
	}
	if len(protocols) > 0 {
		parts = append(parts, "--protocol", quote(strings.Join(protocols, ", ")))
	}
	return strings.Join(append(parts, quote(url)), " ")
}

// Grpcurl writes a gRPC call as a grpcurl command.
func Grpcurl(target, body string, header [][2]string, g GRPC) string {
	parts := []string{"grpcurl"}
	if g.Plaintext {
		parts = append(parts, "-plaintext")
	}
	if g.Descriptor != "" {
		parts = append(parts, "-protoset", quote(g.Descriptor))
	}
	if g.Authority != "" {
		parts = append(parts, "-authority", quote(g.Authority))
	}
	for _, h := range append(append([][2]string{}, header...), g.Metadata...) {
		parts = append(parts, "-H", quote(h[0]+": "+h[1]))
	}
	if strings.TrimSpace(body) != "" {
		parts = append(parts, "-d", quote(compact(body)))
	}
	return strings.Join(append(parts, quote(target), g.Method), " ")
}

// compact puts a JSON body on one line, so the command fits one.
func compact(body string) string {
	var v any
	if json.Unmarshal([]byte(body), &v) != nil {
		return body
	}
	b, _ := json.Marshal(v)
	return string(b)
}
