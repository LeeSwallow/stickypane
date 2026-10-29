// Command fakebridge speaks the bridge protocol for the client's tests.
//
// Environment:
//
//	FAKE_PROTOCOL   the protocol hello reports (default 1)
//	FAKE_STARTS     a file; one byte is appended at each start
//	FAKE_PROTO_FROM from this start on (1-based) hello reports protocol 2
//	FAKE_GRANDCHILD set: start a long-lived child that holds stdout
//	FAKE_DIE_UNTIL  die at the first echo while the file FAKE_STARTS
//	                holds at most this many bytes
//
// Methods besides hello: echo (answers its params), fail (an error
// answer), emit (answers, then sends a board.changed notification),
// flood (params {"n":N}: N notifications, then the answer), stall (stops
// reading input, for ever), pause (params {"ms":N}: answers, then reads
// nothing for N ms), sink (answers {}, whatever its params), lines (answers
// {"lines":L,"bad":B}: the lines read so far, and those not JSON).
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"
)

type req struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "sleep" {
		time.Sleep(20 * time.Second)
		return
	}
	if os.Getenv("FAKE_GRANDCHILD") != "" {
		c := exec.Command(os.Args[0], "sleep")
		c.Env = append(os.Environ(), "FAKE_GRANDCHILD=")
		c.Stdout, c.Stderr = os.Stdout, os.Stderr
		c.Start()
	}
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
	if from, _ := strconv.Atoi(os.Getenv("FAKE_PROTO_FROM")); from > 0 && starts >= from {
		proto = 2
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
	sc.Buffer(nil, 64<<20)
	lines, bad := 0, 0
	for sc.Scan() {
		lines++
		var r req
		if json.Unmarshal(sc.Bytes(), &r) != nil {
			bad++
			send(map[string]any{"id": nil, "error": map[string]string{"code": "bad_request", "message": "not json"}})
			continue
		}
		switch r.Method {
		case "pause":
			var p struct{ MS int }
			json.Unmarshal(r.Params, &p)
			send(map[string]any{"id": r.ID, "result": map[string]any{}})
			time.Sleep(time.Duration(p.MS) * time.Millisecond)
		case "sink":
			send(map[string]any{"id": r.ID, "result": map[string]any{}})
		case "lines":
			send(map[string]any{"id": r.ID, "result": map[string]int{"lines": lines, "bad": bad}})
		case "hello":
			send(map[string]any{"id": r.ID, "result": map[string]any{"protocol": proto, "version": "fake", "project": wd}})
		case "echo":
			if dieUntil > 0 && starts <= dieUntil {
				fmt.Fprintln(os.Stderr, "boom: the fake died")
				os.Exit(3)
			}
			result := r.Params
			if len(result) == 0 {
				result = json.RawMessage("{}")
			}
			send(map[string]any{"id": r.ID, "result": result})
		case "stall":
			time.Sleep(time.Hour)
		case "flood":
			var p struct{ N int }
			json.Unmarshal(r.Params, &p)
			for i := 0; i < p.N; i++ {
				b, _ := json.Marshal(map[string]any{"method": "board.changed", "params": map[string]any{"i": i}})
				out.Write(append(b, '\n'))
			}
			out.Flush()
			send(map[string]any{"id": r.ID, "result": map[string]any{}})
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
