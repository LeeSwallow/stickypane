// Package rpctest runs a small gRPC server for tests: users.v1.Users from
// prototest, with Get (unary; id 404 is NOT_FOUND) and List (a stream of
// three), and server reflection when asked. It echoes the x-trace-id
// metadata back as the X-Got-Trace header.
package rpctest

import (
	"bufio"
	"encoding/binary"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/proto"
	"github.com/LeeSwallow/stickypane/internal/proto/prototest"
)

// readMsgs reads the length-prefixed messages of a gRPC request body.
func readMsgs(t testing.TB, r io.Reader) [][]byte {
	t.Helper()
	var out [][]byte
	br := bufio.NewReader(r)
	for {
		var head [5]byte
		if _, err := io.ReadFull(br, head[:]); err != nil {
			return out
		}
		msg := make([]byte, binary.BigEndian.Uint32(head[1:]))
		io.ReadFull(br, msg)
		out = append(out, msg)
	}
}

func frame(msg []byte) []byte {
	return append(binary.BigEndian.AppendUint32([]byte{0}, uint32(len(msg))), msg...)
}

func user(id uint64, name string) []byte {
	b := proto.AppendVarint(nil, 1, id)
	b = proto.AppendString(b, 2, name)
	return proto.AppendVarint(b, 4, 1) // role: ADMIN
}

// Server is a gRPC server written the way the protocol is: HTTP/2 without
// TLS, length-prefixed messages and a grpc-status trailer. reflection turns
// on grpc.reflection.v1.
func Server(t testing.TB, reflection bool) string {
	t.Helper()
	mux := http.NewServeMux()
	reply := func(w http.ResponseWriter, msgs ...[]byte) {
		w.Header().Set("Content-Type", "application/grpc")
		w.Header().Set("Trailer", "Grpc-Status, Grpc-Message")
		for _, m := range msgs {
			w.Write(frame(m))
		}
		w.Header().Set("Grpc-Status", "0")
	}
	fail := func(w http.ResponseWriter, code int, msg string) {
		// A trailers-only response: the status comes in the headers.
		w.Header().Set("Content-Type", "application/grpc")
		w.Header().Set("Grpc-Status", strconv.Itoa(code))
		w.Header().Set("Grpc-Message", url.PathEscape(msg))
		w.WriteHeader(http.StatusOK)
	}
	mux.HandleFunc("/users.v1.Users/Get", func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor != 2 || r.Header.Get("Content-Type") != "application/grpc" || r.Header.Get("Te") != "trailers" {
			http.Error(w, "not grpc", http.StatusBadRequest)
			return
		}
		w.Header().Set("X-Got-Trace", r.Header.Get("X-Trace-Id"))
		msgs := readMsgs(t, r.Body)
		fields, _ := proto.Fields(msgs[0])
		id := fields[0].Int
		if id == 404 {
			fail(w, 5, "user 404 not found")
			return
		}
		reply(w, user(id, "min"))
	})
	mux.HandleFunc("/users.v1.Users/List", func(w http.ResponseWriter, r *http.Request) {
		readMsgs(t, r.Body)
		reply(w, user(1, "ann"), user(2, "bo"), user(3, "cy"))
	})
	if reflection {
		mux.HandleFunc("/grpc.reflection.v1.ServerReflection/ServerReflectionInfo", func(w http.ResponseWriter, r *http.Request) {
			var out [][]byte
			for _, m := range readMsgs(t, r.Body) {
				fields, _ := proto.Fields(m)
				for _, f := range fields {
					if f.Num == 4 && string(f.Data) == "users.v1.Users" {
						resp := proto.AppendBytes(nil, 4, proto.AppendBytes(nil, 1, prototest.UsersFile()))
						out = append(out, resp)
					}
				}
			}
			reply(w, out...)
		})
	}
	srv := httptest.NewUnstartedServer(mux)
	var p http.Protocols
	p.SetHTTP1(true)
	p.SetUnencryptedHTTP2(true)
	srv.Config.Protocols = &p
	srv.Start()
	t.Cleanup(srv.Close)
	return strings.TrimPrefix(srv.URL, "http://")
}
