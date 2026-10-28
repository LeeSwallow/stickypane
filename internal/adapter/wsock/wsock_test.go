package wsock

import (
	"bufio"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// echo is a test server: it accepts the handshake, says hello, and sends
// back every text or binary message; a ping gets its pong from the client
// side's read loop, which is what Conn does.
func echo(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer t" {
			http.Error(w, "no", http.StatusUnauthorized)
			return
		}
		key := r.Header.Get("Sec-WebSocket-Key")
		conn, rw, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()
		proto := ""
		if strings.Contains(r.Header.Get("Sec-WebSocket-Protocol"), "chat.v2") {
			proto = "Sec-WebSocket-Protocol: chat.v2\r\n"
		}
		rw.WriteString("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " + accept(key) + "\r\n" + proto + "\r\n")
		writeFrame(rw.Writer, true, opText, []byte(`{"type":"welcome"}`), false)
		rw.Flush()
		for {
			f, err := readFrame(rw.Reader)
			if err != nil {
				return
			}
			switch f.op {
			case opText, opBinary:
				writeFrame(rw.Writer, true, f.op, append([]byte("echo:"), f.data...), false)
			case opPing:
				writeFrame(rw.Writer, true, opPong, f.data, false)
			case opClose:
				writeFrame(rw.Writer, true, opClose, f.data, false)
				rw.Flush()
				return
			}
			rw.Flush()
		}
	}))
}

func TestASessionSendsReceivesPingsAndCloses(t *testing.T) {
	srv := echo(t)
	defer srv.Close()
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/chat"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, err := Dial(ctx, url, http.Header{"Authorization": {"Bearer t"}}, []string{"chat.v2", "json"})
	if err != nil {
		t.Fatal(err)
	}
	if c.Protocol != "chat.v2" || c.Status != 101 {
		t.Errorf("protocol %q, status %d", c.Protocol, c.Status)
	}
	m, err := c.Read(time.Second)
	if err != nil || m.Kind != Text || string(m.Data) != `{"type":"welcome"}` {
		t.Fatalf("first message = %+v, %v", m, err)
	}
	if err := c.Send(Text, []byte("hello")); err != nil {
		t.Fatal(err)
	}
	if m, err := c.Read(time.Second); err != nil || string(m.Data) != "echo:hello" {
		t.Errorf("echo = %+v, %v", m, err)
	}
	big := bytes.Repeat([]byte("x"), 70000) // a 64-bit length
	if err := c.Send(Binary, big); err != nil {
		t.Fatal(err)
	}
	if m, err := c.Read(time.Second); err != nil || m.Kind != Binary || len(m.Data) != len(big)+5 {
		t.Errorf("big echo = %d bytes, %v", len(m.Data), err)
	}
	if err := c.Send(Ping, []byte("beat")); err != nil {
		t.Fatal(err)
	}
	if m, err := c.Read(time.Second); err != nil || m.Kind != Pong || string(m.Data) != "beat" {
		t.Errorf("pong = %+v, %v", m, err)
	}
	if err := c.Close(1000, "bye"); err != nil {
		t.Fatal(err)
	}
	if m, err := c.Read(time.Second); err != nil || m.Kind != Close || m.Code != 1000 || string(m.Data) != "bye" {
		t.Errorf("close = %+v, %v", m, err)
	}
}

func TestAReadThatWaitsTooLongSaysSo(t *testing.T) {
	srv := echo(t)
	defer srv.Close()
	c, err := Dial(context.Background(), "ws"+strings.TrimPrefix(srv.URL, "http"), http.Header{"Authorization": {"Bearer t"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close(1000, "")
	c.Read(time.Second) // the welcome
	if _, err := c.Read(50 * time.Millisecond); !IsTimeout(err) {
		t.Errorf("err = %v, want a timeout", err)
	}
	// The connection still works after a timeout.
	if err := c.Send(Text, []byte("still here")); err != nil {
		t.Fatal(err)
	}
	if m, err := c.Read(time.Second); err != nil || string(m.Data) != "echo:still here" {
		t.Errorf("after a timeout: %+v, %v", m, err)
	}
}

func TestARefusedHandshakeGivesTheStatus(t *testing.T) {
	srv := echo(t)
	defer srv.Close()
	_, err := Dial(context.Background(), "ws"+strings.TrimPrefix(srv.URL, "http"), nil, nil)
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Errorf("err = %v, want the 401", err)
	}
}

func TestFramesRoundTripMaskedAndInPieces(t *testing.T) {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	writeFrame(w, false, opText, []byte("hel"), true)
	writeFrame(w, true, opPing, []byte("p"), true) // a control frame between the pieces
	writeFrame(w, true, opContinuation, []byte("lo"), true)
	w.Flush()
	raw := buf.Bytes()
	if raw[1]&0x80 == 0 {
		t.Fatal("a client frame is masked")
	}
	if bytes.Contains(raw, []byte("hel")) {
		t.Error("the masked payload should not show")
	}
	r := bufio.NewReader(bytes.NewReader(raw))
	var got []string
	for {
		f, err := readFrame(r)
		if err != nil {
			break
		}
		got = append(got, f.op.String()+":"+string(f.data))
	}
	if strings.Join(got, " ") != "text:hel ping:p continuation:lo" {
		t.Errorf("frames = %v", got)
	}
}

func TestTheAcceptKeyIsTheRFCs(t *testing.T) {
	// RFC 6455, section 1.3.
	if got := accept("dGhlIHNhbXBsZSBub25jZQ=="); got != "s3pPLMBiTxaQ9kYGzzhZRbK+xOo=" {
		t.Errorf("accept = %q", got)
	}
}

func TestAcceptServesTheOtherEnd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := Accept(w, r, "chat.v2")
		if err != nil {
			return
		}
		defer c.End()
		for {
			m, err := c.Read(2 * time.Second)
			if err != nil || m.Kind == Close {
				c.Close(1000, "bye")
				return
			}
			if m.Kind == Text {
				c.Send(Text, []byte("got "+string(m.Data)))
			}
		}
	}))
	defer srv.Close()
	c, err := Dial(context.Background(), "ws"+strings.TrimPrefix(srv.URL, "http"), nil, []string{"json", "chat.v2"})
	if err != nil || c.Protocol != "chat.v2" {
		t.Fatalf("dial: %v, protocol %q", err, c.Protocol)
	}
	c.Send(Text, []byte("hi"))
	if m, err := c.Read(time.Second); err != nil || string(m.Data) != "got hi" {
		t.Errorf("reply = %+v, %v", m, err)
	}
	c.Close(1000, "")
	if m, err := c.Read(time.Second); err != nil || m.Kind != Close || string(m.Data) != "bye" {
		t.Errorf("close = %+v, %v", m, err)
	}
}
