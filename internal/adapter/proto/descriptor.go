package proto

import (
	"fmt"
	"strings"
)

// Schema is what a set of descriptors says: the messages, enums and
// services by their full names ("users.v1.User", without a leading dot).
type Schema struct {
	Messages map[string]*Message
	Enums    map[string]*Enum
	Services map[string]*Service
	files    map[string]bool
	deps     []string // the files named as dependencies, loaded or not
}

// Message describes a message.
type Message struct {
	Name     string
	Fields   []*FieldDesc
	MapEntry bool
	byNum    map[int]*FieldDesc
	byName   map[string]*FieldDesc
}

// FieldDesc describes a field of a message.
type FieldDesc struct {
	Name     string
	JSONName string
	Num      int
	Kind     Kind
	Repeated bool
	TypeName string // the message or enum, for Kind MessageKind and EnumKind
}

// Enum describes an enum: its values by number and by name.
type Enum struct {
	Name     string
	ByNum    map[int32]string
	ByName   map[string]int32
	FirstNum int32
}

// Service describes a service and its methods.
type Service struct {
	Name    string
	Methods map[string]*Method
}

// Method describes a method of a service.
type Method struct {
	Name, Input, Output string
	ClientStream        bool
	ServerStream        bool
}

// Kind is a field's type, as descriptor.proto numbers them.
type Kind int

// The field types.
const (
	DoubleKind   Kind = 1
	FloatKind    Kind = 2
	Int64Kind    Kind = 3
	Uint64Kind   Kind = 4
	Int32Kind    Kind = 5
	Fixed64Kind  Kind = 6
	Fixed32Kind  Kind = 7
	BoolKind     Kind = 8
	StringKind   Kind = 9
	GroupKind    Kind = 10
	MessageKind  Kind = 11
	BytesKind    Kind = 12
	Uint32Kind   Kind = 13
	EnumKind     Kind = 14
	Sfixed32Kind Kind = 15
	Sfixed64Kind Kind = 16
	Sint32Kind   Kind = 17
	Sint64Kind   Kind = 18
)

// NewSchema is an empty schema to add files to.
func NewSchema() *Schema {
	return &Schema{Messages: map[string]*Message{}, Enums: map[string]*Enum{}, Services: map[string]*Service{}, files: map[string]bool{}}
}

// AddSet adds the files of a FileDescriptorSet, as protoc
// --descriptor_set_out (and buf build -o) write it: a .protoset file.
func (s *Schema) AddSet(b []byte) error {
	fields, err := Fields(b)
	if err != nil {
		return err
	}
	for _, f := range fields {
		if f.Num == 1 && f.Type == Bytes {
			if err := s.AddFile(f.Data); err != nil {
				return err
			}
		}
	}
	return nil
}

// AddFile adds one FileDescriptorProto, as server reflection sends them.
func (s *Schema) AddFile(b []byte) error {
	fields, err := Fields(b)
	if err != nil {
		return fmt.Errorf("a file descriptor: %w", err)
	}
	var name, pkg string
	for _, f := range fields {
		switch f.Num {
		case 1:
			name = string(f.Data)
		case 2:
			pkg = string(f.Data)
		case 3:
			s.deps = append(s.deps, string(f.Data))
		}
	}
	if s.files[name] {
		return nil
	}
	s.files[name] = true
	for _, f := range fields {
		switch f.Num {
		case 4:
			if err := s.addMessage(pkg, f.Data); err != nil {
				return err
			}
		case 5:
			s.addEnum(pkg, f.Data)
		case 6:
			s.addService(pkg, f.Data)
		}
	}
	return nil
}

// Missing is the files named as dependencies that are not in the schema.
func (s *Schema) Missing() []string {
	var out []string
	for _, d := range s.deps {
		if !s.files[d] {
			out = append(out, d)
		}
	}
	return out
}

func join(scope, name string) string {
	if scope == "" {
		return name
	}
	return scope + "." + name
}

