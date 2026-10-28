// Package proto reads and writes protocol buffers without generated code:
// the wire format (wire.go), the descriptors that describe messages
// (descriptor.go), and the mapping between a message and its JSON
// (json.go), with or without a descriptor. It is what a .http note's gRPC
// request needs to turn the JSON it is written in into the bytes a server
// reads, and back.
package proto

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// WireType is how a field's value is laid out.
type WireType int

// The wire types in use. Groups (3 and 4) are not read.
const (
	Varint  WireType = 0
	Fixed64 WireType = 1
	Bytes   WireType = 2
	Fixed32 WireType = 5
)

// Field is one field as read from the wire, before a descriptor says what
// it means.
type Field struct {
	Num  int
	Type WireType
	Int  uint64 // a varint, or the bits of a fixed32 or fixed64
	Data []byte // a length-delimited value
}

// ErrMalformed is what Fields returns for bytes that are not a message.
var ErrMalformed = errors.New("not a protocol buffer message")

// Fields splits a message into its fields, in the order they appear.
func Fields(b []byte) ([]Field, error) {
	var out []Field
	for len(b) > 0 {
		tag, n := binary.Uvarint(b)
		if n <= 0 || tag>>3 == 0 || tag>>3 > math.MaxInt32 {
			return out, ErrMalformed
		}
		b = b[n:]
		f := Field{Num: int(tag >> 3), Type: WireType(tag & 7)}
		switch f.Type {
		case Varint:
			v, n := binary.Uvarint(b)
			if n <= 0 {
				return out, ErrMalformed
			}
			f.Int, b = v, b[n:]
		case Fixed64:
			if len(b) < 8 {
				return out, ErrMalformed
			}
			f.Int, b = binary.LittleEndian.Uint64(b), b[8:]
		case Fixed32:
			if len(b) < 4 {
				return out, ErrMalformed
			}
			f.Int, b = uint64(binary.LittleEndian.Uint32(b)), b[4:]
		case Bytes:
			l, n := binary.Uvarint(b)
			if n <= 0 || l > uint64(len(b)-n) {
				return out, ErrMalformed
			}
			f.Data, b = b[n:n+int(l)], b[n+int(l):]
		default:
			return out, fmt.Errorf("%w: wire type %d", ErrMalformed, f.Type)
		}
		out = append(out, f)
	}
	return out, nil
}

// AppendTag appends a field's tag.
func AppendTag(b []byte, num int, t WireType) []byte {
	return binary.AppendUvarint(b, uint64(num)<<3|uint64(t))
}

// AppendVarint appends a varint field.
func AppendVarint(b []byte, num int, v uint64) []byte {
	return binary.AppendUvarint(AppendTag(b, num, Varint), v)
}

// AppendBytes appends a length-delimited field: a string, bytes or a
// message.
func AppendBytes(b []byte, num int, v []byte) []byte {
	b = binary.AppendUvarint(AppendTag(b, num, Bytes), uint64(len(v)))
	return append(b, v...)
}

// AppendString appends a string field.
func AppendString(b []byte, num int, s string) []byte { return AppendBytes(b, num, []byte(s)) }

// AppendFixed32 appends a fixed32 field.
func AppendFixed32(b []byte, num int, v uint32) []byte {
	return binary.LittleEndian.AppendUint32(AppendTag(b, num, Fixed32), v)
}

// AppendFixed64 appends a fixed64 field.
func AppendFixed64(b []byte, num int, v uint64) []byte {
	return binary.LittleEndian.AppendUint64(AppendTag(b, num, Fixed64), v)
}

// zigzag encodes a signed number for sint32 and sint64.
func zigzag(v int64) uint64 { return uint64(v<<1) ^ uint64(v>>63) }

// unzigzag decodes what zigzag encoded.
func unzigzag(v uint64) int64 { return int64(v>>1) ^ -int64(v&1) }
