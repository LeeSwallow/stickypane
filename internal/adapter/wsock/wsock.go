// Package wsock is a WebSocket client (RFC 6455) on the standard library:
// the opening handshake, frames, and messages read with a time limit. It is
// what a .http note's "@websocket" request talks through. It does what a
// script of sends and waits needs, and no more: no compression, no server.
package wsock

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// MaxMessage is the largest message Read takes; a larger one ends the
// connection.
const MaxMessage = 16 << 20

// Kind is what a message is.
type Kind int

// The kinds of message. Ping, Pong and Close are control messages.
const (
	Text Kind = iota + 1
	Binary
	Ping
	Pong
	Close
)

func (k Kind) String() string {
	return [...]string{"", "text", "binary", "ping", "pong", "close"}[k]
}

// Message is one message, whole, as it was read or sent.
type Message struct {
	Kind Kind
	Data []byte
	Code int // a close message's status code
	At   time.Time
}

// ErrTimeout is what Read returns when nothing came in time.
var ErrTimeout = errors.New("no message in time")

// IsTimeout reports whether err is a Read that waited in vain.
func IsTimeout(err error) bool { return errors.Is(err, ErrTimeout) }

// Conn is an open WebSocket connection.
type Conn struct {
	Status   int         // the handshake's status: 101
	Protocol string      // the subprotocol the server chose
	Header   http.Header // the handshake's response headers

	conn   net.Conn
	server bool // a server does not mask what it sends
	w      *bufio.Writer
	wmu    sync.Mutex
	in     chan Message
	failed chan error
	once   sync.Once
}

// Dial opens a connection to a ws:// or wss:// URL, with the headers given
// and the subprotocols offered. ctx limits the handshake only.
func Dial(ctx context.Context, rawURL string, header http.Header, protocols []string) (*Conn, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	secure := false
	switch strings.ToLower(u.Scheme) {
	case "ws", "http":
	case "wss", "https":
		secure = true
	default:
		return nil, fmt.Errorf("%s is not a ws:// or wss:// URL", rawURL)
	}
	addr := u.Host
	if u.Port() == "" {
		addr = net.JoinHostPort(u.Hostname(), map[bool]string{false: "80", true: "443"}[secure])
	}
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	if secure {
		tc := tls.Client(conn, &tls.Config{ServerName: u.Hostname(), NextProtos: []string{"http/1.1"}})
		if err := tc.HandshakeContext(ctx); err != nil {
			conn.Close()
			return nil, err
		}
		conn = tc
	}
	if dl, ok := ctx.Deadline(); ok {
		conn.SetDeadline(dl)
	}
	c, err := handshake(conn, u, header, protocols)
	if err != nil {
		conn.Close()
		return nil, err
	}
	conn.SetDeadline(time.Time{})
	return c, nil
}

// handshake sends the upgrade request and reads the answer.
func handshake(conn net.Conn, u *url.URL, header http.Header, protocols []string) (*Conn, error) {
	nonce := make([]byte, 16)
	rand.Read(nonce)
	key := base64.StdEncoding.EncodeToString(nonce)
	path := u.RequestURI()
	host := u.Host
	var b strings.Builder
	fmt.Fprintf(&b, "GET %s HTTP/1.1\r\n", path)
	for k, vs := range header {
		if strings.EqualFold(k, "Host") && len(vs) > 0 {
			host = vs[0]
		}
	}
	fmt.Fprintf(&b, "Host: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\n", host, key)
	if len(protocols) > 0 {
		fmt.Fprintf(&b, "Sec-WebSocket-Protocol: %s\r\n", strings.Join(protocols, ", "))
	}
	for k, vs := range header {
		if strings.EqualFold(k, "Host") {
			continue
		}
		for _, v := range vs {
			fmt.Fprintf(&b, "%s: %s\r\n", k, v)
		}
	}
	b.WriteString("\r\n")
	if _, err := io.WriteString(conn, b.String()); err != nil {
		return nil, err
	}
	r := bufio.NewReader(conn)
	req := &http.Request{Method: http.MethodGet, URL: u}
	resp, err := http.ReadResponse(r, req)
	if err != nil {
		return nil, fmt.Errorf("handshake: %w", err)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 200))
		resp.Body.Close()
		msg := fmt.Sprintf("handshake: %s", resp.Status)
		if s := strings.TrimSpace(string(body)); s != "" {
			msg += ": " + s
		}
		return nil, errors.New(msg)
	}
	if got := resp.Header.Get("Sec-WebSocket-Accept"); got != accept(key) {
		return nil, fmt.Errorf("handshake: the server's Sec-WebSocket-Accept %q does not answer the key", got)
	}
	c := &Conn{
		Status:   resp.StatusCode,
		Protocol: resp.Header.Get("Sec-WebSocket-Protocol"),
		Header:   resp.Header,
		conn:     conn,
		w:        bufio.NewWriter(conn),
		in:       make(chan Message, 64),
		failed:   make(chan error, 1),
	}
	go c.readLoop(r)
	return c, nil
}

