package proto

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Encode turns a message written as protobuf JSON into its bytes, using
// the message's descriptor: field names or JSON names, int64 as a string or
// a number, enums by name or number, bytes in base64, maps as objects.
func (s *Schema) Encode(message string, data []byte) ([]byte, error) {
	m, ok := s.Messages[message]
	if !ok {
		return nil, fmt.Errorf("no message %s in the descriptors", message)
	}
	v, err := decodeJSON(data)
	if err != nil {
		return nil, err
	}
	obj, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s is written as a JSON object, {...}", message)
	}
	return s.encodeMessage(m, obj)
}

func decodeJSON(data []byte) (any, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return map[string]any{}, nil
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	var v any
	if err := d.Decode(&v); err != nil {
		return nil, fmt.Errorf("the body is not JSON: %w", err)
	}
	return v, nil
}

func (s *Schema) encodeMessage(m *Message, obj map[string]any) ([]byte, error) {
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := m.byName[keys[i]], m.byName[keys[j]]
		if a != nil && b != nil {
			return a.Num < b.Num
		}
		return keys[i] < keys[j]
	})
	var out []byte
	for _, k := range keys {
		fd := m.byName[k]
		if fd == nil {
			names := make([]string, len(m.Fields))
			for i, f := range m.Fields {
				names[i] = f.JSONName
			}
			return nil, fmt.Errorf("%s has no field %q; it has %s", m.Name, k, strings.Join(names, ", "))
		}
		v := obj[k]
		if v == nil {
			continue
		}
		var err error
		if out, err = s.encodeField(out, fd, v); err != nil {
			return nil, fmt.Errorf("%s.%s: %w", m.Name, fd.JSONName, err)
		}
	}
	return out, nil
}

func (s *Schema) encodeField(out []byte, fd *FieldDesc, v any) ([]byte, error) {
	if entry := s.Messages[fd.TypeName]; fd.Kind == MessageKind && entry != nil && entry.MapEntry {
		obj, ok := v.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("a map is written as a JSON object")
		}
		keys := make([]string, 0, len(obj))
		for k := range obj {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			var e []byte
			var err error
			if e, err = s.encodeValue(e, entry.byNum[1], k, true); err != nil {
				return nil, err
			}
			if e, err = s.encodeValue(e, entry.byNum[2], obj[k], false); err != nil {
				return nil, err
			}
			out = AppendBytes(out, fd.Num, e)
		}
		return out, nil
	}
	if !fd.Repeated {
		return s.encodeValue(out, fd, v, false)
	}
	list, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("a repeated field is written as a JSON array")
	}
	if packable(fd.Kind) {
		var packed []byte
		for _, item := range list {
			var err error
			// Encode as a field, then drop the tag: packed values are bare.
			one, err := s.encodeValue(nil, &FieldDesc{Num: 1, Kind: fd.Kind, TypeName: fd.TypeName}, item, false)
			if err != nil {
				return nil, err
			}
			packed = append(packed, one[1:]...)
		}
		return AppendBytes(out, fd.Num, packed), nil
	}
	for _, item := range list {
		var err error
		if out, err = s.encodeValue(out, fd, item, false); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// packable reports whether repeated values of a kind are packed, as proto3
// writes them.
func packable(k Kind) bool {
	return k != StringKind && k != BytesKind && k != MessageKind && k != GroupKind
}

// encodeValue appends one value. key says it comes from a map's key, which
// JSON always writes as a string.
func (s *Schema) encodeValue(out []byte, fd *FieldDesc, v any, key bool) ([]byte, error) {
	switch fd.Kind {
	case StringKind:
		str, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("want a string, got %s", show(v))
		}
		return AppendString(out, fd.Num, str), nil
	case BytesKind:
		str, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("want base64 text, got %s", show(v))
		}
		b, err := base64.StdEncoding.DecodeString(str)
		if err != nil {
			if b, err = base64.URLEncoding.DecodeString(str); err != nil {
				return nil, fmt.Errorf("%q is not base64", str)
			}
		}
		return AppendBytes(out, fd.Num, b), nil
	case BoolKind:
		b, ok := v.(bool)
		if !ok && key {
			b, ok = v == "true", v == "true" || v == "false"
		}
		if !ok {
			return nil, fmt.Errorf("want true or false, got %s", show(v))
		}
		return AppendVarint(out, fd.Num, map[bool]uint64{false: 0, true: 1}[b]), nil
	case EnumKind:
		e := s.Enums[fd.TypeName]
		if name, ok := v.(string); ok && e != nil {
			n, ok := e.ByName[name]
			if !ok {
				names := make([]string, 0, len(e.ByName))
				for k := range e.ByName {
					names = append(names, k)
				}
				sort.Strings(names)
				return nil, fmt.Errorf("%s has no value %q; it has %s", e.Name, name, strings.Join(names, ", "))
			}
			return AppendVarint(out, fd.Num, uint64(int64(n))), nil
		}
		n, err := integer(v)
		if err != nil {
			return nil, err
		}
		return AppendVarint(out, fd.Num, uint64(n)), nil
	case MessageKind, GroupKind:
		obj, ok := v.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("want an object, got %s", show(v))
		}
		m := s.Messages[fd.TypeName]
		if m == nil {
			return nil, fmt.Errorf("no message %s in the descriptors", fd.TypeName)
		}
		b, err := s.encodeMessage(m, obj)
		if err != nil {
			return nil, err
		}
		return AppendBytes(out, fd.Num, b), nil
	case DoubleKind, FloatKind:
		f, err := float(v)
		if err != nil {
			return nil, err
		}
		if fd.Kind == FloatKind {
			return AppendFixed32(out, fd.Num, math.Float32bits(float32(f))), nil
		}
		return AppendFixed64(out, fd.Num, math.Float64bits(f)), nil
	}
	if fd.Kind == Uint64Kind || fd.Kind == Fixed64Kind || fd.Kind == Uint32Kind || fd.Kind == Fixed32Kind {
		u, err := unsigned(v)
		if err != nil {
			return nil, err
		}
		switch fd.Kind {
		case Fixed64Kind:
			return AppendFixed64(out, fd.Num, u), nil
		case Fixed32Kind:
			return AppendFixed32(out, fd.Num, uint32(u)), nil
		}
		return AppendVarint(out, fd.Num, u), nil
	}
	n, err := integer(v)
	if err != nil {
		return nil, err
	}
	switch fd.Kind {
	case Sint32Kind, Sint64Kind:
		return AppendVarint(out, fd.Num, zigzag(n)), nil
	case Sfixed32Kind:
		return AppendFixed32(out, fd.Num, uint32(int32(n))), nil
	case Sfixed64Kind:
		return AppendFixed64(out, fd.Num, uint64(n)), nil
	}
	return AppendVarint(out, fd.Num, uint64(n)), nil // int32 and int64; negatives take ten bytes
}

