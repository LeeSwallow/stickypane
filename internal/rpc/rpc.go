// Package rpc calls gRPC methods on the standard library: HTTP/2 (h2c
// without TLS, which net/http speaks since Go 1.24), length-prefixed
// messages and the grpc-status trailer. Messages are written and read as
// JSON through internal/proto, with the descriptors from a .protoset file,
// from the server's reflection service, or, without either, by field
// number. Unary calls and server streams; a client stream is not sent.
package rpc

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/LeeSwallow/stickypane/internal/proto"
)

// maxMessage is the largest response message read.
const maxMessage = 16 << 20

// Call is one call to make.
type Call struct {
	Target     string // host:port, or a URL whose scheme says whether TLS is used
	Method     string // package.Service/Method
	Plaintext  bool   // HTTP/2 without TLS
	Authority  string // the :authority, when not the target
	Metadata   [][2]string
	Body       []byte // the request message as JSON
	Descriptor []byte // a FileDescriptorSet; nil asks the server
	Reflection bool   // ask the server for its descriptors when there is no Descriptor
	Timeout    time.Duration
}

// Result is what came back.
type Result struct {
	Code      int    // the gRPC status code: 0 is OK
	CodeName  string // "OK", "NOT_FOUND"...
	Message   string // the status message
	Header    http.Header
	Trailer   http.Header
	Messages  [][]byte // each response message as JSON
	Streaming bool     // the method streams its responses
	Schema    string   // where the descriptors came from: descriptor, reflection, raw
	Took      time.Duration
}

// codes are the gRPC status codes by number.
var codes = []string{"OK", "CANCELLED", "UNKNOWN", "INVALID_ARGUMENT", "DEADLINE_EXCEEDED", "NOT_FOUND",
	"ALREADY_EXISTS", "PERMISSION_DENIED", "RESOURCE_EXHAUSTED", "FAILED_PRECONDITION", "ABORTED",
	"OUT_OF_RANGE", "UNIMPLEMENTED", "INTERNAL", "UNAVAILABLE", "DATA_LOSS", "UNAUTHENTICATED"}

// CodeName names a status code.
func CodeName(code int) string {
	if code >= 0 && code < len(codes) {
		return codes[code]
	}
	return "CODE_" + strconv.Itoa(code)
}

// HTTPStatus is the HTTP status that means what a gRPC code means, as the
// gRPC-HTTP gateway maps them, so a call ends like a request on the board.
func HTTPStatus(code int) int {
	switch code {
	case 0:
		return 200
	case 1:
		return 499
	case 3, 9, 11:
		return 400
	case 4:
		return 504
	case 5:
		return 404
	case 6, 10:
		return 409
	case 7:
		return 403
	case 8:
		return 429
	case 12:
		return 501
	case 14:
		return 503
	case 16:
		return 401
	}
	return 500
}

// reserved are the metadata keys the transport writes itself.
var reserved = map[string]bool{"content-type": true, "te": true, "user-agent": true, "host": true, ":authority": true}

