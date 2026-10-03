package proto_test

import (
	"encoding/json"
	"strings"
	"testing"

	. "github.com/LeeSwallow/stickypane/internal/proto"
	"github.com/LeeSwallow/stickypane/internal/proto/prototest"
)

func usersSchema(t *testing.T) *Schema {
	t.Helper()
	s := NewSchema()
	if err := s.AddSet(prototest.UsersSet()); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestADescriptorSetDescribesMessagesEnumsAndServices(t *testing.T) {
	s := usersSchema(t)
	u := s.Messages["users.v1.User"]
	if u == nil || len(u.Fields) != 12 || u.Fields[11].Name != "user_id" || u.Fields[11].JSONName != "userId" {
		t.Fatalf("User = %+v", u)
	}
	if e := s.Messages["users.v1.User.ScoresEntry"]; e == nil || !e.MapEntry {
		t.Errorf("a map's entry message is marked: %+v", e)
	}
	if s.Enums["users.v1.Role"].ByNum[1] != "ADMIN" {
		t.Errorf("Role = %+v", s.Enums["users.v1.Role"])
	}
	m, err := s.Method("users.v1.Users/List")
	if err != nil || m.Input != "users.v1.GetRequest" || m.Output != "users.v1.User" || !m.ServerStream {
		t.Errorf("List = %+v, %v", m, err)
	}
	if _, err := s.Method("/users.v1.Users.Get"); err != nil {
		t.Errorf("a dotted method name is read too: %v", err)
	}
	if _, err := s.Method("users.v1.Users/Delete"); err == nil || !strings.Contains(err.Error(), "no method Delete") {
		t.Errorf("err = %v", err)
	}
}

func TestJSONBecomesTheBytesProtocWouldWrite(t *testing.T) {
	s := usersSchema(t)
	b, err := s.Encode("users.v1.GetRequest", []byte(`{"id": "7"}`))
	if err != nil || string(b) != string(AppendVarint(nil, 1, 7)) {
		t.Errorf("Encode = %x, %v", b, err)
	}
	if b, err := s.Encode("users.v1.GetRequest", []byte(`{"id": 7}`)); err != nil || string(b) != string(AppendVarint(nil, 1, 7)) {
		t.Errorf("an int64 may be written as a number: %x, %v", b, err)
	}
}

func TestAMessageRoundTripsThroughJSON(t *testing.T) {
	s := usersSchema(t)
	in := `{"id":"7","name":"min","tags":["a","b"],"role":"ADMIN","scores":{"go":3},"address":{"city":"Seoul"},"lucky":[3,7],"active":true,"score":1.5,"delta":-2,"avatar":"aGk=","userId":"u-7"}`
	b, err := s.Encode("users.v1.User", []byte(in))
	if err != nil {
		t.Fatal(err)
	}
	out, err := s.Decode("users.v1.User", b)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != in {
		t.Errorf("round trip:\n got %s\nwant %s", out, in)
	}
	// proto3 leaves out what is at its default.
	if out, _ := s.Decode("users.v1.User", nil); string(out) != "{}" {
		t.Errorf("an empty message = %s", out)
	}
}

func TestUnknownNamesAndWrongTypesAreSaidPlainly(t *testing.T) {
	s := usersSchema(t)
	if _, err := s.Encode("users.v1.User", []byte(`{"nme":"min"}`)); err == nil || !strings.Contains(err.Error(), `no field "nme"`) || !strings.Contains(err.Error(), "name") {
		t.Errorf("err = %v", err)
	}
	if _, err := s.Encode("users.v1.User", []byte(`{"active":"yes"}`)); err == nil || !strings.Contains(err.Error(), "active") {
		t.Errorf("err = %v", err)
	}
	if _, err := s.Encode("users.v1.User", []byte(`{"role":"OWNER"}`)); err == nil || !strings.Contains(err.Error(), "ADMIN") {
		t.Errorf("an unknown enum value lists the ones there are: %v", err)
	}
	if _, err := s.Encode("users.v1.Nope", []byte(`{}`)); err == nil {
		t.Error("an unknown message is an error")
	}
}

func TestPackedAndUnpackedRepeatedFieldsBothRead(t *testing.T) {
	s := usersSchema(t)
	unpacked := AppendVarint(AppendVarint(nil, 7, 3), 7, 7)
	if out, err := s.Decode("users.v1.User", unpacked); err != nil || string(out) != `{"lucky":[3,7]}` {
		t.Errorf("unpacked = %s, %v", out, err)
	}
}

func TestWithoutADescriptorFieldsGoByNumber(t *testing.T) {
	in := `{"1":7,"2":"min","3":{"1":"Seoul"},"4":[1,2]}`
	b, err := EncodeRaw([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	out, err := DecodeRaw(b)
	if err != nil {
		t.Fatal(err)
	}
	var got, want any
	json.Unmarshal(out, &got)
	json.Unmarshal([]byte(`{"1":7,"2":"min","3":{"1":"Seoul"},"4":[1,2]}`), &want)
	if g, w := mustJSON(got), mustJSON(want); g != w {
		t.Errorf("raw round trip:\n got %s\nwant %s", out, w)
	}
	if _, err := EncodeRaw([]byte(`{"id":7}`)); err == nil || !strings.Contains(err.Error(), "number") {
		t.Errorf("a raw message is keyed by field numbers: %v", err)
	}
}

func mustJSON(v any) string { b, _ := json.Marshal(v); return string(b) }
