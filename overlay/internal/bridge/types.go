// Package bridge is the app's client for `stickypane bridge`: it runs the
// program as a child, speaks protocol 1 with it over standard input and
// output (one JSON object a line), and starts it again when it dies. The
// types mirror the main module's internal/bridge, which this module cannot
// import.
package bridge

import (
	"encoding/json"
	"errors"
)

// Protocol is the version of the messages this client speaks.
const Protocol = 1

// The codes an error answer carries.
const (
	CodeBadRequest = "bad_request"
	CodeNotFound   = "not_found"
	CodeStale      = "stale"
	CodeFailed     = "failed"
)

// MethodRestarted is the notification the client itself sends after it
// started a dead bridge again: whatever the screen holds may be out of
// date, so it loads the board again.
const MethodRestarted = "bridge.restarted"

var (
	// ErrProtocol: the bridge speaks another version of the messages.
	ErrProtocol = errors.New("bridge: protocol mismatch")
	// ErrBridgeDown: the bridge died more often than it is started again.
	ErrBridgeDown = errors.New("bridge: down")
	// ErrConnectionLost: the bridge died while the call waited for its
	// answer. The call may or may not have been done; the next call goes to
	// the new bridge.
	ErrConnectionLost = errors.New("bridge: connection lost")
	// ErrClosed: the client was closed.
	ErrClosed = errors.New("bridge: closed")
	// ErrNotFound: no stickypane program was found.
	ErrNotFound = errors.New("bridge: stickypane not found")
)

// Request is one line to the bridge.
type Request struct {
	ID     int64           `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// Response is the answer to one request: a result or an error.
type Response struct {
	ID     json.RawMessage `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *Error          `json:"error,omitempty"`
}

// Notification is a line the bridge sends of its own accord (board.changed).
type Notification struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// Error is an error answer.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string { return e.Message }

// Hello is the answer to the hello method.
type Hello struct {
	Protocol int    `json:"protocol"`
	Version  string `json:"version"`
	Project  string `json:"project"`
}
