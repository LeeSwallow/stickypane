// Command fakebridge speaks the bridge protocol for the client's tests.
//
// Environment:
//
//	FAKE_PROTOCOL   the protocol hello reports (default 1)
//	FAKE_STARTS     a file; one byte is appended at each start
//	FAKE_DIE_UNTIL  die at the first echo while the file FAKE_STARTS
//	                holds at most this many bytes
//
// Methods besides hello: echo (answers its params), fail (an error
// answer), emit (answers, then sends a board.changed notification).
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

type req struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

func main() {
	proto := 1
	if s := os.Getenv("FAKE_PROTOCOL"); s != "" {
		proto, _ = strconv.Atoi(s)
	}
	starts := 0
	if p := os.Getenv("FAKE_STARTS"); p != "" {
		f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0o644)
		if err == nil {
			f.Write([]byte{'x'})
			if fi, err := f.Stat(); err == nil {
				starts = int(fi.Size())
			}
			f.Close()
		}
	}
	dieUntil, _ := strconv.Atoi(os.Getenv("FAKE_DIE_UNTIL"))
	wd, _ := os.Getwd()
	out := bufio.NewWriter(os.Stdout)
	send := func(v any) {
		b, _ := json.Marshal(v)
		out.Write(append(b, '\n'))
		out.Flush()
	}
	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(nil, 1<<20)
	for sc.Scan() {
		var r req
		if json.Unmarshal(sc.Bytes(), &r) != nil {
			send(map[string]any{"id": nil, "error": map[string]string{"code": "bad_request", "message": "not json"}})
			continue
		}
		switch r.Method {
		case "hello":
			send(map[string]any{"id": r.ID, "result": map[string]any{"protocol": proto, "version": "fake", "project": wd}})
		case "echo":
			if dieUntil > 0 && starts <= dieUntil {
				os.Exit(3)
			}
			result := r.Params
			if len(result) == 0 {
				result = json.RawMessage("{}")
			}
			send(map[string]any{"id": r.ID, "result": result})
		case "fail":
			send(map[string]any{"id": r.ID, "error": map[string]string{"code": "stale", "message": "the note changed"}})
		case "emit":
			send(map[string]any{"id": r.ID, "result": map[string]any{}})
			send(map[string]any{"method": "board.changed", "params": map[string]any{"removed": []string{"a.md"}}})
		default:
			send(map[string]any{"id": r.ID, "error": map[string]string{"code": "bad_request", "message": fmt.Sprintf("no method %q", r.Method)}})
		}
	}
}