func show(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// integer reads a whole number written as a number or a string.
func integer(v any) (int64, error) {
	switch v := v.(type) {
	case json.Number:
		return strconv.ParseInt(v.String(), 10, 64)
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("want a whole number, got %q", v)
		}
		return n, nil
	}
	return 0, fmt.Errorf("want a whole number, got %s", show(v))
}

func unsigned(v any) (uint64, error) {
	s := ""
	switch v := v.(type) {
	case json.Number:
		s = v.String()
	case string:
		s = v
	default:
		return 0, fmt.Errorf("want a whole number, got %s", show(v))
	}
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("want a positive whole number, got %s", s)
	}
	return n, nil
}

func float(v any) (float64, error) {
	switch v := v.(type) {
	case json.Number:
		return v.Float64()
	case string:
		switch v {
		case "NaN":
			return math.NaN(), nil
		case "Infinity":
			return math.Inf(1), nil
		case "-Infinity":
			return math.Inf(-1), nil
		}
		return strconv.ParseFloat(v, 64)
	}
	return 0, fmt.Errorf("want a number, got %s", show(v))
}

// Decode turns a message's bytes into protobuf JSON, fields in the order
// of their numbers. Fields the descriptor does not know are kept under
// their number, so nothing is hidden.
func (s *Schema) Decode(message string, b []byte) ([]byte, error) {
	m, ok := s.Messages[message]
	if !ok {
		return nil, fmt.Errorf("no message %s in the descriptors", message)
	}
	var out bytes.Buffer
	if err := s.decodeMessage(&out, m, b); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func (s *Schema) decodeMessage(out *bytes.Buffer, m *Message, b []byte) error {
	fields, err := Fields(b)
	if err != nil {
		return err
	}
	byNum := map[int][]Field{}
	var unknown []int
	for _, f := range fields {
		if m.byNum[f.Num] == nil && len(byNum[f.Num]) == 0 {
			unknown = append(unknown, f.Num)
		}
		byNum[f.Num] = append(byNum[f.Num], f)
	}
	descs := slices.Clone(m.Fields)
	sort.Slice(descs, func(i, j int) bool { return descs[i].Num < descs[j].Num })
	out.WriteByte('{')
	first := true
	sep := func(name string) {
		if !first {
			out.WriteByte(',')
		}
		first = false
		quoted, _ := json.Marshal(name)
		out.Write(quoted)
		out.WriteByte(':')
	}
	for _, fd := range descs {
		got := byNum[fd.Num]
		if len(got) == 0 {
			continue
		}
		sep(fd.JSONName)
		if err := s.decodeField(out, fd, got); err != nil {
			return fmt.Errorf("%s.%s: %w", m.Name, fd.JSONName, err)
		}
	}
	for _, n := range unknown {
		sep(strconv.Itoa(n))
		rawValues(out, byNum[n])
	}
	out.WriteByte('}')
	return nil
}

func (s *Schema) decodeField(out *bytes.Buffer, fd *FieldDesc, got []Field) error {
	if entry := s.Messages[fd.TypeName]; fd.Kind == MessageKind && entry != nil && entry.MapEntry {
		out.WriteByte('{')
		for i, f := range got {
			parts, err := Fields(f.Data)
			if err != nil {
				return err
			}
			var k, v bytes.Buffer
			for _, p := range parts {
				switch p.Num {
				case 1:
					s.decodeValue(&k, entry.byNum[1], p)
				case 2:
					s.decodeValue(&v, entry.byNum[2], p)
				}
			}
			key := k.String()
			if !strings.HasPrefix(key, `"`) {
				key = strconv.Quote(key)
			}
			if v.Len() == 0 {
				v.WriteString("null")
			}
			if i > 0 {
				out.WriteByte(',')
			}
			out.WriteString(key + ":" + v.String())
		}
		out.WriteByte('}')
		return nil
	}
	// Packed scalars come as one length-delimited field: open them up.
	var values []Field
	for _, f := range got {
		if f.Type == Bytes && packable(fd.Kind) {
			values = append(values, unpack(fd.Kind, f.Data)...)
			continue
		}
		values = append(values, f)
	}
	if !fd.Repeated {
		return s.decodeValue(out, fd, values[len(values)-1])
	}
	out.WriteByte('[')
	for i, f := range values {
		if i > 0 {
			out.WriteByte(',')
		}
		if err := s.decodeValue(out, fd, f); err != nil {
			return err
		}
	}
	out.WriteByte(']')
	return nil
}

// unpack reads the values of a packed field.
func unpack(k Kind, b []byte) []Field {
	var out []Field
	for len(b) > 0 {
		switch k {
		case DoubleKind, Fixed64Kind, Sfixed64Kind:
			if len(b) < 8 {
				return out
			}
			out, b = append(out, Field{Type: Fixed64, Int: binary.LittleEndian.Uint64(b)}), b[8:]
		case FloatKind, Fixed32Kind, Sfixed32Kind:
			if len(b) < 4 {
				return out
			}
			out, b = append(out, Field{Type: Fixed32, Int: uint64(binary.LittleEndian.Uint32(b))}), b[4:]
		default:
			v, n := binary.Uvarint(b)
			if n <= 0 {
				return out
			}
			out, b = append(out, Field{Type: Varint, Int: v}), b[n:]
		}
	}
	return out
}

func (s *Schema) decodeValue(out *bytes.Buffer, fd *FieldDesc, f Field) error {
	if fd == nil {
		rawValues(out, []Field{f})
		return nil
	}
	q := func(v string) { b, _ := json.Marshal(v); out.Write(b) }
	switch fd.Kind {
	case StringKind:
		q(string(f.Data))
	case BytesKind:
		q(base64.StdEncoding.EncodeToString(f.Data))
	case MessageKind, GroupKind:
		m := s.Messages[fd.TypeName]
		if m == nil {
			rawValues(out, []Field{f})
			return nil
		}
		return s.decodeMessage(out, m, f.Data)
	case BoolKind:
		out.WriteString(strconv.FormatBool(f.Int != 0))
	case EnumKind:
		if e := s.Enums[fd.TypeName]; e != nil {
			if name, ok := e.ByNum[int32(f.Int)]; ok {
				q(name)
				return nil
			}
		}
		out.WriteString(strconv.FormatInt(int64(int32(f.Int)), 10))
	case DoubleKind:
		number(out, math.Float64frombits(f.Int))
	case FloatKind:
		number(out, float64(math.Float32frombits(uint32(f.Int))))
	case Int64Kind, Sfixed64Kind:
		q(strconv.FormatInt(int64(f.Int), 10))
	case Uint64Kind, Fixed64Kind:
		q(strconv.FormatUint(f.Int, 10))
	case Sint64Kind:
		q(strconv.FormatInt(unzigzag(f.Int), 10))
	case Sint32Kind:
		out.WriteString(strconv.FormatInt(unzigzag(f.Int), 10))
	case Uint32Kind, Fixed32Kind:
		out.WriteString(strconv.FormatUint(uint64(uint32(f.Int)), 10))
	default: // int32, sfixed32
		out.WriteString(strconv.FormatInt(int64(int32(f.Int)), 10))
	}
	return nil
}

func number(out *bytes.Buffer, f float64) {
	switch {
	case math.IsNaN(f):
		out.WriteString(`"NaN"`)
	case math.IsInf(f, 1):
		out.WriteString(`"Infinity"`)
	case math.IsInf(f, -1):
		out.WriteString(`"-Infinity"`)
	default:
		out.WriteString(strconv.FormatFloat(f, 'g', -1, 64))
	}
}

// EncodeRaw turns JSON keyed by field numbers into a message, for a server
// whose descriptors are not at hand: {"1": 7, "2": "min", "3": {"1": "x"}}.
// Whole numbers are varints, other numbers doubles, strings and objects
// length-delimited, arrays repeated fields.
func EncodeRaw(data []byte) ([]byte, error) {
	v, err := decodeJSON(data)
	if err != nil {
		return nil, err
	}
	obj, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("the message is written as a JSON object, {...}")
	}
	return encodeRaw(obj)
}