// Accept answers a WebSocket handshake on the server side and returns the
// connection, for tests and small servers. It picks the first of protocols
// the client offered.
func Accept(w http.ResponseWriter, r *http.Request, protocols ...string) (*Conn, error) {
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") || r.Header.Get("Sec-WebSocket-Key") == "" {
		http.Error(w, "a WebSocket handshake is expected", http.StatusBadRequest)
		return nil, errors.New("not a WebSocket handshake")
	}
	chosen := ""
	for _, offered := range strings.Split(r.Header.Get("Sec-WebSocket-Protocol"), ",") {
		for _, p := range protocols {
			if strings.TrimSpace(offered) == p && chosen == "" {
				chosen = p
			}
		}
	}
	conn, rw, err := w.(http.Hijacker).Hijack()
	if err != nil {
		return nil, err
	}
	head := "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " + accept(r.Header.Get("Sec-WebSocket-Key")) + "\r\n"
	if chosen != "" {
		head += "Sec-WebSocket-Protocol: " + chosen + "\r\n"
	}
	if _, err := rw.WriteString(head + "\r\n"); err != nil {
		conn.Close()
		return nil, err
	}
	if err := rw.Flush(); err != nil {
		conn.Close()
		return nil, err
	}
	c := &Conn{Status: http.StatusSwitchingProtocols, Protocol: chosen, conn: conn, server: true, w: rw.Writer,
		in: make(chan Message, 64), failed: make(chan error, 1)}
	go c.readLoop(rw.Reader)
	return c, nil
}

// accept is the Sec-WebSocket-Accept value that answers key.
func accept(key string) string {
	h := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(h[:])
}

// readLoop reads frames until the connection ends, joins the pieces of a
// message, answers pings, and hands each message to Read.
func (c *Conn) readLoop(r *bufio.Reader) {
	var parts []byte
	var kind opcode
	for {
		f, err := readFrame(r)
		if err != nil {
			c.failed <- err
			close(c.in)
			return
		}
		now := time.Now()
		switch f.op {
		case opPing:
			c.write(true, opPong, f.data)
			c.in <- Message{Kind: Ping, Data: f.data, At: now}
		case opPong:
			c.in <- Message{Kind: Pong, Data: f.data, At: now}
		case opClose:
			m := Message{Kind: Close, At: now}
			if len(f.data) >= 2 {
				m.Code = int(binary.BigEndian.Uint16(f.data))
				m.Data = f.data[2:]
			}
			c.in <- m
		case opText, opBinary, opContinuation:
			if f.op != opContinuation {
				kind, parts = f.op, nil
			}
			parts = append(parts, f.data...)
			if len(parts) > MaxMessage {
				c.failed <- fmt.Errorf("a message over %d bytes", MaxMessage)
				close(c.in)
				return
			}
			if f.fin {
				k := Text
				if kind == opBinary {
					k = Binary
				}
				c.in <- Message{Kind: k, Data: parts, At: now}
				parts = nil
			}
		}
	}
}

// Read returns the next message, waiting at most wait. A ping is answered
// by itself and returned too, so the caller can show it.
func (c *Conn) Read(wait time.Duration) (Message, error) {
	t := time.NewTimer(wait)
	defer t.Stop()
	select {
	case m, ok := <-c.in:
		if !ok {
			err := <-c.failed
			c.failed <- err // the next Read says the same
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				err = errors.New("the connection is closed")
			}
			return Message{}, err
		}
		return m, nil
	case <-t.C:
		return Message{}, ErrTimeout
	}
}

