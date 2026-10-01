// Package mcp serves the notes folder to agents over the Model Context
// Protocol: JSON-RPC 2.0 messages, one per line, on standard input and output.
// It implements only what a tool server needs: initialize, ping, tools/list
// and tools/call.
package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/LeeSwallow/stickypane/internal/api"
)

// defaultProtocol is answered when the client does not name a version.
const defaultProtocol = "2025-06-18"

const (
	maxWait   = 600 // seconds read_answers waits at most
	waitEvery = 200 * time.Millisecond
)

// JSON-RPC error codes.
const (
	codeParse    = -32700
	codeNoMethod = -32601
	codeParams   = -32602
)

// Server answers MCP requests for one notes folder.
type Server struct {
	API     *api.API
	Guide   string // the agent guide, returned by the "guide" tool
	Version string
}

type request struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type toolResult struct {
	Content []content `json:"content"`
	IsError bool      `json:"isError"`
}

type tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

func object(required []string, props map[string]any) map[string]any {
	schema := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func str(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

// tools is what tools/list returns, in this order.
var tools = []tool{
	{
		Name:        "list_notes",
		Description: "List the notes on the user's stickypane board: file name, title, type, whether it is open on screen, size and pin.",
		InputSchema: object(nil, map[string]any{}),
	},
	{
		Name:        "read_note",
		Description: "Read one note's Markdown file, front matter included. Read a note before rewriting it: the user may have changed it from the board.",
		InputSchema: object([]string{"name"}, map[string]any{"name": str(`The note's file name, such as "plan" or "plan.md".`)}),
	},
	{
		Name:        "write_note",
		Description: "Create a note or replace its content. The content is Markdown and may start with front matter. How the user arranged an existing note (open, size, color, pin) is kept unless you set it. Call the guide tool for the formats of boards, checklists, logs, charts and forms.",
		InputSchema: object([]string{"name", "content"}, map[string]any{
			"name":    str(`The note's file name, such as "plan" or "plan.md". Prefix a number ("10-plan") to control the order.`),
			"content": str("The whole note as Markdown."),
			"type":    map[string]any{"type": "string", "enum": []string{"note", "board", "checklist", "log", "chart", "form"}, "description": "The note's shape. Omit for a plain note."},
			"title":   str("Shown in the title bar and the note's border."),
			"open":    map[string]any{"type": "boolean", "description": "true puts the note on the screen now."},
			"size":    map[string]any{"type": "string", "enum": []string{"page", "half", "card"}, "description": "page is the whole width, half is half of it, card is a small sticky note."},
		}),
	},
	{
		Name:        "read_answers",
		Description: "Read what the user chose and wrote in a form note (type: form) and which button they pressed. With wait_seconds it waits up to that long for a button to be pressed, then returns the answers so far.",
		InputSchema: object([]string{"name"}, map[string]any{
			"name":         str(`The form's file name, such as "deploy" or "deploy.md".`),
			"wait_seconds": map[string]any{"type": "number", "description": "How long to wait for a button, at most 600. Omit to read the answers now."},
		}),
	},
	{
		Name:        "guide",
		Description: "How to write notes for the board: the note shapes and their file formats.",
		InputSchema: object(nil, map[string]any{}),
	},
}

// Serve answers requests from in until it ends. A bad line gets an error
// response and does not stop the server.
func (s *Server) Serve(in io.Reader, out io.Writer) error {
	r := bufio.NewReader(in)
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	for {
		line, err := r.ReadBytes('\n')
		if len(bytes.TrimSpace(line)) > 0 {
			if resp := s.handle(line); resp != nil {
				if werr := enc.Encode(resp); werr != nil {
					return werr
				}
			}
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// handle answers one message. Notifications, which carry no id, get no answer.
func (s *Server) handle(line []byte) *response {
	var req request
	if err := json.Unmarshal(line, &req); err != nil {
		return &response{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &rpcError{codeParse, "the message is not valid JSON"}}
	}
	if req.ID == nil {
		return nil
	}
	resp := &response{JSONRPC: "2.0", ID: req.ID}
	switch req.Method {
	case "initialize":
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		_ = json.Unmarshal(req.Params, &p)
		if p.ProtocolVersion == "" {
			p.ProtocolVersion = defaultProtocol
		}
		resp.Result = map[string]any{
			"protocolVersion": p.ProtocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "stickypane", "version": s.Version},
		}
	case "ping":
		resp.Result = struct{}{}
	case "tools/list":
		resp.Result = map[string]any{"tools": tools}
	case "tools/call":
		result, rpcErr := s.call(req.Params)
		resp.Result, resp.Error = result, rpcErr
		if rpcErr != nil {
			resp.Result = nil
		}
	default:
		resp.Error = &rpcError{codeNoMethod, "unknown method " + req.Method}
	}
	return resp
}

// call runs a tool. A tool that fails returns a result marked isError, which
// the agent can read and act on; only an unknown tool or unreadable
// arguments is a protocol error.
func (s *Server) call(params json.RawMessage) (*toolResult, *rpcError) {
	var p struct {
		Name      string `json:"name"`
		Arguments struct {
			Name    string  `json:"name"`
			Content *string `json:"content"`
			Type    string  `json:"type"`
			Title   string  `json:"title"`
			Size    string  `json:"size"`
			Open    bool    `json:"open"`
			Wait    float64 `json:"wait_seconds"`
		} `json:"arguments"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &rpcError{codeParams, "cannot read the tool arguments: " + err.Error()}
	}
	text, err := "", error(nil)
	switch a := p.Arguments; p.Name {
	case "list_notes":
		infos, lerr := s.API.List()
		if err = lerr; err == nil {
			b, _ := json.Marshal(infos)
			text = string(b)
		}
	case "read_note":
		b, rerr := s.API.Show(a.Name)
		text, err = string(b), rerr
	case "write_note":
		if a.Content == nil {
			// Without this check a call that forgot its content would
			// replace the note with nothing and report success.
			err = errors.New("write_note needs content: the whole note as Markdown")
			break
		}
		file, werr := s.API.Write(a.Name, api.Options{Type: a.Type, Title: a.Title, Size: a.Size, Open: a.Open}, []byte(*a.Content))
		text, err = "wrote "+file, werr
	case "read_answers":
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(min(max(a.Wait, 0), maxWait)*float64(time.Second)))
		got, aerr := s.API.Wait(ctx, a.Name, waitEvery)
		cancel()
		if errors.Is(aerr, context.DeadlineExceeded) {
			aerr = nil // the answers so far are the result
		}
		text, err = got.String(), aerr
	case "guide":
		text = s.Guide
	default:
		return nil, &rpcError{codeParams, "unknown tool " + p.Name}
	}
	if err != nil {
		return &toolResult{Content: []content{{Type: "text", Text: err.Error()}}, IsError: true}, nil
	}
	return &toolResult{Content: []content{{Type: "text", Text: text}}}, nil
}