func encodeRaw(obj map[string]any) ([]byte, error) {
	nums := make([]int, 0, len(obj))
	for k := range obj {
		n, err := strconv.Atoi(k)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("without descriptors a field is named by its number, not %q: add # @grpc-descriptor or turn on server reflection", k)
		}
		nums = append(nums, n)
	}
	sort.Ints(nums)
	var out []byte
	for _, n := range nums {
		v := obj[strconv.Itoa(n)]
		items := []any{v}
		if list, ok := v.([]any); ok {
			items = list
		}
		for _, item := range items {
			var err error
			if out, err = rawValue(out, n, item); err != nil {
				return nil, fmt.Errorf("field %d: %w", n, err)
			}
		}
	}
	return out, nil
}

func rawValue(out []byte, n int, v any) ([]byte, error) {
	switch v := v.(type) {
	case nil:
		return out, nil
	case bool:
		return AppendVarint(out, n, map[bool]uint64{false: 0, true: 1}[v]), nil
	case string:
		return AppendString(out, n, v), nil
	case json.Number:
		if i, err := strconv.ParseInt(v.String(), 10, 64); err == nil {
			return AppendVarint(out, n, uint64(i)), nil
		}
		f, err := v.Float64()
		if err != nil {
			return nil, err
		}
		return AppendFixed64(out, n, math.Float64bits(f)), nil
	case map[string]any:
		b, err := encodeRaw(v)
		if err != nil {
			return nil, err
		}
		return AppendBytes(out, n, b), nil
	}
	return nil, fmt.Errorf("cannot write %s without a descriptor", show(v))
}