// Send sends one message: Text, Binary, Ping or Pong.
func (c *Conn) Send(kind Kind, data []byte) error {
	op := map[Kind]opcode{Text: opText, Binary: opBinary, Ping: opPing, Pong: opPong}[kind]
	if op == 0 && kind != Text {
		return fmt.Errorf("cannot send a %s message with Send", kind)
	}
	return c.write(true, op, data)
}

// Close sends a close message with a status code and a reason. The
// server's close comes back through Read; End then drops the connection.
func (c *Conn) Close(code int, reason string) error {
	data := make([]byte, 2, 2+len(reason))
	binary.BigEndian.PutUint16(data, uint16(code))
	return c.write(true, opClose, append(data, reason...))
}

// End drops the connection without a close message.
func (c *Conn) End() error {
	var err error
	c.once.Do(func() { err = c.conn.Close() })
	return err
}

func (c *Conn) write(fin bool, op opcode, data []byte) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err := writeFrame(c.w, fin, op, data, !c.server); err != nil {
		return err
	}
	return c.w.Flush()
}

// opcode is a frame's opcode.
type opcode byte

const (
	opContinuation opcode = 0x0
	opText         opcode = 0x1
	opBinary       opcode = 0x2
	opClose        opcode = 0x8
	opPing         opcode = 0x9
	opPong         opcode = 0xA
)

func (o opcode) String() string {
	switch o {
	case opContinuation:
		return "continuation"
	case opText:
		return "text"
	case opBinary:
		return "binary"
	case opClose:
		return "close"
	case opPing:
		return "ping"
	case opPong:
		return "pong"
	}
	return fmt.Sprintf("opcode %d", byte(o))
}

// frame is one frame, unmasked.
type frame struct {
	fin  bool
	op   opcode
	data []byte
}

// readFrame reads one frame.
func readFrame(r *bufio.Reader) (frame, error) {
	var head [2]byte
	if _, err := io.ReadFull(r, head[:]); err != nil {
		return frame{}, err
	}
	f := frame{fin: head[0]&0x80 != 0, op: opcode(head[0] & 0x0F)}
	masked := head[1]&0x80 != 0
	n := uint64(head[1] & 0x7F)
	switch n {
	case 126:
		var b [2]byte
		if _, err := io.ReadFull(r, b[:]); err != nil {
			return f, err
		}
		n = uint64(binary.BigEndian.Uint16(b[:]))
	case 127:
		var b [8]byte
		if _, err := io.ReadFull(r, b[:]); err != nil {
			return f, err
		}
		n = binary.BigEndian.Uint64(b[:])
	}
	if n > MaxMessage {
		return f, fmt.Errorf("a frame of %d bytes is over %d", n, MaxMessage)
	}
	var mask [4]byte
	if masked {
		if _, err := io.ReadFull(r, mask[:]); err != nil {
			return f, err
		}
	}
	f.data = make([]byte, n)
	if _, err := io.ReadFull(r, f.data); err != nil {
		return f, err
	}
	if masked {
		for i := range f.data {
			f.data[i] ^= mask[i%4]
		}
	}
	return f, nil
}

// writeFrame writes one frame; a client masks every frame it sends.
func writeFrame(w *bufio.Writer, fin bool, op opcode, data []byte, mask bool) error {
	b0 := byte(op)
	if fin {
		b0 |= 0x80
	}
	head := []byte{b0, 0}
	n := len(data)
	switch {
	case n < 126:
		head[1] = byte(n)
	case n <= 0xFFFF:
		head[1] = 126
		head = binary.BigEndian.AppendUint16(head, uint16(n))
	default:
		head[1] = 127
		head = binary.BigEndian.AppendUint64(head, uint64(n))
	}
	payload := data
	if mask {
		head[1] |= 0x80
		var key [4]byte
		rand.Read(key[:])
		head = append(head, key[:]...)
		payload = make([]byte, n)
		for i := range data {
			payload[i] = data[i] ^ key[i%4]
		}
	}
	if _, err := w.Write(head); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}
