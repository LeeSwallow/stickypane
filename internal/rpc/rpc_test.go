package rpc

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/LeeSwallow/stickypane/internal/proto/prototest"
	"github.com/LeeSwallow/stickypane/internal/rpc/rpctest"
)

func call(t *testing.T, c Call) Result {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := Invoke(ctx, c)
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	return res
}

func TestAUnaryCallWithADescriptorSpeaksJSON(t *testing.T) {
	target := rpctest.Server(t, false)
	res := call(t, Call{
		Target: target, Method: "users.v1.Users/Get", Plaintext: true,
		Metadata:   [][2]string{{"x-trace-id", "demo-1"}},
		Body:       []byte(`{"id": "7"}`),
		Descriptor: prototest.UsersSet(),
	})
	if res.Code != 0 || res.CodeName != "OK" || res.Schema != "descriptor" {
		t.Errorf("status %d %s, schema %s", res.Code, res.CodeName, res.Schema)
	}
	if len(res.Messages) != 1 || string(res.Messages[0]) != `{"id":"7","name":"min","role":"ADMIN"}` {
		t.Errorf("messages = %q", res.Messages)
	}
	if res.Header.Get("X-Got-Trace") != "demo-1" {
		t.Errorf("the metadata should reach the server: %v", res.Header)
	}
	if res.Streaming {
		t.Error("Get is unary")
	}
}

func TestReflectionFindsTheDescriptors(t *testing.T) {
	target := rpctest.Server(t, true)
	res := call(t, Call{Target: target, Method: "users.v1.Users/Get", Plaintext: true, Reflection: true, Body: []byte(`{"id": 9}`)})
	if res.Schema != "reflection" || string(res.Messages[0]) != `{"id":"9","name":"min","role":"ADMIN"}` {
		t.Errorf("schema %s, messages %q", res.Schema, res.Messages)
	}
}

func TestWithoutDescriptorsFieldsGoByNumber(t *testing.T) {
	target := rpctest.Server(t, false)
	res := call(t, Call{Target: target, Method: "users.v1.Users/Get", Plaintext: true, Reflection: true, Body: []byte(`{"1": 7}`)})
	if !strings.HasPrefix(res.Schema, "raw") || string(res.Messages[0]) != `{"1":7,"2":"min","4":1}` {
		t.Errorf("schema %s, messages %q", res.Schema, res.Messages)
	}
}

func TestAServerStreamGivesEveryMessage(t *testing.T) {
	target := rpctest.Server(t, false)
	res := call(t, Call{Target: target, Method: "users.v1.Users/List", Plaintext: true, Body: []byte(`{}`), Descriptor: prototest.UsersSet()})
	if !res.Streaming || len(res.Messages) != 3 || string(res.Messages[2]) != `{"id":"3","name":"cy","role":"ADMIN"}` {
		t.Errorf("streaming %v, messages %q", res.Streaming, res.Messages)
	}
}

func TestAnErrorStatusIsAResultNotAFailure(t *testing.T) {
	target := rpctest.Server(t, false)
	res := call(t, Call{Target: target, Method: "users.v1.Users/Get", Plaintext: true, Body: []byte(`{"id": "404"}`), Descriptor: prototest.UsersSet()})
	if res.Code != 5 || res.CodeName != "NOT_FOUND" || res.Message != "user 404 not found" || HTTPStatus(res.Code) != 404 {
		t.Errorf("status %d %s %q", res.Code, res.CodeName, res.Message)
	}
}

func TestMistakesAreSaidBeforeSending(t *testing.T) {
	target := rpctest.Server(t, false)
	for _, c := range []Call{
		{Target: target, Method: "users.v1.Users/Get", Plaintext: true, Body: []byte(`{"nme": 1}`), Descriptor: prototest.UsersSet()},
		{Target: target, Method: "users.v1.Users/Nope", Plaintext: true, Body: []byte(`{}`), Descriptor: prototest.UsersSet()},
		{Target: target, Method: "Get", Plaintext: true, Body: []byte(`{}`)},
		{Target: target, Method: "users.v1.Users/Get", Plaintext: true, Metadata: [][2]string{{"content-type", "x"}}, Body: []byte(`{}`)},
	} {
		if _, err := Invoke(context.Background(), c); err == nil {
			t.Errorf("%+v should fail", c)
		}
	}
}