// DecodeRaw turns a message into JSON keyed by field numbers, as protoc
// --decode_raw does: a length-delimited value is text when it reads as
// text, a message when it parses as one, and base64 otherwise.
func DecodeRaw(b []byte) ([]byte, error) {
	fields, err := Fields(b)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	rawMessage(&out, fields)
	return out.Bytes(), nil
}

func rawMessage(out *bytes.Buffer, fields []Field) {
	byNum := map[int][]Field{}
	var order []int
	for _, f := range fields {
		if len(byNum[f.Num]) == 0 {
			order = append(order, f.Num)
		}
		byNum[f.Num] = append(byNum[f.Num], f)
	}
	out.WriteByte('{')
	for i, n := range order {
		if i > 0 {
			out.WriteByte(',')
		}
		fmt.Fprintf(out, `"%d":`, n)
		rawValues(out, byNum[n])
	}
	out.WriteByte('}')
}

func rawValues(out *bytes.Buffer, fs []Field) {
	if len(fs) > 1 {
		out.WriteByte('[')
	}
	for i, f := range fs {
		if i > 0 {
			out.WriteByte(',')
		}
		switch f.Type {
		case Varint:
			out.WriteString(strconv.FormatUint(f.Int, 10))
		case Fixed64:
			number(out, math.Float64frombits(f.Int))
		case Fixed32:
			number(out, float64(math.Float32frombits(uint32(f.Int))))
		case Bytes:
			if printable(f.Data) {
				b, _ := json.Marshal(string(f.Data))
				out.Write(b)
			} else if inner, err := Fields(f.Data); err == nil && len(inner) > 0 {
				rawMessage(out, inner)
			} else {
				b, _ := json.Marshal(base64.StdEncoding.EncodeToString(f.Data))
				out.Write(b)
			}
		}
	}
	if len(fs) > 1 {
		out.WriteByte(']')
	}
}

// printable reports whether bytes read as text a person wrote.
func printable(b []byte) bool {
	if !utf8.Valid(b) {
		return false
	}
	for _, r := range string(b) {
		if !unicode.IsPrint(r) && r != '\n' && r != '\t' {
			return false
		}
	}
	return true
}