func (s *Schema) addMessage(scope string, b []byte) error {
	fields, err := Fields(b)
	if err != nil {
		return err
	}
	m := &Message{byNum: map[int]*FieldDesc{}, byName: map[string]*FieldDesc{}}
	for _, f := range fields {
		if f.Num == 1 {
			m.Name = join(scope, string(f.Data))
		}
	}
	for _, f := range fields {
		switch f.Num {
		case 2:
			fd, err := field(f.Data)
			if err != nil {
				return err
			}
			m.Fields = append(m.Fields, fd)
			m.byNum[fd.Num] = fd
			m.byName[fd.Name] = fd
			m.byName[fd.JSONName] = fd
		case 3:
			if err := s.addMessage(m.Name, f.Data); err != nil {
				return err
			}
		case 4:
			s.addEnum(m.Name, f.Data)
		case 7: // MessageOptions
			opts, _ := Fields(f.Data)
			for _, o := range opts {
				if o.Num == 7 && o.Int != 0 {
					m.MapEntry = true
				}
			}
		}
	}
	s.Messages[m.Name] = m
	return nil
}

func field(b []byte) (*FieldDesc, error) {
	fields, err := Fields(b)
	if err != nil {
		return nil, err
	}
	fd := &FieldDesc{}
	for _, f := range fields {
		switch f.Num {
		case 1:
			fd.Name = string(f.Data)
		case 3:
			fd.Num = int(f.Int)
		case 4:
			fd.Repeated = f.Int == 3
		case 5:
			fd.Kind = Kind(f.Int)
		case 6:
			fd.TypeName = strings.TrimPrefix(string(f.Data), ".")
		case 10:
			fd.JSONName = string(f.Data)
		}
	}
	if fd.JSONName == "" {
		fd.JSONName = camel(fd.Name)
	}
	return fd, nil
}

// camel is the JSON name protoc gives a field: "user_id" is "userId".
func camel(name string) string {
	var b strings.Builder
	up := false
	for _, r := range name {
		if r == '_' {
			up = true
			continue
		}
		if up && r >= 'a' && r <= 'z' {
			r -= 'a' - 'A'
		}
		up = false
		b.WriteRune(r)
	}
	return b.String()
}

func (s *Schema) addEnum(scope string, b []byte) {
	fields, _ := Fields(b)
	e := &Enum{ByNum: map[int32]string{}, ByName: map[string]int32{}}
	first := true
	for _, f := range fields {
		switch f.Num {
		case 1:
			e.Name = join(scope, string(f.Data))
		case 2:
			vf, _ := Fields(f.Data)
			var name string
			var num int32
			for _, v := range vf {
				switch v.Num {
				case 1:
					name = string(v.Data)
				case 2:
					num = int32(v.Int)
				}
			}
			if _, seen := e.ByNum[num]; !seen {
				e.ByNum[num] = name
			}
			e.ByName[name] = num
			if first {
				e.FirstNum, first = num, false
			}
		}
	}
	s.Enums[e.Name] = e
}

func (s *Schema) addService(scope string, b []byte) {
	fields, _ := Fields(b)
	svc := &Service{Methods: map[string]*Method{}}
	for _, f := range fields {
		switch f.Num {
		case 1:
			svc.Name = join(scope, string(f.Data))
		case 2:
			mf, _ := Fields(f.Data)
			m := &Method{}
			for _, v := range mf {
				switch v.Num {
				case 1:
					m.Name = string(v.Data)
				case 2:
					m.Input = strings.TrimPrefix(string(v.Data), ".")
				case 3:
					m.Output = strings.TrimPrefix(string(v.Data), ".")
				case 5:
					m.ClientStream = v.Int != 0
				case 6:
					m.ServerStream = v.Int != 0
				}
			}
			svc.Methods[m.Name] = m
		}
	}
	s.Services[svc.Name] = svc
}

// Method finds "pkg.Service/Method" (a leading slash and "pkg.Service.Method"
// are read too).
func (s *Schema) Method(full string) (*Method, error) {
	full = strings.TrimPrefix(full, "/")
	svc, name, ok := strings.Cut(full, "/")
	if !ok {
		i := strings.LastIndex(full, ".")
		if i < 0 {
			return nil, fmt.Errorf("%q is not a method: write package.Service/Method", full)
		}
		svc, name = full[:i], full[i+1:]
	}
	service, ok := s.Services[svc]
	if !ok {
		return nil, fmt.Errorf("no service %s in the descriptors", svc)
	}
	m, ok := service.Methods[name]
	if !ok {
		return nil, fmt.Errorf("service %s has no method %s", svc, name)
	}
	return m, nil
}