// Invoke makes the call. A status other than OK is a Result, not an error;
// an error means the call could not be made.
func Invoke(ctx context.Context, c Call) (Result, error) {
	var res Result
	base, plaintext, err := baseURL(c.Target, c.Plaintext)
	if err != nil {
		return res, err
	}
	method := strings.TrimPrefix(c.Method, "/")
	if !strings.Contains(method, "/") {
		if i := strings.LastIndex(method, "."); i > 0 && strings.Contains(method[:i], ".") {
			method = method[:i] + "/" + method[i+1:]
		} else {
			return res, fmt.Errorf("%q is not a method: write package.Service/Method", c.Method)
		}
	}
	for _, kv := range c.Metadata {
		k := strings.ToLower(kv[0])
		if reserved[k] || strings.HasPrefix(k, "grpc-") {
			return res, fmt.Errorf("metadata %q is the transport's own; leave it out", kv[0])
		}
	}
	client := &http.Client{Transport: transport(plaintext)}

	// The descriptors: given, asked for, or none.
	var schema *proto.Schema
	switch {
	case len(c.Descriptor) > 0:
		schema = proto.NewSchema()
		if err := schema.AddSet(c.Descriptor); err != nil {
			return res, fmt.Errorf("the descriptor file: %w", err)
		}
		res.Schema = "descriptor"
	case c.Reflection:
		svc, _, _ := strings.Cut(method, "/")
		s, err := reflect(ctx, client, base, c.Authority, svc)
		if err == nil {
			schema, res.Schema = s, "reflection"
		} else {
			res.Schema = "raw (" + err.Error() + ")"
		}
	default:
		res.Schema = "raw"
	}

	var body []byte
	var output string
	if schema != nil {
		m, err := schema.Method(method)
		if err != nil {
			return res, err
		}
		if m.ClientStream {
			return res, fmt.Errorf("%s takes a stream of messages, which is not sent yet", method)
		}
		res.Streaming, output = m.ServerStream, m.Output
		if body, err = schema.Encode(m.Input, c.Body); err != nil {
			return res, err
		}
	} else if body, err = proto.EncodeRaw(c.Body); err != nil {
		return res, err
	}

	started := time.Now()
	msgs, header, trailer, err := exchange(ctx, client, base+"/"+method, c.Authority, c.Metadata, c.Timeout, [][]byte{body})
	res.Took = time.Since(started)
	res.Header, res.Trailer = header, trailer
	if err != nil {
		return res, err
	}
	res.Code, res.Message = status(header, trailer)
	res.CodeName = CodeName(res.Code)
	for _, m := range msgs {
		var out []byte
		var err error
		if schema != nil {
			out, err = schema.Decode(output, m)
		} else {
			out, err = proto.DecodeRaw(m)
		}
		if err != nil {
			return res, fmt.Errorf("a response message: %w", err)
		}
		res.Messages = append(res.Messages, out)
	}
	if len(res.Messages) > 1 {
		res.Streaming = true
	}
	return res, nil
}

// baseURL reads a target: "host:port" (TLS unless plaintext), or a URL.
func baseURL(target string, plaintext bool) (string, bool, error) {
	target = strings.TrimSpace(strings.TrimSuffix(target, "/"))
	switch {
	case target == "":
		return "", false, errors.New("a gRPC call needs a target: GRPC host:port")
	case strings.HasPrefix(target, "http://"), strings.HasPrefix(target, "grpc://"):
		return "http://" + strings.SplitN(target, "://", 2)[1], true, nil
	case strings.HasPrefix(target, "https://"), strings.HasPrefix(target, "grpcs://"):
		return "https://" + strings.SplitN(target, "://", 2)[1], false, nil
	case plaintext:
		return "http://" + target, true, nil
	}
	return "https://" + target, false, nil
}

// transport speaks HTTP/2 only: over TLS, or in the clear.
func transport(plaintext bool) *http.Transport {
	var p http.Protocols
	if plaintext {
		p.SetUnencryptedHTTP2(true)
	} else {
		p.SetHTTP2(true)
	}
	return &http.Transport{Protocols: &p, ForceAttemptHTTP2: true, ResponseHeaderTimeout: 30 * time.Second}
}

