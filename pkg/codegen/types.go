package codegen

import (
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// --- IR data types (rendered by templates) ---

// Field is one struct field.
type Field struct {
	GoName string
	Type   string // Go type expression, in dto-package terms
	JSON   string // full struct tag body, e.g. `json:"name,omitempty"`
}

// EnumValue is one enum constant.
type EnumValue struct {
	Name    string
	Literal string // rendered literal: `"red"` or `1`
}

// NamedType is a top-level generated type in the dto package.
type NamedType struct {
	Name string
	Doc  string

	// exactly one of the following is populated
	Struct []Field
	Enum   *EnumDef
	Alias  string // for `type Name = Alias`

	// APIError marks a generated error type; when set the template emits the
	// APIError methods (StatusCode/Body/Error).
	APIError    bool
	Status      int
	MessageExpr string // expression returned by Error(), e.g. e.Message or a literal
}

// EnumDef is the enum payload of a NamedType.
type EnumDef struct {
	Base   string // "string" or "int64"
	Values []EnumValue
}

// --- type mapper ---

type mapper struct {
	byName map[string]*NamedType
	order  []string
}

func newMapper() *mapper {
	return &mapper{byName: map[string]*NamedType{}}
}

// named returns the collected types sorted by name for deterministic output.
func (m *mapper) named() []NamedType {
	names := make([]string, len(m.order))
	copy(names, m.order)
	sort.Strings(names)

	out := make([]NamedType, 0, len(names))
	for _, n := range names {
		out = append(out, *m.byName[n])
	}
	return out
}

func (m *mapper) has(name string) bool {
	_, ok := m.byName[name]
	return ok
}

func (m *mapper) put(nt *NamedType) *NamedType {
	if existing, ok := m.byName[nt.Name]; ok {
		return existing
	}
	m.byName[nt.Name] = nt
	m.order = append(m.order, nt.Name)
	return nt
}

// goType maps a schema to a Go type expression (in dto-package terms) and
// registers any named types it needs. hint names synthesized inline types.
func (m *mapper) goType(ref *openapi3.SchemaRef, hint string) string {
	if ref == nil || ref.Value == nil {
		return "any"
	}
	if ref.Ref != "" {
		name := componentName(ref.Ref)
		m.registerNamed(name, ref.Value)
		return name
	}

	s := ref.Value
	if len(s.Enum) > 0 {
		name := pascal(hint)
		m.registerEnum(name, s)
		return name
	}

	switch {
	case s.Type.Is("string"):
		return "string"
	case s.Type.Is("integer"):
		if s.Format == "int32" {
			return "int32"
		}
		return "int64"
	case s.Type.Is("number"):
		if s.Format == "float" {
			return "float32"
		}
		return "float64"
	case s.Type.Is("boolean"):
		return "bool"
	case s.Type.Is("array"):
		return "[]" + m.goType(s.Items, hint+"Item")
	case s.Type.Is("object") || len(s.Properties) > 0:
		name := pascal(hint)
		m.registerStruct(name, s)
		return name
	default:
		return "any"
	}
}

// registerNamed registers a schema referenced by name (component or synthesized)
// as either an enum or a struct.
func (m *mapper) registerNamed(name string, s *openapi3.Schema) {
	if m.has(name) {
		return
	}
	if len(s.Enum) > 0 {
		m.registerEnum(name, s)
		return
	}
	m.registerStruct(name, s)
}

func (m *mapper) registerEnum(name string, s *openapi3.Schema) {
	if m.has(name) {
		return
	}
	base := "string"
	if s.Type.Is("integer") {
		base = "int64"
	}
	def := &EnumDef{Base: base}
	for _, v := range s.Enum {
		lit := `"` + toStr(v) + `"`
		if base == "int64" {
			lit = toStr(v)
		}
		def.Values = append(def.Values, EnumValue{
			Name:    enumValueName(name, v),
			Literal: lit,
		})
	}
	m.put(&NamedType{Name: name, Enum: def})
}

func (m *mapper) registerStruct(name string, s *openapi3.Schema) {
	if m.has(name) {
		return
	}
	// Register first (empty) so self-referential schemas terminate.
	nt := m.put(&NamedType{Name: name})
	nt.Struct = m.fields(name, s)
}

// fields builds struct fields from a schema's properties in sorted order.
func (m *mapper) fields(owner string, s *openapi3.Schema) []Field {
	required := map[string]bool{}
	for _, r := range s.Required {
		required[r] = true
	}

	names := make([]string, 0, len(s.Properties))
	for n := range s.Properties {
		names = append(names, n)
	}
	sort.Strings(names)

	fields := make([]Field, 0, len(names))
	for _, prop := range names {
		propRef := s.Properties[prop]
		goName := pascal(prop)
		base := m.goType(propRef, owner+goName)

		optional := !required[prop]
		nullable := propRef.Value != nil && propRef.Value.Nullable
		if (optional || nullable) && pointerable(base) {
			base = "*" + base
		}

		tag := "json:\"" + prop
		if optional {
			tag += ",omitempty"
		}
		tag += "\""

		fields = append(fields, Field{GoName: goName, Type: base, JSON: tag})
	}
	return fields
}

// pointerable reports whether wrapping the type in a pointer is sensible.
func pointerable(t string) bool {
	return !strings.HasPrefix(t, "[]") && !strings.HasPrefix(t, "map[") && !strings.HasPrefix(t, "*")
}
