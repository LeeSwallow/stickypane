// Package prototest holds what the tests of proto and its users share: a
// descriptor written with the package's own encoder, so no protoc is needed.
package prototest

import . "github.com/LeeSwallow/stickypane/internal/proto"

// UsersFile is the descriptor protoc would write for:
//
//	syntax = "proto3";
//	package users.v1;
//	enum Role { ROLE_UNSPECIFIED = 0; ADMIN = 1; }
//	message GetRequest { int64 id = 1; }
//	message Address { string city = 1; }
//	message User {
//	  int64 id = 1; string name = 2; repeated string tags = 3; Role role = 4;
//	  map<string, int32> scores = 5; Address address = 6; repeated int32 lucky = 7;
//	  bool active = 8; double score = 9; sint32 delta = 10; bytes avatar = 11;
//	  string user_id = 12;
//	}
//	service Users { rpc Get(GetRequest) returns (User); rpc List(GetRequest) returns (stream User); }
func UsersFile() []byte {
	fieldDesc := func(name string, num int, kind Kind, repeated bool, typeName string) []byte {
		var b []byte
		b = AppendString(b, 1, name)
		b = AppendVarint(b, 3, uint64(num))
		label := uint64(1)
		if repeated {
			label = 3
		}
		b = AppendVarint(b, 4, label)
		b = AppendVarint(b, 5, uint64(kind))
		if typeName != "" {
			b = AppendString(b, 6, typeName)
		}
		return b
	}
	message := func(name string, fields ...[]byte) []byte {
		b := AppendString(nil, 1, name)
		for _, f := range fields {
			b = AppendBytes(b, 2, f)
		}
		return b
	}
	entry := message("ScoresEntry", fieldDesc("key", 1, StringKind, false, ""), fieldDesc("value", 2, Int32Kind, false, ""))
	entry = AppendBytes(entry, 7, AppendVarint(nil, 7, 1)) // map_entry
	user := message("User",
		fieldDesc("id", 1, Int64Kind, false, ""),
		fieldDesc("name", 2, StringKind, false, ""),
		fieldDesc("tags", 3, StringKind, true, ""),
		fieldDesc("role", 4, EnumKind, false, ".users.v1.Role"),
		fieldDesc("scores", 5, MessageKind, true, ".users.v1.User.ScoresEntry"),
		fieldDesc("address", 6, MessageKind, false, ".users.v1.Address"),
		fieldDesc("lucky", 7, Int32Kind, true, ""),
		fieldDesc("active", 8, BoolKind, false, ""),
		fieldDesc("score", 9, DoubleKind, false, ""),
		fieldDesc("delta", 10, Sint32Kind, false, ""),
		fieldDesc("avatar", 11, BytesKind, false, ""),
		fieldDesc("user_id", 12, StringKind, false, ""),
	)
	user = AppendBytes(user, 3, entry)
	value := func(name string, n int) []byte { return AppendVarint(AppendString(nil, 1, name), 2, uint64(n)) }
	role := AppendBytes(AppendBytes(AppendString(nil, 1, "Role"), 2, value("ROLE_UNSPECIFIED", 0)), 2, value("ADMIN", 1))
	method := func(name string, stream bool) []byte {
		b := AppendString(AppendString(AppendString(nil, 1, name), 2, ".users.v1.GetRequest"), 3, ".users.v1.User")
		if stream {
			b = AppendVarint(b, 6, 1)
		}
		return b
	}
	service := AppendBytes(AppendBytes(AppendString(nil, 1, "Users"), 2, method("Get", false)), 2, method("List", true))

	var f []byte
	f = AppendString(f, 1, "users.proto")
	f = AppendString(f, 2, "users.v1")
	f = AppendBytes(f, 4, message("GetRequest", fieldDesc("id", 1, Int64Kind, false, "")))
	f = AppendBytes(f, 4, message("Address", fieldDesc("city", 1, StringKind, false, "")))
	f = AppendBytes(f, 4, user)
	f = AppendBytes(f, 5, role)
	f = AppendBytes(f, 6, service)
	return f
}


// UsersSet is UsersFile as a .protoset: a FileDescriptorSet of one file.
func UsersSet() []byte { return AppendBytes(nil, 1, UsersFile()) }