// exchange sends messages to a method and reads every message back.
func exchange(ctx context.Context, client *http.Client, u, authority string, md [][2]string, timeout time.Duration, msgs [][]byte) ([][]byte, http.Header, http.Header, error) {
	var body bytes.Buffer
	for _, m := range msgs {
		body.WriteByte(0)
		binary.Write(&body, binary.BigEndian, uint32(len(m)))
		body.Write(m)
	}
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, &body)
	if err != nil {
		return nil, nil, nil, err
	}
	req.Header.Set("Content-Type", "application/grpc")
	req.Header.Set("Te", "trailers")
	req.Header.Set("User-Agent", "stickypane")
	if timeout > 0 {
		req.Header.Set("Grpc-Timeout", strconv.FormatInt(timeout.Milliseconds(), 10)+"m")
	}
	for _, kv := range md {
		req.Header.Add(kv[0], kv[1])
	}
	if authority != "" {
		req.Host = authority
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, nil, err
	}
	defer resp.Body.Close()
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "application/grpc") {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 200))
		return nil, resp.Header, nil, fmt.Errorf("not a gRPC answer: %s %s", resp.Status, strings.TrimSpace(string(snippet)))
	}
	var out [][]byte
	r := bufio.NewReader(resp.Body)
	for {
		var head [5]byte
		if _, err := io.ReadFull(r, head[:]); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return out, resp.Header, resp.Trailer, err
		}
		if head[0]&1 != 0 {
			return out, resp.Header, resp.Trailer, errors.New("the server compressed a message, which is not read")
		}
		n := binary.BigEndian.Uint32(head[1:])
		if n > maxMessage {
			return out, resp.Header, resp.Trailer, fmt.Errorf("a message of %d bytes is over %d", n, maxMessage)
		}
		m := make([]byte, n)
		if _, err := io.ReadFull(r, m); err != nil {
			return out, resp.Header, resp.Trailer, err
		}
		out = append(out, m)
	}
	return out, resp.Header, resp.Trailer, nil
}

// status reads grpc-status and grpc-message from the trailers, or from the
// headers of a trailers-only answer.
func status(header, trailer http.Header) (int, string) {
	for _, h := range []http.Header{trailer, header} {
		if v := h.Get("Grpc-Status"); v != "" {
			code, err := strconv.Atoi(v)
			if err != nil {
				return 2, "a grpc-status that is not a number: " + v
			}
			msg, err := url.PathUnescape(h.Get("Grpc-Message"))
			if err != nil {
				msg = h.Get("Grpc-Message")
			}
			return code, msg
		}
	}
	return 2, "the server gave no grpc-status"
}

// reflect asks the server's reflection service for the descriptors of a
// service and of the files they need, trying v1 and then v1alpha.
func reflect(ctx context.Context, client *http.Client, base, authority, service string) (*proto.Schema, error) {
	var last error
	for _, svc := range []string{"grpc.reflection.v1.ServerReflection", "grpc.reflection.v1alpha.ServerReflection"} {
		s, err := reflectWith(ctx, client, base+"/"+svc+"/ServerReflectionInfo", authority, service)
		if err == nil {
			return s, nil
		}
		last = err
	}
	return nil, fmt.Errorf("no server reflection: %w", last)
}

func reflectWith(ctx context.Context, client *http.Client, u, authority, service string) (*proto.Schema, error) {
	schema := proto.NewSchema()
	ask := [][]byte{proto.AppendString(nil, 4, service)} // file_containing_symbol
	for round := 0; round < 8 && len(ask) > 0; round++ {
		msgs, header, trailer, err := exchange(ctx, client, u, authority, nil, 10*time.Second, ask)
		if err != nil {
			return nil, err
		}
		if code, msg := status(header, trailer); code != 0 {
			return nil, fmt.Errorf("%s %s", CodeName(code), msg)
		}
		for _, m := range msgs {
			fields, err := proto.Fields(m)
			if err != nil {
				return nil, err
			}
			for _, f := range fields {
				switch f.Num {
				case 4: // file_descriptor_response
					files, _ := proto.Fields(f.Data)
					for _, file := range files {
						if file.Num == 1 {
							if err := schema.AddFile(file.Data); err != nil {
								return nil, err
							}
						}
					}
				case 7: // error_response
					parts, _ := proto.Fields(f.Data)
					for _, p := range parts {
						if p.Num == 2 {
							return nil, errors.New(string(p.Data))
						}
					}
					return nil, errors.New("the reflection service answered with an error")
				}
			}
		}
		ask = nil
		for _, name := range schema.Missing() {
			ask = append(ask, proto.AppendString(nil, 3, name)) // file_by_filename
		}
	}
	if len(schema.Services) == 0 {
		return nil, fmt.Errorf("the server does not describe %s", service)
	}
	return schema, nil
}
